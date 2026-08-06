package service

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/cache"
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"

	"fusionmail/pkg/crypto"

	"github.com/redis/go-redis/v9"
)

// SettingService 配置管理服务
type SettingService struct {
	repo      *repository.SettingRepository
	cache     *cache.SettingCache
	encryptor crypto.Encryptor
}

// NewSettingService 创建配置服务实例
func NewSettingService(
	repo *repository.SettingRepository,
	redisClient *redis.Client,
	encryptor crypto.Encryptor,
) *SettingService {
	// 创建二级缓存：Redis TTL 30分钟，本地缓存10分钟
	cache := cache.NewSettingCache(
		redisClient,
		30*time.Minute, // Redis TTL
		10*time.Minute, // 本地缓存TTL
	)

	return &SettingService{
		repo:      repo,
		cache:     cache,
		encryptor: encryptor,
	}
}

// GetOptions 获取配置选项
type GetOptions struct {
	IncludeSensitive bool // 是否包含敏感配置（仅管理员）
	OnlyPublic       bool // 仅获取公开配置
}

// Get 获取单个配置项（自动处理解密）
func (s *SettingService) Get(ctx context.Context, userID *int64, category, key string, opts *GetOptions) (string, error) {
	// 从数据库获取配置
	setting, err := s.repo.Get(ctx, userID, category, key)
	if err != nil {
		return "", fmt.Errorf("setting service: failed to get setting [%s/%s] for user %v: %w", category, key, userID, err)
	}

	if setting == nil {
		return "", nil
	}

	// 权限检查
	if opts != nil {
		if opts.OnlyPublic && !setting.IsPublic {
			return "", dto.NewAPIError(dto.ErrSettingForbidden)
		}
		if opts.IncludeSensitive && setting.IsSensitive {
			// 管理员权限检查（这里可以添加更严格的权限检查）
			// 实际项目中应该检查用户角色
		}
	}

	// 处理敏感数据
	if setting.IsSensitive {
		// 解密敏感数据
		decrypted, err := s.encryptor.Decrypt(setting.Value)
		if err != nil {
			return "", dto.NewAPIErrorWithDetails(
				dto.ErrSettingDecryptFailed,
				"配置数据解密失败",
				map[string]interface{}{"key": key, "category": category, "error": err.Error()},
			)
		}
		return decrypted, nil
	}

	return setting.Value, nil
}

// GetByCategory 获取分类下的所有配置（带缓存）
func (s *SettingService) GetByCategory(ctx context.Context, userID *int64, category string, opts *GetOptions) (map[string]string, error) {
	// 1. 尝试从缓存获取
	cached, err := s.cache.Get(ctx, userID, category)
	if err == nil {
		// 缓存命中，需要过滤敏感数据
		filtered := make(map[string]string)
		for key, value := range cached.Settings {
			// 检查是否是敏感配置
			if s.isSensitive(category, key) {
				if opts != nil && !opts.IncludeSensitive {
					continue // 跳过敏感配置
				}
				// 解密敏感数据
				decrypted, err := s.encryptor.Decrypt(value)
				if err != nil {
					return nil, dto.NewAPIErrorWithDetails(
						dto.ErrSettingDecryptFailed,
						"配置数据解密失败",
						map[string]interface{}{"key": key, "category": category, "error": err.Error()},
					)
				}
				filtered[key] = decrypted
			} else {
				// 非敏感配置
				if opts == nil || !opts.OnlyPublic || s.isPublic(category, key) {
					filtered[key] = value
				}
			}
		}
		return filtered, nil
	}

	// 2. 缓存未命中，从数据库获取
	settings, err := s.repo.GetByCategory(ctx, userID, category)
	if err != nil {
		return nil, fmt.Errorf("setting service: failed to get settings for category [%s] for user %v: %w", category, userID, err)
	}

	result := make(map[string]string)
	for _, setting := range settings {
		key := setting.Key

		// 如果已经有这个 key（用户级配置），跳过系统级配置
		if _, exists := result[key]; exists {
			continue
		}

		// 权限检查
		if opts != nil && opts.OnlyPublic && !setting.IsPublic {
			continue
		}

		var value string
		if setting.IsSensitive {
			if opts != nil && !opts.IncludeSensitive {
				continue // 跳过敏感数据
			}
			// 解密敏感数据
			decrypted, err := s.encryptor.Decrypt(setting.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt %s: %w", key, err)
			}
			value = decrypted
		} else {
			value = setting.Value
		}

		result[key] = value
	}

	// 3. 存储到缓存（全量数据，加密存储）
	cacheData := make(map[string]string)
	for _, setting := range settings {
		var value string
		if setting.IsSensitive {
			// 加密存储到缓存
			encrypted, err := s.encryptor.Encrypt(setting.Value)
			if err != nil {
				// 记录错误但继续
				continue
			}
			value = encrypted
		} else {
			value = setting.Value
		}
		cacheData[setting.Key] = value
	}

	if err := s.cache.Set(ctx, userID, category, cacheData); err != nil {
		// 缓存失败不影响主流程
	}

	return result, nil
}

// Set 设置配置项（更新数据库和缓存）
func (s *SettingService) Set(ctx context.Context, userID *int64, category, key, value string, opts *GetOptions) error {
	// 判断是否为敏感配置
	isSensitive := s.isSensitive(category, key)

	// 如果是敏感配置，需要加密
	if isSensitive {
		// 加密敏感数据
		encrypted, err := s.encryptor.Encrypt(value)
		if err != nil {
			return dto.NewAPIErrorWithDetails(
				dto.ErrSettingEncryptFailed,
				"配置数据加密失败",
				map[string]interface{}{"key": key, "category": category, "error": err.Error()},
			)
		}
		value = encrypted
	}

	// 设置默认valueType
	valueType := s.getValueType(category, key)

	// 存储到数据库
	if err := s.repo.Set(ctx, userID, category, key, value, isSensitive, valueType); err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}

	// 刷新缓存
	if err := s.refreshCache(ctx, userID, category); err != nil {
		return fmt.Errorf("failed to refresh cache: %w", err)
	}

	return nil
}

// BatchSet 批量设置配置项
func (s *SettingService) BatchSet(ctx context.Context, userID *int64, category string, settings map[string]string, opts *GetOptions) error {
	// 准备批量插入的数据
	isSensitiveMap := make(map[string]bool)
	valueTypeMap := make(map[string]string)
	processedSettings := make(map[string]string)

	for key, value := range settings {
		isSensitive := s.isSensitive(category, key)
		isSensitiveMap[key] = isSensitive
		valueTypeMap[key] = s.getValueType(category, key)

		// 如果是敏感配置，需要加密
		if isSensitive {
			encrypted, err := s.encryptor.Encrypt(value)
			if err != nil {
				return fmt.Errorf("failed to encrypt %s: %w", key, err)
			}
			processedSettings[key] = encrypted
		} else {
			processedSettings[key] = value
		}
	}

	// 批量存储到数据库
	if err := s.repo.BatchSet(ctx, userID, category, processedSettings, isSensitiveMap, valueTypeMap); err != nil {
		return fmt.Errorf("failed to batch set settings: %w", err)
	}

	// 刷新缓存
	if err := s.refreshCache(ctx, userID, category); err != nil {
		return fmt.Errorf("failed to refresh cache: %w", err)
	}

	return nil
}

// Delete 删除配置项（更新数据库和缓存）
func (s *SettingService) Delete(ctx context.Context, userID *int64, category, key string) error {
	// 从数据库删除
	if err := s.repo.Delete(ctx, userID, category, key); err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	// 刷新缓存
	if err := s.refreshCache(ctx, userID, category); err != nil {
		return fmt.Errorf("failed to refresh cache: %w", err)
	}

	return nil
}

// DeleteByCategory 删除整个分类的配置
func (s *SettingService) DeleteByCategory(ctx context.Context, userID *int64, category string) error {
	if err := s.repo.DeleteByCategory(ctx, userID, category); err != nil {
		return fmt.Errorf("failed to delete settings: %w", err)
	}

	// 删除缓存
	if err := s.cache.Delete(ctx, userID, category); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// GetPublic 获取公开配置
func (s *SettingService) GetPublic(ctx context.Context, userID *int64, category string) (map[string]string, error) {
	opts := &GetOptions{
		OnlyPublic: true,
	}
	return s.GetByCategory(ctx, userID, category, opts)
}

// GetSystem 获取系统级配置
func (s *SettingService) GetSystem(ctx context.Context, category, key string) (string, error) {
	opts := &GetOptions{
		IncludeSensitive: true, // 系统配置可能包含敏感数据
	}
	return s.Get(ctx, nil, category, key, opts)
}

// SetSystem 设置系统级配置
func (s *SettingService) SetSystem(ctx context.Context, category, key, value string) error {
	return s.Set(ctx, nil, category, key, value, &GetOptions{
		IncludeSensitive: true,
	})
}

// GetSystemByCategory 批量获取系统级配置（按分类）
func (s *SettingService) GetSystemByCategory(ctx context.Context, category string) (map[string]string, error) {
	return s.GetByCategory(ctx, nil, category, &GetOptions{
		IncludeSensitive: true,
	})
}

// BatchSetSystem 批量设置系统级配置
func (s *SettingService) BatchSetSystem(ctx context.Context, category string, settings map[string]string) error {
	return s.BatchSet(ctx, nil, category, settings, &GetOptions{
		IncludeSensitive: true,
	})
}

// GetUser 获取用户级配置
func (s *SettingService) GetUser(ctx context.Context, userID int64, category, key string) (string, error) {
	opts := &GetOptions{
		IncludeSensitive: false,
	}
	return s.Get(ctx, &userID, category, key, opts)
}

// SetUser 设置用户级配置
func (s *SettingService) SetUser(ctx context.Context, userID int64, category, key, value string) error {
	return s.Set(ctx, &userID, category, key, value, &GetOptions{})
}

// Reset 重置配置为默认值
func (s *SettingService) Reset(ctx context.Context, userID *int64, category, key string) error {
	// 获取默认值
	defaultValue := s.getDefaultValue(category, key)
	if defaultValue == "" {
		return fmt.Errorf("no default value for setting %s:%s", category, key)
	}

	return s.Set(ctx, userID, category, key, defaultValue, &GetOptions{})
}

// Search 搜索配置项
func (s *SettingService) Search(ctx context.Context, userID *int64, keyword, category string, onlyPublic bool) ([]*model.Setting, error) {
	settings, err := s.repo.Search(ctx, userID, keyword, category, onlyPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to search settings: %w", err)
	}

	// 解密敏感配置
	for _, setting := range settings {
		if setting.IsSensitive {
			decrypted, err := s.encryptor.Decrypt(setting.Value)
			if err != nil {
				// 跳过解密失败的数据
				continue
			}
			setting.Value = decrypted
		}
	}

	return settings, nil
}

// WarmUp 预热缓存
func (s *SettingService) WarmUp(ctx context.Context, userID *int64, categories []string) error {
	// 预热缓存
	if err := s.cache.WarmUp(ctx, categories); err != nil {
		return fmt.Errorf("failed to warm up cache: %w", err)
	}

	// 加载常用分类的数据到缓存
	for _, category := range categories {
		if _, err := s.GetByCategory(ctx, userID, category, &GetOptions{}); err != nil {
			// 记录错误但继续
			fmt.Printf("Warning: failed to warm up category %s: %v\n", category, err)
		}
	}

	return nil
}

// GetStats 获取缓存统计信息
func (s *SettingService) GetStats() map[string]interface{} {
	cacheStats := s.cache.GetStats()

	// 添加更多有用的统计信息
	stats := map[string]interface{}{
		"cache":          cacheStats,
		"timestamp":      time.Now().Unix(),
		"service_status": "active",
	}

	// 添加数据库统计（可选）
	if dbStats, err := s.repo.GetStats(); err == nil {
		stats["database"] = dbStats
	}

	return stats
}

// refreshCache 刷新缓存
func (s *SettingService) refreshCache(ctx context.Context, userID *int64, category string) error {
	settings, err := s.repo.GetByCategory(ctx, userID, category)
	if err != nil {
		return err
	}

	cacheData := make(map[string]string)
	for _, setting := range settings {
		// 如果已经有这个 key（用户级配置），跳过系统级配置
		if _, exists := cacheData[setting.Key]; exists {
			continue
		}

		// 数据库中存储的值已经是加密后的（如果是敏感数据）
		// 直接存入缓存，不需要再次加密
		cacheData[setting.Key] = setting.Value
	}

	return s.cache.Set(ctx, userID, category, cacheData)
}

// isSensitive 检查配置是否敏感
func (s *SettingService) isSensitive(category, key string) bool {
	// 敏感配置映射
	sensitiveMap := map[string]map[string]bool{
		"oauth": {
			"gmail_client_secret":     true,
			"microsoft_client_secret": true,
		},
		"security": {
			"jwt_secret":      true,
			"encryption_key":  true,
			"master_password": true,
			"webhook_secret":  true,
		},
		"smtp": {
			"smtp_password": true,
		},
		"api": {
			"secret_keys": true,
		},
	}

	if categoryMap, ok := sensitiveMap[category]; ok {
		return categoryMap[key]
	}
	return false
}

// isPublic 检查配置是否公开
func (s *SettingService) isPublic(category, key string) bool {
	// 公开配置映射
	publicMap := map[string][]string{
		"ui": {
			"theme", "language", "email_page_size", "default_view",
		},
		"sync": {
			"enable_auto_sync", "sync_interval",
		},
		"notification": {
			"enable_desktop_notification", "enable_email_notification",
		},
	}

	if keys, ok := publicMap[category]; ok {
		for _, k := range keys {
			if k == key {
				return true
			}
		}
	}
	return false
}

// getValueType 获取配置值类型
func (s *SettingService) getValueType(category, key string) string {
	// 根据配置名称推断类型
	if key == "enable_auto_sync" || key == "enable_desktop_notification" || key == "enable_email_notification" {
		return "boolean"
	}
	if key == "email_page_size" || key == "sync_interval" || key == "rate_limit_site" || key == "rate_limit_public" {
		return "number"
	}
	if key == "secret_keys" || key == "oauth_refresh_tokens" {
		return "json"
	}
	return "string"
}

// getDefaultValue 获取配置默认值
func (s *SettingService) getDefaultValue(category, key string) string {
	defaultValues := map[string]map[string]string{
		"ui": {
			"theme":           "light",
			"language":        "zh-CN",
			"email_page_size": "50",
			"default_view":    "list",
		},
		"sync": {
			"enable_auto_sync": "true",
			"sync_interval":    "300",
		},
		"notification": {
			"enable_desktop_notification": "true",
			"enable_email_notification":   "true",
		},
		"security": {
			"session_timeout": "1440", // 24小时
		},
	}

	if categoryMap, ok := defaultValues[category]; ok {
		if value, ok := categoryMap[key]; ok {
			return value
		}
	}

	return ""
}
