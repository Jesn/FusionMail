package handler

import (
	"fusionmail/internal/model"
	"fusionmail/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AdapterHandler 适配器管理处理器
type AdapterHandler struct {
	service *service.AdapterService
}

// NewAdapterHandler 创建适配器处理器
func NewAdapterHandler(service *service.AdapterService) *AdapterHandler {
	return &AdapterHandler{
		service: service,
	}
}

// List 获取所有适配器列表
// @Summary 获取适配器列表
// @Description 获取所有邮箱协议适配器列表
// @Tags 适配器管理
// @Produce json
// @Success 200 {object} response.Response{data=[]model.AdapterResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/adapters [get]
func (h *AdapterHandler) List(c *gin.Context) {
	adapters, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 转换为响应格式
	items := make([]*model.AdapterResponse, 0, len(adapters))
	for _, a := range adapters {
		items = append(items, a.ToResponse())
	}

	c.JSON(200, gin.H{"success": true, "data": items})
}

// ListEnabled 获取启用的适配器列表
// @Summary 获取启用的适配器列表
// @Description 获取所有启用的邮箱协议适配器列表
// @Tags 适配器管理
// @Produce json
// @Success 200 {object} response.Response{data=[]model.AdapterResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/adapters/enabled [get]
func (h *AdapterHandler) ListEnabled(c *gin.Context) {
	adapters, err := h.service.ListEnabled(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 转换为响应格式
	items := make([]*model.AdapterResponse, 0, len(adapters))
	for _, a := range adapters {
		items = append(items, a.ToResponse())
	}

	c.JSON(200, gin.H{"success": true, "data": items})
}

// GetByID 通过 ID 获取适配器
// @Summary 获取适配器详情
// @Description 通过 ID 获取指定的适配器配置
// @Tags 适配器管理
// @Produce json
// @Param id path int64 true "适配器 ID"
// @Success 200 {object} response.Response{data=model.AdapterResponse}
// @Failure 404 {object} response.Response
// @Router /api/v1/adapters/{id} [get]
func (h *AdapterHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "无效的适配器 ID: " + err.Error()})
		return
	}

	adapter, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"success": false, "error": "适配器未找到"})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": adapter.ToResponse()})
}

// GetByName 通过名称获取适配器
// @Summary 通过名称获取适配器
// @Description 通过名称获取指定的适配器配置
// @Tags 适配器管理
// @Produce json
// @Param name path string true "适配器名称"
// @Success 200 {object} response.Response{data=model.AdapterResponse}
// @Failure 404 {object} response.Response
// @Router /api/v1/adapters/name/{name} [get]
func (h *AdapterHandler) GetByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, gin.H{"success": false, "error": "适配器名称不能为空"})
		return
	}

	adapter, err := h.service.GetByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(404, gin.H{"success": false, "error": "适配器未找到"})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": adapter.ToResponse()})
}
