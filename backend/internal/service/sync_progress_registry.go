package service

import (
	"fmt"
	"sync"

	"fusionmail/internal/model"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/synclock"
)

// SyncProgressRegistry 持有活跃同步的共享可变状态：锁信息与进度追踪器。
//
// 为什么从 syncService 里拆出来：activeSyncs / activeTrackers / syncMu 原本是
// syncService 的字段，被 executor（写：注册/注销）和控制面（读：取消/查进度、
// 清理卡住任务）双向访问，是拆分调度器与执行器时最核心的耦合点。
// 把这组状态收敛到独立小类型后，Scheduler 与 Executor 都只依赖本类型，
// 互不依赖对方。
type SyncProgressRegistry struct {
	mu             sync.RWMutex                  // 保护 activeSyncs 和 activeTrackers
	activeSyncs    map[string]*synclock.LockInfo // 活跃同步的锁信息
	activeTrackers map[string]ProgressTracker    // 活跃同步的进度追踪器
	logger         *logger.Logger
}

// NewSyncProgressRegistry 创建同步进度注册表。
func NewSyncProgressRegistry() *SyncProgressRegistry {
	return &SyncProgressRegistry{
		activeSyncs:    make(map[string]*synclock.LockInfo),
		activeTrackers: make(map[string]ProgressTracker),
		logger:         logger.NewWithModule("Sync"),
	}
}

// Register 注册活跃同步的锁信息（用于取消同步）。
func (r *SyncProgressRegistry) Register(accountUID string, lockInfo *synclock.LockInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activeSyncs[accountUID] = lockInfo
}

// TryRegister 尝试注册活跃同步的锁信息。
// 若该账户已有活跃同步则返回 false，调用方据此实现内存锁模式下的互斥。
// 检查与注册在同一把写锁内完成，避免"先查后写"的竞态。
func (r *SyncProgressRegistry) TryRegister(accountUID string, lockInfo *synclock.LockInfo) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.activeSyncs[accountUID]; exists {
		return false
	}
	r.activeSyncs[accountUID] = lockInfo
	return true
}

// RegisterTracker 注册活跃同步的进度追踪器（用于查询同步进度）。
func (r *SyncProgressRegistry) RegisterTracker(accountUID string, tracker ProgressTracker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activeTrackers[accountUID] = tracker
}

// EvictStaleLock 清理与 Redis 状态不一致的过期内存锁：
// 当 accountUID 存在内存锁且 isRedisLocked 返回 false（Redis 锁已过期）时，
// 取消并注销该同步。检查与注销在同一把写锁内完成，避免与并发注册产生竞态。
func (r *SyncProgressRegistry) EvictStaleLock(accountUID string, isRedisLocked func() bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existingLock, exists := r.activeSyncs[accountUID]
	if !exists {
		return
	}
	if isRedisLocked() {
		return
	}

	r.logger.Warn("发现过期内存锁，正在清理: account=%s", accountUID)
	if existingLock.CancelFunc != nil {
		existingLock.CancelFunc()
	}
	delete(r.activeSyncs, accountUID)
	delete(r.activeTrackers, accountUID)
}

// Remove 注销活跃同步（锁信息 + 进度追踪器），同步结束时调用。
func (r *SyncProgressRegistry) Remove(accountUID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.activeSyncs, accountUID)
	delete(r.activeTrackers, accountUID)
}

// CancelAndRemove 取消并注销活跃同步，用于清理卡住的同步任务：
// 触发取消函数并移除锁信息与进度追踪器；不存在时为空操作。
func (r *SyncProgressRegistry) CancelAndRemove(accountUID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if lockInfo, exists := r.activeSyncs[accountUID]; exists {
		if lockInfo.CancelFunc != nil {
			lockInfo.CancelFunc()
		}
		delete(r.activeSyncs, accountUID)
		delete(r.activeTrackers, accountUID)
	}
}

// Cancel 取消指定账户的同步（原 syncService.CancelSync 的实现）。
func (r *SyncProgressRegistry) Cancel(accountUID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lockInfo, exists := r.activeSyncs[accountUID]
	if !exists {
		return fmt.Errorf("no active sync for account: %s", accountUID)
	}

	// 调用取消函数
	if lockInfo.CancelFunc != nil {
		lockInfo.CancelFunc()
	}
	r.logger.Info("同步已取消: account=%s", accountUID)

	return nil
}

// GetProgress 获取指定账户的同步进度（原 syncService.GetSyncProgress 的实现）。
func (r *SyncProgressRegistry) GetProgress(accountUID string) *model.SyncProgress {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tracker, exists := r.activeTrackers[accountUID]
	if !exists {
		return nil
	}

	return tracker.GetProgress()
}
