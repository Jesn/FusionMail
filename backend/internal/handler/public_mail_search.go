package handler

import (
	"fmt"
	"log"

	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"

	"github.com/gin-gonic/gin"
)

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
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
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
			Emails: response.NewPublicEmailItems(emails),
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
	Emails []response.PublicEmailItem `json:"emails"`
	Total  int64                      `json:"total"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
	Query  string                     `json:"query"`
}
