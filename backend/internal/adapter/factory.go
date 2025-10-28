package adapter

import (
	"fmt"
)

// Factory 适配器工厂
type Factory struct{}

// NewFactory 创建适配器工厂实例
func NewFactory() *Factory {
	return &Factory{}
}

// CreateProvider 创建邮箱服务提供商适配器
func (f *Factory) CreateProvider(config *Config) (MailProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}

	// 根据协议类型创建对应的适配器
	switch config.Protocol {
	case "imap":
		return NewIMAPAdapter(config)
	case "pop3":
		return NewPOP3Adapter(config)
	case "gmail_api":
		return NewGmailAdapter(config)
	case "graph":
		return NewGraphAdapter(config)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", config.Protocol)
	}
}

// CreateProviderFromAccount 从账户信息创建适配器
// 这是一个便捷方法，用于从数据库账户模型创建适配器
func (f *Factory) CreateProviderFromAccount(provider, protocol string, credentials *Credentials, proxy *ProxyConfig) (MailProvider, error) {
	config := &Config{
		Provider:    provider,
		Protocol:    protocol,
		Credentials: credentials,
		Proxy:       proxy,
		Timeout:     0, // 使用默认超时
	}

	return f.CreateProvider(config)
}

// GetSupportedProtocols 获取支持的协议列表
func (f *Factory) GetSupportedProtocols() []string {
	return []string{
		"imap",
		"pop3",
		"gmail_api",
		"graph",
	}
}

// GetSupportedProviders 获取支持的提供商列表
func (f *Factory) GetSupportedProviders() []string {
	return []string{
		"gmail",
		"outlook",
		"icloud",
		"qq",
		"163",
		"generic", // 通用 IMAP/POP3
	}
}

// GetRecommendedProtocol 获取推荐的协议
// 根据提供商返回推荐使用的协议
func (f *Factory) GetRecommendedProtocol(provider string) string {
	switch provider {
	case "gmail":
		return "gmail_api" // Gmail 优先使用 API
	case "outlook":
		return "graph" // Outlook 优先使用 Graph API
	case "icloud", "qq", "163":
		return "imap" // 其他提供商使用 IMAP
	default:
		return "imap" // 默认使用 IMAP
	}
}

// GetProviderInfo 获取提供商信息
type ProviderInfo struct {
	Name                string   // 提供商名称
	DisplayName         string   // 显示名称
	SupportedProtocols  []string // 支持的协议
	RecommendedProtocol string   // 推荐协议
	RequiresOAuth       bool     // 是否需要 OAuth
	IMAPHost            string   // IMAP 服务器地址
	IMAPPort            int      // IMAP 端口
	POP3Host            string   // POP3 服务器地址
	POP3Port            int      // POP3 端口
}

// GetProviderInfo 获取提供商详细信息
func (f *Factory) GetProviderInfo(provider string) *ProviderInfo {
	infos := map[string]*ProviderInfo{
		"gmail": {
			Name:                "gmail",
			DisplayName:         "Gmail",
			SupportedProtocols:  []string{"gmail_api", "imap"},
			RecommendedProtocol: "gmail_api",
			RequiresOAuth:       true,
			IMAPHost:            "imap.gmail.com",
			IMAPPort:            993,
		},
		"outlook": {
			Name:                "outlook",
			DisplayName:         "Outlook / Hotmail",
			SupportedProtocols:  []string{"graph", "imap"},
			RecommendedProtocol: "graph",
			RequiresOAuth:       true,
			IMAPHost:            "outlook.office365.com",
			IMAPPort:            993,
		},
		"icloud": {
			Name:                "icloud",
			DisplayName:         "iCloud Mail",
			SupportedProtocols:  []string{"imap"},
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.mail.me.com",
			IMAPPort:            993,
		},
		"qq": {
			Name:                "qq",
			DisplayName:         "QQ 邮箱",
			SupportedProtocols:  []string{"imap", "pop3"},
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.qq.com",
			IMAPPort:            993,
			POP3Host:            "pop.qq.com",
			POP3Port:            995,
		},
		"163": {
			Name:                "163",
			DisplayName:         "163 邮箱",
			SupportedProtocols:  []string{"imap", "pop3"},
			RecommendedProtocol: "imap",
			RequiresOAuth:       false,
			IMAPHost:            "imap.163.com",
			IMAPPort:            993,
			POP3Host:            "pop.163.com",
			POP3Port:            995,
		},
	}

	if info, ok := infos[provider]; ok {
		return info
	}

	// 返回通用配置
	return &ProviderInfo{
		Name:                "generic",
		DisplayName:         "通用邮箱",
		SupportedProtocols:  []string{"imap", "pop3"},
		RecommendedProtocol: "imap",
		RequiresOAuth:       false,
	}
}
