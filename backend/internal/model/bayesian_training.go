package model

import "time"

// BayesianTraining 贝叶斯训练数据模型
type BayesianTraining struct {
	ID        int64  `gorm:"primaryKey"`
	UserUID   string `gorm:"size:64;not null;index"` // 用户 UID
	EmailID   string `gorm:"size:255;index"`         // 邮件 ID
	IsSpam    bool   `gorm:"not null;index"`         // 是否为垃圾邮件
	Tokens    string `gorm:"type:text;not null"`     // 特征词（JSON 数组）
	CreatedAt time.Time
}

// TableName 指定表名
func (BayesianTraining) TableName() string {
	return "bayesian_trainings"
}
