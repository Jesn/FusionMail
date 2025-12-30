package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"

	"github.com/google/uuid"
)

// WebAPIProviderService WebAPI Provider 服务
type WebAPIProviderService struct {
	accountRepo  repository.AccountRepository
	providerRepo repository.ProviderRepository
	adapterRepo  repository.AdapterRepository
	emailRepo    repository.EmailRepository
	syncLogRepo  repository.SyncLogRepository
	factory      *webapi.WebAPIAdapterFactory
	cryptoSvc    *crypto.Service
	syncService  *WebAPISyncService
	log          *logger.Logger
}

// NewWebAPIProviderService 创建 WebAPI Provider 服务
func NewWebAPIProviderService(
	accountRepo repository.AccountRepository,
	providerRepo repository.ProviderRepository,
	adapterRepo repository.AdapterRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	cryptoSvc *crypto.Service,
) *WebAPIProviderService {
	return &WebAPIProviderService{
		accountRepo:  accountRepo,
		providerRepo: providerRepo,
		adapterRepo:  adapterRepo,
		emailRepo:    emailRepo,
		syncLogRepo:  syncLogRepo,
		factory:      webapi.NewWebAPIAdapterFactory(),
		cryptoSvc:    cryptoSvc,
		syncService:  NewWebAPISyncService(accountRepo, emailRepo, syncLogRepo),
		log:          logger.NewWithModule("WebAPIProviderService"),
	}
}

// ============================================
// CRUD 操作
// ============================================

// Create 创建 WebAPI Provider（实际创建 EmailAccount）
func (s *WebAPIProviderService) Create(ctx context.Context, name, serviceType, authDataJSON string) (*model.EmailAccount, error) {
	// 1. 验证服务类型
	if !s.factory.IsServiceTypeSupported(serviceType) {
		return nil, fmt.Errorf("不支持的服务类型: %s", serviceType)
	}

	// 2. 验证配置
	if err := s.factory.ValidateConfig(serviceType, authDataJSON); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 3. 查找 WebAPI Provider
	provider, err := s.findWebAPIProvider(ctx, serviceType)
	if err != nil {
		return nil, fmt.Errorf("查找 Provider 失败: %w", err)
	}

	// 4. 加密认证数据
	encryptedAuthData, err := s.cryptoSvc.Encrypt([]byte(authDataJSON))
	if err != nil {
		return nil, fmt.Errorf("加密认证数据失败: %w", err)
	}

	// 5. 从配置中提取邮箱地址（用于显示）
	email := s.extractEmailFromConfig(serviceType, authDataJSON)
	if email == "" {
		email = fmt.Sprintf("%s-%s", serviceType, uuid.New().String()[:8])
	}

	// 6. 查找 WebAPI 适配器
	adapterID, err := s.findWebAPIAdapterID(ctx)
	if err != nil {
		return nil, fmt.Errorf("查找 WebAPI 适配器失败: %w", err)
	}

	// 7. 创建 EmailAccount
	account := &model.EmailAccount{
		UID:                  uuid.New().String(),
		Email:                email,
		ProviderID:           provider.ID,
		AdapterID:            adapterID,
		EncryptedCredentials: string(encryptedAuthData),
		Status:               "active",
		LastSyncStatus:       "idle",
		SyncEnabled:          true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("创建账户失败: %w", err)
	}

	s.log.Info("创建 WebAPI Provider 成功: uid=%s, name=%s, type=%s", account.UID, name, serviceType)
	return account, nil
}

// List 获取 WebAPI Provider 列表
func (s *WebAPIProviderService) List(ctx context.Context, page, pageSize int) ([]*model.EmailAccount, int64, error) {
	// 查找所有 WebAPI Provider
	providers, err := s.findAllWebAPIProviders(ctx)
	if err != nil {
		return nil, 0, err
	}

	if len(providers) == 0 {
		return []*model.EmailAccount{}, 0, nil
	}

	// 获取 Provider IDs
	providerIDs := make([]int64, len(providers))
	for i, p := range providers {
		providerIDs[i] = p.ID
	}

	// 查询关联的 EmailAccount
	accounts, total, err := s.accountRepo.FindByProviderIDs(ctx, providerIDs, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询账户失败: %w", err)
	}

	return accounts, total, nil
}

// GetByUID 通过 UID 获取 WebAPI Provider
func (s *WebAPIProviderService) GetByUID(ctx context.Context, uid string) (*model.EmailAccount, error) {
	account, err := s.accountRepo.FindByUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}

	// 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	return account, nil
}

// Update 更新 WebAPI Provider
func (s *WebAPIProviderService) Update(ctx context.Context, uid, displayName, authDataJSON string) (*model.EmailAccount, error) {
	// 1. 获取现有账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 2. 更新认证数据
	if authDataJSON != "" {
		// 获取服务类型
		serviceType, err := s.getServiceTypeFromAccount(ctx, account)
		if err != nil {
			return nil, err
		}

		// 验证配置
		if err := s.factory.ValidateConfig(serviceType, authDataJSON); err != nil {
			return nil, fmt.Errorf("配置验证失败: %w", err)
		}

		// 加密
		encryptedAuthData, err := s.cryptoSvc.Encrypt([]byte(authDataJSON))
		if err != nil {
			return nil, fmt.Errorf("加密认证数据失败: %w", err)
		}
		account.EncryptedCredentials = encryptedAuthData

		// 更新邮箱地址
		email := s.extractEmailFromConfig(serviceType, authDataJSON)
		if email != "" {
			account.Email = email
		}
	}

	account.UpdatedAt = time.Now()

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("更新账户失败: %w", err)
	}

	s.log.Info("更新 WebAPI Provider 成功: uid=%s", uid)
	return account, nil
}

// Delete 删除 WebAPI Provider
func (s *WebAPIProviderService) Delete(ctx context.Context, uid string) error {
	// 验证账户存在且为 WebAPI 类型
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	if err := s.accountRepo.Delete(ctx, account.ID); err != nil {
		return fmt.Errorf("删除账户失败: %w", err)
	}

	s.log.Info("删除 WebAPI Provider 成功: uid=%s", uid)
	return nil
}

// ============================================
// 连接测试
// ============================================

// TestConnectionResult 连接测试结果
type TestConnectionResult struct {
	Success     bool   `json:"success"`               // 是否成功
	Message     string `json:"message"`               // 消息
	ServiceName string `json:"service_name"`          // 服务名称
	EmailCount  int    `json:"email_count,omitempty"` // 邮件数量（如果成功）
	Error       string `json:"error,omitempty"`       // 错误信息
}

// TestConnection 测试连接（使用配置）
func (s *WebAPIProviderService) TestConnection(ctx context.Context, serviceType, authDataJSON string) (*TestConnectionResult, error) {
	// 1. 验证服务类型
	if !s.factory.IsServiceTypeSupported(serviceType) {
		return &TestConnectionResult{
			Success: false,
			Message: "不支持的服务类型",
			Error:   fmt.Sprintf("服务类型 %s 不支持", serviceType),
		}, nil
	}

	// 2. 验证配置
	if err := s.factory.ValidateConfig(serviceType, authDataJSON); err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "配置验证失败",
			Error:   err.Error(),
		}, nil
	}

	// 3. 创建适配器
	adapter, err := s.factory.CreateAdapter(serviceType, authDataJSON)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "创建适配器失败",
			Error:   err.Error(),
		}, nil
	}

	// 4. 测试连接（尝试拉取少量邮件）
	emails, err := adapter.FetchEmails(ctx, time.Now().Add(-24*time.Hour), 1)
	if err != nil {
		return &TestConnectionResult{
			Success:     false,
			Message:     "连接测试失败",
			ServiceName: adapter.GetServiceName(),
			Error:       err.Error(),
		}, nil
	}

	return &TestConnectionResult{
		Success:     true,
		Message:     "连接测试成功",
		ServiceName: adapter.GetServiceName(),
		EmailCount:  len(emails),
	}, nil
}

// TestConnectionByUID 测试已存在 Provider 的连接
func (s *WebAPIProviderService) TestConnectionByUID(ctx context.Context, uid string) (*TestConnectionResult, error) {
	// 1. 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 2. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "解密认证数据失败",
			Error:   err.Error(),
		}, nil
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "获取服务类型失败",
			Error:   err.Error(),
		}, nil
	}

	// 4. 测试连接
	return s.TestConnection(ctx, serviceType, string(authDataBytes))
}

// ============================================
// 同步操作
// ============================================

// TriggerSync 手动触发同步
func (s *WebAPIProviderService) TriggerSync(ctx context.Context, uid string) error {
	// 1. 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return err
	}

	// 2. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return fmt.Errorf("解密认证数据失败: %w", err)
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return err
	}

	// 4. 创建适配器
	adapter, err := s.factory.CreateAdapter(serviceType, string(authDataBytes))
	if err != nil {
		return fmt.Errorf("创建适配器失败: %w", err)
	}

	// 5. 异步执行同步
	go func() {
		syncCtx := context.Background()
		_, syncErr := s.syncService.SyncProvider(syncCtx, adapter, account.UID)
		if syncErr != nil {
			s.log.Error("同步失败: uid=%s, error=%v", uid, syncErr)
		}
	}()

	s.log.Info("触发同步: uid=%s", uid)
	return nil
}

// SyncStatus 同步状态
type SyncStatus struct {
	Status       string     `json:"status"`                   // 同步状态
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`   // 上次同步时间
	LastSyncedID string     `json:"last_synced_id,omitempty"` // 上次同步的 ID
	EmailCount   int64      `json:"email_count"`              // 邮件数量
	ErrorMessage string     `json:"error_message,omitempty"`  // 错误信息
}

// GetSyncStatus 获取同步状态
func (s *WebAPIProviderService) GetSyncStatus(ctx context.Context, uid string) (*SyncStatus, error) {
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 获取邮件数量
	emailCount, err := s.emailRepo.CountByAccount(ctx, uid)
	if err != nil {
		emailCount = 0
	}

	return &SyncStatus{
		Status:       account.LastSyncStatus,
		LastSyncAt:   account.LastSyncAt,
		EmailCount:   emailCount,
		ErrorMessage: account.LastSyncError,
	}, nil
}

// ============================================
// 辅助方法
// ============================================

// findWebAPIProvider 查找 WebAPI Provider
func (s *WebAPIProviderService) findWebAPIProvider(ctx context.Context, serviceType string) (*model.Provider, error) {
	// 根据服务类型查找对应的 Provider
	// WebAPI Provider 的 Metadata 中包含 service_type
	providers, err := s.providerRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range providers {
		provider := p // 创建副本以获取指针
		if model.IsWebAPIProvider(&provider) {
			config, err := model.ParseWebAPIProviderConfig(provider.Metadata)
			if err != nil {
				continue
			}
			if config.ServiceType == serviceType {
				return &provider, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到服务类型 %s 对应的 Provider", serviceType)
}

// findAllWebAPIProviders 查找所有 WebAPI Provider
func (s *WebAPIProviderService) findAllWebAPIProviders(ctx context.Context) ([]*model.Provider, error) {
	providers, err := s.providerRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var webAPIProviders []*model.Provider
	for _, p := range providers {
		provider := p // 创建副本以获取指针
		if model.IsWebAPIProvider(&provider) {
			webAPIProviders = append(webAPIProviders, &provider)
		}
	}

	return webAPIProviders, nil
}

// findWebAPIAdapterID 查找 WebAPI 适配器 ID
func (s *WebAPIProviderService) findWebAPIAdapterID(ctx context.Context) (int64, error) {
	adapter, err := s.adapterRepo.FindByName(ctx, model.AdapterNameWebAPI)
	if err != nil {
		return 0, fmt.Errorf("查找 WebAPI 适配器失败: %w", err)
	}
	if adapter == nil {
		return 0, errors.New("WebAPI 适配器不存在")
	}
	return adapter.ID, nil
}

// isWebAPIAccount 检查账户是否为 WebAPI 类型
func (s *WebAPIProviderService) isWebAPIAccount(ctx context.Context, account *model.EmailAccount) bool {
	if account == nil || account.ProviderID == 0 {
		return false
	}

	provider, err := s.providerRepo.FindByID(ctx, account.ProviderID)
	if err != nil {
		return false
	}

	return model.IsWebAPIProvider(provider)
}

// getServiceTypeFromAccount 从账户获取服务类型
func (s *WebAPIProviderService) getServiceTypeFromAccount(ctx context.Context, account *model.EmailAccount) (string, error) {
	if account == nil || account.ProviderID == 0 {
		return "", errors.New("账户无效")
	}

	provider, err := s.providerRepo.FindByID(ctx, account.ProviderID)
	if err != nil {
		return "", fmt.Errorf("查找 Provider 失败: %w", err)
	}

	return model.GetWebAPIServiceType(provider)
}

// extractEmailFromConfig 从配置中提取邮箱地址
func (s *WebAPIProviderService) extractEmailFromConfig(serviceType, authDataJSON string) string {
	switch serviceType {
	case model.WebAPIServiceTypeCloudflareTempEmail:
		var config model.CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err == nil {
			if config.Email != "" {
				return config.Email
			}
			if config.Domain != "" {
				return fmt.Sprintf("admin@%s", config.Domain)
			}
		}

	case model.WebAPIServiceTypeCloudMail:
		var config model.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err == nil {
			if len(config.Accounts) > 0 {
				return config.Accounts[0].Email
			}
		}

	case model.WebAPIServiceTypeCustom:
		var config model.CustomWebAPIAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err == nil {
			if config.TargetEmail != "" {
				return config.TargetEmail
			}
			return config.ServiceName
		}
	}

	return ""
}
