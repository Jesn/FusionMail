package handler

import (
	"fusionmail/internal/adapter/webapi"

	"github.com/gin-gonic/gin"
)

// WebAPIServicesHandler WebAPI 服务模板处理器
type WebAPIServicesHandler struct {
	factory *webapi.WebAPIAdapterFactory
}

// NewWebAPIServicesHandler 创建 WebAPI 服务模板处理器
func NewWebAPIServicesHandler() *WebAPIServicesHandler {
	return &WebAPIServicesHandler{
		factory: webapi.NewWebAPIAdapterFactory(),
	}
}

// ListServices 获取支持的服务列表
// @Summary 获取支持的 WebAPI 服务列表
// @Description 获取所有支持的 WebAPI 邮箱服务类型及其模板配置
// @Tags WebAPI Services
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/webapi/services [get]
func (h *WebAPIServicesHandler) ListServices(c *gin.Context) {
	templates := h.factory.GetServiceTemplates()

	// 转换为响应格式
	services := make([]ServiceInfo, 0, len(templates))
	for _, t := range templates {
		services = append(services, ServiceInfo{
			ServiceType:   t.ServiceType,
			Name:          t.Name,
			Description:   t.Description,
			AccessModes:   t.AccessModes,
			AuthFields:    convertAuthFields(t.AuthFields),
			DefaultConfig: t.DefaultConfig,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    services,
	})
}

// GetServiceDetail 获取服务详情
// @Summary 获取 WebAPI 服务详情
// @Description 获取指定服务类型的详细配置模板
// @Tags WebAPI Services
// @Produce json
// @Param service_type path string true "服务类型"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/webapi/services/{service_type} [get]
func (h *WebAPIServicesHandler) GetServiceDetail(c *gin.Context) {
	serviceType := c.Param("service_type")
	if serviceType == "" {
		c.JSON(400, gin.H{"success": false, "error": "服务类型不能为空"})
		return
	}

	template := h.factory.GetServiceTemplate(serviceType)
	if template == nil {
		c.JSON(404, gin.H{"success": false, "error": "服务类型不存在"})
		return
	}

	service := ServiceInfo{
		ServiceType:   template.ServiceType,
		Name:          template.Name,
		Description:   template.Description,
		AccessModes:   template.AccessModes,
		AuthFields:    convertAuthFields(template.AuthFields),
		DefaultConfig: template.DefaultConfig,
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    service,
	})
}

// GetSupportedTypes 获取支持的服务类型列表
// @Summary 获取支持的服务类型
// @Description 获取所有支持的 WebAPI 服务类型名称列表
// @Tags WebAPI Services
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/webapi/services/types [get]
func (h *WebAPIServicesHandler) GetSupportedTypes(c *gin.Context) {
	types := h.factory.GetSupportedServiceTypes()

	c.JSON(200, gin.H{
		"success": true,
		"data":    types,
	})
}

// ValidateConfig 验证配置
// @Summary 验证 WebAPI 配置
// @Description 验证指定服务类型的配置是否有效
// @Tags WebAPI Services
// @Accept json
// @Produce json
// @Param request body ValidateConfigRequest true "请求参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/webapi/services/validate [post]
func (h *WebAPIServicesHandler) ValidateConfig(c *gin.Context) {
	var req ValidateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": "请求参数无效: " + err.Error()})
		return
	}

	err := h.factory.ValidateConfig(req.ServiceType, req.AuthData)
	if err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"error":   err.Error(),
			"valid":   false,
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"valid":   true,
		"message": "配置验证通过",
	})
}

// ============================================
// 响应结构体
// ============================================

// ServiceInfo 服务信息
type ServiceInfo struct {
	ServiceType   string                 `json:"service_type"`             // 服务类型
	Name          string                 `json:"name"`                     // 服务名称
	Description   string                 `json:"description"`              // 描述
	AccessModes   []string               `json:"access_modes"`             // 支持的访问模式
	AuthFields    []AuthFieldInfo        `json:"auth_fields"`              // 认证字段
	DefaultConfig map[string]interface{} `json:"default_config,omitempty"` // 默认配置
}

// AuthFieldInfo 认证字段信息
type AuthFieldInfo struct {
	Name        string `json:"name"`        // 字段名
	Label       string `json:"label"`       // 显示标签
	Type        string `json:"type"`        // 字段类型
	Required    bool   `json:"required"`    // 是否必填
	Placeholder string `json:"placeholder"` // 占位符
	HelpText    string `json:"help_text"`   // 帮助文本
}

// ValidateConfigRequest 验证配置请求
type ValidateConfigRequest struct {
	ServiceType string `json:"service_type" binding:"required"` // 服务类型
	AuthData    string `json:"auth_data" binding:"required"`    // 认证数据 JSON
}

// convertAuthFields 转换认证字段
func convertAuthFields(fields []webapi.AuthField) []AuthFieldInfo {
	result := make([]AuthFieldInfo, len(fields))
	for i, f := range fields {
		result[i] = AuthFieldInfo{
			Name:        f.Name,
			Label:       f.Label,
			Type:        f.Type,
			Required:    f.Required,
			Placeholder: f.Placeholder,
			HelpText:    f.HelpText,
		}
	}
	return result
}
