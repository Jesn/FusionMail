package handler

import (
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/service/spam"

	"github.com/gin-gonic/gin"
)

// EmailListHandler 白名单/黑名单 API 处理器
type EmailListHandler struct {
	emailListService *spam.EmailListService
}

// NewEmailListHandler 创建白名单/黑名单处理器实例
func NewEmailListHandler(emailListService *spam.EmailListService) *EmailListHandler {
	return &EmailListHandler{
		emailListService: emailListService,
	}
}

// AddToWhitelistRequest 添加到白名单请求
type AddToWhitelistRequest struct {
	Target string `json:"target" binding:"required"` // 邮箱地址或域名
	Reason string `json:"reason"`                    // 添加原因
}

// AddToBlacklistRequest 添加到黑名单请求
type AddToBlacklistRequest struct {
	Target string `json:"target" binding:"required"` // 邮箱地址或域名
	Reason string `json:"reason"`                    // 添加原因
}

// GetWhitelist 获取白名单
// @Summary 获取白名单
// @Description 获取用户的白名单列表
// @Tags emaillist
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} dto.Response
// @Router /api/v1/emaillist/whitelist [get]
func (h *EmailListHandler) GetWhitelist(c *gin.Context) {
	// 获取用户 UID（从认证中间件获取）
	// 注意：认证中间件设置的是 "userID"，值为用户名（如 "admin"）
	userUID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未授权")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取白名单
	lists, total, err := h.emailListService.GetWhitelist(c.Request.Context(), userUID.(string), offset, pageSize)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.PaginatedSuccessResponse(c, lists, total, page, pageSize)
}

// AddToWhitelist 添加到白名单
// @Summary 添加到白名单
// @Description 添加邮箱地址或域名到白名单
// @Tags emaillist
// @Accept json
// @Produce json
// @Param body body AddToWhitelistRequest true "白名单信息"
// @Success 201 {object} dto.Response
// @Router /api/v1/emaillist/whitelist [post]
func (h *EmailListHandler) AddToWhitelist(c *gin.Context) {
	// 获取用户 UID（从认证中间件获取）
	userUID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未授权")
		return
	}

	// 解析请求
	var req AddToWhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 添加到白名单
	list, err := h.emailListService.AddToWhitelist(c.Request.Context(), userUID.(string), req.Target, req.Reason)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	c.JSON(201, dto.Response{
		Success: true,
		Data:    list,
	})
}

// DeleteFromWhitelist 从白名单中删除
// @Summary 从白名单中删除
// @Description 从白名单中删除指定条目
// @Tags emaillist
// @Accept json
// @Produce json
// @Param id path int true "条目 ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/emaillist/whitelist/{id} [delete]
func (h *EmailListHandler) DeleteFromWhitelist(c *gin.Context) {
	// 获取用户 UID（从认证中间件获取）
	userUID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未授权")
		return
	}

	// 获取条目 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "条目 ID 格式无效")
		return
	}

	// 从白名单中删除
	if err := h.emailListService.RemoveFromWhitelist(c.Request.Context(), id, userUID.(string)); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// GetBlacklist 获取黑名单
// @Summary 获取黑名单
// @Description 获取用户的黑名单列表
// @Tags emaillist
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} dto.Response
// @Router /api/v1/emaillist/blacklist [get]
func (h *EmailListHandler) GetBlacklist(c *gin.Context) {
	// 获取用户 UID（从认证中间件获取）
	userUID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未授权")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取黑名单
	lists, total, err := h.emailListService.GetBlacklist(c.Request.Context(), userUID.(string), offset, pageSize)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.PaginatedSuccessResponse(c, lists, total, page, pageSize)
}

// AddToBlacklist 添加到黑名单
// @Summary 添加到黑名单
// @Description 添加邮箱地址或域名到黑名单
// @Tags emaillist
// @Accept json
// @Produce json
// @Param body body AddToBlacklistRequest true "黑名单信息"
// @Success 201 {object} dto.Response
// @Router /api/v1/emaillist/blacklist [post]
func (h *EmailListHandler) AddToBlacklist(c *gin.Context) {
	// 获取用户 UID（从认证中间件获取）
	userUID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未授权")
		return
	}

	// 解析请求
	var req AddToBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 添加到黑名单
	list, err := h.emailListService.AddToBlacklist(c.Request.Context(), userUID.(string), req.Target, req.Reason)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	c.JSON(201, dto.Response{
		Success: true,
		Data:    list,
	})
}

// DeleteFromBlacklist 从黑名单中删除
// @Summary 从黑名单中删除
// @Description 从黑名单中删除指定条目
// @Tags emaillist
// @Accept json
// @Produce json
// @Param id path int true "条目 ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/emaillist/blacklist/{id} [delete]
func (h *EmailListHandler) DeleteFromBlacklist(c *gin.Context) {
	// 获取用户 UID（从认证中间件获取）
	userUID, exists := c.Get("userID")
	if !exists {
		dto.UnauthorizedResponse(c, "未授权")
		return
	}

	// 获取条目 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "条目 ID 格式无效")
		return
	}

	// 从黑名单中删除
	if err := h.emailListService.RemoveFromBlacklist(c.Request.Context(), id, userUID.(string)); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "删除成功"})
}
