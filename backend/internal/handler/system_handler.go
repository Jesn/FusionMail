package handler

import (
	"net/http"
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统管理处理器
type SystemHandler struct {
	systemService *service.SystemService
}

// NewSystemHandler 创建系统管理处理器
func NewSystemHandler(systemService *service.SystemService) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
	}
}

// GetHealth 获取系统健康状态
// @Summary 获取系统健康状态
// @Description 检查系统各组件的健康状态
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=service.SystemHealthResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/system/health [get]
func (h *SystemHandler) GetHealth(c *gin.Context) {
	health, err := h.systemService.GetSystemHealth(c.Request.Context())
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "获取系统健康状态失败")
		return
	}

	dto.SuccessResponse(c, health)
}

// GetStats 获取系统统计信息
// @Summary 获取系统统计信息
// @Description 获取系统运行统计信息，包括邮件数量、账户数量等
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=SystemStatsResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/system/stats [get]
func (h *SystemHandler) GetStats(c *gin.Context) {
	stats, err := h.systemService.GetSystemStats(c.Request.Context())
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "获取系统统计信息失败")
		return
	}

	dto.SuccessResponse(c, stats)
}

// GetSyncStatus 获取同步状态
// @Summary 获取同步状态
// @Description 获取所有账户的同步状态信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]SyncStatusResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/sync/status [get]
func (h *SystemHandler) GetSyncStatus(c *gin.Context) {
	status, err := h.systemService.GetSyncStatus(c.Request.Context())
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "获取同步状态失败")
		return
	}

	dto.SuccessResponse(c, status)
}

// GetSyncLogs 获取同步日志
// @Summary 获取同步日志
// @Description 获取系统同步日志，支持分页和筛选
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param account_uid query string false "账户UID筛选"
// @Param status query string false "状态筛选(success/failed)"
// @Success 200 {object} response.Response{data=SyncLogsResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/sync/logs [get]
func (h *SystemHandler) GetSyncLogs(c *gin.Context) {
	// 解析查询参数
	page := parseIntParam(c, "page", 1)
	pageSize := parseIntParam(c, "page_size", 20)
	accountUID := c.Query("account_uid")
	status := c.Query("status")

	// 验证参数
	if page < 1 {
		dto.BadRequestResponse(c, "页码必须大于0")
		return
	}
	if pageSize < 1 || pageSize > 100 {
		dto.BadRequestResponse(c, "每页数量必须在1-100之间")
		return
	}

	logs, total, err := h.systemService.GetSyncLogs(c.Request.Context(), page, pageSize, accountUID, status)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "获取同步日志失败")
		return
	}

	dto.PaginatedSuccessResponse(c, logs, total, page, pageSize)
}

// parseIntParam 解析整数参数
func parseIntParam(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
