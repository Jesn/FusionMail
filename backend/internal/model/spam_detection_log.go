package model

import "time"

// SpamDetectionLog 垃圾邮件检测日志模型
type SpamDetectionLog struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	EmailID          string    `gorm:"size:255;index" json:"email_id"`         // 邮件 ID
	IsSpam           bool      `gorm:"not null;index" json:"is_spam"`          // 是否为垃圾邮件
	FinalScore       float64   `gorm:"not null" json:"final_score"`            // 最终评分
	PreFilterScore   float64   `gorm:"default:0" json:"pre_filter_score"`      // 预过滤层评分
	RuleScore        float64   `gorm:"default:0" json:"rule_score"`            // 规则引擎评分
	BayesianScore    float64   `gorm:"default:0" json:"bayesian_score"`        // 贝叶斯分类评分
	DetectionDetails string    `gorm:"type:text" json:"detection_details"`     // 检测详情（JSON）
	ProcessingTimeMs int64     `json:"processing_time_ms"`                     // 处理时间（毫秒）
	CreatedAt        time.Time `gorm:"index:idx_created_at,sort:desc" json:"created_at"`
}

// TableName 指定表名
func (SpamDetectionLog) TableName() string {
	return "spam_detection_logs"
}
