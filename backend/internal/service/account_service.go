package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"

	"github.com/google/uuid"
)

// AccountService 账户管理服务接口
type AccountService interface {
	// Create 创建账户
	Create(ctx context.Context, req *CreateAccountRequest) (*response.AccountResponse, error)

	// GetByUID 根据 UID 获取账户
	GetByUID(ctx context.Context, uid string) (*response.AccountResponse, error)

	// GetByEmail 根据邮箱地址获取账户
	GetByEmail(ctx context.Context, email string) (*response.AccountResponse, error)

	// List 获取账户列表
	List(ctx context.Context) ([]*response.AccountResponse, error)

	// ListWithFilter 带筛选条件的账户列表
	ListWithFilter(ctx context.Context, filter *AccountListFilter) (*AccountListResponse, error)

	// Update 更新账户
	Update(ctx context.Context, uid string, req *UpdateAccountRequest) (*response.AccountResponse, error)

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

	// BatchEnableAccounts 批量启用账户
	BatchEnableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error)

	// BatchDisableAccounts 批量禁用账户
	BatchDisableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error)

	// ClearSyncError 清除同步错误状态
	ClearSyncError(ctx context.Context, uid string) error

	// ListDeleted 获取回收站中的账号（仅软删除的）
	ListDeleted(ctx context.Context) ([]*response.AccountResponse, error)

	// Restore 恢复软删除的账号
	Restore(ctx context.Context, uid string) error

	// ForceDelete 永久删除账号（包括所有相关数据）
	ForceDelete(ctx context.Context, uid string) error

	// CleanupTrash 清理回收站（删除超过指定天数的软删除账号）
	CleanupTrash(ctx context.Context, days int) (int, error)

	// 新增：预加载关联的查询方法
	GetByUIDWithRelations(ctx context.Context, uid string) (*model.EmailAccount, error)
	ListWithRelations(ctx context.Context) ([]*model.EmailAccount, error)
	ListSyncEnabledWithRelations(ctx context.Context) ([]*model.EmailAccount, error)

	// 新增：根据邮箱自动匹配 Provider
	MatchProviderByEmail(ctx context.Context, email string) (*model.Provider, error)
}

// CreateAccountRequest 创建账户请求
type CreateAccountRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Provider     string `json:"provider" binding:"required"`  // 提供商名称（向后兼容）
	Protocol     string `json:"protocol" binding:"required"`  // 协议类型（向后兼容）
	AuthType     string `json:"auth_type" binding:"required"` // 认证类型（向后兼容）
	Password     string `json:"password"`
	SyncEnabled  bool   `json:"sync_enabled"`
	SyncInterval int    `json:"sync_interval"`
	// 新增：Provider 和 Adapter 外键
	ProviderID int64 `json:"provider_id,omitempty"` // 关联的提供商 ID
	AdapterID  int64 `json:"adapter_id,omitempty"`  // 用户选择的适配器 ID
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
	// 首次同步优化配置 (Requirements: 6.1, 6.2)
	FirstSyncDays    int `json:"first_sync_days,omitempty"`     // 首次同步天数，0 表示全量，默认 7
	BatchSize        int `json:"batch_size,omitempty"`          // 每批处理数量，默认 100
	MaxEmailsPerSync int `json:"max_emails_per_sync,omitempty"` // 单次同步最大邮件数，默认 5000
	// 分组 ID
	GroupID *int64 `json:"group_id,omitempty"` // 所属分组 ID，null 表示未分组
	// SMTP 发件功能
	SMTPEnabled bool `json:"smtp_enabled,omitempty"` // 是否启用 SMTP 发件功能
}

// UpdateAccountRequest 更新账户请求
type UpdateAccountRequest struct {
	Email        *string `json:"email,omitempty"`
	Password     *string `json:"password,omitempty"`
	SyncEnabled  *bool   `json:"sync_enabled,omitempty"`
	SyncInterval *int    `json:"sync_interval,omitempty"`
	// 分组 ID
	GroupID *int64 `json:"group_id"` // 所属分组 ID，null 表示未分组
	// 通用邮箱配置字段
	IMAPHost   *string `json:"imap_host,omitempty"`
	IMAPPort   *int    `json:"imap_port,omitempty"`
	POP3Host   *string `json:"pop3_host,omitempty"`
	POP3Port   *int    `json:"pop3_port,omitempty"`
	Encryption *string `json:"encryption,omitempty"`
	// 删除策略
	ServerDeletePolicy *string `json:"server_delete_policy,omitempty"` // 'off' 或 'soft'
	// 首次同步优化配置 (Requirements: 6.1, 6.2)
	FirstSyncDays    *int `json:"first_sync_days,omitempty"`     // 首次同步天数，0 表示全量
	BatchSize        *int `json:"batch_size,omitempty"`          // 每批处理数量
	MaxEmailsPerSync *int `json:"max_emails_per_sync,omitempty"` // 单次同步最大邮件数
}

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	Success     int                        `json:"success"`      // 成功数量
	Failed      int                        `json:"failed"`       // 失败数量
	Total       int                        `json:"total"`        // 总数量
	FailedItems []BatchOperationFailedItem `json:"failed_items"` // 失败项详情
}

// BatchOperationFailedItem 批量操作失败项
type BatchOperationFailedItem struct {
	UID   string `json:"uid"`   // 账户 UID
	Email string `json:"email"` // 邮箱地址
	Error string `json:"error"` // 错误信息
}

// accountService 账户管理服务实现
type accountService struct {
	accountRepo    repository.AccountRepository
	emailRepo      repository.EmailRepository
	providerRepo   repository.ProviderRepository
	adapterRepo    repository.AdapterRepository
	adapterFactory *adapter.Factory
	cryptoService  *crypto.Service
	logger         *logger.Logger
}

// NewAccountService 创建账户管理服务实例
func NewAccountService(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	providerRepo repository.ProviderRepository,
	adapterFactory *adapter.Factory,
	cryptoService *crypto.Service,
) (AccountService, error) {
	return &accountService{
		accountRepo:    accountRepo,
		emailRepo:      emailRepo,
		providerRepo:   providerRepo,
		adapterFactory: adapterFactory,
		cryptoService:  cryptoService,
		logger:         logger.NewWithModule("Account"),
	}, nil
}

// NewAccountServiceWithAdapterRepo 创建带 AdapterRepository 的账户管理服务实例
func NewAccountServiceWithAdapterRepo(
	accountRepo repository.AccountRepository,
	emailRepo repository.EmailRepository,
	providerRepo repository.ProviderRepository,
	adapterRepo repository.AdapterRepository,
	adapterFactory *adapter.Factory,
	cryptoService *crypto.Service,
) (AccountService, error) {
	return &accountService{
		accountRepo:    accountRepo,
		emailRepo:      emailRepo,
		providerRepo:   providerRepo,
		adapterRepo:    adapterRepo,
		adapterFactory: adapterFactory,
		cryptoService:  cryptoService,
		logger:         logger.NewWithModule("Account"),
	}, nil
}

// Create 创建账户
func (s *accountService) Create(ctx context.Context, req *CreateAccountRequest) (*response.AccountResponse, error) {
	// 检查邮箱是否已存在（仅包含未软删除账户）
	existing, _ := s.accountRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, dto.NewAPIErrorWithMessage(
			dto.ErrAccountExists,
			fmt.Sprintf("邮箱账户 %s 已存在", req.Email),
		)
	}

	// 检查是否存在同邮箱的软删除账户，如果有则清理
	deletedAccounts, err := s.accountRepo.FindDeletedByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check soft-deleted accounts: %w", err)
	}
	for _, acc := range deletedAccounts {
		if acc == nil {
			continue
		}

		// 删除该账号下的所有邮件
		if err := s.emailRepo.DeleteByAccountUID(ctx, acc.UID); err != nil {
			return nil, fmt.Errorf("failed to delete emails for soft-deleted account: %w", err)
		}

		// 永久删除软删除账号
		if err := s.accountRepo.ForceDelete(ctx, acc.UID); err != nil {
			return nil, fmt.Errorf("failed to force delete soft-deleted account: %w", err)
		}
	}

	// 生成唯一 UID
	uid := uuid.New().String()

	// 根据认证类型加密凭证
	var encryptedCredentials string

	if req.AuthType == "quick" {
		// 短效认证：加密 JSON 格式的凭证
		credentials := map[string]any{
			"email":         req.Email,
			"auth_type":     "quick",
			"refresh_token": req.RefreshToken,
			"client_id":     req.ClientID,
		}

		credentialsJSON, err := json.Marshal(credentials)
		if err != nil {
			s.logger.Error("序列化凭证失败: %v", err)
			return nil, fmt.Errorf("failed to marshal credentials: %w", err)
		}

		encryptedCredentials, err = s.cryptoService.Encrypt(credentialsJSON)
		if err != nil {
			s.logger.Error("加密凭证失败: %v", err)
			return nil, fmt.Errorf("encryption error: %w", err)
		}
	} else {
		// 密码认证：直接加密密码
		encryptedCredentials, err = s.cryptoService.Encrypt([]byte(req.Password))
		if err != nil {
			s.logger.Error("加密密码失败: %v", err)
			return nil, fmt.Errorf("encryption error: %w", err)
		}
	}

	// 从 Provider 获取默认配置
	var providerConfig *model.Provider
	var providerID int64
	var adapterID int64

	// 优先使用 provider_id，否则通过名称查找
	if req.ProviderID > 0 {
		providerID = req.ProviderID
		providerConfig, _ = s.providerRepo.FindByID(ctx, req.ProviderID)
	} else if req.Provider != "" && req.Provider != "generic" {
		var err error
		providerConfig, err = s.providerRepo.FindByName(ctx, req.Provider)
		if err != nil {
			s.logger.Error("查找 Provider 失败: provider=%s, error=%v", req.Provider, err)
			return nil, dto.NewAPIErrorWithMessage(
				dto.ErrInvalidRequest,
				fmt.Sprintf("不支持的邮箱提供商: %s", req.Provider),
			)
		}
		if providerConfig != nil {
			providerID = providerConfig.ID
		}
	}

	// 确保 providerID 有效（数据库字段是 not null）
	if providerID == 0 {
		return nil, dto.NewAPIErrorWithMessage(
			dto.ErrInvalidRequest,
			"必须指定有效的邮箱提供商",
		)
	}

	// 设置 adapter_id
	if req.AdapterID > 0 {
		adapterID = req.AdapterID
	} else if providerConfig != nil && providerConfig.DefaultAdapterID > 0 {
		// 使用 Provider 的默认适配器
		adapterID = providerConfig.DefaultAdapterID
	}

	// 确保 adapterID 有效（数据库字段是 not null）
	if adapterID == 0 {
		return nil, dto.NewAPIErrorWithMessage(
			dto.ErrInvalidRequest,
			"必须指定有效的适配器，请检查提供商配置",
		)
	}

	// 创建账户模型
	account := &model.EmailAccount{
		UID:                  uid,
		Email:                req.Email,
		EncryptedCredentials: encryptedCredentials,
		SyncEnabled:          req.SyncEnabled,
		SyncInterval:         req.SyncInterval,
		Status:               "active",
		// 删除策略（默认关闭）
		ServerDeletePolicy: "off",
		// 首次同步优化配置 (Requirements: 6.1)
		FirstSyncDays:    req.FirstSyncDays,
		BatchSize:        req.BatchSize,
		MaxEmailsPerSync: req.MaxEmailsPerSync,
		// 分组 ID
		GroupID:   req.GroupID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 只有当 providerID > 0 时才设置外键，避免外键约束错误
	if providerID > 0 {
		account.ProviderID = providerID
	}
	if adapterID > 0 {
		account.AdapterID = adapterID
	}

	// 设置 SMTP 启用状态（如果 Provider 存在且有 SMTP 配置）
	if providerConfig != nil && providerConfig.SMTPHost != "" && providerConfig.SMTPPort > 0 {
		account.SMTPEnabled = req.SMTPEnabled
		s.logger.Debug("Provider 支持 SMTP: provider=%s, host=%s, port=%d, enabled=%v",
			req.Provider, providerConfig.SMTPHost, providerConfig.SMTPPort, req.SMTPEnabled)
	}

	// 设置默认值
	if account.SyncInterval == 0 {
		account.SyncInterval = 2 // 默认 2 分钟
	}

	// 设置首次同步配置默认值 (Requirements: 6.3)
	if account.FirstSyncDays == 0 {
		account.FirstSyncDays = model.DefaultFirstSyncDays
	}
	if account.BatchSize == 0 {
		account.BatchSize = model.DefaultBatchSize
	}
	if account.MaxEmailsPerSync == 0 {
		account.MaxEmailsPerSync = model.DefaultMaxEmailsPerSync
	}

	// 如果请求中指定了删除策略，则使用请求中的值
	if req.ServerDeletePolicy != "" {
		account.ServerDeletePolicy = req.ServerDeletePolicy
	}

	// 保存到数据库
	if err := s.accountRepo.Create(ctx, account); err != nil {
		s.logger.Error("创建账户失败: email=%s, error=%v", req.Email, err)
		return nil, fmt.Errorf("database error: %w", err)
	}

	s.logger.Info("账户创建成功: uid=%s, email=%s", account.UID, account.Email)
	return toAccountResponse(account), nil
}

// GetByUID 根据 UID 获取账户
func (s *accountService) GetByUID(ctx context.Context, uid string) (*response.AccountResponse, error) {
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		s.logger.Error("查询账户失败: uid=%s, error=%v", uid, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, dto.NewAPIError(dto.ErrAccountNotFound)
	}
	return toAccountResponse(account), nil
}

// GetByEmail 根据邮箱地址获取账户
func (s *accountService) GetByEmail(ctx context.Context, email string) (*response.AccountResponse, error) {
	account, err := s.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		s.logger.Error("查询账户失败: email=%s, error=%v", email, err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if account == nil {
		return nil, nil // 返回 nil，不返回错误，让调用者处理
	}
	return toAccountResponse(account), nil
}

// toAccountResponse 将 model.EmailAccount 转换为 AccountResponse DTO
func toAccountResponse(a *model.EmailAccount) *response.AccountResponse {
	if a == nil {
		return nil
	}
	return &response.AccountResponse{
		ID:                 a.ID,
		UID:                a.UID,
		Email:              a.Email,
		ProviderID:         a.ProviderID,
		ProviderRef:        toProviderRef(a.ProviderRef),
		AdapterID:          a.AdapterID,
		AdapterRef:         toAdapterRef(a.AdapterRef),
		SMTPEnabled:        a.SMTPEnabled,
		ProxyEnabled:       a.ProxyEnabled,
		ProxyType:          a.ProxyType,
		ProxyHost:          a.ProxyHost,
		ProxyPort:          a.ProxyPort,
		ProxyUsername:      a.ProxyUsername,
		Status:             a.Status,
		AutoDisabledAt:     a.AutoDisabledAt,
		DisableReason:      a.DisableReason,
		SyncEnabled:        a.SyncEnabled,
		SyncInterval:       a.SyncInterval,
		LastSyncAt:         a.LastSyncAt,
		LastSyncStatus:     a.LastSyncStatus,
		LastSyncError:      a.LastSyncError,
		FirstSyncDays:      a.FirstSyncDays,
		BatchSize:          a.BatchSize,
		MaxEmailsPerSync:   a.MaxEmailsPerSync,
		ServerDeletePolicy: a.ServerDeletePolicy,
		GroupID:            a.GroupID,
		ParentAccountUID:   a.ParentAccountUID,
		TotalEmails:        a.TotalEmails,
		UnreadCount:        a.UnreadCount,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
	}
}

func toProviderRef(p *model.Provider) *response.Provider {
	if p == nil {
		return nil
	}
	return &response.Provider{
		ID:          p.ID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
	}
}

func toAdapterRef(a *model.Adapter) *response.Adapter {
	if a == nil {
		return nil
	}
	return &response.Adapter{
		ID:   a.ID,
		Name: a.Name,
	}
}

func toAccountResponseList(accounts []*model.EmailAccount) []*response.AccountResponse {
	result := make([]*response.AccountResponse, len(accounts))
	for i, a := range accounts {
		result[i] = toAccountResponse(a)
	}
	return result
}

// List 获取账户列表
func (s *accountService) List(ctx context.Context) ([]*response.AccountResponse, error) {
	// 获取所有账户（不分页），预加载 Provider 和 Adapter 关联
	accounts, _, err := s.accountRepo.ListWithRelations(ctx, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	return toAccountResponseList(accounts), nil
}

// AccountListFilter 账户列表筛选参数（从 repository 导出）
type AccountListFilter = repository.AccountListFilter

// AccountListResponse 账户列表响应
type AccountListResponse struct {
	Accounts   []*response.AccountResponse `json:"accounts"`
	Total      int64                       `json:"total"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"page_size"`
	TotalPages int                         `json:"total_pages"`
}

// ListWithFilter 带筛选条件的账户列表
func (s *accountService) ListWithFilter(ctx context.Context, filter *AccountListFilter) (*AccountListResponse, error) {
	// 默认值
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	accounts, total, err := s.accountRepo.ListWithFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &AccountListResponse{
		Accounts:   toAccountResponseList(accounts),
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Update 更新账户
