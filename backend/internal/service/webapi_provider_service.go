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
// WebAPIProviderCRUDService 负责 WebAPI Provider 的增删改查、认证配置与内部查询辅助。
//
// 为什么从 WebAPIProviderService 里拆出来：原类型跨 6 个文件、1661 行、31 个方法，
// 同时承担 CRUD、认证配置、连接测试、同步触发、子账户对账五类职责。
// Go 没有 partial class，跨文件拆分只带来"文件级"错觉。
type WebAPIProviderCRUDService struct {
	accountRepo  repository.AccountRepository
	providerRepo repository.ProviderRepository
	adapterRepo  repository.AdapterRepository
	factory      *webapi.WebAPIAdapterFactory
	cryptoSvc    *crypto.Service
	log          *logger.Logger
}

// NewWebAPIProviderCRUDService 创建 WebAPI Provider CRUD 服务。
func NewWebAPIProviderCRUDService(
	accountRepo repository.AccountRepository,
	providerRepo repository.ProviderRepository,
	adapterRepo repository.AdapterRepository,
	cryptoSvc *crypto.Service,
) *WebAPIProviderCRUDService {
	return &WebAPIProviderCRUDService{
		accountRepo:  accountRepo,
		providerRepo: providerRepo,
		adapterRepo:  adapterRepo,
		factory:      webapi.NewWebAPIAdapterFactory(),
		cryptoSvc:    cryptoSvc,
		log:          logger.NewWithModule("WebAPIProviderCRUD"),
	}
}

// WebAPIProviderConnectionService 负责 WebAPI 账户的连接测试与同步触发。
type WebAPIProviderConnectionService struct {
	accountRepo  repository.AccountRepository
	providerRepo repository.ProviderRepository
	emailRepo    repository.EmailRepository
	factory      *webapi.WebAPIAdapterFactory
	cryptoSvc    *crypto.Service
	syncService  *WebAPISyncService
	log          *logger.Logger
}

// NewWebAPIProviderConnectionService 创建 WebAPI Provider 连接服务。
func NewWebAPIProviderConnectionService(
	accountRepo repository.AccountRepository,
	providerRepo repository.ProviderRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	cryptoSvc *crypto.Service,
) *WebAPIProviderConnectionService {
	return &WebAPIProviderConnectionService{
		accountRepo:  accountRepo,
		providerRepo: providerRepo,
		emailRepo:    emailRepo,
		factory:      webapi.NewWebAPIAdapterFactory(),
		cryptoSvc:    cryptoSvc,
		syncService:  NewWebAPISyncService(accountRepo, emailRepo, syncLogRepo),
		log:          logger.NewWithModule("WebAPIProviderConnection"),
	}
}

// WebAPIProviderChildAccountService 负责 WebAPI 子账户的查询与对账。
type WebAPIProviderChildAccountService struct {
	accountRepo  repository.AccountRepository
	providerRepo repository.ProviderRepository
	emailRepo    repository.EmailRepository
	factory      *webapi.WebAPIAdapterFactory
	cryptoSvc    *crypto.Service
	log          *logger.Logger
}

// NewWebAPIProviderChildAccountService 创建 WebAPI Provider 子账户服务。
func NewWebAPIProviderChildAccountService(
	accountRepo repository.AccountRepository,
	providerRepo repository.ProviderRepository,
	emailRepo repository.EmailRepository,
	cryptoSvc *crypto.Service,
) *WebAPIProviderChildAccountService {
	return &WebAPIProviderChildAccountService{
		accountRepo:  accountRepo,
		providerRepo: providerRepo,
		emailRepo:    emailRepo,
		factory:      webapi.NewWebAPIAdapterFactory(),
		cryptoSvc:    cryptoSvc,
		log:          logger.NewWithModule("WebAPIProviderChildAccount"),
	}
}

// NewWebAPIProviderService 构造同时实现全部能力的组合 WebAPI Provider 服务。
//
// 内部由三个聚焦的类型组合而成，而不是一个 1661 行的大类型。
func NewWebAPIProviderService(
	accountRepo repository.AccountRepository,
	providerRepo repository.ProviderRepository,
	adapterRepo repository.AdapterRepository,
	emailRepo repository.EmailRepository,
	syncLogRepo repository.SyncLogRepository,
	cryptoSvc *crypto.Service,
) *WebAPIProviderService {
	return &WebAPIProviderService{
		crud:         NewWebAPIProviderCRUDService(accountRepo, providerRepo, adapterRepo, cryptoSvc),
		connection:   NewWebAPIProviderConnectionService(accountRepo, providerRepo, emailRepo, syncLogRepo, cryptoSvc),
		childAccount: NewWebAPIProviderChildAccountService(accountRepo, providerRepo, emailRepo, cryptoSvc),
	}
}

// WebAPIProviderService 把三个聚焦服务组合在一起。
//
// 保留它是因为 handler 依赖具体类型并使用了 15 个方法（跨全部三组）。
// 下面的方法全是纯转发，不含业务逻辑——这样"上帝类型"不会重新长出上帝实现。
type WebAPIProviderService struct {
	crud         *WebAPIProviderCRUDService
	connection   *WebAPIProviderConnectionService
	childAccount *WebAPIProviderChildAccountService
}

// ============================================
// CRUD 操作
// ============================================

// Create 创建 WebAPI Provider（实际创建 EmailAccount）
func (s *WebAPIProviderCRUDService) Create(ctx context.Context, name, serviceType, authDataJSON string, groupID *int64, syncInterval int, syncEnabled bool) (*model.EmailAccount, error) {
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
func (s *WebAPIProviderCRUDService) List(ctx context.Context, page, pageSize int) ([]*model.EmailAccount, int64, error) {
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

// loadWebAPIAccount 按 UID 读取账户并校验其为 WebAPI 类型。
//
// 提升为包级函数供 CRUD / Connection 两个服务共用：
// 若挂在 CRUD 上，Connection 就得反向依赖 CRUD 服务，
// 那只是把上帝对象换成了上帝依赖。
func loadWebAPIAccount(
	ctx context.Context,
	accountRepo repository.AccountRepository,
	providerRepo repository.ProviderRepository,
	uid string,
) (*model.EmailAccount, error) {
	account, err := accountRepo.FindByUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}
	if !isWebAPIAccount(ctx, providerRepo, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}
	return account, nil
}

// GetByUID 通过 UID 获取 WebAPI Provider
func (s *WebAPIProviderCRUDService) GetByUID(ctx context.Context, uid string) (*model.EmailAccount, error) {
	return loadWebAPIAccount(ctx, s.accountRepo, s.providerRepo, uid)
}

// Update 更新 WebAPI Provider
func (s *WebAPIProviderCRUDService) Update(ctx context.Context, uid, displayName, authDataJSON string) (*model.EmailAccount, error) {
	// 1. 获取现有账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 2. 更新认证数据
	if authDataJSON != "" {
		// 获取服务类型
		serviceType, err := getServiceTypeFromAccount(ctx, s.providerRepo, account)
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
func (s *WebAPIProviderCRUDService) Delete(ctx context.Context, uid string) error {
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

// 以下为转发方法：WebAPIProviderService 只做委托，不含业务逻辑。
func (c *WebAPIProviderService) Create(ctx context.Context, name, serviceType, authDataJSON string, groupID *int64, syncInterval int, syncEnabled bool) (*model.EmailAccount, error) {
	return c.crud.Create(ctx, name, serviceType, authDataJSON, groupID, syncInterval, syncEnabled)
}
func (c *WebAPIProviderService) List(ctx context.Context, page, pageSize int) ([]*model.EmailAccount, int64, error) {
	return c.crud.List(ctx, page, pageSize)
}
func (c *WebAPIProviderService) GetByUID(ctx context.Context, uid string) (*model.EmailAccount, error) {
	return c.crud.GetByUID(ctx, uid)
}
func (c *WebAPIProviderService) Update(ctx context.Context, uid, displayName, authDataJSON string) (*model.EmailAccount, error) {
	return c.crud.Update(ctx, uid, displayName, authDataJSON)
}
func (c *WebAPIProviderService) Delete(ctx context.Context, uid string) error {
	return c.crud.Delete(ctx, uid)
}
func (c *WebAPIProviderService) GetAccountConfig(ctx context.Context, accountUID string) (*AccountConfigResponse, error) {
	return c.crud.GetAccountConfig(ctx, accountUID)
}
func (c *WebAPIProviderService) UpdateAccountConfig(ctx context.Context, accountUID, serviceType, authDataJSON string) error {
	return c.crud.UpdateAccountConfig(ctx, accountUID, serviceType, authDataJSON)
}
func (c *WebAPIProviderService) TestConnection(ctx context.Context, serviceType, authDataJSON string) (*TestConnectionResult, error) {
	return c.connection.TestConnection(ctx, serviceType, authDataJSON)
}
func (c *WebAPIProviderService) TestConnectionByUID(ctx context.Context, uid string) (*TestConnectionResult, error) {
	return c.connection.TestConnectionByUID(ctx, uid)
}
func (c *WebAPIProviderService) TriggerSync(ctx context.Context, uid string) error {
	return c.connection.TriggerSync(ctx, uid)
}
func (c *WebAPIProviderService) GetSyncStatus(ctx context.Context, uid string) (*SyncStatus, error) {
	return c.connection.GetSyncStatus(ctx, uid)
}
func (c *WebAPIProviderService) GetChildAccounts(ctx context.Context, parentUID string, q ChildAccountListQuery) (*ChildAccountListResult, error) {
	return c.childAccount.GetChildAccounts(ctx, parentUID, q)
}
func (c *WebAPIProviderService) ReconcileChildAccounts(ctx context.Context, parentUID string) (*ChildReconcileResult, error) {
	return c.childAccount.ReconcileChildAccounts(ctx, parentUID)
}
func (c *WebAPIProviderService) GetSubAccounts(ctx context.Context, accountUID string) ([]*SubAccountInfo, error) {
	return c.childAccount.GetSubAccounts(ctx, accountUID)
}
func (c *WebAPIProviderService) GetCloudMailAccounts(ctx context.Context, accountUID string) ([]*CloudMailAccountInfo, error) {
	return c.childAccount.GetCloudMailAccounts(ctx, accountUID)
}
func (c *WebAPIProviderService) FetchCloudflareTempEmailSettings(ctx context.Context, baseURL, jwtToken string) (*CloudflareTempEmailSettings, error) {
	return c.childAccount.FetchCloudflareTempEmailSettings(ctx, baseURL, jwtToken)
}
