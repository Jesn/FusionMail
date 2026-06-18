package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fusionmail/internal/model"
)

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
