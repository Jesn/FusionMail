package service

import (
	"encoding/json"
	"sync"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/sse"
)

// SSE 事件类型常量
const (
	SSEEventSyncStarted   = "sync_started"
	SSEEventSyncProgress  = "sync_progress"
	SSEEventSyncCompleted = "sync_completed"
	SSEEventSyncFailed    = "sync_failed"
	SSEEventSyncCancelled = "sync_cancelled"
)

// ProgressTracker 进度追踪接口
// Requirements: 2.1, 2.2, 2.3, 2.4
type ProgressTracker interface {
	// Start 开始追踪
	Start(accountUID string, estimatedTotal int, isFirstSync bool)

	// Update 更新进度
	Update(processed, newEmails, updated, failed int)

	// IncrementBatch 增加批次计数
	IncrementBatch()

	// SetPhase 设置当前阶段
	SetPhase(phase string)

	// Complete 完成追踪
	Complete()

	// Fail 标记失败
	Fail(err error)

	// Cancel 标记取消
	Cancel()

	// GetProgress 获取当前进度
	GetProgress() *model.SyncProgress
}

// DefaultProgressTracker 默认进度追踪器实现
type DefaultProgressTracker struct {
	mu               sync.RWMutex
	progress         *model.SyncProgress
	progressInterval int       // 进度通知间隔（邮件数）
	lastNotifyCount  int       // 上次通知时的处理数
	lastNotifyTime   time.Time // 上次通知时间
	notifyIntervalMs int64     // 时间间隔（毫秒），默认 5000
}

// NewProgressTracker 创建新的进度追踪器
func NewProgressTracker(progressInterval int) *DefaultProgressTracker {
	if progressInterval <= 0 {
		progressInterval = model.DefaultProgressInterval
	}
	return &DefaultProgressTracker{
		progressInterval: progressInterval,
		notifyIntervalMs: 5000, // 5 秒
	}
}

// Start 开始追踪
// Requirements: 2.1 - 发送 sync_started 事件
func (t *DefaultProgressTracker) Start(accountUID string, estimatedTotal int, isFirstSync bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress = model.NewSyncProgress(accountUID, isFirstSync)
	t.progress.TotalEstimated = estimatedTotal
	t.lastNotifyCount = 0
	t.lastNotifyTime = time.Now()

	// 发送 sync_started SSE 事件
	t.broadcastEvent(SSEEventSyncStarted, t.buildStartedEvent())
}

// Update 更新进度
// Requirements: 2.2 - 每 10 封邮件或每 5 秒发送进度事件
func (t *DefaultProgressTracker) Update(processed, newEmails, updated, failed int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return
	}

	t.progress.UpdateProgress(processed, newEmails, updated, failed)

	// 检查是否需要发送进度通知
	countDiff := processed - t.lastNotifyCount
	timeDiff := time.Since(t.lastNotifyTime).Milliseconds()

	if countDiff >= t.progressInterval || timeDiff >= t.notifyIntervalMs {
		t.broadcastEvent(SSEEventSyncProgress, t.buildProgressEvent())
		t.lastNotifyCount = processed
		t.lastNotifyTime = time.Now()
	}
}

// IncrementBatch 增加批次计数
func (t *DefaultProgressTracker) IncrementBatch() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		t.progress.IncrementBatch()
	}
}

// SetPhase 设置当前阶段
func (t *DefaultProgressTracker) SetPhase(phase string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		t.progress.SetPhase(phase)
	}
}

// Complete 完成追踪
// Requirements: 2.3 - 发送 sync_completed 事件
func (t *DefaultProgressTracker) Complete() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return
	}

	t.progress.MarkCompleted()

	// 发送 sync_completed SSE 事件
	t.broadcastEvent(SSEEventSyncCompleted, t.buildCompletedEvent())
}

// Fail 标记失败
// Requirements: 2.4 - 发送 sync_failed 事件
func (t *DefaultProgressTracker) Fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return
	}

	t.progress.MarkFailed(err)

	// 发送 sync_failed SSE 事件
	t.broadcastEvent(SSEEventSyncFailed, t.buildFailedEvent())
}

// Cancel 标记取消
func (t *DefaultProgressTracker) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return
	}

	t.progress.MarkCancelled()

	// 发送 sync_cancelled SSE 事件
	t.broadcastEvent(SSEEventSyncCancelled, t.buildCancelledEvent())
}

// GetProgress 获取当前进度
func (t *DefaultProgressTracker) GetProgress() *model.SyncProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.progress == nil {
		return nil
	}
	return t.progress.Clone()
}

// broadcastEvent 广播 SSE 事件
func (t *DefaultProgressTracker) broadcastEvent(eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	sse.Broadcast(eventType, string(jsonData))
}

// SyncStartedEvent 同步开始事件结构
type SyncStartedEvent struct {
	Type           string `json:"type"`
	AccountUID     string `json:"account_uid"`
	IsFirstSync    bool   `json:"is_first_sync"`
	EstimatedTotal int    `json:"estimated_total"`
	StartedAt      string `json:"started_at"`
}

// buildStartedEvent 构建同步开始事件
func (t *DefaultProgressTracker) buildStartedEvent() *SyncStartedEvent {
	return &SyncStartedEvent{
		Type:           SSEEventSyncStarted,
		AccountUID:     t.progress.AccountUID,
		IsFirstSync:    t.progress.IsFirstSync,
		EstimatedTotal: t.progress.TotalEstimated,
		StartedAt:      t.progress.StartedAt.Format(time.RFC3339),
	}
}

// SyncProgressEvent 同步进度事件结构
type SyncProgressEvent struct {
	Type           string `json:"type"`
	AccountUID     string `json:"account_uid"`
	Processed      int    `json:"processed"`
	TotalEstimated int    `json:"total_estimated"`
	NewEmails      int    `json:"new_emails"`
	CurrentBatch   int    `json:"current_batch"`
	TotalBatches   int    `json:"total_batches"`
	Percent        int    `json:"percent"`
}

// buildProgressEvent 构建同步进度事件
func (t *DefaultProgressTracker) buildProgressEvent() *SyncProgressEvent {
	return &SyncProgressEvent{
		Type:           SSEEventSyncProgress,
		AccountUID:     t.progress.AccountUID,
		Processed:      t.progress.Processed,
		TotalEstimated: t.progress.TotalEstimated,
		NewEmails:      t.progress.NewEmails,
		CurrentBatch:   t.progress.CurrentBatch,
		TotalBatches:   t.progress.TotalBatches,
		Percent:        t.progress.GetPercent(),
	}
}

// SyncCompletedEvent 同步完成事件结构
type SyncCompletedEvent struct {
	Type          string `json:"type"`
	AccountUID    string `json:"account_uid"`
	TotalSynced   int    `json:"total_synced"`
	NewEmails     int    `json:"new_emails"`
	UpdatedEmails int    `json:"updated_emails"`
	DurationMs    int64  `json:"duration_ms"`
}

// buildCompletedEvent 构建同步完成事件
func (t *DefaultProgressTracker) buildCompletedEvent() *SyncCompletedEvent {
	return &SyncCompletedEvent{
		Type:          SSEEventSyncCompleted,
		AccountUID:    t.progress.AccountUID,
		TotalSynced:   t.progress.Processed,
		NewEmails:     t.progress.NewEmails,
		UpdatedEmails: t.progress.UpdatedEmails,
		DurationMs:    t.progress.GetDurationMs(),
	}
}

// SyncFailedEvent 同步失败事件结构
type SyncFailedEvent struct {
	Type            string               `json:"type"`
	AccountUID      string               `json:"account_uid"`
	Error           string               `json:"error"`
	PartialProgress *PartialProgressInfo `json:"partial_progress"`
}

// PartialProgressInfo 部分进度信息
type PartialProgressInfo struct {
	Processed int `json:"processed"`
	NewEmails int `json:"new_emails"`
}

// buildFailedEvent 构建同步失败事件
func (t *DefaultProgressTracker) buildFailedEvent() *SyncFailedEvent {
	return &SyncFailedEvent{
		Type:       SSEEventSyncFailed,
		AccountUID: t.progress.AccountUID,
		Error:      t.progress.ErrorMessage,
		PartialProgress: &PartialProgressInfo{
			Processed: t.progress.Processed,
			NewEmails: t.progress.NewEmails,
		},
	}
}

// SyncCancelledEvent 同步取消事件结构
type SyncCancelledEvent struct {
	Type            string               `json:"type"`
	AccountUID      string               `json:"account_uid"`
	PartialProgress *PartialProgressInfo `json:"partial_progress"`
}

// buildCancelledEvent 构建同步取消事件
func (t *DefaultProgressTracker) buildCancelledEvent() *SyncCancelledEvent {
	return &SyncCancelledEvent{
		Type:       SSEEventSyncCancelled,
		AccountUID: t.progress.AccountUID,
		PartialProgress: &PartialProgressInfo{
			Processed: t.progress.Processed,
			NewEmails: t.progress.NewEmails,
		},
	}
}
