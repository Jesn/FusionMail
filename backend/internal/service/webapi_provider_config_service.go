package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"fusionmail/internal/dto/webapiconfig"
	"fusionmail/internal/model"
)

// AccountConfigResponse 账户配置响应
type AccountConfigResponse struct {
	ServiceType string      `json:"service_type"` // 服务类型
	AuthData    interface{} `json:"auth_data"`    // 认证数据（脱敏后）
}

// GetAccountConfig 获取账户的 WebAPI 配置（脱敏）
func (s *WebAPIProviderCRUDService) GetAccountConfig(ctx context.Context, accountUID string) (*AccountConfigResponse, error) {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return nil, fmt.Errorf("账户未找到: %w", err)
	}

	// 2. 验证是否为 WebAPI 类型
	if !isWebAPIAccount(ctx, s.providerRepo, account) {
		return nil, errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取服务类型
	serviceType, err := getServiceTypeFromAccount(ctx, s.providerRepo, account)
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
func (s *WebAPIProviderCRUDService) UpdateAccountConfig(ctx context.Context, accountUID, serviceType, authDataJSON string) error {
	// 1. 获取账户
	account, err := s.accountRepo.FindByUID(ctx, accountUID)
	if err != nil {
		return fmt.Errorf("账户未找到: %w", err)
	}

	// 2. 验证是否为 WebAPI 类型
	if !isWebAPIAccount(ctx, s.providerRepo, account) {
		return errors.New("该账户不是 WebAPI 类型")
	}

	// 3. 获取当前服务类型
	currentServiceType, err := getServiceTypeFromAccount(ctx, s.providerRepo, account)
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

// sensitiveFieldPattern 匹配需要脱敏的结构体字段名（大小写不敏感）。
//
// 采用"按名字默认脱敏"而不是"逐个字段列举"：新增认证字段时默认就被脱敏，
// 忘记登记只会导致过度脱敏（用户看到 *** 而非真实值），而不会泄露凭据。
// 这是安全控制应有的 fail-closed 方向。
var sensitiveFieldPattern = regexp.MustCompile(`(?i)(token|password|secret|apikey|api_key|credential)`)

// maskSensitiveFields 递归脱敏结构体中名字命中 sensitiveFieldPattern 的字符串字段。
// 支持嵌套结构体与结构体指针；非导出字段（CanSet 为 false）跳过。
func maskSensitiveFields(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	maskSensitiveValue(rv.Elem())
}

func maskSensitiveValue(rv reflect.Value) {
	switch rv.Kind() {
	case reflect.Struct:
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Field(i)
			if !f.CanSet() {
				continue
			}
			switch f.Kind() {
			case reflect.String:
				if sensitiveFieldPattern.MatchString(rt.Field(i).Name) && f.String() != "" {
					f.SetString(maskString(f.String()))
				}
			case reflect.Struct:
				maskSensitiveValue(f)
			case reflect.Ptr:
				if !f.IsNil() && f.Elem().Kind() == reflect.Struct {
					maskSensitiveValue(f.Elem())
				}
			}
		}
	}
}

// sanitizeAuthData 脱敏认证数据（隐藏敏感信息）
//
// 三个分支统一走 maskSensitiveFields，避免"逐个列举敏感字段"导致的漏网：
// 历史实现里 CloudMail 的 Password 与 Cloudflare 的 WebhookSecret 都因此明文返回过。
func (s *WebAPIProviderCRUDService) sanitizeAuthData(serviceType, authDataJSON string) (interface{}, error) {
	switch serviceType {
	case webapiconfig.WebAPIServiceTypeCloudflareTempEmail:
		var config webapiconfig.CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, err
		}
		maskSensitiveFields(&config)
		return config, nil

	case webapiconfig.WebAPIServiceTypeCloudMail:
		var config webapiconfig.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, err
		}
		maskSensitiveFields(&config)
		return config, nil

	case webapiconfig.WebAPIServiceTypeCustom:
		var config webapiconfig.CustomWebAPIAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return nil, err
		}
		maskSensitiveFields(&config)
		return config, nil

	default:
		return nil, fmt.Errorf("不支持的服务类型: %s", serviceType)
	}
}

// mergeAuthData 合并认证数据（保留未修改的敏感字段）
func (s *WebAPIProviderCRUDService) mergeAuthData(ctx context.Context, account *model.EmailAccount, serviceType, newAuthDataJSON string) (string, error) {
	// 解密原有认证数据
	oldAuthDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return "", fmt.Errorf("解密原有认证数据失败: %w", err)
	}

	switch serviceType {
	case webapiconfig.WebAPIServiceTypeCloudflareTempEmail:
		var oldConfig, newConfig webapiconfig.CloudflareTempEmailAuthData
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

	case webapiconfig.WebAPIServiceTypeCloudMail:
		var oldConfig, newConfig webapiconfig.CloudMailAuthData
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

	case webapiconfig.WebAPIServiceTypeCustom:
		var oldConfig, newConfig webapiconfig.CustomWebAPIAuthData
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
