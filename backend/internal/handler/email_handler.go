package handler

import (
	"fusionmail/internal/repository"
	"fusionmail/internal/service"
	"net/http"
	"strconv"

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
// @Tags emails
// @Accept json
// @Produce json
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
// @Success 200 {object} service.EmailListResponse
// @Router /api/v1/emails [get]
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

	// 默认不显示已删除的邮件
	isDeleted := false
	filter.IsDeleted = &isDeleted

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务层
	result, err := h.emailService.GetEmailList(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetEmailByID 获取邮件详情
// @Summary 获取邮件详情
// @Description 根据 ID 获取邮件的完整信息，包括附件
// @Tags emails
// @Accept json
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} model.Email
// @Router /api/v1/emails/{id} [get]
func (h *EmailHandler) GetEmailByID(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid email id",
		})
		return
	}

	// 调用服务层
	email, err := h.emailService.GetEmailByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "email not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "email not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, email)
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "search query is required",
		})
		return
	}

	accountUID := c.Query("account_uid")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务层
	result, err := h.emailService.SearchEmails(c.Request.Context(), query, accountUID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.emailService.MarkAsRead(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "emails marked as read",
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.emailService.MarkAsUnread(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "emails marked as unread",
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid email id",
		})
		return
	}

	if err := h.emailService.ToggleStar(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "star status toggled",
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid email id",
		})
		return
	}

	if err := h.emailService.ArchiveEmail(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "email archived",
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid email id",
		})
		return
	}

	if err := h.emailService.DeleteEmail(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "email deleted",
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unread_count": count,
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "account_uid is required",
		})
		return
	}

	stats, err := h.emailService.GetAccountStats(c.Request.Context(), accountUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
