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
type SMTPConfigRequest struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	Encryption string `json:"encryption"` // none/tls/starttls
	Username   string `json:"username"`
	Password   string `json:"password"`
	Enabled    bool   `json:"enabled"`
}

// SMTPConfigResponse SMTP 配置响应
type SMTPConfigResponse struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Encryption string `json:"encryption"`
	Username   string `json:"username"`
	Enabled    bool   `json:"enabled"`
	// 不返回密码
}

// UpdateSMTPConfig 更新账户的 SMTP 配置
// Requirements: 3.1
func (s *SMTPConfigService) UpdateSMTPConfig(ctx context.Context, accountUID string, req *SMTPConfigRequest) error {
	// 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	// 更新 SMTP 配置
	account.SMTPHost = req.Host
	account.SMTPPort = req.Port
	account.SMTPEncryption = req.Encryption
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
func (s *SMTPConfigService) GetSMTPConfig(ctx context.Context, accountUID string) (*SMTPConfigResponse, error) {
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return &SMTPConfigResponse{
		Host:       account.SMTPHost,
		Port:       account.SMTPPort,
		Encryption: account.SMTPEncryption,
		Username:   account.SMTPUsername,
		Enabled:    account.SMTPEnabled,
	}, nil
}

// TestSMTPConnection 测试 SMTP 连接
// Requirements: 3.2, 3.3
func (s *SMTPConfigService) TestSMTPConnection(ctx context.Context, accountUID string) error {
	// 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	// 检查 SMTP 是否已配置
	if account.SMTPHost == "" {
		return fmt.Errorf("SMTP 未配置")
	}

	// 解密密码
	password := ""
	if account.EncryptedSMTPPassword != "" {
		decrypted, err := s.cryptoService.Decrypt(account.EncryptedSMTPPassword)
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		password = string(decrypted)
	}

	// 创建 SMTP 发送器
	config := &adapter.SMTPConfig{
		Host:       account.SMTPHost,
		Port:       account.SMTPPort,
		Encryption: account.SMTPEncryption,
		Username:   account.SMTPUsername,
		Password:   password,
	}

	sender := adapter.NewSMTPSender(config, account.Email)

	// 测试连接
	if err := sender.TestConnection(ctx); err != nil {
		return fmt.Errorf("SMTP 连接测试失败: %w", err)
	}

	return nil
}

// TestSMTPConnectionWithConfig 使用提供的配置测试 SMTP 连接（不保存）
// Requirements: 3.2
func (s *SMTPConfigService) TestSMTPConnectionWithConfig(ctx context.Context, email string, req *SMTPConfigRequest) error {
	config := &adapter.SMTPConfig{
		Host:       req.Host,
		Port:       req.Port,
		Encryption: req.Encryption,
		Username:   req.Username,
		Password:   req.Password,
	}

	sender := adapter.NewSMTPSender(config, email)

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
