package handler

import (
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/dto/request"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"
	"fusionmail/internal/sse"

	"github.com/gin-gonic/gin"
)

// EmailHandler 邮件 API 处理器
type EmailHandler struct {
	emailService service.EmailService
}

// NewEmailHandler 创建邮件处理器实例
func NewEmailHandler(emailService service.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

// GetEmailList 获取邮件列表
// @Summary 获取邮件列表
// @Description 获取邮件列表，支持分页、筛选和排序
// @Tags 邮件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param account_uid query string false "账户 UID"
// @Param is_read query bool false "是否已读"
// @Param is_starred query bool false "是否星标"
// @Param is_archived query bool false "是否归档"
// @Param from_address query string false "发件人地址（模糊匹配）"
// @Param subject query string false "主题（模糊匹配）"
// @Param start_date query string false "开始日期（YYYY-MM-DD）"
// @Param end_date query string false "结束日期（YYYY-MM-DD）"
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /emails [get]
func (h *EmailHandler) GetEmailList(c *gin.Context) {
	// 解析查询参数
	filter := &repository.EmailFilter{
		AccountUID:  c.Query("account_uid"),
		FromAddress: c.Query("from_address"),
		Subject:     c.Query("subject"),
		StartDate:   c.Query("start_date"),
		EndDate:     c.Query("end_date"),
	}

	// 解析布尔值参数
	if isReadStr := c.Query("is_read"); isReadStr != "" {
		isRead := isReadStr == "true"
		filter.IsRead = &isRead
	}
	if isStarredStr := c.Query("is_starred"); isStarredStr != "" {
		isStarred := isStarredStr == "true"
		filter.IsStarred = &isStarred
	}
	if isArchivedStr := c.Query("is_archived"); isArchivedStr != "" {
		isArchived := isArchivedStr == "true"
		filter.IsArchived = &isArchived
	}

	// 处理是否显示已删除邮件：未传入时默认不显示；传入时按参数
	if isDeletedStr := c.Query("is_deleted"); isDeletedStr != "" {
		isDeleted := isDeletedStr == "true"
		filter.IsDeleted = &isDeleted
	} else {
		isDeleted := false
		filter.IsDeleted = &isDeleted
	}

	// 处理是否显示垃圾邮件：未传入时默认不显示；传入时按参数
	if isSpamStr := c.Query("is_spam"); isSpamStr != "" {
		isSpam := isSpamStr == "true"
		filter.IsSpam = &isSpam
	} else {
		isSpam := false
		filter.IsSpam = &isSpam
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务层
	result, err := h.emailService.GetEmailList(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, result)
}

// GetEmailByID 获取邮件详情
// @Summary 获取邮件详情
// @Description 根据 ID 获取邮件的完整信息，包括附件
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} response.Response
// @Router /api/v1/emails/{id} [get]
func (h *EmailHandler) GetEmailByID(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "邮件 ID 格式无效")
		return
	}

	// 调用服务层
	email, err := h.emailService.GetEmailByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 仅当从垃圾箱进入时才允许查看已删除邮件详情（include_deleted=true 或 from=trash）
	if email.IsDeleted {
		includeDeleted := c.Query("include_deleted")
		from := c.Query("from")
		if includeDeleted != "true" && from != "trash" {
			// 为避免泄露信息，返回 404（与未找到邮件一致）
			dto.HandleServiceError(c, dto.NewAPIError(dto.ErrEmailNotFound))
			return
		}
	}

	dto.SuccessResponse(c, email)
}

// SearchEmails 搜索邮件
// @Summary 搜索邮件
// @Description 全文搜索邮件（主题、发件人、正文）
// @Tags emails
// @Accept json
// @Produce json
// @Param q query string true "搜索关键词"
// @Param account_uid query string false "账户 UID"
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Success 200 {object} service.EmailListResponse
// @Router /api/v1/emails/search [get]
func (h *EmailHandler) SearchEmails(c *gin.Context) {
	// 解析查询参数
	query := c.Query("q")
	if query == "" {
		dto.BadRequestResponse(c, "搜索关键词不能为空")
		return
	}

	accountUID := c.Query("account_uid")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务层
	result, err := h.emailService.SearchEmails(c.Request.Context(), query, accountUID, page, pageSize)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, result)
}

// MarkAsReadRequest 标记已读请求
type MarkAsReadRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// MarkAsRead 标记邮件为已读
// @Summary 标记邮件为已读
// @Description 批量标记邮件为已读（仅本地状态）
// @Tags emails
// @Accept json
// @Produce json
// @Param body body MarkAsReadRequest true "邮件 ID 列表"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/mark-read [post]
func (h *EmailHandler) MarkAsRead(c *gin.Context) {
	var req MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	if err := h.emailService.MarkAsRead(c.Request.Context(), req.IDs); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "邮件已标记为已读")
}

// MarkAsUnread 标记邮件为未读
// @Summary 标记邮件为未读
// @Description 批量标记邮件为未读（仅本地状态）
// @Tags emails
// @Accept json
// @Produce json
// @Param body body MarkAsReadRequest true "邮件 ID 列表"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/mark-unread [post]
func (h *EmailHandler) MarkAsUnread(c *gin.Context) {
	var req MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	if err := h.emailService.MarkAsUnread(c.Request.Context(), req.IDs); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "邮件已标记为未读")
}

// MarkAllAsRead 全部标记为已读
// @Summary 全部标记为已读
// @Description 批量标记所有未读邮件为已读（仅本地状态），可选择指定账号或全部账号
// @Tags emails
// @Accept json
// @Produce json
// @Param body body request.MarkAllAsReadRequest true "标记请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/emails/mark-all-read [post]
func (h *EmailHandler) MarkAllAsRead(c *gin.Context) {
	var req request.MarkAllAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	count, err := h.emailService.MarkAllAsRead(c.Request.Context(), req.AccountUID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessResponse(c, gin.H{
		"message": "标记成功",
		"count":   count,
	})
}

// ToggleStar 切换星标状态
// @Summary 切换星标状态
// @Description 切换邮件的星标状态（仅本地状态）
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/{id}/toggle-star [post]
func (h *EmailHandler) ToggleStar(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "邮件 ID 格式无效")
		return
	}

	if err := h.emailService.ToggleStar(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "星标状态已切换")
}

// ArchiveEmail 归档邮件
// @Summary 归档邮件
// @Description 归档邮件（仅本地状态）
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/{id}/archive [post]
func (h *EmailHandler) ArchiveEmail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "邮件 ID 格式无效")
		return
	}

	if err := h.emailService.ArchiveEmail(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "邮件已归档")
}

// DeleteEmail 删除邮件
// @Summary 删除邮件
// @Description 删除邮件（软删除，仅本地状态）
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/{id} [delete]
func (h *EmailHandler) DeleteEmail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "邮件 ID 格式无效")
		return
	}

	if err := h.emailService.DeleteEmail(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "邮件已删除")
}

// RestoreEmail 恢复已删除邮件
// @Summary 恢复已删除邮件
// @Description 将邮件从本地垃圾箱恢复（仅本地状态）
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/{id}/restore [post]
func (h *EmailHandler) RestoreEmail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "邮件 ID 格式无效")
		return
	}

	if err := h.emailService.RestoreEmail(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "邮件已恢复")
}

// GetUnreadCount 获取未读邮件数
// @Summary 获取未读邮件数
// @Description 获取指定账户或全部账户的未读邮件数
// @Tags emails
// @Accept json
// @Produce json
// @Param account_uid query string false "账户 UID"
// @Success 200 {object} map[string]int64
// @Router /api/v1/emails/unread-count [get]
func (h *EmailHandler) GetUnreadCount(c *gin.Context) {
	accountUID := c.Query("account_uid")

	count, err := h.emailService.GetUnreadCount(c.Request.Context(), accountUID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{
		"unread_count": count,
	})
}

// GetGlobalStats 获取全局邮件统计
// @Summary 获取全局邮件统计
// @Description 获取全局范围内的邮件统计信息
// @Tags emails
// @Accept json
// @Produce json
// @Success 200 {object} service.GlobalEmailStats
// @Router /api/v1/emails/stats [get]
func (h *EmailHandler) GetGlobalStats(c *gin.Context) {
	stats, err := h.emailService.GetGlobalStats(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}
	dto.SuccessResponse(c, stats)
}

// GetAccountStats 获取账户邮件统计
// @Summary 获取账户邮件统计
// @Description 获取指定账户的邮件统计信息
// @Tags emails
// @Accept json
// @Produce json
// @Param account_uid path string true "账户 UID"
// @Success 200 {object} service.AccountEmailStats
// @Router /api/v1/emails/stats/{account_uid} [get]
func (h *EmailHandler) GetAccountStats(c *gin.Context) {
	accountUID := c.Param("account_uid")
	if accountUID == "" {
		dto.BadRequestResponse(c, "账户 UID 不能为空")
		return
	}

	stats, err := h.emailService.GetAccountStats(c.Request.Context(), accountUID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, stats)
}

// PermanentDeleteEmail 永久删除邮件
// @Summary 永久删除邮件
// @Description 永久删除回收站中的邮件（物理删除，不可恢复）
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/emails/{id}/permanent [delete]
func (h *EmailHandler) PermanentDeleteEmail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "邮件 ID 格式无效")
		return
	}

	if err := h.emailService.PermanentDeleteEmail(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessWithMessage(c, nil, "邮件已永久删除")
}

// BatchPermanentDeleteRequest 批量永久删除请求
type BatchPermanentDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// BatchPermanentDeleteEmails 批量永久删除邮件
// @Summary 批量永久删除邮件
// @Description 批量永久删除回收站中的邮件（物理删除，不可恢复）
// @Tags emails
// @Accept json
// @Produce json
// @Param body body BatchPermanentDeleteRequest true "邮件 ID 列表"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/emails/permanent-delete [post]
func (h *EmailHandler) BatchPermanentDeleteEmails(c *gin.Context) {
	var req BatchPermanentDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	deletedCount, err := h.emailService.BatchPermanentDeleteEmails(c.Request.Context(), req.IDs)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessResponse(c, gin.H{
		"message":       "邮件已永久删除",
		"deleted_count": deletedCount,
	})
}

// EmptyTrash 清空回收站
// @Summary 清空回收站
// @Description 永久删除回收站中的所有邮件（物理删除，不可恢复）
// @Tags emails
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/emails/empty-trash [post]
func (h *EmailHandler) EmptyTrash(c *gin.Context) {
	deletedCount, err := h.emailService.EmptyTrash(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// SSE: broadcast count-change signal
	sse.Broadcast("email_counts_maybe_changed", "{}")

	dto.SuccessResponse(c, gin.H{
		"message":       "回收站已清空",
		"deleted_count": deletedCount,
	})
}
