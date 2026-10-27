package model

import "time"

// SenderReputation 发件人信誉模型
type SenderReputation struct {
	ID              int64   `gorm:"primaryKey"`
	Email           string  `gorm:"size:255;uniqueIndex;not null"`                                                      // 发件人邮箱
	Domain          string  `gorm:"size:255;index"`                                                                     // 发件人域名
	ReputationScore float64 `gorm:"default:50;index"`                                                                   // 信誉评分（0-100）
	TrustLevel      string  `gorm:"size:20;index;check:trust_level IN ('trusted', 'neutral', 'suspicious', 'blocked')"` // 信任级别
	TotalEmails     int64   `gorm:"default:0"`                                                                          // 总邮件数
	SpamCount       int64   `gorm:"default:0"`                                                                          // 垃圾邮件数
	HamCount        int64   `gorm:"default:0"`                                                                          // 正常邮件数
	RBLStatus       string  `gorm:"size:20;check:rbl_status IN ('clean', 'listed', 'unknown')"`                         // RBL 状态
	RBLCheckedAt    *time.Time
	RBLLists        string `gorm:"type:text"` // RBL 列表（JSON 数组）
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName 指定表名
func (SenderReputation) TableName() string {
	return "sender_reputations"
}
