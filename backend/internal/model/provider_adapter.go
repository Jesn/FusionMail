package model

import (
	"time"
)

// ProviderAdapter Provider 与 Adapter 的多对多关联模型
// 支持一个 Provider 关联多个 Adapter（如 Gmail 同时支持 OAuth2 和 IMAP）
type ProviderAdapter struct {
	ProviderID int64     `gorm:"primaryKey" json:"provider_id"` // 关联的提供商 ID
	AdapterID  int64     `gorm:"primaryKey" json:"adapter_id"`  // 关联的适配器 ID
	Priority   int       `gorm:"default:0" json:"priority"`     // 优先级，0 为最高（默认推荐）
	CreatedAt  time.Time `json:"created_at"`
	Adapter    *Adapter  `gorm:"foreignKey:AdapterID" json:"adapter,omitempty"` // 关联的适配器
}

// TableName 指定表名
func (ProviderAdapter) TableName() string {
	return "provider_adapters"
}

// ProviderAdapterResponse 用于 API 响应的 ProviderAdapter 结构
type ProviderAdapterResponse struct {
	ProviderID int64            `json:"provider_id"`
	AdapterID  int64            `json:"adapter_id"`
	Priority   int              `json:"priority"`
	Adapter    *AdapterResponse `json:"adapter,omitempty"`
}

// ToResponse 将 ProviderAdapter 转换为 ProviderAdapterResponse
func (pa *ProviderAdapter) ToResponse() *ProviderAdapterResponse {
	resp := &ProviderAdapterResponse{
		ProviderID: pa.ProviderID,
		AdapterID:  pa.AdapterID,
		Priority:   pa.Priority,
	}

	if pa.Adapter != nil {
		resp.Adapter = pa.Adapter.ToResponse()
	}

	return resp
}

// IsDefault 检查是否为默认（最高优先级）适配器
func (pa *ProviderAdapter) IsDefault() bool {
	return pa.Priority == 0
}
