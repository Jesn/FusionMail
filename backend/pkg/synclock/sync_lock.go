package synclock

import (
	"context"
	"fmt"
	"time"

	"fusionmail/pkg/logger"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// 模块日志记录器
var lockLog = logger.NewWithModule("SyncLock")

// 同步锁配置常量
const (
	// DefaultLockTTL 默认锁过期时间（5分钟）
	// 如果同步任务超过这个时间没有续期，锁会自动释放
	DefaultLockTTL = 5 * time.Minute

	// DefaultSyncTimeout 默认同步超时时间（10分钟）
	// 单次同步任务的最大执行时间
	DefaultSyncTimeout = 10 * time.Minute

	// LockRenewalInterval 锁续期间隔（锁TTL的1/3）
	LockRenewalInterval = DefaultLockTTL / 3

	// LockKeyPrefix Redis 锁键前缀
	LockKeyPrefix = "sync:lock:"
)

// SyncLock 同步锁管理器
// 使用 Redis 实现分布式锁，支持自动续期和超时释放
type SyncLock struct {
	redisClient *redis.Client
	lockTTL     time.Duration
	syncTimeout time.Duration
}

// LockInfo 锁信息
type LockInfo struct {
	AccountUID string
	LockID     string             // 唯一锁标识，用于安全释放
	AcquiredAt time.Time          // 获取锁的时间
	CancelFunc context.CancelFunc // 取消函数
	StopRenew  chan struct{}      // 停止续期信号
}

// NewSyncLock 创建同步锁管理器
func NewSyncLock(redisClient *redis.Client) *SyncLock {
	return &SyncLock{
		redisClient: redisClient,
		lockTTL:     DefaultLockTTL,
		syncTimeout: DefaultSyncTimeout,
	}
}

// NewSyncLockWithConfig 创建带自定义配置的同步锁管理器
func NewSyncLockWithConfig(redisClient *redis.Client, lockTTL, syncTimeout time.Duration) *SyncLock {
	return &SyncLock{
		redisClient: redisClient,
		lockTTL:     lockTTL,
		syncTimeout: syncTimeout,
	}
}

// AcquireLock 获取同步锁
// 返回锁信息和带超时的 context
// 如果锁已被占用，返回错误
func (s *SyncLock) AcquireLock(ctx context.Context, accountUID string) (*LockInfo, context.Context, error) {
	lockKey := LockKeyPrefix + accountUID
	lockID := uuid.New().String()

	// 尝试获取锁（使用 SETNX）
	success, err := s.redisClient.SetNX(ctx, lockKey, lockID, s.lockTTL).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !success {
		// 锁已被占用，检查锁是否过期（防止死锁）
		ttl, err := s.redisClient.TTL(ctx, lockKey).Result()
		if err != nil {
			return nil, nil, fmt.Errorf("sync already in progress for account: %s", accountUID)
		}

		// 如果锁没有 TTL（-1 表示永不过期），说明是旧版本的锁，强制删除
		if ttl == -1 {
			lockLog.Warn("发现无 TTL 的锁，强制释放: account=%s", accountUID)
			s.redisClient.Del(ctx, lockKey)
			// 重试获取锁
			success, err = s.redisClient.SetNX(ctx, lockKey, lockID, s.lockTTL).Result()
			if err != nil || !success {
				return nil, nil, fmt.Errorf("sync already in progress for account: %s", accountUID)
			}
		} else {
			return nil, nil, fmt.Errorf("sync already in progress for account: %s (TTL: %v)", accountUID, ttl)
		}
	}

	// 创建带超时的 context
	syncCtx, cancelFunc := context.WithTimeout(ctx, s.syncTimeout)

	// 创建锁信息
	lockInfo := &LockInfo{
		AccountUID: accountUID,
		LockID:     lockID,
		AcquiredAt: time.Now(),
		CancelFunc: cancelFunc,
		StopRenew:  make(chan struct{}),
	}

	// 启动锁续期协程
	go s.renewLock(syncCtx, lockInfo)

	lockLog.Debug("获取同步锁成功: account=%s, lockID=%s, TTL=%v, timeout=%v",
		accountUID, lockID, s.lockTTL, s.syncTimeout)

	return lockInfo, syncCtx, nil
}

// ReleaseLock 释放同步锁
// 只有持有正确 lockID 的调用者才能释放锁
func (s *SyncLock) ReleaseLock(ctx context.Context, lockInfo *LockInfo) error {
	if lockInfo == nil {
		return nil
	}

	// 停止续期协程
	close(lockInfo.StopRenew)

	// 取消 context
	if lockInfo.CancelFunc != nil {
		lockInfo.CancelFunc()
	}

	lockKey := LockKeyPrefix + lockInfo.AccountUID

	// 使用 Lua 脚本确保只删除自己的锁（防止误删其他进程的锁）
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, s.redisClient, []string{lockKey}, lockInfo.LockID).Int()
	if err != nil {
		lockLog.Warn("释放锁失败: account=%s, err=%v", lockInfo.AccountUID, err)
		return err
	}

	if result == 1 {
		duration := time.Since(lockInfo.AcquiredAt)
		lockLog.Debug("释放同步锁成功: account=%s, duration=%v", lockInfo.AccountUID, duration)
	} else {
		lockLog.Warn("锁已被释放或属于其他进程: account=%s", lockInfo.AccountUID)
	}

	return nil
}

// renewLock 自动续期锁
// 在锁过期前定期续期，防止长时间同步任务被误判为超时
func (s *SyncLock) renewLock(ctx context.Context, lockInfo *LockInfo) {
	ticker := time.NewTicker(LockRenewalInterval)
	defer ticker.Stop()

	lockKey := LockKeyPrefix + lockInfo.AccountUID

	for {
		select {
		case <-ctx.Done():
			// context 被取消，停止续期
			return
		case <-lockInfo.StopRenew:
			// 收到停止信号
			return
		case <-ticker.C:
			// 续期锁（只有当锁仍然属于我们时才续期）
			script := redis.NewScript(`
				if redis.call("get", KEYS[1]) == ARGV[1] then
					return redis.call("pexpire", KEYS[1], ARGV[2])
				else
					return 0
				end
			`)

			result, err := script.Run(ctx, s.redisClient, []string{lockKey}, lockInfo.LockID, int(s.lockTTL.Milliseconds())).Int()
			if err != nil {
				lockLog.Warn("续期锁失败: account=%s, err=%v", lockInfo.AccountUID, err)
				return
			}

			if result == 0 {
				lockLog.Warn("锁已丢失，停止续期: account=%s", lockInfo.AccountUID)
				return
			}

			lockLog.Debug("续期锁成功: account=%s, TTL=%v", lockInfo.AccountUID, s.lockTTL)
		}
	}
}

// IsLocked 检查账户是否被锁定
func (s *SyncLock) IsLocked(ctx context.Context, accountUID string) (bool, error) {
	lockKey := LockKeyPrefix + accountUID
	exists, err := s.redisClient.Exists(ctx, lockKey).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ForceReleaseLock 强制释放锁（仅用于管理/调试）
func (s *SyncLock) ForceReleaseLock(ctx context.Context, accountUID string) error {
	lockKey := LockKeyPrefix + accountUID
	_, err := s.redisClient.Del(ctx, lockKey).Result()
	if err != nil {
		return err
	}
	lockLog.Warn("强制释放锁: account=%s", accountUID)
	return nil
}

// GetLockTTL 获取锁的剩余 TTL
func (s *SyncLock) GetLockTTL(ctx context.Context, accountUID string) (time.Duration, error) {
	lockKey := LockKeyPrefix + accountUID
	return s.redisClient.TTL(ctx, lockKey).Result()
}
