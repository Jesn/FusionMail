package model

import (
	"time"
)

// Webhook Webhook 配置模型
type Webhook struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`

	// Webhook 配置
	URL     string `gorm:"type:text;not null"`
	Method  string `gorm:"size:10;default:'POST'"`
	Headers string `gorm:"type:text"` // 自定义 HTTP 头（JSON）
	Enabled bool   `gorm:"default:true;index"`

	// 触发条件
	Events  string `gorm:"type:text;not null"` // 监听的事件类型（JSON 数组）
	Filters string `gorm:"type:text"`          // 过滤条件（JSON）

	// 重试配置
	RetryEnabled   bool   `gorm:"default:true"`
	MaxRetries     int    `gorm:"default:3"`
	RetryIntervals string `gorm:"type:text;default:'[10, 30, 60]'"` // 重试间隔（秒）

	// 统计信息
	TotalCalls   int `gorm:"default:0"`
	SuccessCalls int `gorm:"default:0"`
	FailedCalls  int `gorm:"default:0"`
	LastCalledAt *time.Time
	LastStatus   string `gorm:"size:20"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (Webhook) TableName() string {
	return "webhooks"
}

// WebhookLog Webhook 调用日志模型
type WebhookLog struct {
	ID        int64 `gorm:"primaryKey"`
	WebhookID int64 `gorm:"not null;index"`

	// 请求信息
	RequestURL     string `gorm:"type:text;not null"`
	RequestMethod  string `gorm:"size:10"`
	RequestHeaders string `gorm:"type:text"`
	RequestBody    string `gorm:"type:text"`

	// 响应信息
	ResponseStatus int
	ResponseBody   string `gorm:"type:text"`
	ResponseTimeMs int

	// 结果
	Success      bool
	ErrorMessage string `gorm:"type:text"`
	RetryCount   int    `gorm:"default:0"`

	CreatedAt time.Time `gorm:"index:idx_created_at,sort:desc"`
}

// TableName 指定表名
func (WebhookLog) TableName() string {
	return "webhook_logs"
}
