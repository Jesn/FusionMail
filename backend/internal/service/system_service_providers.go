package service

import (
	"context"
	"time"

	"fusionmail/internal/adapter"
)

// GetSupportedProviders 获取支持的邮箱提供商列表
// 优先从缓存读取，缓存未命中时从数据库读取，数据库查询失败时使用硬编码配置作为降级
func (s *SystemService) GetSupportedProviders(ctx context.Context) ([]ProviderInfo, error) {
	// 1. 尝试从 Redis 缓存读取
	cacheKey := "providers:cache"
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var providers []ProviderInfo
			if err := s.redis.Get(ctx, cacheKey).Scan(&providers); err == nil {
				s.logger.Debug("从缓存获取提供商配置", "count", len(providers))
				return providers, nil
			}
		}
	}

	s.logger.Info("缓存未命中，从数据库获取提供商配置")

	// 2. 从数据库读取启用状态的提供商
	dbProviders, err := s.providerRepo.FindEnabled(ctx)
	if err != nil {
		s.logger.Error("从数据库获取提供商失败", "error", err)
		s.logger.Warn("使用硬编码的降级配置")
		// 降级：使用硬编码的默认配置
		return s.getFallbackProviders(), nil
	}

	// 3. 转换为 DTO
	var providers []ProviderInfo
	for _, p := range dbProviders {
		// 解析 supported_protocols JSON
		protocols, err := p.GetSupportedProtocols()
		if err != nil {
			s.logger.Warn("解析协议配置失败", "provider", p.Name, "error", err)
			continue
		}

		providerInfo := ProviderInfo{
			ID:                  p.ID,
			Name:                p.Name,
			DisplayName:         p.DisplayName,
			ProviderType:        providerNameToType(p.Name), // 从 name 推断类型（向后兼容）
			SupportedProtocols:  protocols,
			RecommendedProtocol: p.RecommendedProtocol,
			RequiresOAuth:       p.RequiresOAuth,
			Enabled:             p.Enabled,
			IMAPHost:            p.IMAPHost,
			IMAPPort:            p.IMAPPort,
			POP3Host:            p.POP3Host,
			POP3Port:            p.POP3Port,
			SMTPHost:            p.SMTPHost,
			SMTPPort:            p.SMTPPort,
			IMAPEncryption:      p.IMAPEncryption,
			POP3Encryption:      p.POP3Encryption,
			SMTPEncryption:      p.SMTPEncryption,
			EmailDomains:        p.EmailDomains,
			Description:         p.Description,
		}
		providers = append(providers, providerInfo)
	}

	s.logger.Info("从数据库获取提供商配置成功", "count", len(providers))

	// 4. 写入缓存（1小时 TTL）
	if s.redis != nil && len(providers) > 0 {
		err := s.redis.Set(ctx, cacheKey, providers, time.Hour).Err()
		if err != nil {
			s.logger.Warn("写入提供商配置缓存失败", "error", err)
		} else {
			s.logger.Debug("写入提供商配置到缓存", "cache_key", cacheKey, "ttl", "1h")
		}
	}

	return providers, nil
}

// getFallbackProviders 获取降级配置（保留原来的硬编码配置作为fallback）
func (s *SystemService) getFallbackProviders() []ProviderInfo {
	s.logger.Info("使用硬编码的降级配置")

	factory := adapter.NewFactory()
	// 获取所有支持的提供商
	providerNames := factory.GetSupportedProviders()

	var providers []ProviderInfo
	for _, name := range providerNames {
		info := factory.GetProviderInfo(name)
		if info != nil {
			providerInfo := ProviderInfo{
				ID:          0, // Fallback providers don't have real IDs
				Name:        name,
				DisplayName: info.DisplayName,
				ProviderType: func(name string) int {
					switch name {
					case "gmail":
						return 1 // ProviderTypeGmail
					case "outlook":
						return 2 // ProviderTypeOutlook
					case "icloud":
						return 3 // ProviderTypeIcloud
					case "qq":
						return 4 // ProviderTypeQQ
					case "163":
						return 5 // ProviderType163
					case "generic":
						return 6 // ProviderTypeGeneric
					default:
						return 6 // 默认使用通用类型
					}
				}(name), // 使用映射函数
				SupportedProtocols:  info.SupportedProtocols,
				RecommendedProtocol: info.RecommendedProtocol,
				RequiresOAuth:       info.RequiresOAuth,
				Enabled:             true, // Fallback providers are always enabled
				IMAPHost:            info.IMAPHost,
				IMAPPort:            info.IMAPPort,
				POP3Host:            info.POP3Host,
				POP3Port:            info.POP3Port,
			}
			providers = append(providers, providerInfo)
		}
	}

	s.logger.Warn("降级配置返回", "count", len(providers), "source", "hardcoded")
	return providers
}

// InvalidateProviderCache 失效提供商配置缓存
// 当提供商配置发生变更时调用此方法清除缓存
func (s *SystemService) InvalidateProviderCache(ctx context.Context) {
	cacheKey := "providers:cache"
	if s.redis != nil {
		err := s.redis.Del(ctx, cacheKey).Err()
		if err != nil {
			s.logger.Error("清除提供商配置缓存失败", "cache_key", cacheKey, "error", err)
		} else {
			s.logger.Info("清除提供商配置缓存成功", "cache_key", cacheKey)
		}
	}
}

// providerNameToType 将提供商名称转换为类型枚举值（用于向后兼容）
func providerNameToType(name string) int {
	switch name {
	case "gmail":
		return 1 // ProviderTypeGmail
	case "outlook":
		return 2 // ProviderTypeOutlook
	case "icloud":
		return 3 // ProviderTypeIcloud
	case "qq":
		return 4 // ProviderTypeQQ
	case "163":
		return 5 // ProviderType163
	default:
		return 6 // ProviderTypeGeneric
	}
}
