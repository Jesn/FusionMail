package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID           int64  `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;size:100;not null"`
	Email        string `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string `gorm:"size:255;not null"` // 不在 JSON 中返回密码

	// 个人信息
	DisplayName string `gorm:"size:255"`
	AvatarURL   string `gorm:"type:text"`

	// 设置
	Timezone string `gorm:"size:50;default:'UTC'"`
	Language string `gorm:"size:10;default:'en'"`
	Theme    string `gorm:"size:20;default:'light'"` // light/dark

	// 角色和权限
	Role string `gorm:"size:20;default:'user'"` // admin/user

	// 状态
	IsActive      bool `gorm:"default:true"`
	EmailVerified bool `gorm:"default:false"`

	// 安全
	LastLoginAt         *time.Time
	LastLoginIP         string `gorm:"size:45"`
	FailedLoginAttempts int    `gorm:"default:0"`
	LockedUntil         *time.Time
	SessionVersion      int64 `gorm:"default:0;not null"`

	// 双因素认证 (2FA)
	TwoFactorEnabled   bool   `gorm:"default:false"`
	TwoFactorSecret    string `gorm:"type:text"`     // TOTP 密钥（加密存储）
	TwoFactorBackup    string `gorm:"type:text"`     // 恢复码（哈希存储）
	TwoFactorVerified  bool   `gorm:"default:false"` // 是否已验证完成设置
	TwoFactorEnabledAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
