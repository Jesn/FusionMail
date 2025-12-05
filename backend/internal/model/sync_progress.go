package model

import (
	"encoding/json"
	"time"
)

// 同步状态常量
const (
	SyncStatusStarted   = "started"     // 同步已开始
	SyncStatusProgress  = "in_progress" // 同步进行中
	SyncStatusCompleted = "completed"   // 同步已完成
	SyncStatusFailed    = "failed"      // 同步失败
	SyncStatusCancelled = "cancelled"   // 同步已取消
)

// 同步阶段常量
const (
	SyncPhaseFetching   = "fetching"   // 正在拉取邮件
	SyncPhaseProcessing = "processing" // 正在处理邮件
	SyncPhaseFinalizing = "finalizing" // 正在完成同步
)

// SyncProgress 同步进度结构
// 用于追踪和报告同步操作的实时进度
// Requirements: 2.1, 2.2, 2.3, 2.4
type SyncProgress struct {
	AccountUID     string    `json:"account_uid"`             // 账户 UID
	Status         string    `json:"status"`                  // 同步状态
	Phase          string    `json:"phase"`                   // 当前阶段
	TotalEstimated int       `json:"total_estimated"`         // 预估总数
	Processed      int       `json:"processed"`               // 已处理数
	NewEmails      int       `json:"new_emails"`              // 新邮件数
	UpdatedEmails  int       `json:"updated_emails"`          // 更新邮件数
	FailedEmails   int       `json:"failed_emails"`           // 失败邮件数
	CurrentBatch   int       `json:"current_batch"`           // 当前批次
	TotalBatches   int       `json:"total_batches"`           // 总批次数
	IsFirstSync    bool      `json:"is_first_sync"`           // 是否首次同步
	StartedAt      time.Time `json:"started_at"`              // 开始时间
	LastUpdateAt   time.Time `json:"last_update_at"`          // 最后更新时间
	ErrorMessage   string    `json:"error_message,omitempty"` // 错误信息
}

// NewSyncProgress 创建新的同步进度实例
func NewSyncProgress(accountUID string, isFirstSync bool) *SyncProgress {
	now := time.Now()
	return &SyncProgress{
		AccountUID:   accountUID,
		Status:       SyncStatusStarted,
		Phase:        SyncPhaseFetching,
		IsFirstSync:  isFirstSync,
		StartedAt:    now,
		LastUpdateAt: now,
	}
}

// SetEstimatedTotal 设置预估总数和批次信息
func (p *SyncProgress) SetEstimatedTotal(total int, batchSize int) {
	p.TotalEstimated = total
	if batchSize > 0 && total > 0 {
		p.TotalBatches = (total + batchSize - 1) / batchSize // 向上取整
	}
	p.LastUpdateAt = time.Now()
}

// UpdateProgress 更新处理进度
func (p *SyncProgress) UpdateProgress(processed, newEmails, updated, failed int) {
	p.Processed = processed
	p.NewEmails = newEmails
	p.UpdatedEmails = updated
	p.FailedEmails = failed
	p.Status = SyncStatusProgress
	p.LastUpdateAt = time.Now()
}

// IncrementBatch 增加当前批次
func (p *SyncProgress) IncrementBatch() {
	p.CurrentBatch++
	p.LastUpdateAt = time.Now()
}

// SetPhase 设置当前阶段
func (p *SyncProgress) SetPhase(phase string) {
	p.Phase = phase
	p.LastUpdateAt = time.Now()
}

// MarkCompleted 标记同步完成
func (p *SyncProgress) MarkCompleted() {
	p.Status = SyncStatusCompleted
	p.Phase = SyncPhaseFinalizing
	p.LastUpdateAt = time.Now()
}

// MarkFailed 标记同步失败
func (p *SyncProgress) MarkFailed(err error) {
	p.Status = SyncStatusFailed
	if err != nil {
		p.ErrorMessage = err.Error()
	}
	p.LastUpdateAt = time.Now()
}

// MarkCancelled 标记同步取消
func (p *SyncProgress) MarkCancelled() {
	p.Status = SyncStatusCancelled
	p.LastUpdateAt = time.Now()
}

// GetPercent 获取完成百分比
func (p *SyncProgress) GetPercent() int {
	if p.TotalEstimated <= 0 {
		return 0
	}
	percent := (p.Processed * 100) / p.TotalEstimated
	if percent > 100 {
		percent = 100
	}
	return percent
}

// GetDurationMs 获取已用时间（毫秒）
func (p *SyncProgress) GetDurationMs() int64 {
	return time.Since(p.StartedAt).Milliseconds()
}

// IsTerminal 判断是否为终止状态
func (p *SyncProgress) IsTerminal() bool {
	return p.Status == SyncStatusCompleted ||
		p.Status == SyncStatusFailed ||
		p.Status == SyncStatusCancelled
}

// ToJSON 序列化为 JSON 字符串
func (p *SyncProgress) ToJSON() string {
	data, err := json.Marshal(p)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// FromJSON 从 JSON 字符串反序列化
func (p *SyncProgress) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), p)
}

// Clone 创建进度的副本
func (p *SyncProgress) Clone() *SyncProgress {
	return &SyncProgress{
		AccountUID:     p.AccountUID,
		Status:         p.Status,
		Phase:          p.Phase,
		TotalEstimated: p.TotalEstimated,
		Processed:      p.Processed,
		NewEmails:      p.NewEmails,
		UpdatedEmails:  p.UpdatedEmails,
		FailedEmails:   p.FailedEmails,
		CurrentBatch:   p.CurrentBatch,
		TotalBatches:   p.TotalBatches,
		IsFirstSync:    p.IsFirstSync,
		StartedAt:      p.StartedAt,
		LastUpdateAt:   p.LastUpdateAt,
		ErrorMessage:   p.ErrorMessage,
	}
}
