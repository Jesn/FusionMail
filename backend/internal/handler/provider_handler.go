package handler

import (
	"fusionmail/internal/dto"
	"fusionmail/internal/dto/request"
	"fusionmail/internal/model"
	"fusionmail/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProviderHandler Provider 管理处理器
type ProviderHandler struct {
	service *service.ProviderService
}

// NewProviderHandler 创建 Provider 处理器
func NewProviderHandler(service *service.ProviderService) *ProviderHandler {
	return &ProviderHandler{
		service: service,
	}
}

// Create 创建 Provider
// @Summary 创建 Provider 配置
// @Description 为系统添加新的邮箱提供商配置
// @Tags Provider管理
// @Accept json
// @Produce json
// @Param request body request.ProviderRequest true "请求参数"
// @Success 201 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/providers [post]
func (h *ProviderHandler) Create(c *gin.Context) {
	// 绑定到请求 DTO 而不是 model.Provider：后者带 gorm tag，
	// 会让存储层字段与关联模型（ProviderAdapter / Adapter）进入 API 契约。
	var req request.ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	// 透传 adapter_ids：服务层在事务内同步 ProviderAdapter 关联
	created, err := h.service.Create(c.Request.Context(), req.ToModel(), req.AdapterIDs)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 转换为响应 DTO，带上 supported_adapters 关联
	resp, err := created.ToResponse()
	if err != nil {
		dto.InternalServerErrorResponse(c, "转换响应失败: ")
		return
	}

	dto.SuccessResponse(c, resp)
}

// List 获取所有 Provider 列表
// @Summary 获取 Provider 列表
// @Description 获取所有邮箱提供商配置列表
// @Tags Provider管理
// @Produce json
// @Success 200 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/providers/all [get]
func (h *ProviderHandler) List(c *gin.Context) {
	providers, err := h.service.List(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, providers)
}

// ListWithPagination 分页获取 Provider 列表
// @Summary 分页获取 Provider 列表
// @Description 分页获取所有邮箱提供商配置列表
// @Tags Provider管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} dto.Response{data=object{items=[]model.ProviderResponse,total=int64,page=int,page_size=int}}
// @Failure 500 {object} dto.Response
// @Router /api/v1/providers [get]
func (h *ProviderHandler) ListWithPagination(c *gin.Context) {
	// 获取分页参数
	page := 1
	pageSize := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	// 查询数据
	providers, total, err := h.service.ListWithPagination(c.Request.Context(), page, pageSize)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*model.ProviderResponse, 0, len(providers))
	for _, p := range providers {
		resp, err := p.ToResponse()
		if err != nil {
			dto.InternalServerErrorResponse(c, "Failed to convert provider: ")
			return
		}
		items = append(items, resp)
	}

	// 返回结果
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetByID 通过ID获取 Provider
// @Summary 获取 Provider 详情
// @Description 通过ID获取指定的邮箱提供商配置
// @Tags Provider管理
// @Produce json
// @Param id path int64 true "Provider ID"
// @Success 201 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/providers/{id} [get]
func (h *ProviderHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "Invalid provider ID: " + err.Error()})
		return
	}

	provider, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	dto.SuccessResponse(c, provider)
}

// UpdateByID 更新 Provider
// @Summary 更新 Provider 配置
// @Description 更新指定ID的邮箱提供商配置
// @Tags Provider管理
// @Accept json
// @Produce json
// @Param id path int64 true "Provider ID"
// @Param request body request.ProviderRequest true "请求参数"
// @Success 201 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/providers/{id} [put]
func (h *ProviderHandler) UpdateByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "Invalid provider ID: " + err.Error()})
		return
	}

	var req request.ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误")
		return
	}

	// 透传 adapter_ids：服务层在事务内同步 ProviderAdapter 关联（空/nil 表示清空）
	updated, err := h.service.UpdateByID(c.Request.Context(), id, req.ToModel(), req.AdapterIDs)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 转换为响应 DTO，带上 supported_adapters 关联
	resp, err := updated.ToResponse()
	if err != nil {
		dto.InternalServerErrorResponse(c, "转换响应失败: ")
		return
	}

	dto.SuccessResponse(c, resp)
}

// DeleteByID 删除 Provider
// @Summary 删除 Provider 配置
// @Description 删除指定ID的邮箱提供商配置
// @Tags Provider管理
// @Produce json
// @Param id path int64 true "Provider ID"
// @Success 200 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/providers/{id} [delete]
func (h *ProviderHandler) DeleteByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "Invalid provider ID: " + err.Error()})
		return
	}

	err = h.service.DeleteByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{"success": true})
}

// FindByDomain 根据邮箱域名查找 Provider
// @Summary 根据域名查找 Provider
// @Description 根据邮箱域名查找匹配的邮箱提供商配置
// @Tags Provider管理
// @Produce json
// @Param domain query string true "邮箱域名"
// @Success 200 {object} dto.Response{data=model.ProviderResponse}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/providers/by-domain [get]
func (h *ProviderHandler) FindByDomain(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		dto.BadRequestResponse(c, "域名参数不能为空")
		return
	}

	provider, err := h.service.FindByDomain(c.Request.Context(), domain)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	if provider == nil {
		dto.NotFoundResponse(c, "未找到匹配的提供商")
		return
	}

	resp, err := provider.ToResponse()
	if err != nil {
		dto.InternalServerErrorResponse(c, "转换响应失败: ")
		return
	}

	dto.SuccessResponse(c, resp)
}

// FindByEmail 根据邮箱地址查找 Provider
// @Summary 根据邮箱查找 Provider
// @Description 根据邮箱地址自动匹配邮箱提供商配置
// @Tags Provider管理
// @Produce json
// @Param email query string true "邮箱地址"
// @Success 200 {object} dto.Response{data=model.ProviderResponse}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/providers/by-email [get]
func (h *ProviderHandler) FindByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		dto.BadRequestResponse(c, "邮箱参数不能为空")
		return
	}

	provider, err := h.service.FindByEmail(c.Request.Context(), email)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	if provider == nil {
		dto.NotFoundResponse(c, "未找到匹配的提供商")
		return
	}

	resp, err := provider.ToResponse()
	if err != nil {
		dto.InternalServerErrorResponse(c, "转换响应失败: ")
		return
	}

	dto.SuccessResponse(c, resp)
}

// GetWithAdapters 获取 Provider 并预加载适配器
// @Summary 获取 Provider 及其适配器
// @Description 获取指定 Provider 及其支持的所有适配器
// @Tags Provider管理
// @Produce json
// @Param id path int64 true "Provider ID"
// @Success 200 {object} dto.Response{data=model.ProviderResponse}
// @Failure 404 {object} dto.Response
// @Router /api/v1/providers/{id}/adapters [get]
func (h *ProviderHandler) GetWithAdapters(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "无效的 Provider ID: " + err.Error()})
		return
	}

	provider, err := h.service.GetWithAdapters(c.Request.Context(), id)
	if err != nil {
		dto.NotFoundResponse(c, "Provider 未找到")
		return
	}

	resp, err := provider.ToResponse()
	if err != nil {
		dto.InternalServerErrorResponse(c, "转换响应失败: ")
		return
	}

	dto.SuccessResponse(c, resp)
}

// ListWithAdapters 获取所有 Provider 并预加载适配器
// @Summary 获取所有 Provider 及其适配器
// @Description 获取所有 Provider 及其支持的适配器列表
// @Tags Provider管理
// @Produce json
// @Success 200 {object} dto.Response{data=[]model.ProviderResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/providers/with-adapters [get]
func (h *ProviderHandler) ListWithAdapters(c *gin.Context) {
	providers, err := h.service.ListWithAdapters(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*model.ProviderResponse, 0, len(providers))
	for _, p := range providers {
		resp, err := p.ToResponse()
		if err != nil {
			dto.InternalServerErrorResponse(c, "转换响应失败: ")
			return
		}
		items = append(items, resp)
	}

	dto.SuccessResponse(c, items)
}
