package service

import (
	"context"
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
func (s *WebAPIProviderService) Create(ctx context.Context, name, serviceType, authDataJSON string, groupID *int64, syncInterval int, syncEnabled bool) (*model.EmailAccount, error) {
	// 1. 验证服务类型
	if !s.factory.IsServiceTypeSupported(serviceType) {
		return nil, fmt.Errorf("不支持的服务类型: %s", serviceType)
	}

	// 2. 验证并规范化配置（去除空格、末尾斜杠等）
	normalizedAuthData, err := s.factory.ValidateAndNormalizeConfig(serviceType, authDataJSON)
	if err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 3. 查找 WebAPI Provider
	provider, err := s.findWebAPIProvider(ctx, serviceType)
	if err != nil {
		return nil, fmt.Errorf("查找 Provider 失败: %w", err)
	}

	// 4. 加密认证数据（使用规范化后的数据）
	encryptedAuthData, err := s.cryptoSvc.Encrypt([]byte(normalizedAuthData))
	if err != nil {
		return nil, fmt.Errorf("加密认证数据失败: %w", err)
	}

	// 5. 确定显示名称（Email 字段）
	// 优先级：1. 用户设置的显示名称 2. 从配置中提取的邮箱地址 3. 自动生成
	email := ""
	if name != "" {
		// 使用用户设置的显示名称
		email = name
	} else {
		// 尝试从配置中提取邮箱地址（使用规范化后的数据）
		email = s.extractEmailFromConfig(serviceType, normalizedAuthData)
	}
	if email == "" {
		// 如果都没有，则自动生成
		email = fmt.Sprintf("%s-%s", serviceType, uuid.New().String()[:8])
	}

	// 6. 设置同步间隔默认值
	if syncInterval <= 0 {
		syncInterval = 2 // 默认 2 分钟
	}

	// 7. 查找 WebAPI 适配器
	adapterID, err := s.findWebAPIAdapterID(ctx)
	if err != nil {
		return nil, fmt.Errorf("查找 WebAPI 适配器失败: %w", err)
	}

	// 8. 从配置中提取同步模式
	syncMode := s.extractSyncModeFromConfig(normalizedAuthData)

	// 9. 创建 EmailAccount
	account := &model.EmailAccount{
		UID:                  uuid.New().String(),
		Email:                email,
		ProviderID:           provider.ID,
		AdapterID:            adapterID,
		EncryptedCredentials: string(encryptedAuthData),
		Status:               "active",
		LastSyncStatus:       "idle",
		SyncEnabled:          syncEnabled,
		SyncInterval:         syncInterval,
		SyncModeField:        syncMode, // 设置同步模式
		GroupID:              groupID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("创建账户失败: %w", err)
	}

	s.log.Info("创建 WebAPI Provider 成功: uid=%s, email=%s, type=%s, groupID=%v", account.UID, email, serviceType, groupID)
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

		// 验证并规范化配置（去除空格、末尾斜杠等）
		normalizedAuthData, err := s.factory.ValidateAndNormalizeConfig(serviceType, authDataJSON)
		if err != nil {
			return nil, fmt.Errorf("配置验证失败: %w", err)
		}

		// 加密（使用规范化后的数据）
		encryptedAuthData, err := s.cryptoSvc.Encrypt([]byte(normalizedAuthData))
		if err != nil {
			return nil, fmt.Errorf("加密认证数据失败: %w", err)
		}
		account.EncryptedCredentials = encryptedAuthData

		// 更新邮箱地址（使用规范化后的数据）
		email := s.extractEmailFromConfig(serviceType, normalizedAuthData)
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
