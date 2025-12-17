package handler

import (
	"context"
	"fmt"
	"log"
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

// getProviderByParam 根据参数获取提供商（通过 provider_id）
func (h *OAuth2ClientHandler) getProviderByParam(ctx context.Context, param string) (*model.Provider, error) {
	// 解析为数字 ID
	if providerId, parseErr := strconv.ParseInt(param, 10, 64); parseErr == nil {
		// 通过 ID 获取提供商
		return h.providerService.GetByID(ctx, providerId)
	}
	// 无法解析为数字，返回错误
	return nil, fmt.Errorf("invalid provider parameter: %s (expected provider ID)", param)
}

// Create 创建 OAuth2 客户端配置
// @Summary 创建 OAuth2 客户端配置
// @Description 为指定邮箱提供商创建新的 OAuth2 客户端配置
// @Tags OAuth2客户端管理
// @Accept json
// @Produce json
// @Param request body model.OAuth2ClientCreateRequest true "请求参数"
// @Success 201 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
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
// @Success 200 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
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
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
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
// @Success 200 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
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
// @Success 200 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 500 {object} response.Response
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
// @Param provider_type path string true "邮箱提供商类型 (1=Gmail, 2=Outlook)"
// @Success 200 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/oauth2/clients/provider/{provider_type} [get]
func (h *OAuth2ClientHandler) GetByProvider(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] GetByProvider panicked: %v", r)
			dto.InternalServerErrorResponse(c, "Internal server error")
		}
	}()

	providerParam := c.Param("provider_type")

	log.Printf("[DEBUG] GetByProvider called with provider_param: %s", providerParam)

	// 检查处理器是否为nil
	if h == nil {
		log.Printf("[ERROR] OAuth2ClientHandler is nil!")
		dto.InternalServerErrorResponse(c, "Handler not initialized")
		return
	}

	if h.service == nil {
		log.Printf("[ERROR] OAuth2ClientService is nil!")
		dto.InternalServerErrorResponse(c, "Service not initialized")
		return
	}

	if h.providerService == nil {
		log.Printf("[ERROR] ProviderService is nil!")
		dto.InternalServerErrorResponse(c, "Provider service not initialized")
		return
	}

	// 通过参数获取提供商（支持数字ID和字符串名称）
	provider, providerErr := h.getProviderByParam(c.Request.Context(), providerParam)
	if providerErr != nil {
		log.Printf("[DEBUG] Provider not found for param %s: %v", providerParam, providerErr)
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	log.Printf("[DEBUG] Provider found: ID=%d, Name=%s", provider.ID, provider.Name)

	log.Printf("[DEBUG] Using provider_id query: %d", provider.ID)
	// 使用 provider_id 查询
	clients, err := h.service.GetByProvider(c.Request.Context(), provider.ID)
	if err != nil {
		log.Printf("[ERROR] Failed to get OAuth2 clients for provider %s: %v", providerParam, err)
		dto.HandleServiceError(c, err)
		return
	}

	log.Printf("[DEBUG] Found %d OAuth2 clients", len(clients))

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
// @Param provider_type path string true "邮箱提供商类型 (1=Gmail, 2=Outlook)"
// @Success 200 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/oauth2/clients/provider/{provider_name}/default [get]
func (h *OAuth2ClientHandler) GetDefault(c *gin.Context) {
	providerParam := c.Param("provider_type")

	// 通过参数获取提供商
	provider, providerErr := h.getProviderByParam(c.Request.Context(), providerParam)
	if providerErr != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}
	// 使用 provider_id 查询
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
// @Param provider_type path string true "邮箱提供商类型 (1=Gmail, 2=Outlook)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/oauth2/clients/{id}/default [post]
func (h *OAuth2ClientHandler) SetDefault(c *gin.Context) {
	id, parseErr := strconv.ParseInt(c.Param("id"), 10, 64)
	if parseErr != nil {
		dto.BadRequestResponse(c, "Invalid client ID")
		return
	}

	providerParam := c.Param("provider_type")

	// 通过参数获取提供商（支持数字ID和字符串名称）
	provider, providerErr := h.getProviderByParam(c.Request.Context(), providerParam)
	if providerErr != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}

	// 使用 provider_id 设置默认
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
// @Param provider_type path string true "邮箱提供商类型 (1=Gmail, 2=Outlook)"
// @Param client_id query int64 false "指定的客户端 ID（可选）"
// @Success 200 {object} response.Response{data=model.OAuth2ClientResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/oauth2/clients/smart-select [get]
func (h *OAuth2ClientHandler) SmartSelect(c *gin.Context) {
	providerParam := c.Param("provider_type")

	var clientID *int64
	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		if id, err := strconv.ParseInt(clientIDStr, 10, 64); err == nil {
			clientID = &id
		}
	}

	// 通过参数获取提供商
	provider, providerErr := h.getProviderByParam(c.Request.Context(), providerParam)
	if providerErr != nil {
		dto.NotFoundResponse(c, "Provider not found")
		return
	}
	// 使用 provider_id 查询
	client, err := h.service.SmartSelect(c.Request.Context(), provider.ID, clientID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, client.ToResponse())
}
