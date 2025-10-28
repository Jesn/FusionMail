package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID           int64  `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:100;not null" json:"username"`
	Email        string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	PasswordHash string `gorm:"size:255;not null" json:"-"` // 不在 JSON 中返回密码

	// 个人信息
	DisplayName string `gorm:"size:255" json:"display_name"`
	AvatarURL   string `gorm:"type:text" json:"avatar_url"`

	// 设置
	Timezone string `gorm:"size:50;default:'UTC'" json:"timezone"`
	Language string `gorm:"size:10;default:'en'" json:"language"`
	Theme    string `gorm:"size:20;default:'light'" json:"theme"` // light/dark

	// 状态
	IsActive      bool `gorm:"default:true" json:"is_active"`
	EmailVerified bool `gorm:"default:false" json:"email_verified"`

	// 安全
	LastLoginAt         *time.Time `json:"last_login_at"`
	LastLoginIP         string     `gorm:"size:45" json:"last_login_ip"`
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
