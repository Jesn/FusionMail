package handler

import (
	"context"
	"errors"
	"fmt"
	"log"

	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

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
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
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
			Emails: response.NewPublicEmailItems(emails),
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
	Emails []response.PublicEmailItem `json:"emails"`
	Total  int64                      `json:"total"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
}
