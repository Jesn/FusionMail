package response

import (
	"fusionmail/internal/model"
	"time"
)

// AdapterRecord 与 model.Adapter 逐字段 JSON 等价的响应 DTO。
//
// model.Adapter 未声明 json tag，序列化键为 Go 字段名，这里用显式 tag
// 保持一致。仅用于 Swagger 契约声明，避免带 gorm tag 的模型进入 API 文档。
type AdapterRecord struct {
	ID          int64     `json:"ID"`
	Name        string    `json:"Name"`
	DisplayName string    `json:"DisplayName"`
	AuthType    string    `json:"AuthType"`
	Description string    `json:"Description"`
	IsEnabled   bool      `json:"IsEnabled"`
	CreatedAt   time.Time `json:"CreatedAt"`
	UpdatedAt   time.Time `json:"UpdatedAt"`
}

// ProviderAdapterRecord 与 model.ProviderAdapter 逐字段 JSON 等价的响应 DTO。
type ProviderAdapterRecord struct {
	ProviderID int64          `json:"ProviderID"`
	AdapterID  int64          `json:"AdapterID"`
	Priority   int            `json:"Priority"`
	CreatedAt  time.Time      `json:"CreatedAt"`
	Adapter    *AdapterRecord `json:"Adapter"`
}

// ProviderRecord 与 model.Provider 逐字段 JSON 等价的响应 DTO。
//
// 用于 GET /api/v1/providers/all（原样返回 []model.Provider）。该查询不预加载
// 关联，DefaultAdapter / SupportedAdapters 恒为 null，这里保持同样的键与 null 值。
// 仅用于 Swagger 契约声明；对外暴露 Provider 的新端点请优先使用 model.ProviderResponse
// （经 ToResponse 映射的 snake_case 视图）。
type ProviderRecord struct {
	ID int64 `json:"ID"`

	// 基础信息
	Name        string `json:"Name"`
	DisplayName string `json:"DisplayName"`

	// 适配器关联
	DefaultAdapterID  int64                   `json:"DefaultAdapterID"`
	DefaultAdapter    *AdapterRecord          `json:"DefaultAdapter"`
	EmailDomains      model.StringArray       `json:"EmailDomains"`
	SupportedAdapters []ProviderAdapterRecord `json:"SupportedAdapters"`

	// 协议配置（保留用于向后兼容）
	SupportedProtocols  string `json:"SupportedProtocols"`
	RecommendedProtocol string `json:"RecommendedProtocol"`
	RequiresOAuth       bool   `json:"RequiresOAuth"`

	// 服务器配置
	IMAPHost string `json:"IMAPHost"`
	IMAPPort int    `json:"IMAPPort"`
	POP3Host string `json:"POP3Host"`
	POP3Port int    `json:"POP3Port"`
	SMTPHost string `json:"SMTPHost"`
	SMTPPort int    `json:"SMTPPort"`

	// 加密配置
	IMAPEncryption string `json:"IMAPEncryption"`
	POP3Encryption string `json:"POP3Encryption"`
	SMTPEncryption string `json:"SMTPEncryption"`

	// 管理字段
	Enabled     bool   `json:"Enabled"`
	SortOrder   int    `json:"SortOrder"`
	Description string `json:"Description"`
	Metadata    string `json:"Metadata"`

	// 时间戳
	CreatedAt time.Time `json:"CreatedAt"`
	UpdatedAt time.Time `json:"UpdatedAt"`
}

// NewProviderRecords 批量映射（保持 nil/空切片语义不变）。
func NewProviderRecords(providers []model.Provider) []ProviderRecord {
	if providers == nil {
		return nil
	}
	out := make([]ProviderRecord, 0, len(providers))
	for i := range providers {
		p := &providers[i]
		out = append(out, ProviderRecord{
			ID:          p.ID,
			Name:        p.Name,
			DisplayName: p.DisplayName,

			DefaultAdapterID:  p.DefaultAdapterID,
			DefaultAdapter:    nil, // FindAll 不预加载，运行时恒为 nil
			EmailDomains:      p.EmailDomains,
			SupportedAdapters: nil, // FindAll 不预加载，运行时恒为 nil

			SupportedProtocols:  p.SupportedProtocols,
			RecommendedProtocol: p.RecommendedProtocol,
			RequiresOAuth:       p.RequiresOAuth,

			IMAPHost: p.IMAPHost,
			IMAPPort: p.IMAPPort,
			POP3Host: p.POP3Host,
			POP3Port: p.POP3Port,
			SMTPHost: p.SMTPHost,
			SMTPPort: p.SMTPPort,

			IMAPEncryption: p.IMAPEncryption,
			POP3Encryption: p.POP3Encryption,
			SMTPEncryption: p.SMTPEncryption,

			Enabled:     p.Enabled,
			SortOrder:   p.SortOrder,
			Description: p.Description,
			Metadata:    p.Metadata,

			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return out
}

// NewProviderRecord 单个 Provider 的序列化等价映射（复用批量映射）。
// model.Provider 裸序列化的键为 Go 字段名（无 json tag），ProviderRecord
// 已按此镜像。
func NewProviderRecord(p *model.Provider) *ProviderRecord {
	if p == nil {
		return nil
	}
	records := NewProviderRecords([]model.Provider{*p})
	return &records[0]
}
