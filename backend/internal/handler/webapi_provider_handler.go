package handler

import (
	"strconv"

	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// WebAPIProviderHandler WebAPI Provider 管理处理器
type WebAPIProviderHandler struct {
	service *service.WebAPIProviderService
}

// NewWebAPIProviderHandler 创建 WebAPI Provider 处理器
func NewWebAPIProviderHandler(service *service.WebAPIProviderService) *WebAPIProviderHandler {
	return &WebAPIProviderHandler{
		service: service,
	}
}

// ============================================
// 请求/响应结构体
// ============================================

// CreateWebAPIProviderRequest 创建 WebAPI Provider 请求
type CreateWebAPIProviderRequest struct {
	Name        string `json:"name" binding:"required"`         // Provider 名称
	ServiceType string `json:"service_type" binding:"required"` // 服务类型
	AuthData    string `json:"auth_data" binding:"required"`    // 认证数据 JSON
}

// UpdateWebAPIProviderRequest 更新 WebAPI Provider 请求
type UpdateWebAPIProviderRequest struct {
	Name     string `json:"name,omitempty"`      // Provider 名称
	AuthData string `json:"auth_data,omitempty"` // 认证数据 JSON
}

// TestConnectionRequest 测试连接请求
type TestConnectionRequest struct {
	ServiceType string `json:"service_type" binding:"required"` // 服务类型
	AuthData    string `json:"auth_data" binding:"required"`    // 认证数据 JSON
}

// ============================================
// Handler 方法
// ============================================

// Create 创建 WebAPI Provider
// @Summary 创建 WebAPI Provider
// @Description 创建新的 WebAPI 邮箱服务提供商配置
// @Tags WebAPI Provider
// @Accept json
// @Produce json
// @Param request body CreateWebAPIProviderRequest true "请求参数"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/webapi/providers [post]
func (h *WebAPIProviderHandler) Create(c *gin.Context) {
	var req CreateWebAPIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": "请求参数无效: " + err.Error()})
		return
	}

	// 调用服务层创建
	account, err := h.service.Create(c.Request.Context(), req.Name, req.ServiceType, req.AuthData)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"success": true, "data": account})
}

// List 获取 WebAPI Provider 列表
// @Summary 获取 WebAPI Provider 列表
// @Description 获取所有 WebAPI 邮箱服务提供商配置列表
// @Tags WebAPI Provider
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/webapi/providers [get]
func (h *WebAPIProviderHandler) List(c *gin.Context) {
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
	accounts, total, err := h.service.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"items":     accounts,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetByUID 通过 UID 获取 WebAPI Provider
// @Summary 获取 WebAPI Provider 详情
// @Description 通过 UID 获取指定的 WebAPI Provider 配置
// @Tags WebAPI Provider
// @Produce json
// @Param uid path string true "Provider UID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/webapi/providers/{uid} [get]
func (h *WebAPIProviderHandler) GetByUID(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(400, gin.H{"success": false, "error": "UID 不能为空"})
		return
	}

	account, err := h.service.GetByUID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(404, gin.H{"success": false, "error": "Provider 未找到"})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": account})
}

// Update 更新 WebAPI Provider
// @Summary 更新 WebAPI Provider 配置
// @Description 更新指定 UID 的 WebAPI Provider 配置
// @Tags WebAPI Provider
// @Accept json
// @Produce json
// @Param uid path string true "Provider UID"
// @Param request body UpdateWebAPIProviderRequest true "请求参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/webapi/providers/{uid} [put]
func (h *WebAPIProviderHandler) Update(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(400, gin.H{"success": false, "error": "UID 不能为空"})
		return
	}

	var req UpdateWebAPIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": "请求参数无效: " + err.Error()})
		return
	}

	account, err := h.service.Update(c.Request.Context(), uid, req.Name, req.AuthData)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": account})
}

// Delete 删除 WebAPI Provider
// @Summary 删除 WebAPI Provider
// @Description 删除指定 UID 的 WebAPI Provider 配置
// @Tags WebAPI Provider
// @Produce json
// @Param uid path string true "Provider UID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/webapi/providers/{uid} [delete]
func (h *WebAPIProviderHandler) Delete(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(400, gin.H{"success": false, "error": "UID 不能为空"})
		return
	}

	err := h.service.Delete(c.Request.Context(), uid)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "删除成功"})
}

// TestConnection 测试连接
// @Summary 测试 WebAPI 连接
// @Description 测试 WebAPI 服务的连接是否正常
// @Tags WebAPI Provider
// @Accept json
// @Produce json
// @Param request body TestConnectionRequest true "请求参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/webapi/providers/test [post]
func (h *WebAPIProviderHandler) TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": "请求参数无效: " + err.Error()})
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), req.ServiceType, req.AuthData)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": result})
}

// TestConnectionByUID 测试已存在 Provider 的连接
// @Summary 测试已存在 Provider 的连接
// @Description 测试指定 UID 的 WebAPI Provider 连接是否正常
// @Tags WebAPI Provider
// @Produce json
// @Param uid path string true "Provider UID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/webapi/providers/{uid}/test [post]
func (h *WebAPIProviderHandler) TestConnectionByUID(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(400, gin.H{"success": false, "error": "UID 不能为空"})
		return
	}

	result, err := h.service.TestConnectionByUID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": result})
}

// TriggerSync 手动触发同步
// @Summary 手动触发同步
// @Description 手动触发指定 WebAPI Provider 的邮件同步
// @Tags WebAPI Provider
// @Produce json
// @Param uid path string true "Provider UID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/webapi/providers/{uid}/sync [post]
func (h *WebAPIProviderHandler) TriggerSync(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(400, gin.H{"success": false, "error": "UID 不能为空"})
		return
	}

	err := h.service.TriggerSync(c.Request.Context(), uid)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "同步任务已启动"})
}

// GetSyncStatus 获取同步状态
// @Summary 获取同步状态
// @Description 获取指定 WebAPI Provider 的同步状态
// @Tags WebAPI Provider
// @Produce json
// @Param uid path string true "Provider UID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/webapi/providers/{uid}/sync/status [get]
func (h *WebAPIProviderHandler) GetSyncStatus(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(400, gin.H{"success": false, "error": "UID 不能为空"})
		return
	}

	status, err := h.service.GetSyncStatus(c.Request.Context(), uid)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": status})
}
