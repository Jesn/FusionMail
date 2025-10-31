package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"

	"github.com/google/uuid"
)

// AccountService 账户管理服务接口
type AccountService interface {
	// Create 创建账户
	Create(ctx context.Context, req *CreateAccountRequest) (*model.Account, error)

	// GetByUID 根据 UID 获取账户
	GetByUID(ctx context.Context, uid string) (*model.Account, error)

	// List 获取账户列表
	List(ctx context.Context) ([]*model.Account, error)

	// Update 更新账户
	Update(ctx context.Context, uid string, req *UpdateAccountRequest) (*model.Account, error)

	// Delete 删除账户
	Delete(ctx context.Context, uid string) error

	// TestConnection 测试账户连接
	TestConnection(ctx context.Context, uid string) error

	// SetStatus 设置账户状态
	SetStatus(ctx context.Context, uid string, status string) error

	// DisableAccount 禁用账户
	DisableAccount(ctx context.Context, uid string) error

	// EnableAccount 启用账户
	EnableAccount(ctx context.Context, uid string) error

	// ClearSyncError 清除同步错误状态
	ClearSyncError(ctx context.Context, uid string) error
}

// CreateAccountRequest 创建账户请求
type CreateAccountRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Provider     string `json:"provider" binding:"required"`
	Protocol     string `json:"protocol" binding:"required"`
	AuthType     string `json:"auth_type" binding:"required"`
	Password     string `json:"password" binding:"required"`
	SyncEnabled  bool   `json:"sync_enabled"`
	SyncInterval int    `json:"sync_interval"`
	// 通用邮箱配置字段
	IMAPHost   string `json:"imap_host,omitempty"`
	IMAPPort   int    `json:"imap_port,omitempty"`
	POP3Host   string `json:"pop3_host,omitempty"`
	POP3Port   int    `json:"pop3_port,omitempty"`
	Encryption string `json:"encryption,omitempty"`
}

// UpdateAccountRequest 更新账户请求
type UpdateAccountRequest struct {
	Email        *string `json:"email,omitempty"`
	Password     *string `json:"password,omitempty"`
	SyncEnabled  *bool   `json:"sync_enabled,omitempty"`
	SyncInterval *int    `json:"sync_interval,omitempty"`
	// 通用邮箱配置字段
	IMAPHost   *string `json:"imap_host,omitempty"`
	IMAPPort   *int    `json:"imap_port,omitempty"`
	POP3Host   *string `json:"pop3_host,omitempty"`
	POP3Port   *int    `json:"pop3_port,omitempty"`
	Encryption *string `json:"encryption,omitempty"`
}

// accountService 账户管理服务实现
type accountService struct {
	accountRepo    repository.AccountRepository
	adapterFactory *adapter.Factory
	encryptor      crypto.Encryptor
}

// NewAccountService 创建账户管理服务实例
func NewAccountService(
	accountRepo repository.AccountRepository,
	adapterFactory *adapter.Factory,
) (AccountService, error) {
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &accountService{
		accountRepo:    accountRepo,
		adapterFactory: adapterFactory,
		encryptor:      encryptor,
	}, nil
}

// Create 创建账户
func (s *accountService) Create(ctx context.Context, req *CreateAccountRequest) (*model.Account, error) {
	// 生成唯一 UID
	uid := uuid.New().String()

	// 加密密码
	encryptedPassword, err := s.encryptor.Encrypt(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	// 创建账户模型
	account := &model.Account{
		UID:                  uid,
		Email:                req.Email,
		Provider:             req.Provider,
		Protocol:             req.Protocol,
		AuthType:             req.AuthType,
		EncryptedCredentials: encryptedPassword,
		SyncEnabled:          req.SyncEnabled,
		SyncInterval:         req.SyncInterval,
		// 通用邮箱配置
		IMAPHost:   req.IMAPHost,
		IMAPPort:   req.IMAPPort,
		POP3Host:   req.POP3Host,
		POP3Port:   req.POP3Port,
		Encryption: req.Encryption,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 设置默认值
	if account.SyncInterval == 0 {
		account.SyncInterval = 5 // 默认 5 分钟
	}

	// 保存到数据库
	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// GetByUID 根据 UID 获取账户
func (s *accountService) GetByUID(ctx context.Context, uid string) (*model.Account, error) {
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found: %s", uid)
	}
	return account, nil
}

// List 获取账户列表
func (s *accountService) List(ctx context.Context) ([]*model.Account, error) {
	// 获取所有账户（不分页）
	accounts, _, err := s.accountRepo.List(ctx, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	return accounts, nil
}

// Update 更新账户
func (s *accountService) Update(ctx context.Context, uid string, req *UpdateAccountRequest) (*model.Account, error) {
	// 获取现有账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Email != nil {
		account.Email = *req.Email
	}
	if req.Password != nil {
		encryptedPassword, err := s.encryptor.Encrypt(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		account.EncryptedCredentials = encryptedPassword
	}
	if req.SyncEnabled != nil {
		account.SyncEnabled = *req.SyncEnabled
	}
	if req.SyncInterval != nil {
		account.SyncInterval = *req.SyncInterval
	}
	// 更新通用邮箱配置
	if req.IMAPHost != nil {
		account.IMAPHost = *req.IMAPHost
	}
	if req.IMAPPort != nil {
		account.IMAPPort = *req.IMAPPort
	}
	if req.POP3Host != nil {
		account.POP3Host = *req.POP3Host
	}
	if req.POP3Port != nil {
		account.POP3Port = *req.POP3Port
	}
	if req.Encryption != nil {
		account.Encryption = *req.Encryption
	}

	account.UpdatedAt = time.Now()

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return account, nil
}

// Delete 删除账户
func (s *accountService) Delete(ctx context.Context, uid string) error {
	// 先获取账户以获得 ID
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	if err := s.accountRepo.Delete(ctx, account.ID); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

// TestConnection 测试账户连接
func (s *accountService) TestConnection(ctx context.Context, uid string) error {
	// 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	// 解密密码
	password, err := s.encryptor.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	// 创建凭证
	credentials := &adapter.Credentials{
		Email:    account.Email,
		Password: password,
		AuthType: account.AuthType,
	}

	// 设置服务器配置
	switch account.Provider {
	case "icloud":
		credentials.Host = "imap.mail.me.com"
		credentials.Port = 993
		credentials.TLS = true
	case "qq":
		credentials.Host = "imap.qq.com"
		credentials.Port = 993
		credentials.TLS = true
	case "163":
		credentials.Host = "imap.163.com"
		credentials.Port = 993
		credentials.TLS = true
	case "gmail":
		credentials.Host = "imap.gmail.com"
		credentials.Port = 993
		credentials.TLS = true
	case "outlook":
		credentials.Host = "outlook.office365.com"
		credentials.Port = 993
		credentials.TLS = true
	case "generic":
		// 使用用户配置的服务器信息
		if account.Protocol == "imap" {
			credentials.Host = account.IMAPHost
			credentials.Port = account.IMAPPort
		} else if account.Protocol == "pop3" {
			credentials.Host = account.POP3Host
			credentials.Port = account.POP3Port
		}
		
		// 智能修复常见的配置错误
		if credentials.Host == "mail.linuxdo.org" {
			log.Printf("Auto-fixing incorrect host: %s -> mail.linux.do", credentials.Host)
			credentials.Host = "mail.linux.do"
		}
		
		// 设置加密方式
		switch account.Encryption {
		case "ssl":
			credentials.TLS = true
		case "starttls":
			credentials.StartTLS = true
		case "none":
			credentials.TLS = false
			credentials.StartTLS = false
		default:
			credentials.TLS = true // 默认使用 SSL
		}
		
		// 验证必要的配置
		if credentials.Host == "" || credentials.Port == 0 {
			return fmt.Errorf("generic provider requires host and port configuration")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", account.Provider)
	}

	// 创建适配器
	provider, err := s.adapterFactory.CreateProviderFromAccount(
		account.Provider,
		account.Protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// 测试连接
	return provider.TestConnection(ctx)
}

// SetStatus 设置账户状态
func (s *accountService) SetStatus(ctx context.Context, uid string, status string) error {
	// 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	// 更新状态
	account.Status = status
	account.UpdatedAt = time.Now()

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	return nil
}

// DisableAccount 禁用账户
func (s *accountService) DisableAccount(ctx context.Context, uid string) error {
	return s.SetStatus(ctx, uid, "disabled")
}

// EnableAccount 启用账户
func (s *accountService) EnableAccount(ctx context.Context, uid string) error {
	return s.SetStatus(ctx, uid, "active")
}

// ClearSyncError 清除同步错误状态
func (s *accountService) ClearSyncError(ctx context.Context, uid string) error {
	// 使用 repository 的 UpdateSyncStatus 方法清除错误
	return s.accountRepo.UpdateSyncStatus(ctx, uid, "", "")
}
