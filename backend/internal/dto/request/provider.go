package request

import "fusionmail/internal/model"

// ProviderRequest 是创建/更新邮箱提供商的请求体。
//
// 为什么需要它：handler 此前直接把请求体绑定到 model.Provider（带 gorm tag 的
// 持久化模型），导致：
//   - 请求契约暴露了存储层字段（created_at / updated_at）
//   - 级联带出关联模型 model.ProviderAdapter 与 model.Adapter 进入 API 契约
//   - 客户端可以尝试写入 ID、关联对象等不该由请求决定的字段
//
// 逐字段显式列出可写字段：新增 model 字段默认不会成为可写入口。
type ProviderRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`

	DefaultAdapterID int64             `json:"default_adapter_id"`
	EmailDomains     model.StringArray `json:"email_domains"`

	// 支持的适配器 ID 列表（Provider-Adapter 多对多关联）
	// 由服务层在事务内写入 provider_adapters 表；空/nil 表示清空关联
	AdapterIDs []int64 `json:"adapter_ids"`

	RecommendedProtocol string `json:"recommended_protocol"`
	RequiresOAuth       bool   `json:"requires_oauth"`

	// 协议服务器配置
	IMAPHost string `json:"imap_host"`
	IMAPPort int    `json:"imap_port"`
	POP3Host string `json:"pop3_host"`
	POP3Port int    `json:"pop3_port"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`

	IMAPEncryption string `json:"imap_encryption"`
	POP3Encryption string `json:"pop3_encryption"`
	SMTPEncryption string `json:"smtp_encryption"`

	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
	Metadata    string `json:"metadata,omitempty"`
}

// ToModel 把请求 DTO 映射为存储模型。
//
// 只赋值可写字段：ID 与关联对象由服务层决定，不从请求体带入。
func (r *ProviderRequest) ToModel() *model.Provider {
	return &model.Provider{
		Name:        r.Name,
		DisplayName: r.DisplayName,

		DefaultAdapterID: r.DefaultAdapterID,
		EmailDomains:     r.EmailDomains,

		RecommendedProtocol: r.RecommendedProtocol,
		RequiresOAuth:       r.RequiresOAuth,

		IMAPHost: r.IMAPHost,
		IMAPPort: r.IMAPPort,
		POP3Host: r.POP3Host,
		POP3Port: r.POP3Port,
		SMTPHost: r.SMTPHost,
		SMTPPort: r.SMTPPort,

		IMAPEncryption: r.IMAPEncryption,
		POP3Encryption: r.POP3Encryption,
		SMTPEncryption: r.SMTPEncryption,

		Enabled:     r.Enabled,
		SortOrder:   r.SortOrder,
		Description: r.Description,
		Metadata:    r.Metadata,
	}
}
