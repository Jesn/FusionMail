package webapi

import (
	"encoding/json"
	"fmt"
	"sync"

	"fusionmail/internal/model"
)

// AdapterCreator 适配器创建函数类型
type AdapterCreator func(authDataJSON string) (WebAPIProvider, error)

// ConfigValidator 配置验证函数类型
type ConfigValidator func(authDataJSON string) error

// ============================================
// 适配器注册表
// ============================================

var (
	// 适配器创建函数注册表
	adapterCreators = make(map[string]AdapterCreator)
	// 配置验证函数注册表
	configValidators = make(map[string]ConfigValidator)
	// 服务模板注册表
	serviceTemplates = make(map[string]*ServiceTemplate)
	// 注册表锁
	registryMu sync.RWMutex
)

// RegisterAdapter 注册适配器创建函数
// 各适配器包在 init() 中调用此函数注册自己
func RegisterAdapter(serviceType string, creator AdapterCreator) {
	registryMu.Lock()
	defer registryMu.Unlock()
	adapterCreators[serviceType] = creator
}

// RegisterConfigValidator 注册配置验证函数
func RegisterConfigValidator(serviceType string, validator ConfigValidator) {
	registryMu.Lock()
	defer registryMu.Unlock()
	configValidators[serviceType] = validator
}

// RegisterServiceTemplate 注册服务模板
func RegisterServiceTemplate(template *ServiceTemplate) {
	registryMu.Lock()
	defer registryMu.Unlock()
	serviceTemplates[template.ServiceType] = template
}

// ============================================
// WebAPI 适配器工厂
// ============================================

// WebAPIAdapterFactory WebAPI 适配器工厂
// 根据服务类型创建对应的适配器实例
type WebAPIAdapterFactory struct{}

// NewWebAPIAdapterFactory 创建 WebAPI 适配器工厂
func NewWebAPIAdapterFactory() *WebAPIAdapterFactory {
	return &WebAPIAdapterFactory{}
}

// CreateAdapter 根据服务类型和配置创建适配器
// serviceType: 服务类型（cloudflare_temp_email, cloud_mail, custom）
// authDataJSON: 认证数据 JSON 字符串
func (f *WebAPIAdapterFactory) CreateAdapter(serviceType string, authDataJSON string) (WebAPIProvider, error) {
	registryMu.RLock()
	creator, ok := adapterCreators[serviceType]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("不支持的服务类型: %s", serviceType)
	}

	return creator(authDataJSON)
}

// CreateAdapterFromProvider 从 Provider 和 EmailAccount 创建适配器
// 便捷方法，自动从 Provider.Metadata 获取服务类型
func (f *WebAPIAdapterFactory) CreateAdapterFromProvider(provider *model.Provider, authDataJSON string) (WebAPIProvider, error) {
	if provider == nil {
		return nil, WrapError(ErrCodeConfigError, "provider 不能为空", nil)
	}

	// 从 Provider.Metadata 获取服务类型
	serviceType, err := model.GetWebAPIServiceType(provider)
	if err != nil {
		return nil, WrapError(ErrCodeConfigError, "获取服务类型失败", err)
	}

	return f.CreateAdapter(serviceType, authDataJSON)
}

// GetSupportedServiceTypes 获取支持的服务类型列表
func (f *WebAPIAdapterFactory) GetSupportedServiceTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	types := make([]string, 0, len(adapterCreators))
	for t := range adapterCreators {
		types = append(types, t)
	}
	return types
}

// IsServiceTypeSupported 检查服务类型是否支持
func (f *WebAPIAdapterFactory) IsServiceTypeSupported(serviceType string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	_, ok := adapterCreators[serviceType]
	return ok
}

// ValidateConfig 验证配置
func (f *WebAPIAdapterFactory) ValidateConfig(serviceType string, authDataJSON string) error {
	registryMu.RLock()
	validator, ok := configValidators[serviceType]
	registryMu.RUnlock()

	if !ok {
		// 如果没有注册验证器，尝试使用通用验证
		return f.validateConfigGeneric(serviceType, authDataJSON)
	}

	return validator(authDataJSON)
}

// validateConfigGeneric 通用配置验证
func (f *WebAPIAdapterFactory) validateConfigGeneric(serviceType string, authDataJSON string) error {
	switch serviceType {
	case model.WebAPIServiceTypeCloudflareTempEmail:
		var config model.CloudflareTempEmailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return WrapError(ErrCodeConfigError, "解析配置失败", err)
		}
		return config.Validate()

	case model.WebAPIServiceTypeCloudMail:
		var config model.CloudMailAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return WrapError(ErrCodeConfigError, "解析配置失败", err)
		}
		return config.Validate()

	case model.WebAPIServiceTypeCustom:
		var config model.CustomWebAPIAuthData
		if err := json.Unmarshal([]byte(authDataJSON), &config); err != nil {
			return WrapError(ErrCodeConfigError, "解析配置失败", err)
		}
		return config.Validate()

	default:
		return fmt.Errorf("不支持的服务类型: %s", serviceType)
	}
}

// ============================================
// 服务模板
// ============================================

// ServiceTemplate 服务模板
type ServiceTemplate struct {
	ServiceType   string                 `json:"service_type"`             // 服务类型
	Name          string                 `json:"name"`                     // 服务名称
	Description   string                 `json:"description"`              // 描述
	AccessModes   []string               `json:"access_modes"`             // 支持的访问模式
	AuthFields    []AuthField            `json:"auth_fields"`              // 认证字段
	DefaultConfig map[string]interface{} `json:"default_config,omitempty"` // 默认配置
}

// AuthField 认证字段定义
type AuthField struct {
	Name        string `json:"name"`        // 字段名
	Label       string `json:"label"`       // 显示标签
	Type        string `json:"type"`        // 字段类型：text, password, select, textarea
	Required    bool   `json:"required"`    // 是否必填
	Placeholder string `json:"placeholder"` // 占位符
	HelpText    string `json:"help_text"`   // 帮助文本
}

// GetServiceTemplates 获取所有服务模板
func (f *WebAPIAdapterFactory) GetServiceTemplates() []ServiceTemplate {
	registryMu.RLock()
	defer registryMu.RUnlock()

	templates := make([]ServiceTemplate, 0, len(serviceTemplates))
	for _, t := range serviceTemplates {
		templates = append(templates, *t)
	}
	return templates
}

// GetServiceTemplate 获取指定服务的模板
func (f *WebAPIAdapterFactory) GetServiceTemplate(serviceType string) *ServiceTemplate {
	registryMu.RLock()
	defer registryMu.RUnlock()

	return serviceTemplates[serviceType]
}
