package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"fusionmail/internal/model"
)

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
