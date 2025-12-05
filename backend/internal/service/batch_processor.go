package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/pkg/logger"
)

// BatchProcessor 分批处理接口
// Requirements: 3.1, 3.2
type BatchProcessor interface {
	// ProcessBatch 处理一批邮件
	ProcessBatch(ctx context.Context, emails []*adapter.Email) (*BatchResult, error)

	// GetProgress 获取当前进度
	GetProgress() *model.SyncProgress

	// Cancel 取消处理
	Cancel() error

	// IsCancelled 检查是否已取消
	IsCancelled() bool
}

// BatchResult 批次处理结果
type BatchResult struct {
	Processed int     // 已处理数
	New       int     // 新邮件数
	Updated   int     // 更新邮件数
	Failed    int     // 失败数
	Errors    []error // 错误列表
}

// NewBatchResult 创建新的批次结果
func NewBatchResult() *BatchResult {
	return &BatchResult{
		Errors: make([]error, 0),
	}
}

// Merge 合并另一个批次结果
func (r *BatchResult) Merge(other *BatchResult) {
	if other == nil {
		return
	}
	r.Processed += other.Processed
	r.New += other.New
	r.Updated += other.Updated
	r.Failed += other.Failed
	r.Errors = append(r.Errors, other.Errors...)
}

// EmailSaver 邮件保存接口（由 EmailService 实现）
type EmailSaver interface {
	SaveEmail(ctx context.Context, accountUID string, email *adapter.Email) (isNew bool, err error)
}

// ProgressPersister 进度持久化接口
type ProgressPersister interface {
	SaveProgress(ctx context.Context, accountUID string, cursor string, progress *model.SyncProgress) error
}

// DefaultBatchProcessor 默认分批处理器实现
type DefaultBatchProcessor struct {
	mu             sync.RWMutex
	accountUID     string
	emailSaver     EmailSaver
	persister      ProgressPersister
	tracker        ProgressTracker
	config         *model.SyncConfig
	cancelled      bool
	cancelChan     chan struct{}
	totalProcessed int
	totalNew       int
	totalUpdated   int
	totalFailed    int
	currentCursor  string
}

// NewBatchProcessor 创建新的分批处理器
func NewBatchProcessor(
	accountUID string,
	emailSaver EmailSaver,
	persister ProgressPersister,
	tracker ProgressTracker,
	config *model.SyncConfig,
) *DefaultBatchProcessor {
	if config == nil {
		config = model.DefaultSyncConfig()
	}
	return &DefaultBatchProcessor{
		accountUID: accountUID,
		emailSaver: emailSaver,
		persister:  persister,
		tracker:    tracker,
		config:     config,
		cancelChan: make(chan struct{}),
	}
}

// ProcessBatch 处理一批邮件
// Requirements: 3.1 - 分批处理邮件
func (p *DefaultBatchProcessor) ProcessBatch(ctx context.Context, emails []*adapter.Email) (*BatchResult, error) {
	result := NewBatchResult()

	if len(emails) == 0 {
		return result, nil
	}

	// 检查是否已取消
	if p.IsCancelled() {
		return result, errors.New("batch processing cancelled")
	}

	for _, email := range emails {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-p.cancelChan:
			return result, errors.New("batch processing cancelled")
		default:
		}

		// 保存邮件
		isNew, err := p.emailSaver.SaveEmail(ctx, p.accountUID, email)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
			logger.Error("保存邮件失败", "provider_id", email.ProviderID, "error", err)
			continue
		}

		result.Processed++
		if isNew {
			result.New++
		} else {
			result.Updated++
		}

		// 更新总计数
		p.mu.Lock()
		p.totalProcessed++
		if isNew {
			p.totalNew++
		} else {
			p.totalUpdated++
		}
		p.mu.Unlock()

		// 更新进度追踪器
		if p.tracker != nil {
			p.tracker.Update(p.totalProcessed, p.totalNew, p.totalUpdated, p.totalFailed)
		}
	}

	// 增加批次计数
	if p.tracker != nil {
		p.tracker.IncrementBatch()
	}

	return result, nil
}

// GetProgress 获取当前进度
func (p *DefaultBatchProcessor) GetProgress() *model.SyncProgress {
	if p.tracker != nil {
		return p.tracker.GetProgress()
	}
	return nil
}

// Cancel 取消处理
func (p *DefaultBatchProcessor) Cancel() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancelled {
		return nil
	}

	p.cancelled = true
	close(p.cancelChan)

	if p.tracker != nil {
		p.tracker.Cancel()
	}

	return nil
}

// IsCancelled 检查是否已取消
func (p *DefaultBatchProcessor) IsCancelled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cancelled
}

// SetCursor 设置当前游标
func (p *DefaultBatchProcessor) SetCursor(cursor string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentCursor = cursor
}

// GetCursor 获取当前游标
func (p *DefaultBatchProcessor) GetCursor() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentCursor
}

// PersistProgress 持久化进度
// Requirements: 3.2 - 每批处理后保存进度
func (p *DefaultBatchProcessor) PersistProgress(ctx context.Context) error {
	if p.persister == nil {
		return nil
	}

	progress := p.GetProgress()
	cursor := p.GetCursor()

	return p.persister.SaveProgress(ctx, p.accountUID, cursor, progress)
}

// GetTotalStats 获取总统计信息
func (p *DefaultBatchProcessor) GetTotalStats() (processed, newEmails, updated, failed int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.totalProcessed, p.totalNew, p.totalUpdated, p.totalFailed
}

// RetryWithBackoff 带退避的重试
// Requirements: 3.4 - 失败重试机制
func RetryWithBackoff(ctx context.Context, fn func() error, maxRetries int, baseDelayMs int) error {
	if maxRetries <= 0 {
		maxRetries = model.DefaultRetryCount
	}
	if baseDelayMs <= 0 {
		baseDelayMs = model.DefaultRetryBackoffMs
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			logger.Warn("操作失败，准备重试",
				"attempt", i+1,
				"max_retries", maxRetries,
				"error", err,
			)

			// 计算退避延迟：baseDelay * 2^i
			delay := time.Duration(baseDelayMs*(1<<i)) * time.Millisecond

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		return nil
	}
	return lastErr
}

// BatchEmailsIntoBatches 将邮件列表分割成批次
func BatchEmailsIntoBatches(emails []*adapter.Email, batchSize int) [][]*adapter.Email {
	if batchSize <= 0 {
		batchSize = model.DefaultBatchSize
	}

	var batches [][]*adapter.Email
	for i := 0; i < len(emails); i += batchSize {
		end := i + batchSize
		if end > len(emails) {
			end = len(emails)
		}
		batches = append(batches, emails[i:end])
	}
	return batches
}

// CalculateTotalBatches 计算总批次数
func CalculateTotalBatches(totalEmails, batchSize int) int {
	if batchSize <= 0 {
		batchSize = model.DefaultBatchSize
	}
	if totalEmails <= 0 {
		return 0
	}
	return (totalEmails + batchSize - 1) / batchSize
}
