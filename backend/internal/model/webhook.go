package model

import (
	"time"
)

// Webhook Webhook 配置模型
type Webhook struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:255;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// Webhook 配置
	URL     string `gorm:"type:text;not null" json:"url"`
	Method  string `gorm:"size:10;default:'POST'" json:"method"`
	Headers string `gorm:"type:text" json:"headers"` // 自定义 HTTP 头（JSON）
	Enabled bool   `gorm:"default:true;index" json:"enabled"`

	// 触发条件
	Events  string `gorm:"type:text;not null" json:"events"` // 监听的事件类型（JSON 数组）
	Filters string `gorm:"type:text" json:"filters"`         // 过滤条件（JSON）

	// 重试配置
	RetryEnabled   bool   `gorm:"default:true" json:"retry_enabled"`
	MaxRetries     int    `gorm:"default:3" json:"max_retries"`
	RetryIntervals string `gorm:"type:text;default:'[10, 30, 60]'" json:"retry_intervals"` // 重试间隔（秒）

	// 统计信息
	TotalCalls   int        `gorm:"default:0" json:"total_calls"`
	SuccessCalls int        `gorm:"default:0" json:"success_calls"`
	FailedCalls  int        `gorm:"default:0" json:"failed_calls"`
	LastCalledAt *time.Time `json:"last_called_at"`
	LastStatus   string     `gorm:"size:20" json:"last_status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Webhook) TableName() string {
	return "webhooks"
}

// WebhookLog Webhook 调用日志模型
type WebhookLog struct {
	ID        int64 `gorm:"primaryKey" json:"id"`
	WebhookID int64 `gorm:"not null;index" json:"webhook_id"`

	// 请求信息
	RequestURL     string `gorm:"type:text;not null" json:"request_url"`
	RequestMethod  string `gorm:"size:10" json:"request_method"`
	RequestHeaders string `gorm:"type:text" json:"request_headers"`
	RequestBody    string `gorm:"type:text" json:"request_body"`

	// 响应信息
	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `gorm:"type:text" json:"response_body"`
	ResponseTimeMs int    `json:"response_time_ms"`

	// 结果
	Success      bool   `json:"success"`
	ErrorMessage string `gorm:"type:text" json:"error_message"`
	RetryCount   int    `gorm:"default:0" json:"retry_count"`

	CreatedAt time.Time `gorm:"index:idx_created_at,sort:desc" json:"created_at"`
}

// TableName 指定表名
func (WebhookLog) TableName() string {
	return "webhook_logs"
}
