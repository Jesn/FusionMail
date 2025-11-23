package handler

import (
	"context"
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// OAuth2ClientHandler OAuth2 客户端管理处理器
type OAuth2ClientHandler struct {
	service         *service.OAuth2ClientService
	providerService *service.ProviderService
}

// NewOAuth2ClientHandler 创建 OAuth2 客户端处理器
func NewOAuth2ClientHandler(service *service.OAuth2ClientService, providerService *service.ProviderService) *OAuth2ClientHandler {
	return &OAuth2ClientHandler{
		service:         service,
		providerService: providerService,
	}
}

// getProviderByParam 根据参数获取提供商（支持数字ID和字符串名称）
func (h *OAuth2ClientHandler) getProviderByParam(ctx context.Context, param string) (*model.Provider, error) {
	// 检测参数是数字ID还是名称
	if providerId, parseErr := strconv.ParseInt(param, 10, 64); parseErr == nil {
		// 是数字ID，直接通过ID获取提供商
		return h.providerService.GetByID(ctx, providerId)
	} else {
		// 是字符串名称，通过名称获取提供商
		return h.providerService.GetByName(ctx, param)
	}
}

// Create 创建 OAuth2 客户端配置
// @Summary 创建 OAuth2 客户端配置
// @Description 为指定邮箱提供商创建新的 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param request body model.OAuth2ClientCreateRequest true "请求参数"
// @Success 201 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients [post]
func (h *OAuth2ClientHandler) Create(c *gin.Context) {
	var req model.OAuth2ClientCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "Invalid request parameters: "+err.Error())
		return
	}

	client, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, client.ToResponse())
}

// Update 更新 OAuth2 客户端配置
// @Summary 更新 OAuth2 客户端配置
// @Description 更新指定 ID 的 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param id path int64 true "客户端 ID"
// @Param request body model.OAuth2ClientUpdateRequest true "请求参数"
// @Success 200 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/{id} [put]
func (h *OAuth2ClientHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "Invalid client ID")
		return
	}

	var req model.OAuth2ClientUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "Invalid request parameters: "+err.Error())
		return
	}

	client, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, client.ToResponse())
}

// Delete 删除 OAuth2 客户端配置
// @Summary 删除 OAuth2 客户端配置
// @Description 删除指定 ID 的 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param id path int64 true "客户端 ID"
// @Success 200 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/{id} [delete]
func (h *OAuth2ClientHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "Invalid client ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "OAuth2 client deleted successfully"})
}

// GetByID 获取单个 OAuth2 客户端配置
// @Summary 获取单个 OAuth2 客户端配置
// @Description 根据 ID 获取 OAuth2 客户端配置详情
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param id path int64 true "客户端 ID"
// @Success 200 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/{id} [get]
func (h *OAuth2ClientHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "Invalid client ID")
		return
	}

	client, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, client.ToResponse())
}

// List 分页获取 OAuth2 客户端配置列表
// @Summary 分页获取 OAuth2 客户端配置列表
// @Description 分页查询所有 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param page query int false "页码（默认1）"
// @Param page_size query int false "每页数量（默认20）"
// @Success 200 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients [get]
func (h *OAuth2ClientHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	clients, total, err := h.service.List(c.Request.Context(), page, pageSize)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 转换为响应格式
	var responses []model.OAuth2ClientResponse
	for _, client := range clients {
		responses = append(responses, *client.ToResponse())
	}

	dto.SuccessResponse(c, gin.H{
		"data":       responses,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetByProvider 获取指定提供商的所有客户端配置
// @Summary 获取指定提供商的所有客户端配置
// @Description 获取指定邮箱提供商的所有 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param provider_name path string true "邮箱提供商名称"
// @Success 200 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/provider/{provider_name} [get]
func (h *OAuth2ClientHandler) GetByProvider(c *gin.Context) {
	providerParam := c.Param("provider_name")

	// 通过参数获取提供商（支持数字ID和字符串名称）
	provider, err := h.getProviderByParam(c.Request.Context(), providerParam)
	if err != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	clients, err := h.service.GetByProvider(c.Request.Context(), provider.ID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 转换为响应格式
	var responses []model.OAuth2ClientResponse
	for _, client := range clients {
		responses = append(responses, *client.ToResponse())
	}

	dto.SuccessResponse(c, responses)
}

// GetDefault 获取指定提供商的默认客户端配置
// @Summary 获取指定提供商的默认客户端配置
// @Description 获取指定邮箱提供商的默认 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param provider_name path string true "邮箱提供商名称"
// @Success 200 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/provider/{provider_name}/default [get]
func (h *OAuth2ClientHandler) GetDefault(c *gin.Context) {
	providerParam := c.Param("provider_name")

	// 通过参数获取提供商（支持数字ID和字符串名称）
	provider, err := h.getProviderByParam(c.Request.Context(), providerParam)
	if err != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	client, err := h.service.GetDefault(c.Request.Context(), provider.ID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, client.ToResponse())
}

// SetDefault 设置默认客户端配置
// @Summary 设置默认客户端配置
// @Description 将指定客户端设置为提供商的默认配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param id path int64 true "客户端 ID"
// @Param provider_name path string true "邮箱提供商名称"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/{id}/default [post]
func (h *OAuth2ClientHandler) SetDefault(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "Invalid client ID")
		return
	}

	providerParam := c.Param("provider_name")

	// 通过参数获取提供商（支持数字ID和字符串名称）
	provider, err := h.getProviderByParam(c.Request.Context(), providerParam)
	if err != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	if err := h.service.SetDefault(c.Request.Context(), id, provider.ID); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "Default client set successfully"})
}

// SmartSelect 智能选择客户端配置
// @Summary 智能选择客户端配置
// @Description 智能选择 OAuth2 客户端，支持配额管理和自动降级
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param provider_name path string true "邮箱提供商名称"
// @Param client_id query int64 false "指定的客户端 ID（可选）"
// @Success 200 {object} dto.Response{data=model.OAuth2ClientResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/oauth2/clients/smart-select [get]
func (h *OAuth2ClientHandler) SmartSelect(c *gin.Context) {
	providerParam := c.Param("provider_name")

	var clientID *int64
	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		if id, err := strconv.ParseInt(clientIDStr, 10, 64); err == nil {
			clientID = &id
		}
	}

	// 通过参数获取提供商（支持数字ID和字符串名称）
	provider, err := h.getProviderByParam(c.Request.Context(), providerParam)
	if err != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	client, err := h.service.SmartSelect(c.Request.Context(), provider.ID, clientID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, client.ToResponse())
}
