package model

import "time"

// SpamDetectionLog 垃圾邮件检测日志模型
type SpamDetectionLog struct {
	ID               int64   `gorm:"primaryKey"`
	EmailID          string  `gorm:"size:255;index"` // 邮件 ID
	IsSpam           bool    `gorm:"not null;index"` // 是否为垃圾邮件
	FinalScore       float64 `gorm:"not null"`       // 最终评分
	DetectionDetails string  `gorm:"type:text"`      // 检测详情（JSON）
	ProcessingTimeMs int64
	CreatedAt        time.Time `gorm:"index:idx_created_at,sort:desc"`
}

// TableName 指定表名
func (SpamDetectionLog) TableName() string {
	return "spam_detection_logs"
}
