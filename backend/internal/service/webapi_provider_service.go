package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/adapter/webapi/cloudmail"
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

	// 4. 测试连接（直接调用适配器的 TestConnection 方法）
	if err := adapter.TestConnection(ctx); err != nil {
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

	// 5. 为 Cloudflare Temp Email 适配器设置 token 更新回调
	s.setupTokenUpdateCallback(adapter, account.UID, serviceType, string(authDataBytes))

	// 6. 异步执行同步
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

// setupTokenUpdateCallback 为适配器设置 token 更新回调
// 当 Cloudflare Temp Email 的 user_token 被刷新时，自动保存新 token
func (s *WebAPIProviderService) setupTokenUpdateCallback(adapter webapi.WebAPIProvider, accountUID, serviceType, authDataJSON string) {
	// 只有 Cloudflare Temp Email 需要设置回调
	if serviceType != model.WebAPIServiceTypeCloudflareTempEmail {
		return
	}

	// 类型断言获取 Cloudflare 适配器
	cfAdapter, ok := adapter.(interface {
		SetTokenUpdateCallback(func(string) error)
	})
	if !ok {
		return
	}

	// 设置回调函数
	cfAdapter.SetTokenUpdateCallback(func(newUserToken string) error {
		return s.updateUserToken(context.Background(), accountUID, authDataJSON, newUserToken)
	})
}

// updateUserToken 更新账户的 user_token
func (s *WebAPIProviderService) updateUserToken(ctx context.Context, accountUID, oldAuthDataJSON, newUserToken string) error {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("账户未找到: %w", err)
	}

	// 2. 解析原有配置
	var config model.CloudflareTempEmailAuthData
	if err := json.Unmarshal([]byte(oldAuthDataJSON), &config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 3. 更新 user_token
	config.UserToken = newUserToken

	// 4. 序列化新配置
	newAuthDataJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 5. 加密并保存
	encryptedAuthData, err := s.cryptoSvc.Encrypt(newAuthDataJSON)
	if err != nil {
		return fmt.Errorf("加密认证数据失败: %w", err)
	}

	account.EncryptedCredentials = encryptedAuthData
	account.UpdatedAt = time.Now()

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("更新账户失败: %w", err)
	}

	s.log.Info("user_token 已更新: uid=%s", accountUID)
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
		s.log.Error("查询所有 Provider 失败: %v", err)
		return nil, err
	}

	s.log.Info("查找 WebAPI Provider: serviceType=%s, 总 Provider 数量=%d", serviceType, len(providers))

	for i, p := range providers {
		provider := p // 创建副本以获取指针
		protocols, protocolErr := provider.GetSupportedProtocols()
		s.log.Info("检查 Provider[%d]: id=%d, name=%s, protocols=%v, protocolErr=%v, metadata=%s",
			i, provider.ID, provider.Name, protocols, protocolErr, provider.Metadata)

		isWebAPI := model.IsWebAPIProvider(&provider)
		s.log.Info("Provider[%d] IsWebAPIProvider=%v", i, isWebAPI)

		if isWebAPI {
			s.log.Info("找到 WebAPI Provider: id=%d, name=%s", provider.ID, provider.Name)
			config, err := model.ParseWebAPIProviderConfig(provider.Metadata)
			if err != nil {
				s.log.Warn("解析 Provider metadata 失败: id=%d, err=%v", provider.ID, err)
				continue
			}
			s.log.Info("Provider 服务类型: id=%d, configServiceType=%s, targetServiceType=%s",
				provider.ID, config.ServiceType, serviceType)
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
			// Admin 模式：使用第一个配置的域名生成显示邮箱
			domains := config.GetDomainList()
			if len(domains) > 0 {
				return fmt.Sprintf("admin@%s", domains[0])
			}
		}

	case model.WebAPIServiceTypeCloudMail:
		var config model.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err == nil {
			// 优先使用登录邮箱作为显示名称
			if config.Email != "" {
				return config.Email
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

// extractSyncModeFromConfig 从配置中提取同步模式
// 返回 "polling"（默认）或 "webhook"
func (s *WebAPIProviderService) extractSyncModeFromConfig(authDataJSON string) string {
	// 尝试解析为通用 JSON 结构
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
		return model.SyncModePolling
	}

	// 获取 sync_mode 字段
	if syncMode, ok := config["sync_mode"].(string); ok && syncMode != "" {
		if syncMode == model.SyncModeWebhook {
			return model.SyncModeWebhook
		}
	}

	return model.SyncModePolling
}

// ============================================
// 账户配置操作（用于编辑 WebAPI 账户）
// ============================================

// AccountConfigResponse 账户配置响应
type AccountConfigResponse struct {
	ServiceType string      `json:"service_type"` // 服务类型
	AuthData    interface{} `json:"auth_data"`    // 认证数据（脱敏后）
}

// GetAccountConfig 获取账户的 WebAPI 配置（脱敏）
func (s *WebAPIProviderService) GetAccountConfig(ctx context.Context, accountUID string) (*AccountConfigResponse, error) {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}

	// 2. 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	// 4. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("解密认证数据失败: %w", err)
	}

	// 5. 解析并脱敏认证数据
	sanitizedAuthData, err := s.sanitizeAuthData(serviceType, string(authDataBytes))
	if err != nil {
		return nil, fmt.Errorf("处理认证数据失败: %w", err)
	}

	return &AccountConfigResponse{
		ServiceType: serviceType,
		AuthData:    sanitizedAuthData,
	}, nil
}

// UpdateAccountConfig 更新账户的 WebAPI 配置
func (s *WebAPIProviderService) UpdateAccountConfig(ctx context.Context, accountUID, serviceType, authDataJSON string) error {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("账户未找到: %w", err)
	}

	// 2. 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, account) {
		return errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取当前服务类型
	currentServiceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return err
	}

	// 4. 验证服务类型是否匹配
	if serviceType != currentServiceType {
		return fmt.Errorf("服务类型不匹配: 期望 %s, 实际 %s", currentServiceType, serviceType)
	}

	// 5. 合并认证数据（保留未修改的字段）
	mergedAuthData, err := s.mergeAuthData(ctx, account, serviceType, authDataJSON)
	if err != nil {
		return fmt.Errorf("合并认证数据失败: %w", err)
	}

	// 6. 验证配置
	if err := s.factory.ValidateConfig(serviceType, mergedAuthData); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 7. 加密
	encryptedAuthData, err := s.cryptoSvc.Encrypt([]byte(mergedAuthData))
	if err != nil {
		return fmt.Errorf("加密认证数据失败: %w", err)
	}

	// 8. 更新账户
	account.EncryptedCredentials = encryptedAuthData
	account.UpdatedAt = time.Now()

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("更新账户失败: %w", err)
	}

	s.log.Info("更新 WebAPI 账户配置成功: uid=%s", accountUID)
	return nil
}

// sanitizeAuthData 脱敏认证数据（隐藏敏感信息）
func (s *WebAPIProviderService) sanitizeAuthData(serviceType, authDataJSON string) (interface{}, error) {
	switch serviceType {
	case model.WebAPIServiceTypeCloudflareTempEmail:
		var config model.CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, err
		}
		// 脱敏敏感字段
		if config.JWTToken != "" {
			config.JWTToken = maskString(config.JWTToken)
		}
		if config.UserToken != "" {
			config.UserToken = maskString(config.UserToken)
		}
		if config.AdminPassword != "" {
			config.AdminPassword = maskString(config.AdminPassword)
		}
		return config, nil

	case model.WebAPIServiceTypeCloudMail:
		var config model.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, err
		}
		// 脱敏敏感字段
		if config.JWTToken != "" {
			config.JWTToken = maskString(config.JWTToken)
		}
		return config, nil

	case model.WebAPIServiceTypeCustom:
		var config model.CustomWebAPIAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, err
		}
		// 脱敏敏感字段
		if config.Auth.Token != "" {
			config.Auth.Token = maskString(config.Auth.Token)
		}
		if config.Auth.APIKey != "" {
			config.Auth.APIKey = maskString(config.Auth.APIKey)
		}
		if config.Auth.Password != "" {
			config.Auth.Password = maskString(config.Auth.Password)
		}
		return config, nil

	default:
		return nil, fmt.Errorf("不支持的服务类型: %s", serviceType)
	}
}

// mergeAuthData 合并认证数据（保留未修改的敏感字段）
func (s *WebAPIProviderService) mergeAuthData(ctx context.Context, account *model.EmailAccount, serviceType, newAuthDataJSON string) (string, error) {
	// 解密原有认证数据
	oldAuthDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return "", fmt.Errorf("解密原有认证数据失败: %w", err)
	}

	switch serviceType {
	case model.WebAPIServiceTypeCloudflareTempEmail:
		var oldConfig, newConfig model.CloudflareTempEmailAuthData
		if err := json.Unmarshal(oldAuthDataBytes, &oldConfig); err != nil {
			return "", err
		}
		if err := json.Unmarshal([]byte(newAuthDataJSON), &newConfig); err != nil {
			return "", err
		}
		// 如果新值为空或为脱敏值，保留原值
		if newConfig.JWTToken == "" || isMaskedString(newConfig.JWTToken) {
			newConfig.JWTToken = oldConfig.JWTToken
		}
		if newConfig.UserToken == "" || isMaskedString(newConfig.UserToken) {
			newConfig.UserToken = oldConfig.UserToken
		}
		if newConfig.AdminPassword == "" || isMaskedString(newConfig.AdminPassword) {
			newConfig.AdminPassword = oldConfig.AdminPassword
		}
		// 保留其他必要字段
		if newConfig.BaseURL == "" {
			newConfig.BaseURL = oldConfig.BaseURL
		}
		if newConfig.AccessMode == "" {
			newConfig.AccessMode = oldConfig.AccessMode
		}
		if newConfig.Email == "" {
			newConfig.Email = oldConfig.Email
		}
		if newConfig.Domains == "" {
			newConfig.Domains = oldConfig.Domains
		}
		merged, _ := json.Marshal(newConfig)
		return string(merged), nil

	case model.WebAPIServiceTypeCloudMail:
		var oldConfig, newConfig model.CloudMailAuthData
		if err := json.Unmarshal(oldAuthDataBytes, &oldConfig); err != nil {
			return "", err
		}
		if err := json.Unmarshal([]byte(newAuthDataJSON), &newConfig); err != nil {
			return "", err
		}
		// 如果新值为空或为脱敏值，保留原值
		if newConfig.JWTToken == "" || isMaskedString(newConfig.JWTToken) {
			newConfig.JWTToken = oldConfig.JWTToken
		}
		// 保留其他必要字段
		if newConfig.BaseURL == "" {
			newConfig.BaseURL = oldConfig.BaseURL
		}
		// 保留登录凭据（如果新配置中没有提供）
		if newConfig.Email == "" {
			newConfig.Email = oldConfig.Email
		}
		if newConfig.Password == "" || isMaskedString(newConfig.Password) {
			newConfig.Password = oldConfig.Password
		}
		merged, _ := json.Marshal(newConfig)
		return string(merged), nil

	case model.WebAPIServiceTypeCustom:
		var oldConfig, newConfig model.CustomWebAPIAuthData
		if err := json.Unmarshal(oldAuthDataBytes, &oldConfig); err != nil {
			return "", err
		}
		if err := json.Unmarshal([]byte(newAuthDataJSON), &newConfig); err != nil {
			return "", err
		}
		// 如果新值为空或为脱敏值，保留原值
		if newConfig.Auth.Token == "" || isMaskedString(newConfig.Auth.Token) {
			newConfig.Auth.Token = oldConfig.Auth.Token
		}
		if newConfig.Auth.APIKey == "" || isMaskedString(newConfig.Auth.APIKey) {
			newConfig.Auth.APIKey = oldConfig.Auth.APIKey
		}
		if newConfig.Auth.Password == "" || isMaskedString(newConfig.Auth.Password) {
			newConfig.Auth.Password = oldConfig.Auth.Password
		}
		// 保留其他必要字段
		if newConfig.BaseURL == "" {
			newConfig.BaseURL = oldConfig.BaseURL
		}
		if newConfig.ServiceName == "" {
			newConfig.ServiceName = oldConfig.ServiceName
		}
		if newConfig.ListEndpoint == "" {
			newConfig.ListEndpoint = oldConfig.ListEndpoint
		}
		merged, _ := json.Marshal(newConfig)
		return string(merged), nil

	default:
		return newAuthDataJSON, nil
	}
}

// maskString 脱敏字符串（显示前4位和后4位）
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// isMaskedString 检查是否为脱敏字符串
func isMaskedString(s string) bool {
	return len(s) > 0 && (s == "****" || (len(s) >= 12 && s[4:8] == "****"))
}

// ============================================
// 子邮箱账户查询
// ============================================

// ChildAccountInfo 子邮箱账户信息
type ChildAccountInfo struct {
	UID         string     `json:"uid"`          // 账户 UID
	Email       string     `json:"email"`        // 邮箱地址
	Status      string     `json:"status"`       // 状态
	TotalEmails int64      `json:"total_emails"` // 邮件总数
	UnreadCount int        `json:"unread_count"` // 未读数量
	LastSyncAt  *time.Time `json:"last_sync_at"` // 上次同步时间
	CreatedAt   time.Time  `json:"created_at"`   // 创建时间
}

// GetChildAccounts 获取 WebAPI 账户关联的子邮箱列表
// 子邮箱是指 ParentAccountUID 等于父账户 UID 的账户
func (s *WebAPIProviderService) GetChildAccounts(ctx context.Context, parentUID string) ([]*ChildAccountInfo, error) {
	// 1. 验证父账户存在且为 WebAPI 类型
	parentAccount, err := s.accountRepo.FindByUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("父账户未找到: %w", err)
	}
	if parentAccount == nil {
		return nil, errors.New("父账户不存在")
	}

	// 2. 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, parentAccount) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 查找所有子账户（通过 ParentAccountUID 关联）
	childAccountList, err := s.accountRepo.FindByParentAccountUID(ctx, parentUID)
	if err != nil {
		return nil, fmt.Errorf("查询子账户失败: %w", err)
	}

	// 4. 构建子账户信息列表
	var childAccounts []*ChildAccountInfo
	for _, acc := range childAccountList {
		// 获取邮件数量
		emailCount, _ := s.emailRepo.CountByAccount(ctx, acc.UID)

		childAccounts = append(childAccounts, &ChildAccountInfo{
			UID:         acc.UID,
			Email:       acc.Email,
			Status:      acc.Status,
			TotalEmails: emailCount,
			UnreadCount: acc.UnreadCount,
			LastSyncAt:  acc.LastSyncAt,
			CreatedAt:   acc.CreatedAt,
		})
	}

	s.log.Info("获取子邮箱列表: parentUID=%s, 子邮箱数量=%d", parentUID, len(childAccounts))
	return childAccounts, nil
}

// ============================================
// 服务端子邮箱账户列表（通用）
// ============================================

// SubAccountInfo 服务端子邮箱账户信息（通用结构）
type SubAccountInfo struct {
	AccountID int    `json:"account_id"` // 账户 ID（Cloud Mail 使用）
	Email     string `json:"email"`      // 邮箱地址
	Name      string `json:"name"`       // 账户名称
}

// GetSubAccounts 获取服务端的子邮箱账户列表
// 支持 Cloud Mail 和 Cloudflare Temp Email
func (s *WebAPIProviderService) GetSubAccounts(ctx context.Context, accountUID string) ([]*SubAccountInfo, error) {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}
	if account == nil {
		return nil, errors.New("账户不存在")
	}

	// 2. 验证是否为 WebAPI 类型
	if !s.isWebAPIAccount(ctx, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	// 4. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("解密认证数据失败: %w", err)
	}

	// 5. 根据服务类型获取子邮箱列表
	switch serviceType {
	case model.WebAPIServiceTypeCloudMail:
		return s.getCloudMailSubAccounts(ctx, string(authDataBytes))

	case model.WebAPIServiceTypeCloudflareTempEmail:
		return s.getCloudflareTempEmailSubAccounts(ctx, string(authDataBytes))

	default:
		return nil, fmt.Errorf("该服务类型不支持获取子邮箱列表: %s", serviceType)
	}
}

// getCloudMailSubAccounts 获取 Cloud Mail 服务端的子邮箱列表
func (s *WebAPIProviderService) getCloudMailSubAccounts(ctx context.Context, authDataJSON string) ([]*SubAccountInfo, error) {
	// 创建适配器并连接
	adapter, err := s.factory.CreateAdapter(model.WebAPIServiceTypeCloudMail, authDataJSON)
	if err != nil {
		return nil, fmt.Errorf("创建适配器失败: %w", err)
	}

	// 连接到服务
	if err := adapter.Connect(ctx); err != nil {
		return nil, fmt.Errorf("连接服务失败: %w", err)
	}
	defer adapter.Disconnect()

	// 获取账户列表（通过类型断言获取 Cloud Mail 特有方法）
	cloudMailAdapter, ok := adapter.(*cloudmail.CloudMailAdapter)
	if !ok {
		return nil, errors.New("适配器不支持获取账户列表")
	}

	accounts := cloudMailAdapter.GetAccounts()

	// 转换为通用格式
	var result []*SubAccountInfo
	for _, acc := range accounts {
		result = append(result, &SubAccountInfo{
			AccountID: acc.AccountID,
			Email:     acc.Email,
			Name:      acc.Name,
		})
	}

	s.log.Info("获取 Cloud Mail 子邮箱列表: 账户数量=%d", len(result))
	return result, nil
}

// getCloudflareTempEmailSubAccounts 获取 Cloudflare Temp Email 的子邮箱列表
func (s *WebAPIProviderService) getCloudflareTempEmailSubAccounts(ctx context.Context, authDataJSON string) ([]*SubAccountInfo, error) {
	// 解析配置
	var config model.CloudflareTempEmailAuthData
	if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	var result []*SubAccountInfo

	// 根据访问模式返回不同的邮箱列表
	if config.AccessMode == model.WebAPIAccessModeSingle {
		// Single 模式：返回当前配置的邮箱或从 API 获取
		email := config.Email
		if email == "" {
			// 根据认证方式选择不同的 API 获取邮箱地址
			if config.HasUserToken() {
				// user_token 模式：调用 /user_api/bind_address 获取绑定的邮箱列表
				bindAddresses, err := s.fetchCloudflareTempEmailBindAddresses(ctx, &config)
				if err == nil && len(bindAddresses) > 0 {
					// 返回所有绑定的邮箱
					for _, addr := range bindAddresses {
						result = append(result, &SubAccountInfo{
							AccountID: addr.ID,
							Email:     addr.Name,
							Name:      extractNameFromEmail(addr.Name),
						})
					}
					s.log.Info("获取 Cloudflare Temp Email 子邮箱列表 (user_token): 模式=%s, 账户数量=%d", config.AccessMode, len(result))
					return result, nil
				} else if err != nil {
					s.log.Warn("获取绑定邮箱列表失败: %v", err)
				}
			} else if config.JWTToken != "" {
				// jwt_token 模式：调用 /api/settings 获取邮箱地址
				settings, err := s.FetchCloudflareTempEmailSettings(ctx, config.BaseURL, config.JWTToken)
				if err == nil && settings.Email != "" {
					email = settings.Email
				}
			}
		}
		if email != "" {
			result = append(result, &SubAccountInfo{
				AccountID: 0,
				Email:     email,
				Name:      extractNameFromEmail(email),
			})
		}
	} else if config.AccessMode == model.WebAPIAccessModeAdmin {
		// Admin 模式：从邮件列表中提取唯一的收件地址
		subAccounts, err := s.fetchCloudflareTempEmailAddresses(ctx, &config)
		if err != nil {
			s.log.Warn("获取 Cloudflare Temp Email 地址列表失败: %v", err)
			// 如果获取失败，返回配置的域名信息
			domains := config.GetDomainList()
			for i, domain := range domains {
				result = append(result, &SubAccountInfo{
					AccountID: i + 1,
					Email:     fmt.Sprintf("*@%s", domain),
					Name:      fmt.Sprintf("域名: %s", domain),
				})
			}
		} else {
			result = subAccounts
		}
	}

	s.log.Info("获取 Cloudflare Temp Email 子邮箱列表: 模式=%s, 账户数量=%d", config.AccessMode, len(result))
	return result, nil
}

// fetchCloudflareTempEmailAddresses 从 Cloudflare Temp Email API 获取地址列表
func (s *WebAPIProviderService) fetchCloudflareTempEmailAddresses(ctx context.Context, config *model.CloudflareTempEmailAuthData) ([]*SubAccountInfo, error) {
	// 构建请求 URL - 获取邮件列表
	url := config.BaseURL + "/admin/mails?offset=0&limit=100"

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置认证头
	req.Header.Set("x-admin-auth", config.AdminPassword)
	if config.JWTToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.JWTToken)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 - Cloudflare Temp Email Admin API 返回格式
	var response struct {
		Results []struct {
			Address     string   `json:"address"`
			ToAddresses []string `json:"to"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取唯一的邮箱地址
	addressMap := make(map[string]bool)
	for _, item := range response.Results {
		// 优先使用 address 字段
		if item.Address != "" {
			addressMap[item.Address] = true
		}
		// 也检查 to 字段
		for _, addr := range item.ToAddresses {
			if addr != "" {
				addressMap[addr] = true
			}
		}
	}

	// 如果配置了域名过滤，只返回匹配的地址
	var result []*SubAccountInfo
	idx := 1
	for addr := range addressMap {
		// 域名过滤
		if config.HasDomainFilter() && !config.MatchesDomain(addr) {
			continue
		}
		result = append(result, &SubAccountInfo{
			AccountID: idx,
			Email:     addr,
			Name:      extractNameFromEmail(addr),
		})
		idx++
	}

	return result, nil
}

// extractNameFromEmail 从邮箱地址提取名称
func extractNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

// CloudMailAccountInfo Cloud Mail 服务端账户信息（保留兼容性）
type CloudMailAccountInfo = SubAccountInfo

// GetCloudMailAccounts 获取 Cloud Mail 服务端的账户列表（兼容旧 API）
// 已废弃，请使用 GetSubAccounts
func (s *WebAPIProviderService) GetCloudMailAccounts(ctx context.Context, accountUID string) ([]*CloudMailAccountInfo, error) {
	return s.GetSubAccounts(ctx, accountUID)
}

// ============================================
// Cloudflare Temp Email 设置获取
// ============================================

// CloudflareTempEmailSettings Cloudflare Temp Email 设置信息
type CloudflareTempEmailSettings struct {
	Email   string   `json:"email"`             // 邮箱地址
	Domains []string `json:"domains,omitempty"` // 可用域名列表
}

// FetchCloudflareTempEmailSettings 获取 Cloudflare Temp Email 设置
// 通过 JWT Token 调用 /api/settings 接口获取邮箱地址和域名信息
func (s *WebAPIProviderService) FetchCloudflareTempEmailSettings(ctx context.Context, baseURL, jwtToken string) (*CloudflareTempEmailSettings, error) {
	if baseURL == "" {
		return nil, errors.New("base_url 不能为空")
	}
	if jwtToken == "" {
		return nil, errors.New("jwt_token 不能为空")
	}

	// 构建请求 URL
	url := baseURL + "/api/settings"

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("认证失败，请检查 JWT Token 是否有效")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 JSON
	// Cloudflare Temp Email 的 /api/settings 响应格式可能是：
	// { "address": "user@domain.com", "domains": ["domain1.com", "domain2.com"], ... }
	var settingsResp struct {
		Address string   `json:"address"` // 邮箱地址
		Domains []string `json:"domains"` // 可用域名列表
	}

	if err := json.Unmarshal(body, &settingsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 构建返回结果
	settings := &CloudflareTempEmailSettings{
		Email:   settingsResp.Address,
		Domains: settingsResp.Domains,
	}

	s.log.Info("获取 Cloudflare Temp Email 设置成功: email=%s, domains=%v", settings.Email, settings.Domains)
	return settings, nil
}

// BindAddressInfo 绑定邮箱地址信息
type BindAddressInfo struct {
	ID   int    `json:"id"`   // 邮箱 ID
	Name string `json:"name"` // 邮箱地址
}

// fetchCloudflareTempEmailBindAddresses 获取 user_token 模式下绑定的邮箱列表
// 调用 /user_api/bind_address 端点
func (s *WebAPIProviderService) fetchCloudflareTempEmailBindAddresses(ctx context.Context, config *model.CloudflareTempEmailAuthData) ([]*BindAddressInfo, error) {
	// 规范化 URL：去除前后空格和末尾斜杠
	baseURL := strings.TrimSpace(config.BaseURL)
	baseURL = strings.TrimRight(baseURL, "/")

	if baseURL == "" {
		return nil, errors.New("base_url 不能为空")
	}
	if config.UserToken == "" {
		return nil, errors.New("user_token 不能为空")
	}

	// 构建请求 URL
	url := baseURL + "/user_api/bind_address"

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头 - user_token 模式只需要 x-user-token 头
	req.Header.Set("x-user-token", config.UserToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("认证失败，请检查 user_token 是否有效")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应 JSON
	// /user_api/bind_address 响应格式：
	// { "results": [{"id": 623, "name": "ui_jesn89@ui.edu.kg", ...}] }
	var bindResp struct {
		Results []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &bindResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 转换为返回格式
	var result []*BindAddressInfo
	for _, item := range bindResp.Results {
		result = append(result, &BindAddressInfo{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	s.log.Info("获取 Cloudflare Temp Email 绑定邮箱列表成功: 数量=%d", len(result))
	return result, nil
}
