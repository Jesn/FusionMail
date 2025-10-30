package handler

import (
	"strconv"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// WebhookHandler Webhook API 处理器
type WebhookHandler struct {
	webhookService service.WebhookService
	webhookLogRepo repository.WebhookLogRepository
}

// NewWebhookHandler 创建 Webhook 处理器实例
func NewWebhookHandler(webhookService service.WebhookService, webhookLogRepo repository.WebhookLogRepository) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
		webhookLogRepo: webhookLogRepo,
	}
}

// CreateWebhookRequest 创建 Webhook 请求
type CreateWebhookRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	URL            string `json:"url" binding:"required,url"`
	Method         string `json:"method"`
	Headers        string `json:"headers"`
	Events         string `json:"events" binding:"required"`
	Filters        string `json:"filters"`
	RetryEnabled   bool   `json:"retry_enabled"`
	MaxRetries     int    `json:"max_retries"`
	RetryIntervals string `json:"retry_intervals"`
}

// UpdateWebhookRequest 更新 Webhook 请求
type UpdateWebhookRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	URL            string `json:"url" binding:"required,url"`
	Method         string `json:"method"`
	Headers        string `json:"headers"`
	Events         string `json:"events" binding:"required"`
	Filters        string `json:"filters"`
	RetryEnabled   bool   `json:"retry_enabled"`
	MaxRetries     int    `json:"max_retries"`
	RetryIntervals string `json:"retry_intervals"`
	Enabled        bool   `json:"enabled"`
}

// TestWebhookRequest 测试 Webhook 请求
type TestWebhookRequest struct {
	TestData map[string]interface{} `json:"test_data"`
}

// CreateWebhook 创建 Webhook
// @Summary 创建 Webhook
// @Description 创建新的 Webhook 配置
// @Tags webhooks
// @Accept json
// @Produce json
// @Param webhook body CreateWebhookRequest true "Webhook 配置"
// @Success 200 {object} dto.Response
// @Router /api/v1/webhooks [post]
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 创建 Webhook 模型
	webhook := &model.Webhook{
		Name:           req.Name,
		Description:    req.Description,
		URL:            req.URL,
		Method:         req.Method,
		Headers:        req.Headers,
		Events:         req.Events,
		Filters:        req.Filters,
		RetryEnabled:   req.RetryEnabled,
		MaxRetries:     req.MaxRetries,
		RetryIntervals: req.RetryIntervals,
		Enabled:        true, // 默认启用
	}

	// 创建 Webhook
	if err := h.webhookService.Create(c.Request.Context(), webhook); err != nil {
		dto.InternalServerErrorResponse(c, "创建 Webhook 失败: "+err.Error())
		return
	}

	dto.SuccessWithMessage(c, webhook, "Webhook 创建成功")
}

// GetWebhookList 获取 Webhook 列表
// @Summary 获取 Webhook 列表
// @Description 获取 Webhook 列表，支持分页
// @Tags webhooks
// @Accept json
// @Produce json
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Success 200 {object} dto.PaginatedResponse
// @Router /api/v1/webhooks [get]
func (h *WebhookHandler) GetWebhookList(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取 Webhook 列表
	webhooks, total, err := h.webhookService.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取 Webhook 列表失败: "+err.Error())
		return
	}

	dto.PaginatedSuccessResponse(c, webhooks, total, page, pageSize)
}

// GetWebhookByID 获取 Webhook 详情
// @Summary 获取 Webhook 详情
// @Description 根据 ID 获取 Webhook 详情
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/webhooks/{id} [get]
func (h *WebhookHandler) GetWebhookByID(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的 Webhook ID")
		return
	}

	// 获取 Webhook
	webhook, err := h.webhookService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.NotFoundResponse(c, "Webhook 不存在")
		return
	}

	dto.SuccessResponse(c, webhook)
}

// UpdateWebhook 更新 Webhook
// @Summary 更新 Webhook
// @Description 更新 Webhook 配置
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Param webhook body UpdateWebhookRequest true "Webhook 配置"
// @Success 200 {object} dto.Response
// @Router /api/v1/webhooks/{id} [put]
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的 Webhook ID")
		return
	}

	var req UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 获取现有 Webhook
	webhook, err := h.webhookService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.NotFoundResponse(c, "Webhook 不存在")
		return
	}

	// 更新字段
	webhook.Name = req.Name
	webhook.Description = req.Description
	webhook.URL = req.URL
	webhook.Method = req.Method
	webhook.Headers = req.Headers
	webhook.Events = req.Events
	webhook.Filters = req.Filters
	webhook.RetryEnabled = req.RetryEnabled
	webhook.MaxRetries = req.MaxRetries
	webhook.RetryIntervals = req.RetryIntervals
	webhook.Enabled = req.Enabled

	// 更新 Webhook
	if err := h.webhookService.Update(c.Request.Context(), webhook); err != nil {
		dto.InternalServerErrorResponse(c, "更新 Webhook 失败: "+err.Error())
		return
	}

	dto.SuccessWithMessage(c, webhook, "Webhook 更新成功")
}

// DeleteWebhook 删除 Webhook
// @Summary 删除 Webhook
// @Description 删除指定的 Webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/webhooks/{id} [delete]
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的 Webhook ID")
		return
	}

	// 删除 Webhook
	if err := h.webhookService.Delete(c.Request.Context(), id); err != nil {
		dto.InternalServerErrorResponse(c, "删除 Webhook 失败: "+err.Error())
		return
	}

	dto.SuccessWithMessage(c, nil, "Webhook 删除成功")
}

// ToggleWebhook 启用/禁用 Webhook
// @Summary 启用/禁用 Webhook
// @Description 切换 Webhook 的启用状态
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/webhooks/{id}/toggle [post]
func (h *WebhookHandler) ToggleWebhook(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的 Webhook ID")
		return
	}

	// 获取当前 Webhook
	webhook, err := h.webhookService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.NotFoundResponse(c, "Webhook 不存在")
		return
	}

	// 切换状态
	if webhook.Enabled {
		err = h.webhookService.Disable(c.Request.Context(), id)
	} else {
		err = h.webhookService.Enable(c.Request.Context(), id)
	}

	if err != nil {
		dto.InternalServerErrorResponse(c, "切换 Webhook 状态失败: "+err.Error())
		return
	}

	// 返回新状态
	webhook.Enabled = !webhook.Enabled
	message := "Webhook 已禁用"
	if webhook.Enabled {
		message = "Webhook 已启用"
	}

	dto.SuccessWithMessage(c, gin.H{"enabled": webhook.Enabled}, message)
}

// TestWebhook 测试 Webhook
// @Summary 测试 Webhook
// @Description 发送测试请求到 Webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Param test_data body TestWebhookRequest false "测试数据"
// @Success 200 {object} dto.Response
// @Router /api/v1/webhooks/{id}/test [post]
func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的 Webhook ID")
		return
	}

	var req TestWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供测试数据，使用默认数据
		req.TestData = map[string]interface{}{
			"message":   "这是一个测试消息",
			"timestamp": time.Now().Unix(),
		}
	}

	// 测试 Webhook
	if err := h.webhookService.TestWebhook(c.Request.Context(), id, req.TestData); err != nil {
		dto.InternalServerErrorResponse(c, "Webhook 测试失败: "+err.Error())
		return
	}

	dto.SuccessWithMessage(c, nil, "Webhook 测试成功")
}

// GetWebhookLogs 获取 Webhook 调用日志
// @Summary 获取 Webhook 调用日志
// @Description 获取指定 Webhook 的调用日志
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Webhook ID"
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Success 200 {object} dto.PaginatedResponse
// @Router /api/v1/webhooks/{id}/logs [get]
func (h *WebhookHandler) GetWebhookLogs(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的 Webhook ID")
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取日志列表
	logs, total, err := h.webhookLogRepo.FindByWebhookID(c.Request.Context(), id, offset, pageSize)
	if err != nil {
		dto.InternalServerErrorResponse(c, "获取 Webhook 日志失败: "+err.Error())
		return
	}

	dto.PaginatedSuccessResponse(c, logs, total, page, pageSize)
}
