package handler

import (
	"fmt"
	"log"

	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// MarkMailAsReadRequest 标记邮件为已读请求
type MarkMailAsReadRequest struct {
	IDs []int64 `json:"ids" binding:"required"` // 邮件 ID 列表
}

// MarkMailAsReadResponse 标记邮件为已读响应
type MarkMailAsReadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MarkMailAsRead 标记邮件为已读
// @Summary 标记邮件为已读
// @Description 通过 API Key 批量标记邮件为已读（仅本地状态）
// @Tags public
// @Accept json
// @Produce json
// @Param body body MarkMailAsReadRequest true "邮件 ID 列表"
// @Security ApiKeyAuth
// @Success 200 {object} MarkMailAsReadResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/public/mail/mark-read [post]
func (h *PublicHandler) MarkMailAsRead(c *gin.Context) {
	var req MarkMailAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 获取 API Key ID（从中间件设置）
	apiKeyID, exists := c.Get("api_key_id")
	if !exists {
		log.Printf("[ERROR] API Key ID not found in context")
		dto.UnauthorizedResponse(c, "无效的 API Key")
		return
	}

	apiKeyIDInt, ok := apiKeyID.(int64)
	if !ok {
		log.Printf("[ERROR] API Key ID type assertion failed: %v", apiKeyID)
		dto.UnauthorizedResponse(c, "无效的 API Key")
		return
	}

	// 调用服务层标记为已读
	if err := h.emailService.MarkAsRead(c.Request.Context(), req.IDs); err != nil {
		log.Printf("[ERROR] Failed to mark emails as read: %v", err)
		dto.HandleServiceError(c, err)
		return
	}

	// 记录 API Key 使用（异步）
	go func() {
		if err := h.logAPIKeyUsage(c.Request.Context(), apiKeyIDInt, "", len(req.IDs)); err != nil {
			log.Printf("[WARN] Failed to log API Key usage: %v", err)
		}
	}()

	c.JSON(200, MarkMailAsReadResponse{
		Success: true,
		Message: "邮件已标记为已读",
	})
}

// ---------------------------------------------------------------------------
// 发送邮件
// ---------------------------------------------------------------------------

// SendMailRequest 发送邮件请求
type SendMailRequest struct {
	From     string   `json:"from" binding:"required,email"`
	To       []string `json:"to" binding:"required,min=1"`
	Cc       []string `json:"cc"`
	Bcc      []string `json:"bcc"`
	Subject  string   `json:"subject"`
	TextBody string   `json:"text_body"`
	HTMLBody string   `json:"html_body"`
	ReplyTo  string   `json:"reply_to"`
}

// SendMail 发送邮件
// @Summary 发送邮件
// @Description 通过 API Key 使用指定邮箱账户发送邮件
// @Tags public
// @Accept json
// @Produce json
// @Param body body SendMailRequest true "发送邮件请求"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/public/mail/send [post]
func (h *PublicHandler) SendMail(c *gin.Context) {
	var req SendMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.GetByEmail(c.Request.Context(), req.From)
	if err != nil {
		log.Printf("[ERROR] Failed to get account by email: %s, error: %v", req.From, err)
		dto.HandleServiceError(c, err)
		return
	}
	if account == nil {
		dto.NotFoundResponse(c, fmt.Sprintf("邮箱账户不存在: %s", req.From))
		return
	}
	if account.Status != "active" {
		dto.BadRequestResponse(c, fmt.Sprintf("账户已禁用: %s", account.Status))
		return
	}

	serviceReq := &service.SendEmailRequest{
		AccountUID: account.UID,
		To:         req.To,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		Subject:    req.Subject,
		TextBody:   req.TextBody,
		HTMLBody:   req.HTMLBody,
		ReplyTo:    req.ReplyTo,
	}

	result, err := h.sendService.SendEmail(c.Request.Context(), serviceReq)
	if err != nil {
		dto.InternalServerErrorResponse(c, result.Error)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"message_id":      result.MessageID,
			"sent_email_id":   result.SentEmailID,
			"sender_type":     result.SenderType,
			"provider_msg_id": result.ProviderMsgID,
		},
	})
}

// ---------------------------------------------------------------------------
// 获取单封邮件详情
// ---------------------------------------------------------------------------

// GetMailDetailRequest 获取邮件详情请求
type GetMailDetailRequest struct {
	Email string `form:"email" binding:"required,email"`
	ID    int64  `form:"id" binding:"required"`
}

// GetMailDetail 获取单封邮件详情
// @Summary 获取邮件详情
// @Description 通过 API Key 获取指定邮件的详细信息
// @Tags public
// @Accept json
// @Produce json
// @Param email query string true "邮箱地址"
// @Param id query int true "邮件 ID"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/public/mail/detail [get]
func (h *PublicHandler) GetMailDetail(c *gin.Context) {
	var req GetMailDetailRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("[ERROR] Failed to get account by email: %s, error: %v", req.Email, err)
		dto.HandleServiceError(c, err)
		return
	}
	if account == nil {
		dto.NotFoundResponse(c, fmt.Sprintf("邮箱账户不存在: %s", req.Email))
		return
	}

	email, err := h.emailService.GetEmailByID(c.Request.Context(), req.ID)
	if err != nil {
		log.Printf("[ERROR] Failed to get email by id: %d, error: %v", req.ID, err)
		dto.HandleServiceError(c, err)
		return
	}
	if email == nil {
		dto.NotFoundResponse(c, "邮件不存在")
		return
	}

	if email.AccountUID != account.UID {
		dto.NotFoundResponse(c, "邮件不存在")
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    email,
	})
}

// ---------------------------------------------------------------------------
// 删除单封邮件
// ---------------------------------------------------------------------------

// DeleteMailRequest 删除邮件请求
type DeleteMailRequest struct {
	Email string `form:"email" binding:"required,email"`
	ID    int64  `form:"id" binding:"required"`
}

// DeleteMail 删除单封邮件
// @Summary 删除邮件
// @Description 通过 API Key 删除指定邮件（软删除）
// @Tags public
// @Accept json
// @Produce json
// @Param email query string true "邮箱地址"
// @Param id query int true "邮件 ID"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/public/mail/delete [delete]
func (h *PublicHandler) DeleteMail(c *gin.Context) {
	var req DeleteMailRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("[ERROR] Failed to get account by email: %s, error: %v", req.Email, err)
		dto.HandleServiceError(c, err)
		return
	}
	if account == nil {
		dto.NotFoundResponse(c, fmt.Sprintf("邮箱账户不存在: %s", req.Email))
		return
	}

	email, err := h.emailRepo.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		log.Printf("[ERROR] Failed to find email: id=%d, error: %v", req.ID, err)
		dto.InternalServerErrorResponse(c, "查询邮件失败")
		return
	}
	if email == nil {
		dto.NotFoundResponse(c, "邮件不存在")
		return
	}
	if email.AccountUID != account.UID {
		dto.NotFoundResponse(c, "邮件不存在")
		return
	}

	if err := h.emailService.DeleteEmail(c.Request.Context(), req.ID); err != nil {
		log.Printf("[ERROR] Failed to delete email: id=%d, error: %v", req.ID, err)
		dto.HandleServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    gin.H{"message": "Email deleted"},
	})
}

// ---------------------------------------------------------------------------
// 清空收件箱
// ---------------------------------------------------------------------------

// ClearMailboxRequest 清空收件箱请求
type ClearMailboxRequest struct {
	Email string `form:"email" binding:"required,email"`
}

// ClearMailbox 清空收件箱
// @Summary 清空收件箱
// @Description 通过 API Key 清空指定邮箱的所有邮件（软删除）
// @Tags public
// @Accept json
// @Produce json
// @Param email query string true "邮箱地址"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/public/mail/clear [delete]
func (h *PublicHandler) ClearMailbox(c *gin.Context) {
	var req ClearMailboxRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("[ERROR] Failed to get account by email: %s, error: %v", req.Email, err)
		dto.HandleServiceError(c, err)
		return
	}
	if account == nil {
		dto.NotFoundResponse(c, fmt.Sprintf("邮箱账户不存在: %s", req.Email))
		return
	}

	count, err := h.emailRepo.BatchSoftDeleteByAccountUID(c.Request.Context(), account.UID)
	if err != nil {
		log.Printf("[ERROR] Failed to clear mailbox: account=%s, error: %v", account.UID, err)
		dto.InternalServerErrorResponse(c, "清空收件箱失败")
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    gin.H{"count": count},
	})
}

// ---------------------------------------------------------------------------
// 已发送邮件列表
// ---------------------------------------------------------------------------

// ListSentEmailsRequest 已发送邮件列表请求
type ListSentEmailsRequest struct {
	Email    string `form:"email" binding:"required,email"`
	Status   string `form:"status"`
	Search   string `form:"search"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListSentEmails 获取已发送邮件列表
// @Summary 获取已发送邮件列表
// @Description 通过 API Key 获取指定邮箱的已发送邮件列表
// @Tags public
// @Accept json
// @Produce json
// @Param email query string true "邮箱地址"
// @Param status query string false "状态(sent/failed)"
// @Param search query string false "搜索关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/public/mail/sent [get]
func (h *PublicHandler) ListSentEmails(c *gin.Context) {
	var req ListSentEmailsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("[ERROR] Failed to get account by email: %s, error: %v", req.Email, err)
		dto.HandleServiceError(c, err)
		return
	}
	if account == nil {
		dto.NotFoundResponse(c, fmt.Sprintf("邮箱账户不存在: %s", req.Email))
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	listReq := &service.ListSentEmailsRequest{
		AccountUID:  account.UID,
		Status:      req.Status,
		SearchQuery: req.Search,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}

	result, err := h.sentEmailService.ListSentEmails(c.Request.Context(), listReq)
	if err != nil {
		log.Printf("[ERROR] Failed to list sent emails: %v", err)
		dto.HandleServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    result,
	})
}

// ---------------------------------------------------------------------------
// 批量导入邮箱账户
// ---------------------------------------------------------------------------
