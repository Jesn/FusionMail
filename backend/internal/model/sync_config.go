package model

import (
	"errors"
	"fmt"
)

// SyncConfig 同步配置结构
// 用于控制邮件同步行为，包括首次同步天数、批次大小等参数
type SyncConfig struct {
	FirstSyncDays    int `json:"first_sync_days"`     // 首次同步天数，0 表示全量，默认 7
	BatchSize        int `json:"batch_size"`          // 每批处理数量，默认 100
	MaxEmailsPerSync int `json:"max_emails_per_sync"` // 单次同步最大邮件数，默认 5000
	ProgressInterval int `json:"progress_interval"`   // 进度通知间隔（邮件数），默认 10
	RetryCount       int `json:"retry_count"`         // 失败重试次数，默认 3
	RetryBackoffMs   int `json:"retry_backoff_ms"`    // 重试退避基数（毫秒），默认 1000
}

// 同步配置默认值常量
const (
	DefaultFirstSyncDays    = 7
	DefaultBatchSize        = 100
	DefaultMaxEmailsPerSync = 5000
	DefaultProgressInterval = 10
	DefaultRetryCount       = 3
	DefaultRetryBackoffMs   = 1000
)

// 同步配置验证边界常量
const (
	MinFirstSyncDays    = 0   // 0 表示全量同步
	MaxFirstSyncDays    = 365 // 最大 365 天
	MinBatchSize        = 10
	MaxBatchSize        = 500
	MinMaxEmailsPerSync = 100
	MaxMaxEmailsPerSync = 50000
	MinProgressInterval = 1
	MaxProgressInterval = 100
	MinRetryCount       = 0
	MaxRetryCount       = 10
	MinRetryBackoffMs   = 100
	MaxRetryBackoffMs   = 60000
)

// 同步配置验证错误
var (
	ErrInvalidFirstSyncDays    = errors.New("first_sync_days must be between 0 and 365")
	ErrInvalidBatchSize        = errors.New("batch_size must be between 10 and 500")
	ErrInvalidMaxEmailsPerSync = errors.New("max_emails_per_sync must be between 100 and 50000")
	ErrInvalidProgressInterval = errors.New("progress_interval must be between 1 and 100")
	ErrInvalidRetryCount       = errors.New("retry_count must be between 0 and 10")
	ErrInvalidRetryBackoffMs   = errors.New("retry_backoff_ms must be between 100 and 60000")
)

// DefaultSyncConfig 返回默认同步配置
// 满足 Requirements 1.1, 6.3：提供默认配置值
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		FirstSyncDays:    DefaultFirstSyncDays,
		BatchSize:        DefaultBatchSize,
		MaxEmailsPerSync: DefaultMaxEmailsPerSync,
		ProgressInterval: DefaultProgressInterval,
		RetryCount:       DefaultRetryCount,
		RetryBackoffMs:   DefaultRetryBackoffMs,
	}
}

// Validate 验证同步配置的有效性
// 返回第一个遇到的验证错误，如果配置有效则返回 nil
func (c *SyncConfig) Validate() error {
	if c.FirstSyncDays < MinFirstSyncDays || c.FirstSyncDays > MaxFirstSyncDays {
		return fmt.Errorf("%w: got %d", ErrInvalidFirstSyncDays, c.FirstSyncDays)
	}

	if c.BatchSize < MinBatchSize || c.BatchSize > MaxBatchSize {
		return fmt.Errorf("%w: got %d", ErrInvalidBatchSize, c.BatchSize)
	}

	if c.MaxEmailsPerSync < MinMaxEmailsPerSync || c.MaxEmailsPerSync > MaxMaxEmailsPerSync {
		return fmt.Errorf("%w: got %d", ErrInvalidMaxEmailsPerSync, c.MaxEmailsPerSync)
	}

	if c.ProgressInterval < MinProgressInterval || c.ProgressInterval > MaxProgressInterval {
		return fmt.Errorf("%w: got %d", ErrInvalidProgressInterval, c.ProgressInterval)
	}

	if c.RetryCount < MinRetryCount || c.RetryCount > MaxRetryCount {
		return fmt.Errorf("%w: got %d", ErrInvalidRetryCount, c.RetryCount)
	}

	if c.RetryBackoffMs < MinRetryBackoffMs || c.RetryBackoffMs > MaxRetryBackoffMs {
		return fmt.Errorf("%w: got %d", ErrInvalidRetryBackoffMs, c.RetryBackoffMs)
	}

	return nil
}

// MergeWithDefaults 将当前配置与默认配置合并
// 对于零值字段，使用默认值填充
func (c *SyncConfig) MergeWithDefaults() *SyncConfig {
	defaults := DefaultSyncConfig()

	result := &SyncConfig{
		FirstSyncDays:    c.FirstSyncDays,
		BatchSize:        c.BatchSize,
		MaxEmailsPerSync: c.MaxEmailsPerSync,
		ProgressInterval: c.ProgressInterval,
		RetryCount:       c.RetryCount,
		RetryBackoffMs:   c.RetryBackoffMs,
	}

	// FirstSyncDays 为 0 是有效值（表示全量同步），不需要特殊处理

	if result.BatchSize == 0 {
		result.BatchSize = defaults.BatchSize
	}

	if result.MaxEmailsPerSync == 0 {
		result.MaxEmailsPerSync = defaults.MaxEmailsPerSync
	}

	if result.ProgressInterval == 0 {
		result.ProgressInterval = defaults.ProgressInterval
	}

	// RetryCount 为 0 是有效值（表示不重试），不需要特殊处理

	if result.RetryBackoffMs == 0 {
		result.RetryBackoffMs = defaults.RetryBackoffMs
	}

	return result
}

// IsFullSync 判断是否为全量同步
func (c *SyncConfig) IsFullSync() bool {
	return c.FirstSyncDays == 0
}
