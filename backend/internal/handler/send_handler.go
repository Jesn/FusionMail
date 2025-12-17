package handler

import (
	"net/http"
	"strconv"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// SendHandler 发送邮件 API 处理器
type SendHandler struct {
	sendService       *service.SendService
	sentEmailService  *service.SentEmailService
	smtpConfigService *service.SMTPConfigService
}

// NewSendHandler 创建发送邮件处理器
func NewSendHandler(
	sendService *service.SendService,
	sentEmailService *service.SentEmailService,
	smtpConfigService *service.SMTPConfigService,
) *SendHandler {
	return &SendHandler{
		sendService:       sendService,
		sentEmailService:  sentEmailService,
		smtpConfigService: smtpConfigService,
	}
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	AccountUID string   `json:"account_uid" binding:"required"`
	To         []string `json:"to" binding:"required,min=1"`
	Cc         []string `json:"cc"`
	Bcc        []string `json:"bcc"`
	Subject    string   `json:"subject"`
	TextBody   string   `json:"text_body"`
	HTMLBody   string   `json:"html_body"`
	ReplyTo    string   `json:"reply_to"`
}

// SendEmail 发送邮件
// @Summary 发送邮件
// @Description 通过指定账户发送邮件
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param request body SendEmailRequest true "发送邮件请求"
// @Success 200 {object} dto.Response{data=service.SendEmailResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/send [post]
func (h *SendHandler) SendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 构建服务请求
	serviceReq := &service.SendEmailRequest{
		AccountUID: req.AccountUID,
		To:         req.To,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		Subject:    req.Subject,
		TextBody:   req.TextBody,
		HTMLBody:   req.HTMLBody,
		ReplyTo:    req.ReplyTo,
	}

	// 发送邮件
	result, err := h.sendService.SendEmail(c.Request.Context(), serviceReq)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, result.Error)
		return
	}

	dto.SuccessResponse(c, result)
}

// ReplyRequest 回复邮件请求
type ReplyRequest struct {
	AccountUID string `json:"account_uid" binding:"required"`
	TextBody   string `json:"text_body"`
	HTMLBody   string `json:"html_body"`
}

// Reply 回复邮件
// @Summary 回复邮件
// @Description 回复指定邮件
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param id path int true "邮件ID"
// @Param request body ReplyRequest true "回复请求"
// @Success 200 {object} dto.Response{data=service.SendEmailResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/{id}/reply [post]
func (h *SendHandler) Reply(c *gin.Context) {
	emailID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "无效的邮件ID")
		return
	}

	var req ReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	serviceReq := &service.ReplyRequest{
		TextBody: req.TextBody,
		HTMLBody: req.HTMLBody,
	}

	result, err := h.sendService.Reply(c.Request.Context(), emailID, req.AccountUID, serviceReq)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, result)
}

// ReplyAll 全部回复
// @Summary 全部回复
// @Description 回复邮件给所有收件人
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param id path int true "邮件ID"
// @Param request body ReplyRequest true "回复请求"
// @Success 200 {object} dto.Response{data=service.SendEmailResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/{id}/reply-all [post]
func (h *SendHandler) ReplyAll(c *gin.Context) {
	emailID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "无效的邮件ID")
		return
	}

	var req ReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	serviceReq := &service.ReplyRequest{
		TextBody: req.TextBody,
		HTMLBody: req.HTMLBody,
	}

	result, err := h.sendService.ReplyAll(c.Request.Context(), emailID, req.AccountUID, serviceReq)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, result)
}

// ForwardRequest 转发邮件请求
type ForwardRequest struct {
	AccountUID string   `json:"account_uid" binding:"required"`
	To         []string `json:"to" binding:"required,min=1"`
	Cc         []string `json:"cc"`
	TextBody   string   `json:"text_body"`
	HTMLBody   string   `json:"html_body"`
}

// Forward 转发邮件
// @Summary 转发邮件
// @Description 转发指定邮件
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param id path int true "邮件ID"
// @Param request body ForwardRequest true "转发请求"
// @Success 200 {object} dto.Response{data=service.SendEmailResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/{id}/forward [post]
func (h *SendHandler) Forward(c *gin.Context) {
	emailID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "无效的邮件ID")
		return
	}

	var req ForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	serviceReq := &service.ForwardRequest{
		To:       req.To,
		Cc:       req.Cc,
		TextBody: req.TextBody,
		HTMLBody: req.HTMLBody,
	}

	result, err := h.sendService.Forward(c.Request.Context(), emailID, req.AccountUID, serviceReq)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, result)
}

// ListSentEmails 获取已发送邮件列表
// @Summary 获取已发送邮件列表
// @Description 获取已发送邮件列表，支持分页和筛选
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param account_uid query string false "账户UID"
// @Param status query string false "状态(sent/failed)"
// @Param search query string false "搜索关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} dto.Response{data=service.ListSentEmailsResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/sent [get]
func (h *SendHandler) ListSentEmails(c *gin.Context) {
	var req service.ListSentEmailsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	result, err := h.sentEmailService.ListSentEmails(c.Request.Context(), &req)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, result)
}

// GetSentEmail 获取已发送邮件详情
// @Summary 获取已发送邮件详情
// @Description 获取指定已发送邮件的详细信息
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param id path int true "已发送邮件ID"
// @Success 200 {object} dto.Response{data=model.SentEmail}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/sent/{id} [get]
func (h *SendHandler) GetSentEmail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "无效的ID")
		return
	}

	email, err := h.sentEmailService.GetSentEmail(c.Request.Context(), id)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if email == nil {
		dto.ErrorResponse(c, http.StatusNotFound, "邮件不存在")
		return
	}

	dto.SuccessResponse(c, email)
}

// UpdateSMTPConfigRequest SMTP 配置请求
// 注意：host/port/encryption 从 Provider 继承，Account 只需配置用户名和密码
type UpdateSMTPConfigRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// UpdateSMTPConfig 更新 SMTP 配置
// @Summary 更新 SMTP 配置
// @Description 更新账户的 SMTP 发送配置
// @Tags 账户管理
// @Accept json
// @Produce json
// @Param uid path string true "账户UID"
// @Param request body UpdateSMTPConfigRequest true "SMTP配置"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/accounts/{uid}/smtp [put]
func (h *SendHandler) UpdateSMTPConfig(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		dto.ErrorResponse(c, http.StatusBadRequest, "账户UID不能为空")
		return
	}

	var req UpdateSMTPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	serviceReq := &service.SMTPConfigRequest{
		Username: req.Username,
		Password: req.Password,
		Enabled:  req.Enabled,
	}

	if err := h.smtpConfigService.UpdateSMTPConfig(c.Request.Context(), uid, serviceReq); err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "SMTP 配置已更新"})
}

// GetSMTPConfig 获取 SMTP 配置
// @Summary 获取 SMTP 配置
// @Description 获取账户的 SMTP 发送配置
// @Tags 账户管理
// @Accept json
// @Produce json
// @Param uid path string true "账户UID"
// @Success 200 {object} dto.Response{data=service.SMTPConfigResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/accounts/{uid}/smtp [get]
func (h *SendHandler) GetSMTPConfig(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		dto.ErrorResponse(c, http.StatusBadRequest, "账户UID不能为空")
		return
	}

	config, err := h.smtpConfigService.GetSMTPConfig(c.Request.Context(), uid)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, config)
}

// SMTPTestRequest SMTP 测试请求（可选参数）
type SMTPTestRequest struct {
	Password string `json:"password"` // 临时密码，用于测试未保存的配置
	Username string `json:"username"` // 临时用户名，用于测试未保存的配置
}

// TestSMTPConnection 测试 SMTP 连接
// @Summary 测试 SMTP 连接
// @Description 测试账户的 SMTP 连接是否正常，支持传入临时密码进行测试
// @Tags 账户管理
// @Accept json
// @Produce json
// @Param uid path string true "账户UID"
// @Param body body SMTPTestRequest false "测试参数（可选）"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/accounts/{uid}/smtp/test [post]
func (h *SendHandler) TestSMTPConnection(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		dto.ErrorResponse(c, http.StatusBadRequest, "账户UID不能为空")
		return
	}

	// 解析可选的请求体参数
	var req SMTPTestRequest
	// 忽略绑定错误，因为请求体是可选的
	_ = c.ShouldBindJSON(&req)

	if err := h.smtpConfigService.TestSMTPConnection(c.Request.Context(), uid, req.Username, req.Password); err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "SMTP 连接测试成功"})
}

// GetDefaultSMTPConfigs 获取默认 SMTP 配置列表
// @Summary 获取默认 SMTP 配置列表
// @Description 获取常见邮箱服务商的默认 SMTP 配置
// @Tags 账户管理
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/smtp/defaults [get]
func (h *SendHandler) GetDefaultSMTPConfigs(c *gin.Context) {
	configs := service.GetAllDefaultSMTPConfigs()
	dto.SuccessResponse(c, configs)
}

// UploadAttachment 上传附件
// @Summary 上传附件
// @Description 上传邮件附件
// @Tags 邮件发送
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "附件文件"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/attachments [post]
func (h *SendHandler) UploadAttachment(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}

	// 检查文件大小
	if file.Size > adapter.MaxSingleAttachmentSize {
		dto.ErrorResponse(c, http.StatusBadRequest, "文件大小超过限制（最大 25MB）")
		return
	}

	// 打开文件
	f, err := file.Open()
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "无法读取文件")
		return
	}
	defer f.Close()

	// 读取文件内容
	content := make([]byte, file.Size)
	if _, err := f.Read(content); err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "读取文件失败")
		return
	}

	// 返回附件信息（实际应用中可能需要保存到临时存储）
	dto.SuccessResponse(c, gin.H{
		"filename":     file.Filename,
		"size":         file.Size,
		"content_type": file.Header.Get("Content-Type"),
	})
}
