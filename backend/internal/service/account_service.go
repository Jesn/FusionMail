package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
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

	// GetByEmail 根据邮箱地址获取账户
	GetByEmail(ctx context.Context, email string) (*model.Account, error)

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

	// ListWithDeleted 获取所有账号（包括软删除的）
	ListWithDeleted(ctx context.Context) ([]*model.Account, error)

	// Restore 恢复软删除的账号
	Restore(ctx context.Context, uid string) error

	// ForceDelete 永久删除账号
	ForceDelete(ctx context.Context, uid string) error
}

// CreateAccountRequest 创建账户请求
type CreateAccountRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Provider     string `json:"provider" binding:"required"`
	Protocol     string `json:"protocol" binding:"required"`
	AuthType     string `json:"auth_type" binding:"required"`
	Password     string `json:"password"`
	SyncEnabled  bool   `json:"sync_enabled"`
	SyncInterval int    `json:"sync_interval"`
	// 短效认证字段
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	// 通用邮箱配置字段
	IMAPHost   string `json:"imap_host,omitempty"`
	IMAPPort   int    `json:"imap_port,omitempty"`
	POP3Host   string `json:"pop3_host,omitempty"`
	POP3Port   int    `json:"pop3_port,omitempty"`
	Encryption string `json:"encryption,omitempty"`
	// 删除策略
	ServerDeletePolicy string `json:"server_delete_policy,omitempty"` // 'off' 或 'soft'
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
	// 删除策略
	ServerDeletePolicy *string `json:"server_delete_policy,omitempty"` // 'off' 或 'soft'
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
	// 检查邮箱是否已存在
	existing, _ := s.accountRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, dto.NewAPIErrorWithMessage(
			dto.ErrAccountExists,
			fmt.Sprintf("邮箱账户 %s 已存在", req.Email),
		)
	}

	// 生成唯一 UID
	uid := uuid.New().String()

	// 根据认证类型加密凭证
	var encryptedCredentials string
	var err error

	if req.AuthType == "quick" {
		// 短效认证：加密 JSON 格式的凭证
		credentials := map[string]interface{}{
			"email":         req.Email,
			"auth_type":     "quick",
			"refresh_token": req.RefreshToken,
			"client_id":     req.ClientID,
		}

		credentialsJSON, err := json.Marshal(credentials)
		if err != nil {
			log.Printf("failed to marshal credentials: %v", err)
			return nil, fmt.Errorf("failed to marshal credentials: %w", err)
		}

		encryptedCredentials, err = s.encryptor.Encrypt(string(credentialsJSON))
		if err != nil {
			log.Printf("failed to encrypt credentials: %v", err)
			return nil, fmt.Errorf("encryption error: %w", err)
		}
	} else {
		// 密码认证：直接加密密码
		encryptedCredentials, err = s.encryptor.Encrypt(req.Password)
		if err != nil {
			log.Printf("failed to encrypt password: %v", err)
			return nil, fmt.Errorf("encryption error: %w", err)
		}
	}

	// 创建账户模型
	account := &model.Account{
		UID:                  uid,
		Email:                req.Email,
		Provider:             req.Provider,
		Protocol:             req.Protocol,
		AuthType:             req.AuthType,
		EncryptedCredentials: encryptedCredentials,
		SyncEnabled:          req.SyncEnabled,
		SyncInterval:         req.SyncInterval,
		Status:               "active",
		// 通用邮箱配置
		IMAPHost:   req.IMAPHost,
		IMAPPort:   req.IMAPPort,
		POP3Host:   req.POP3Host,
		POP3Port:   req.POP3Port,
		Encryption: req.Encryption,
		// 删除策略（默认关闭）
		ServerDeletePolicy: "off",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 设置默认值
	if account.SyncInterval == 0 {
		account.SyncInterval = 2 // 默认 2 分钟
	}

	// 如果请求中指定了删除策略，则使用请求中的值
	if req.ServerDeletePolicy != "" {
		account.ServerDeletePolicy = req.ServerDeletePolicy
	}

	// 保存到数据库
	if err := s.accountRepo.Create(ctx, account); err != nil {
		log.Printf("failed to create account in database: email=%s, error=%v", req.Email, err)
		return nil, fmt.Errorf("database error: %w", err)
	}

	log.Printf("account created successfully: uid=%s, email=%s", account.UID, account.Email)
	return account, nil
}

// GetByUID 根据 UID 获取账户
func (s *accountService) GetByUID(ctx context.Context, uid string) (*model.Account, error) {
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		log.Printf("database error when finding account: uid=%s, error=%v", uid, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, dto.NewAPIError(dto.ErrAccountNotFound)
	}
	return account, nil
}

// GetByEmail 根据邮箱地址获取账户
func (s *accountService) GetByEmail(ctx context.Context, email string) (*model.Account, error) {
	account, err := s.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		log.Printf("database error when finding account: email=%s, error=%v", email, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, nil // 返回 nil，不返回错误，让调用者处理
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
	// 更新删除策略
	if req.ServerDeletePolicy != nil {
		account.ServerDeletePolicy = *req.ServerDeletePolicy
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
		return err // 已经是 APIError 或系统错误
	}

	// 解密密码
	password, err := s.encryptor.Decrypt(account.EncryptedCredentials)
	if err != nil {
		log.Printf("failed to decrypt credentials: uid=%s, error=%v", uid, err)
		return fmt.Errorf("decryption error: %w", err)
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
			return dto.NewAPIErrorWithMessage(
				dto.ErrAccountInvalid,
				"通用邮箱需要配置服务器地址和端口",
			)
		}
	default:
		return dto.NewAPIErrorWithMessage(
			dto.ErrAccountInvalid,
			fmt.Sprintf("不支持的邮箱提供商: %s", account.Provider),
		)
	}

	// 创建适配器
	provider, err := s.adapterFactory.CreateProviderFromAccount(
		account.Provider,
		account.Protocol,
		credentials,
		nil, // 暂不支持代理
	)
	if err != nil {
		return dto.NewAPIErrorWithMessage(
			dto.ErrAccountInvalid,
			"账户配置无效: "+err.Error(),
		)
	}

	// 测试连接
	if err := provider.TestConnection(ctx); err != nil {
		return dto.NewAPIErrorWithMessage(
			dto.ErrConnectionFailed,
			"连接失败: "+err.Error(),
		)
	}

	return nil
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
// 重置所有自动禁用相关字段，允许账号重新同步
func (s *accountService) EnableAccount(ctx context.Context, uid string) error {
	// 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	// 重置所有禁用相关字段
	account.Status = "active"
	account.ConsecutiveAuthFailures = 0
	account.AutoDisabledAt = nil
	account.DisableReason = ""
	account.LastSyncError = ""
	account.UpdatedAt = time.Now()

	// 保存更新
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to enable account: %w", err)
	}

	log.Printf("[INFO] Manually re-enabled account %s (email: %s)", account.UID, account.Email)
	return nil
}

// ClearSyncError 清除同步错误状态
func (s *accountService) ClearSyncError(ctx context.Context, uid string) error {
	// 使用 repository 的 UpdateSyncStatus 方法清除错误
	return s.accountRepo.UpdateSyncStatus(ctx, uid, "", "")
}

// ListWithDeleted 获取所有账号（包括软删除的）
func (s *accountService) ListWithDeleted(ctx context.Context) ([]*model.Account, error) {
	accounts, err := s.accountRepo.FindAllWithDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts with deleted: %w", err)
	}
	return accounts, nil
}

// Restore 恢复软删除的账号
func (s *accountService) Restore(ctx context.Context, uid string) error {
	// 先检查账号是否存在（包括软删除的）
	account, err := s.accountRepo.FindByUIDIncludingDeleted(ctx, uid)
	if err != nil {
		return err
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 检查是否已恢复
	if account.DeletedAt == nil {
		return dto.NewAPIErrorWithMessage(
			dto.ErrAccountInvalid,
			"账号未被删除",
		)
	}

	// 恢复账号
	if err := s.accountRepo.Restore(ctx, uid); err != nil {
		return fmt.Errorf("failed to restore account: %w", err)
	}

	log.Printf("account restored successfully: uid=%s, email=%s", uid, account.Email)
	return nil
}

// ForceDelete 永久删除账号
func (s *accountService) ForceDelete(ctx context.Context, uid string) error {
	// 先检查账号是否存在（包括软删除的）
	account, err := s.accountRepo.FindByUIDIncludingDeleted(ctx, uid)
	if err != nil {
		return err
	}
	if account == nil {
		return dto.NewAPIError(dto.ErrAccountNotFound)
	}

	// 执行硬删除
	if err := s.accountRepo.ForceDelete(ctx, uid); err != nil {
		return fmt.Errorf("failed to force delete account: %w", err)
	}

	log.Printf("account force deleted: uid=%s, email=%s", uid, account.Email)
	return nil
}
