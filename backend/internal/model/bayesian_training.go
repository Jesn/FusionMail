package model

import "time"

// BayesianTraining 贝叶斯训练数据模型
type BayesianTraining struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserUID   string    `gorm:"size:64;not null;index" json:"user_uid"` // 用户 UID
	EmailID   string    `gorm:"size:255;index" json:"email_id"`         // 邮件 ID
	IsSpam    bool      `gorm:"not null;index" json:"is_spam"`          // 是否为垃圾邮件
	Tokens    string    `gorm:"type:text;not null" json:"tokens"`       // 特征词（JSON 数组）
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (BayesianTraining) TableName() string {
	return "bayesian_trainings"
}
