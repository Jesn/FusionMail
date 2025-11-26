package model

import "time"

// SenderReputation 发件人信誉模型
type SenderReputation struct {
	ID              int64      `gorm:"primaryKey" json:"id"`
	Email           string     `gorm:"size:255;uniqueIndex;not null" json:"email"`                                                                 // 发件人邮箱
	Domain          string     `gorm:"size:255;index" json:"domain"`                                                                               // 发件人域名
	ReputationScore float64    `gorm:"default:50;index" json:"reputation_score"`                                                                   // 信誉评分（0-100）
	TrustLevel      string     `gorm:"size:20;index;check:trust_level IN ('trusted', 'neutral', 'suspicious', 'blocked')" json:"trust_level"`     // 信任级别
	TotalEmails     int64      `gorm:"default:0" json:"total_emails"`                                                                              // 总邮件数
	SpamCount       int64      `gorm:"default:0" json:"spam_count"`                                                                                // 垃圾邮件数
	HamCount        int64      `gorm:"default:0" json:"ham_count"`                                                                                 // 正常邮件数
	RBLStatus       string     `gorm:"size:20;check:rbl_status IN ('clean', 'listed', 'unknown')" json:"rbl_status"`                              // RBL 状态
	RBLCheckedAt    *time.Time `json:"rbl_checked_at"`                                                                                             // RBL 检查时间
	RBLLists        string     `gorm:"type:text" json:"rbl_lists"`                                                                                 // RBL 列表（JSON 数组）
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (SenderReputation) TableName() string {
	return "sender_reputations"
}
