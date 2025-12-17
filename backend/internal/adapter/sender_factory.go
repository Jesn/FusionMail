package adapter

import (
	"fmt"

	"fusionmail/internal/model"
	"fusionmail/pkg/crypto"
)

// SenderFactory 发送器工厂
// 根据账户类型创建对应的发送器
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5
type SenderFactory struct {
	cryptoService *crypto.Service
}

// NewSenderFactory 创建发送器工厂
func NewSenderFactory(encryptionKey string) (*SenderFactory, error) {
	cryptoService, err := crypto.NewService(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto service: %w", err)
	}

	return &SenderFactory{
		cryptoService: cryptoService,
	}, nil
}

// GetSender 根据账户获取对应的发送器
// Requirements: 2.1, 2.2, 2.3
func (f *SenderFactory) GetSender(account *model.EmailAccount, credentials *AccountCredentials) (MailSender, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}

	// 根据提供商类型选择发送器
	switch account.Provider {
	case "gmail":
		return f.createGmailSender(account, credentials)
	case "outlook", "hotmail", "microsoft":
		return f.createGraphSender(account, credentials)
	default:
		// 其他类型（IMAP/POP3/generic）使用 SMTP
		return f.createSMTPSender(account)
	}
}

// GetSenderWithFallback 获取发送器，支持降级
// 如果首选发送器不可用，尝试降级到 SMTP
// Requirements: 2.4, 2.5
func (f *SenderFactory) GetSenderWithFallback(account *model.EmailAccount, credentials *AccountCredentials) (MailSender, error) {
	// 首先尝试获取首选发送器
	sender, err := f.GetSender(account, credentials)
	if err == nil {
		return sender, nil
	}

	// 如果首选发送器创建失败，尝试降级到 SMTP
	if account.SMTPEnabled && account.SMTPHost != "" {
		smtpSender, smtpErr := f.createSMTPSender(account)
		if smtpErr == nil {
			return smtpSender, nil
		}
		// SMTP 也失败，返回原始错误
		return nil, fmt.Errorf("primary sender failed: %v, SMTP fallback failed: %v", err, smtpErr)
	}

	// 没有配置 SMTP，返回原始错误
	return nil, err
}

// createGmailSender 创建 Gmail 发送器
func (f *SenderFactory) createGmailSender(account *model.EmailAccount, credentials *AccountCredentials) (MailSender, error) {
	if credentials == nil {
		return nil, fmt.Errorf("credentials required for Gmail sender")
	}

	if credentials.AccessToken == "" {
		return nil, fmt.Errorf("access token required for Gmail sender")
	}

	config := &SenderConfig{
		AccountUID:   account.UID,
		Email:        account.Email,
		Provider:     account.Provider,
		AuthType:     account.AuthType,
		AccessToken:  credentials.AccessToken,
		RefreshToken: credentials.RefreshToken,
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
	}

	return NewGmailSender(config)
}

// createGraphSender 创建 Microsoft Graph 发送器
func (f *SenderFactory) createGraphSender(account *model.EmailAccount, credentials *AccountCredentials) (MailSender, error) {
	if credentials == nil {
		return nil, fmt.Errorf("credentials required for Graph sender")
	}

	if credentials.AccessToken == "" {
		return nil, fmt.Errorf("access token required for Graph sender")
	}

	config := &SenderConfig{
		AccountUID:   account.UID,
		Email:        account.Email,
		Provider:     account.Provider,
		AuthType:     account.AuthType,
		AccessToken:  credentials.AccessToken,
		RefreshToken: credentials.RefreshToken,
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
	}

	return NewGraphSender(config)
}

// createSMTPSender 创建 SMTP 发送器
func (f *SenderFactory) createSMTPSender(account *model.EmailAccount) (MailSender, error) {
	// 检查 SMTP 是否已配置
	if !account.SMTPEnabled {
		return nil, fmt.Errorf("SMTP not enabled for this account")
	}

	// 从 Provider 或 Account 获取 SMTP 服务器配置
	host, port, encryption := account.GetSMTPConfig()
	if host == "" {
		return nil, fmt.Errorf("SMTP host not configured (check Provider or Account settings)")
	}

	// 解密 SMTP 密码
	password := ""
	if account.EncryptedSMTPPassword != "" {
		decrypted, err := f.cryptoService.Decrypt(account.EncryptedSMTPPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt SMTP password: %w", err)
		}
		password = string(decrypted)
	}

	// 确定用户名
	username := account.SMTPUsername
	if username == "" {
		username = account.Email
	}

	config := &SMTPConfig{
		Host:       host,
		Port:       port,
		Encryption: encryption,
		Username:   username,
		Password:   password,
	}

	return NewSMTPSender(config, account.Email), nil
}

// AccountCredentials 账户凭证（用于 OAuth2 认证的账户）
type AccountCredentials struct {
	AccessToken  string
	RefreshToken string
	ClientID     string
	ClientSecret string
}

// GetSenderType 根据账户类型获取推荐的发送器类型
// Requirements: 2.1, 2.2, 2.3
func GetSenderType(provider string) string {
	switch provider {
	case "gmail":
		return SenderTypeGmailAPI
	case "outlook", "hotmail", "microsoft":
		return SenderTypeGraphAPI
	default:
		return SenderTypeSMTP
	}
}

// IsSMTPRequired 检查账户是否需要 SMTP 配置才能发送邮件
func IsSMTPRequired(provider string) bool {
	switch provider {
	case "gmail", "outlook", "hotmail", "microsoft":
		return false // 这些提供商可以使用 API 发送
	default:
		return true // 其他提供商需要 SMTP
	}
}
