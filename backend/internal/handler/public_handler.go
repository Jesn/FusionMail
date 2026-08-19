package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/repository"
	"fusionmail/internal/service"
	pkgredis "fusionmail/pkg/redis"
	"fusionmail/pkg/synclock"

	"github.com/gin-gonic/gin"
)

// PublicHandler 公共接口处理器
type PublicHandler struct {
	emailService     service.EmailService
	accountService   service.AccountService
	syncService      service.SyncService
	sendService      *service.SendService
	sentEmailService *service.SentEmailService
	emailRepo        repository.EmailRepository
	oauth2Service    *service.OAuth2Service
}

// NewPublicHandler 创建公共接口处理器实例
func NewPublicHandler(
	emailService service.EmailService,
	accountService service.AccountService,
	syncService service.SyncService,
	sendService *service.SendService,
	sentEmailService *service.SentEmailService,
	emailRepo repository.EmailRepository,
	oauth2Service *service.OAuth2Service,
) *PublicHandler {
	return &PublicHandler{
		emailService:     emailService,
		accountService:   accountService,
		syncService:      syncService,
		sendService:      sendService,
		sentEmailService: sentEmailService,
		emailRepo:        emailRepo,
		oauth2Service:    oauth2Service,
	}
}

func isSyncInProgressError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "sync already in progress")
}

func isWebhookModeError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "webhook mode")
}

func isSyncDisabledError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "sync is disabled for account")
}

// waitForSyncComplete 等待指定账号的同步完成（用于 sync already in progress 的场景）
func (h *PublicHandler) waitForSyncComplete(ctx context.Context, accountUID string) error {
	redisClient := pkgredis.GetClient()

	// 优先使用 Redis 锁判断（支持多实例部署）
	if redisClient != nil {
		lock := synclock.NewSyncLock(redisClient)
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			locked, err := lock.IsLocked(ctx, accountUID)
			if err != nil {
				return err
			}
			if !locked {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	}

	// 无 Redis 时降级：使用本地进度追踪器判断（仅适用于单实例）
	if h.syncService == nil {
		return nil
	}

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		if h.syncService.GetSyncProgress(accountUID) == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// ReceiveMailRequest 接收邮件请求
type ReceiveMailRequest struct {
	Email       string `form:"email" binding:"required,email"` // 邮箱地址
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset      int    `form:"offset" binding:"omitempty,min=0"`
	IsRead      *bool  `form:"is_read" binding:"omitempty"`      // 是否已读
	IsStarred   *bool  `form:"is_starred" binding:"omitempty"`   // 是否星标
	IsArchived  *bool  `form:"is_archived" binding:"omitempty"`  // 是否归档
	FromAddress string `form:"from_address" binding:"omitempty"` // 发件人地址
	Subject     string `form:"subject" binding:"omitempty"`      // 主题
	StartDate   string `form:"start_date" binding:"omitempty"`   // 开始日期
	EndDate     string `form:"end_date" binding:"omitempty"`     // 结束日期
}

// ReceiveMail 实时拉取邮件
// @Summary 实时拉取邮件
// @Description 通过 API Key 实时拉取指定邮箱的邮件列表
// @Tags public
// @Accept json
// @Produce json
// @Param email query string true "邮箱地址"
// @Param limit query int false "返回邮件数量（默认 20，最大 100）"
// @Param offset query int false "偏移量（默认 0）"
// @Param is_read query bool false "是否已读"
// @Param is_starred query bool false "是否星标"
// @Param is_archived query bool false "是否归档"
// @Param from_address query string false "发件人地址"
// @Param subject query string false "主题"
// @Param start_date query string false "开始日期（YYYY-MM-DD）"
// @Param end_date query string false "结束日期（YYYY-MM-DD）"
// @Security ApiKeyAuth
// @Success 200 {object} ReceiveMailResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/public/mail/receive [get]
func (h *PublicHandler) ReceiveMail(c *gin.Context) {
	var req ReceiveMailRequest

	// 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
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

	// 根据邮箱地址查找账户
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

	// 检查账户状态
	if account.Status != "active" {
		dto.BadRequestResponse(c, fmt.Sprintf("账户已禁用: %s", account.Status))
		return
	}

	// 实时同步：先从源邮箱服务器拉取最新数据，再返回本地查询结果
	if h.syncService != nil {
		if err := h.syncService.SyncAccount(c.Request.Context(), account.UID); err != nil {
			// 同步超时/取消
			if errors.Is(err, context.DeadlineExceeded) {
				dto.InternalServerErrorResponse(c, "实时同步超时")
				return
			}
			if errors.Is(err, context.Canceled) {
				dto.InternalServerErrorResponse(c, "实时同步已取消")
				return
			}

			// Webhook 模式无需轮询同步，直接返回本地数据
			if isWebhookModeError(err) {
				// no-op
			} else if isSyncInProgressError(err) {
				// 已有同步任务在进行中：等待其完成后再返回（尽量保证“实时”语义）
				if waitErr := h.waitForSyncComplete(c.Request.Context(), account.UID); waitErr != nil {
					dto.InternalServerErrorResponse(c, "等待同步完成失败: "+waitErr.Error())
					return
				}
			} else if isSyncDisabledError(err) {
				// 同步未启用，直接返回本地数据
				log.Printf("[INFO] Sync disabled for account %s, returning local data", account.UID)
			} else {
				log.Printf("[ERROR] Real-time sync failed: account=%s, email=%s, err=%v", account.UID, req.Email, err)
				dto.InternalServerErrorResponse(c, "实时同步失败: "+err.Error())
				return
			}
		}
	}

	// 构建过滤条件
	filter := &service.EmailQueryParams{
		AccountUID:  account.UID,
		IsRead:      req.IsRead,
		IsStarred:   req.IsStarred,
		IsArchived:  req.IsArchived,
		FromAddress: req.FromAddress,
		Subject:     req.Subject,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	// 默认不显示已删除的邮件
	isDeleted := false
	filter.IsDeleted = &isDeleted

	// 调用服务层获取邮件列表
	emails, total, err := h.emailService.GetEmailListWithFilter(c.Request.Context(), filter, req.Offset, req.Limit)
	if err != nil {
		log.Printf("[ERROR] Failed to get email list: %v", err)
		dto.HandleServiceError(c, err)
		return
	}

	// 记录 API Key 使用（异步）
	go func() {
		if err := h.logAPIKeyUsage(c.Request.Context(), apiKeyIDInt, req.Email, len(emails)); err != nil {
			log.Printf("[WARN] Failed to log API Key usage: %v", err)
		}
	}()

	// 返回响应
	response := ReceiveMailResponse{
		Success: true,
		Data: MailListData{
			Emails: emails,
			Total:  total,
			Limit:  req.Limit,
			Offset: req.Offset,
		},
	}

	c.JSON(200, response)
}

// ReceiveMailResponse 接收邮件响应
type ReceiveMailResponse struct {
	Success bool         `json:"success"`
	Data    MailListData `json:"data"`
}

// MailListData 邮件列表数据
type MailListData struct {
	Emails interface{} `json:"emails"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// logAPIKeyUsage 记录 API Key 使用情况（占位符）
func (h *PublicHandler) logAPIKeyUsage(ctx interface{}, apiKeyID int64, email string, emailCount int) error {
	// TODO: 实现 API Key 使用日志记录
	// 可以在这里记录 API Key 的使用情况，用于审计和分析
	log.Printf("[INFO] API Key %d used to fetch %d emails from %s", apiKeyID, emailCount, email)
	return nil
}

// SearchMailRequest 搜索邮件请求
type SearchMailRequest struct {
	Email  string `form:"email" binding:"required,email"` // 邮箱地址
	Query  string `form:"q" binding:"required"`           // 搜索关键词
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int    `form:"offset" binding:"omitempty,min=0"`
}

// SearchMail 搜索邮件
// @Summary 搜索邮件
// @Description 通过 API Key 搜索指定邮箱的邮件
// @Tags public
// @Accept json
// @Produce json
// @Param email query string true "邮箱地址"
// @Param q query string true "搜索关键词"
// @Param limit query int false "返回邮件数量（默认 20，最大 100）"
// @Param offset query int false "偏移量（默认 0）"
// @Security ApiKeyAuth
// @Success 200 {object} SearchMailResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/public/mail/search [get]
func (h *PublicHandler) SearchMail(c *gin.Context) {
	var req SearchMailRequest

	// 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
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

	// 根据邮箱地址查找账户
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

	// 检查账户状态
	if account.Status != "active" {
		dto.BadRequestResponse(c, fmt.Sprintf("账户已禁用: %s", account.Status))
		return
	}

	// 调用服务层搜索邮件
	emails, total, err := h.emailService.SearchEmailsWithFilter(c.Request.Context(), req.Query, account.UID, req.Offset, req.Limit)
	if err != nil {
		log.Printf("[ERROR] Failed to search emails: %v", err)
		dto.HandleServiceError(c, err)
		return
	}

	// 记录 API Key 使用（异步）
	go func() {
		if err := h.logAPIKeyUsage(c.Request.Context(), apiKeyIDInt, req.Email, len(emails)); err != nil {
			log.Printf("[WARN] Failed to log API Key usage: %v", err)
		}
	}()

	// 返回响应
	response := SearchMailResponse{
		Success: true,
		Data: SearchMailData{
			Emails: emails,
			Total:  total,
			Limit:  req.Limit,
			Offset: req.Offset,
			Query:  req.Query,
		},
	}

	c.JSON(200, response)
}

// SearchMailResponse 搜索邮件响应
type SearchMailResponse struct {
	Success bool           `json:"success"`
	Data    SearchMailData `json:"data"`
}

// SearchMailData 搜索邮件数据
type SearchMailData struct {
	Emails interface{} `json:"emails"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Query  string      `json:"query"`
}

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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
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
			"message_id":     result.MessageID,
			"sent_email_id":  result.SentEmailID,
			"sender_type":    result.SenderType,
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
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
	Email      string `form:"email" binding:"required,email"`
	Status     string `form:"status"`
	Search     string `form:"search"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
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

// BatchImportAccounts 通过 API Key 批量导入邮箱账户
// @Summary 批量导入邮箱账户
// @Description 通过 API Key 批量导入 Outlook 邮箱账户，支持自定义分隔符和字段顺序
// @Tags public
// @Accept json
// @Produce json
// @Param body body BatchImportRequest true "批量导入请求"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/public/mail/import-accounts [post]
func (h *PublicHandler) BatchImportAccounts(c *gin.Context) {
	var req BatchImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求格式错误: "+err.Error())
		return
	}

	if len(req.Accounts) == 0 {
		dto.BadRequestResponse(c, "账户列表不能为空")
		return
	}

	if len(req.Accounts) > 50 {
		dto.BadRequestResponse(c, "单次最多导入 50 个账户")
		return
	}

	response := BatchImportResponse{
		Success: 0,
		Failed:  0,
		Results: make([]BatchImportResult, 0, len(req.Accounts)),
	}

	for _, accountString := range req.Accounts {
		result := h.importSingleAccountPublic(c.Request.Context(), accountString, req.Format, req.SyncEnabled, req.SyncInterval, req.GroupID, req.FirstSyncDays)
		response.Results = append(response.Results, result)
		if result.Status == "success" {
			response.Success++
		} else {
			response.Failed++
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    response,
	})
}

// importSingleAccountPublic 公共 API 版本的单个账户导入逻辑
func (h *PublicHandler) importSingleAccountPublic(ctx context.Context, accountString string, format *ImportFormatConfig, syncEnabled *bool, syncInterval *int, groupID *int64, firstSyncDays *int) BatchImportResult {
	var config *adapter.Config
	var err error
	delimiter := adapter.QuickAccountSeparator
	if format != nil && format.Delimiter != "" {
		delimiter = format.Delimiter
	}

	if format != nil && len(format.Fields) > 0 {
		config, err = adapter.ParseAccountStringWithFormat(accountString, format.Delimiter, format.Fields)
	} else {
		config, err = adapter.ParseQuickAccountString(accountString)
	}
	if err != nil {
		return BatchImportResult{
			Email:  extractEmailFromString(accountString, delimiter),
			Status: "failed",
			Error:  "账户格式错误: " + err.Error(),
		}
	}

	if config.Provider == "outlook" {
		if config.Credentials.RefreshToken == "" {
			return BatchImportResult{
				Email:  config.Email,
				Status: "failed",
				Error:  "Outlook 账户缺少刷新令牌",
			}
		}

		err = h.oauth2Service.ValidateMicrosoftAccount(ctx, config.Credentials.RefreshToken, config.Credentials.ClientID)
		if err != nil {
			return BatchImportResult{
				Email:  config.Email,
				Status: "failed",
				Error:  "Outlook 账户验证失败: " + err.Error(),
			}
		}
	}

	syncEnabledVal := true
	syncIntervalVal := 2
	firstSyncDaysVal := 7
	if syncEnabled != nil {
		syncEnabledVal = *syncEnabled
	}
	if syncInterval != nil {
		syncIntervalVal = *syncInterval
	}
	if firstSyncDays != nil {
		firstSyncDaysVal = *firstSyncDays
	}

	createReq := &service.CreateAccountRequest{
		Email:         config.Email,
		Provider:      config.Provider,
		Protocol:      "graph_quick",
		AuthType:      "quick",
		RefreshToken:  config.Credentials.RefreshToken,
		ClientID:      config.Credentials.ClientID,
		Password:      config.Credentials.Password,
		SyncEnabled:   syncEnabledVal,
		SyncInterval:  syncIntervalVal,
		FirstSyncDays: firstSyncDaysVal,
		GroupID:       groupID,
	}

	_, err = h.accountService.Create(ctx, createReq)
	if err != nil {
		return BatchImportResult{
			Email:  config.Email,
			Status: "failed",
			Error:  err.Error(),
		}
	}

	return BatchImportResult{
		Email:  config.Email,
		Status: "success",
	}
}
