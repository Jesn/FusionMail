package service

import (
	"context"
	"fmt"

	"fusionmail/internal/adapter"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
)

// SMTPConfigService SMTP 配置服务
// Requirements: 3.1, 3.2, 3.3, 3.4
type SMTPConfigService struct {
	accountRepo   repository.AccountRepository
	cryptoService *crypto.Service
}

// NewSMTPConfigService 创建 SMTP 配置服务实例
func NewSMTPConfigService(accountRepo repository.AccountRepository, encryptionKey string) (*SMTPConfigService, error) {
	cryptoService, err := crypto.NewService(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto service: %w", err)
	}

	return &SMTPConfigService{
		accountRepo:   accountRepo,
		cryptoService: cryptoService,
	}, nil
}

// SMTPConfigRequest SMTP 配置请求
// 注意：host/port/encryption 从 Provider 继承，Account 只需配置用户名和密码
type SMTPConfigRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// SMTPConfigResponse SMTP 配置响应
type SMTPConfigResponse struct {
	Host         string `json:"smtp_host"`       // 实际使用的 SMTP 服务器（可能来自 Provider）
	Port         int    `json:"smtp_port"`       // 实际使用的端口
	Encryption   string `json:"smtp_encryption"` // 实际使用的加密方式
	Username     string `json:"smtp_username"`
	Enabled      bool   `json:"smtp_enabled"`
	FromProvider bool   `json:"from_provider"` // 服务器配置是否来自 Provider
	ProviderName string `json:"provider_name"` // Provider 名称（如果有）
	// 不返回密码
}

// UpdateSMTPConfig 更新账户的 SMTP 配置
// Requirements: 3.1
// 注意：SMTP 服务器配置（host/port/encryption）从 Provider 获取
// Account 级别只需配置 username、password 和 enabled
func (s *SMTPConfigService) UpdateSMTPConfig(ctx context.Context, accountUID string, req *SMTPConfigRequest) error {
	// 获取账户（预加载 Provider）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	// 更新 SMTP 配置（只更新用户名、密码和启用状态）
	account.SMTPUsername = req.Username
	account.SMTPEnabled = req.Enabled

	// 如果提供了密码，加密存储
	if req.Password != "" {
		encrypted, err := s.cryptoService.Encrypt([]byte(req.Password))
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		account.EncryptedSMTPPassword = encrypted
	}

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	return nil
}

// GetSMTPConfig 获取账户的 SMTP 配置
// 返回实际使用的配置（优先从 Provider 获取服务器配置）
func (s *SMTPConfigService) GetSMTPConfig(ctx context.Context, accountUID string) (*SMTPConfigResponse, error) {
	// 获取账户（预加载 Provider）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	// 使用 GetSMTPConfig 方法获取实际配置（优先 Provider）
	host, port, encryption := account.GetSMTPConfig()

	// 判断配置是否来自 Provider
	fromProvider := account.ProviderRef != nil && account.ProviderRef.SMTPHost != ""
	providerName := ""
	if account.ProviderRef != nil {
		providerName = account.ProviderRef.DisplayName
	}

	return &SMTPConfigResponse{
		Host:         host,
		Port:         port,
		Encryption:   encryption,
		Username:     account.SMTPUsername,
		Enabled:      account.SMTPEnabled,
		FromProvider: fromProvider,
		ProviderName: providerName,
	}, nil
}

// TestSMTPConnection 测试 SMTP 连接
// Requirements: 3.2, 3.3
// tempUsername 和 tempPassword 是可选的临时凭证，用于测试未保存的配置
func (s *SMTPConfigService) TestSMTPConnection(ctx context.Context, accountUID string, tempUsername, tempPassword string) error {
	// 获取账户（预加载 Provider）
	account, err := s.accountRepo.FindByUIDWithRelations(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	// 使用 GetSMTPConfig 方法获取实际配置（优先 Provider）
	host, port, encryption := account.GetSMTPConfig()

	// 检查 SMTP 是否已配置
	if host == "" {
		return fmt.Errorf("SMTP 未配置（请在提供商或账户中配置 SMTP 服务器）")
	}

	// 确定使用的用户名（优先使用临时用户名）
	username := account.SMTPUsername
	if tempUsername != "" {
		username = tempUsername
	}

	// 确定使用的密码（优先使用临时密码）
	password := tempPassword
	if password == "" && account.EncryptedSMTPPassword != "" {
		// 如果没有提供临时密码，尝试使用已保存的密码
		decrypted, err := s.cryptoService.Decrypt(account.EncryptedSMTPPassword)
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		password = string(decrypted)
	}

	// 检查是否有可用的密码
	if password == "" {
		return fmt.Errorf("请提供 SMTP 密码/授权码")
	}

	// 创建 SMTP 发送器
	config := &adapter.SMTPConfig{
		Host:       host,
		Port:       port,
		Encryption: encryption,
		Username:   username,
		Password:   password,
	}

	sender := adapter.NewSMTPSender(config, account.Email)

	// 测试连接
	if err := sender.TestConnection(ctx); err != nil {
		return fmt.Errorf("SMTP 连接测试失败: %w", err)
	}

	return nil
}

// DefaultSMTPConfig 默认 SMTP 配置
type DefaultSMTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Encryption string `json:"encryption"`
}

// GetDefaultSMTPConfig 获取常见邮箱服务商的默认 SMTP 配置
// Requirements: 3.4
func GetDefaultSMTPConfig(provider string) *DefaultSMTPConfig {
	configs := map[string]*DefaultSMTPConfig{
		// 国内邮箱
		"qq": {
			Host:       "smtp.qq.com",
			Port:       465,
			Encryption: "tls",
		},
		"163": {
			Host:       "smtp.163.com",
			Port:       465,
			Encryption: "tls",
		},
		"126": {
			Host:       "smtp.126.com",
			Port:       465,
			Encryption: "tls",
		},
		"sina": {
			Host:       "smtp.sina.com",
			Port:       465,
			Encryption: "tls",
		},
		"sohu": {
			Host:       "smtp.sohu.com",
			Port:       465,
			Encryption: "tls",
		},
		"139": {
			Host:       "smtp.139.com",
			Port:       465,
			Encryption: "tls",
		},
		"aliyun": {
			Host:       "smtp.aliyun.com",
			Port:       465,
			Encryption: "tls",
		},
		// 国际邮箱
		"gmail": {
			Host:       "smtp.gmail.com",
			Port:       587,
			Encryption: "starttls",
		},
		"outlook": {
			Host:       "smtp.office365.com",
			Port:       587,
			Encryption: "starttls",
		},
		"hotmail": {
			Host:       "smtp.office365.com",
			Port:       587,
			Encryption: "starttls",
		},
		"yahoo": {
			Host:       "smtp.mail.yahoo.com",
			Port:       465,
			Encryption: "tls",
		},
		"icloud": {
			Host:       "smtp.mail.me.com",
			Port:       587,
			Encryption: "starttls",
		},
		// 企业邮箱
		"exmail": { // 腾讯企业邮箱
			Host:       "smtp.exmail.qq.com",
			Port:       465,
			Encryption: "tls",
		},
	}

	if config, ok := configs[provider]; ok {
		return config
	}

	// 默认配置
	return &DefaultSMTPConfig{
		Host:       "",
		Port:       587,
		Encryption: "starttls",
	}
}

// GetAllDefaultSMTPConfigs 获取所有默认 SMTP 配置
func GetAllDefaultSMTPConfigs() map[string]*DefaultSMTPConfig {
	return map[string]*DefaultSMTPConfig{
		"qq":      GetDefaultSMTPConfig("qq"),
		"163":     GetDefaultSMTPConfig("163"),
		"126":     GetDefaultSMTPConfig("126"),
		"sina":    GetDefaultSMTPConfig("sina"),
		"sohu":    GetDefaultSMTPConfig("sohu"),
		"139":     GetDefaultSMTPConfig("139"),
		"aliyun":  GetDefaultSMTPConfig("aliyun"),
		"gmail":   GetDefaultSMTPConfig("gmail"),
		"outlook": GetDefaultSMTPConfig("outlook"),
		"hotmail": GetDefaultSMTPConfig("hotmail"),
		"yahoo":   GetDefaultSMTPConfig("yahoo"),
		"icloud":  GetDefaultSMTPConfig("icloud"),
		"exmail":  GetDefaultSMTPConfig("exmail"),
	}
}
