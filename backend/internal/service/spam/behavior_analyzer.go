package spam

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// BehaviorAnalyzer 发信行为分析器
type BehaviorAnalyzer struct {
	cache *redis.Client
}

// BehaviorResult 行为分析结果
type BehaviorResult struct {
	FrequencyScore      int  // 发信频率评分
	SizeScore           int  // 邮件大小评分
	AttachmentScore     int  // 附件类型评分
	TotalScore          int  // 总评分
	IsFrequencyAbnormal bool // 发信频率是否异常
	IsLargeEmail        bool // 是否为大邮件
	HasDangerousAttach  bool // 是否有危险附件
}

// EmailInfo 邮件信息
type EmailInfo struct {
	From            string   // 发件人
	Size            int64    // 邮件大小（字节）
	AttachmentTypes []string // 附件类型列表
}

// NewBehaviorAnalyzer 创建发信行为分析器实例
func NewBehaviorAnalyzer(cache *redis.Client) *BehaviorAnalyzer {
	return &BehaviorAnalyzer{
		cache: cache,
	}
}

// Analyze 分析邮件发信行为
func (b *BehaviorAnalyzer) Analyze(ctx context.Context, email *EmailInfo) (*BehaviorResult, error) {
	result := &BehaviorResult{
		FrequencyScore:      0,
		SizeScore:           0,
		AttachmentScore:     0,
		TotalScore:          0,
		IsFrequencyAbnormal: false,
		IsLargeEmail:        false,
		HasDangerousAttach:  false,
	}

	// 1. 检查发信频率
	frequencyScore, isAbnormal, err := b.checkFrequency(ctx, email.From)
	if err != nil {
		// 频率检查失败，不影响主流程
		frequencyScore = 0
	}
	result.FrequencyScore = frequencyScore
	result.IsFrequencyAbnormal = isAbnormal

	// 2. 检查邮件大小和附件
	sizeScore, isLarge := b.checkEmailSize(email.Size)
	result.SizeScore = sizeScore
	result.IsLargeEmail = isLarge

	// 3. 检查附件类型
	attachmentScore, hasDangerous := b.checkAttachments(email.AttachmentTypes, email.Size)
	result.AttachmentScore = attachmentScore
	result.HasDangerousAttach = hasDangerous

	// 4. 计算总评分
	result.TotalScore = result.FrequencyScore + result.SizeScore + result.AttachmentScore

	return result, nil
}

// checkFrequency 检查发信频率
// 5 分钟内超过 20 封邮件视为异常
func (b *BehaviorAnalyzer) checkFrequency(ctx context.Context, sender string) (int, bool, error) {
	// 使用 Redis 的有序集合记录发信时间
	key := fmt.Sprintf("sender:frequency:%s", sender)
	now := time.Now().Unix()
	fiveMinutesAgo := now - 300 // 5 分钟 = 300 秒

	// 1. 清理 5 分钟前的记录
	b.cache.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", fiveMinutesAgo))

	// 2. 添加当前时间戳
	b.cache.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	})

	// 3. 设置过期时间（10 分钟）
	b.cache.Expire(ctx, key, 10*time.Minute)

	// 4. 统计 5 分钟内的邮件数量
	count, err := b.cache.ZCount(ctx, key, fmt.Sprintf("%d", fiveMinutesAgo), fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return 0, false, err
	}

	// 5. 判断是否异常
	if count > 20 {
		// 超过 20 封，评分增加 20 分
		return 20, true, nil
	}

	return 0, false, nil
}

// checkEmailSize 检查邮件大小
// 超过 10MB 且包含可执行附件，评分增加 30 分
func (b *BehaviorAnalyzer) checkEmailSize(size int64) (int, bool) {
	// 10MB = 10 * 1024 * 1024 字节
	const tenMB = 10 * 1024 * 1024

	if size > tenMB {
		return 0, true // 返回 0 分，但标记为大邮件（需要结合附件类型判断）
	}

	return 0, false
}

// checkAttachments 检查附件类型
// 包含可执行附件，评分增加 25-30 分
func (b *BehaviorAnalyzer) checkAttachments(attachmentTypes []string, emailSize int64) (int, bool) {
	if len(attachmentTypes) == 0 {
		return 0, false
	}

	// 危险的附件类型
	dangerousTypes := map[string]bool{
		".exe": true,
		".bat": true,
		".cmd": true,
		".com": true,
		".scr": true,
		".pif": true,
		".vbs": true,
		".js":  true,
		".jar": true,
		".zip": true, // 压缩包可能包含恶意文件
		".rar": true,
		".7z":  true,
		".iso": true,
		".msi": true,
		".dll": true,
		".sys": true,
		".app": true, // macOS 应用
		".dmg": true, // macOS 磁盘镜像
		".pkg": true, // macOS 安装包
		".deb": true, // Linux 包
		".rpm": true, // Linux 包
		".sh":  true, // Shell 脚本
		".ps1": true, // PowerShell 脚本
		".reg": true, // 注册表文件
	}

	hasDangerous := false
	for _, attachType := range attachmentTypes {
		if dangerousTypes[attachType] {
			hasDangerous = true
			break
		}
	}

	if !hasDangerous {
		return 0, false
	}

	// 有危险附件，基础评分 25 分
	score := 25

	// 如果邮件大小超过 10MB，额外增加 5 分
	const tenMB = 10 * 1024 * 1024
	if emailSize > tenMB {
		score = 30
	}

	return score, true
}

// GetSenderFrequency 获取发件人的发信频率（用于调试和监控）
func (b *BehaviorAnalyzer) GetSenderFrequency(ctx context.Context, sender string) (int64, error) {
	key := fmt.Sprintf("sender:frequency:%s", sender)
	now := time.Now().Unix()
	fiveMinutesAgo := now - 300

	count, err := b.cache.ZCount(ctx, key, fmt.Sprintf("%d", fiveMinutesAgo), fmt.Sprintf("%d", now)).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ResetSenderFrequency 重置发件人的发信频率（用于测试或管理）
func (b *BehaviorAnalyzer) ResetSenderFrequency(ctx context.Context, sender string) error {
	key := fmt.Sprintf("sender:frequency:%s", sender)
	return b.cache.Del(ctx, key).Err()
}
