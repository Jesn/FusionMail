package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 分组相关错误定义
var (
	ErrGroupNameRequired = errors.New("分组名称不能为空")
	ErrGroupNameTooLong  = errors.New("分组名称不能超过100个字符")
	ErrGroupNameExists   = errors.New("分组名称已存在")
	ErrGroupNotFound     = errors.New("分组不存在")
)

// AccountGroup 账号分组模型
type AccountGroup struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"size:100;not null" json:"name"`          // 分组名称
	Description  string         `gorm:"type:text" json:"description,omitempty"` // 分组描述
	DisplayOrder int            `gorm:"default:0" json:"display_order"`         // 显示顺序
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (AccountGroup) TableName() string {
	return "account_groups"
}

// Validate 验证分组数据
func (g *AccountGroup) Validate() error {
	// 去除首尾空格
	g.Name = strings.TrimSpace(g.Name)

	// 检查名称是否为空
	if g.Name == "" {
		return ErrGroupNameRequired
	}

	// 检查名称长度
	if len(g.Name) > 100 {
		return ErrGroupNameTooLong
	}

	return nil
}

// BeforeCreate GORM 钩子：创建前验证
func (g *AccountGroup) BeforeCreate(tx *gorm.DB) error {
	return g.Validate()
}

// BeforeUpdate GORM 钩子：更新前验证
func (g *AccountGroup) BeforeUpdate(tx *gorm.DB) error {
	return g.Validate()
}

// AccountGroupWithCount 带账号数量的分组
type AccountGroupWithCount struct {
	AccountGroup
	AccountCount int `json:"account_count"`
}

// GroupListResponse 分组列表响应（包含统计信息）
type GroupListResponse struct {
	Groups         []*AccountGroupWithCount `json:"groups"`          // 分组列表
	TotalCount     int                      `json:"total_count"`     // 所有账号总数
	UngroupedCount int                      `json:"ungrouped_count"` // 未分组账号数
}

// AccountGroupWithAccounts 带账号列表的分组
type AccountGroupWithAccounts struct {
	AccountGroup
	Accounts []*EmailAccount `json:"accounts,omitempty"`
}
