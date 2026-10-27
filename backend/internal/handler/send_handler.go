package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// SendHandler 发送邮件 API 处理器
type SendHandler struct {
	sendService       *service.SendService
	sentEmailService  *service.SentEmailService
	smtpConfigService *service.SMTPConfigService
	tempStore         *AttachmentTempStore // 待发送附件的临时文件生命周期
}

// NewSendHandler 创建发送邮件处理器
func NewSendHandler(
	sendService *service.SendService,
	sentEmailService *service.SentEmailService,
	smtpConfigService *service.SMTPConfigService,
	tempStore *AttachmentTempStore,
) *SendHandler {
	return &SendHandler{
		sendService:       sendService,
		sentEmailService:  sentEmailService,
		smtpConfigService: smtpConfigService,
		tempStore:         tempStore,
	}
}

// AttachmentInput 客户端回传的待发送附件（引用 UploadAttachment 返回的 temp_path）。
type AttachmentInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	TempPath    string `json:"temp_path" binding:"required"`
}

// toOutgoingAttachments 把客户端附件引用读取为 service 层附件。
// 读取失败（temp_path 非法/越界/超限）整体报错，不让半份附件静默发出。
func (h *SendHandler) toOutgoingAttachments(inputs []AttachmentInput) ([]adapter.OutgoingAttachment, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	out := make([]adapter.OutgoingAttachment, 0, len(inputs))
	var total int64
	for i, in := range inputs {
		content, err := h.tempStore.Read(in.TempPath)
		if err != nil {
			return nil, fmt.Errorf("附件 #%d 无效: %w", i+1, err)
		}
		total += int64(len(content))
		if total > adapter.MaxTotalAttachmentSize {
			return nil, fmt.Errorf("附件总大小超过限制（最大 %d 字节）", adapter.MaxTotalAttachmentSize)
		}
		out = append(out, adapter.OutgoingAttachment{
			Filename:    filepath.Base(in.Filename), // 客户端文件名只取 base，防路径注入
			ContentType: in.ContentType,
			Content:     content,
		})
	}
	return out, nil
}

// cleanupAttachments 发送结束后清理临时文件（无论成败）。
func (h *SendHandler) cleanupAttachments(inputs []AttachmentInput) {
	for _, in := range inputs {
		h.tempStore.Remove(in.TempPath)
	}
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	AccountUID  string            `json:"account_uid" binding:"required"`
	To          []string          `json:"to" binding:"required,min=1"`
	Cc          []string          `json:"cc"`
	Bcc         []string          `json:"bcc"`
	Subject     string            `json:"subject"`
	TextBody    string            `json:"text_body"`
	HTMLBody    string            `json:"html_body"`
	Attachments []AttachmentInput `json:"attachments"`
	ReplyTo     string            `json:"reply_to"`
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
	attachments, err := h.toOutgoingAttachments(req.Attachments)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	defer h.cleanupAttachments(req.Attachments) // 发送结束（成败均）清理临时文件

	serviceReq := &service.SendEmailRequest{
		AccountUID:  req.AccountUID,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		TextBody:    req.TextBody,
		HTMLBody:    req.HTMLBody,
		Attachments: attachments,
		ReplyTo:     req.ReplyTo,
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
	AccountUID  string            `json:"account_uid" binding:"required"`
	TextBody    string            `json:"text_body"`
	HTMLBody    string            `json:"html_body"`
	Attachments []AttachmentInput `json:"attachments"`
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

	attachments, err := h.toOutgoingAttachments(req.Attachments)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	defer h.cleanupAttachments(req.Attachments)

	serviceReq := &service.ReplyRequest{
		TextBody:    req.TextBody,
		HTMLBody:    req.HTMLBody,
		Attachments: attachments,
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
	AccountUID  string            `json:"account_uid" binding:"required"`
	To          []string          `json:"to" binding:"required,min=1"`
	Cc          []string          `json:"cc"`
	TextBody    string            `json:"text_body"`
	HTMLBody    string            `json:"html_body"`
	Attachments []AttachmentInput `json:"attachments"`
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

	attachments, err := h.toOutgoingAttachments(req.Attachments)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	defer h.cleanupAttachments(req.Attachments)

	serviceReq := &service.ForwardRequest{
		To:          req.To,
		Cc:          req.Cc,
		TextBody:    req.TextBody,
		HTMLBody:    req.HTMLBody,
		Attachments: attachments,
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
// @Success 200 {object} dto.Response{data=response.SentEmailResponse}
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
		// 不把 err.Error() 直接返回给前端：会泄露 SQL 片段与内部路径
		dto.HandleServiceError(c, err)
		return
	}

	if email == nil {
		dto.NotFoundResponse(c, "邮件不��在")
		return
	}

	// 返回 DTO 而不是 *model.SentEmail：后者带 gorm tag，
	// 会让 created_at / updated_at / deleted_at 等存储层字段进入 API 契约。
	dto.SuccessResponse(c, response.NewSentEmailResponse(email))
}

// DeleteSentEmail 删除已发送邮件
// @Summary 删除已发送邮件
// @Description 删除指定的已发送邮件记录
// @Tags 邮件发送
// @Accept json
// @Produce json
// @Param id path int true "已发送邮件ID"
// @Success 200 {object} dto.Response{data=response.MessageResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/emails/sent/{id} [delete]
func (h *SendHandler) DeleteSentEmail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.ErrorResponse(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.sentEmailService.DeleteSentEmail(c.Request.Context(), id); err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "已发送邮件已删除"})
}

// UpdateSMTPConfigRequest SMTP 配置请求
// 注意：SMTP 使用与 IMAP/POP3 相同的邮箱地址和密码
// Account 级别只需配置启用状态
type UpdateSMTPConfigRequest struct {
	Enabled bool `json:"enabled"`
}

// UpdateSMTPConfig 更新 SMTP 配置
// @Summary 更新 SMTP 配置
// @Description 更新账户的 SMTP 发送配置（SMTP 使用与接收邮件相同的凭证）
// @Tags 账户管理
// @Accept json
// @Produce json
// @Param uid path string true "账户UID"
// @Param request body UpdateSMTPConfigRequest true "SMTP配置"
// @Success 200 {object} dto.Response{data=response.MessageResponse}
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
		Enabled: req.Enabled,
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
// @Success 200 {object} dto.Response{data=response.MessageResponse}
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
// @Success 200 {object} dto.Response{data=response.DefaultSMTPConfigMap}
// @Router /api/v1/smtp/defaults [get]
func (h *SendHandler) GetDefaultSMTPConfigs(c *gin.Context) {
	configs := service.GetAllDefaultSMTPConfigs()
	mapped := make(map[string]response.DefaultSMTPConfigResponse, len(configs))
	for k, v := range configs {
		mapped[k] = response.DefaultSMTPConfigResponse{Host: v.Host, Port: v.Port, Encryption: v.Encryption}
	}
	dto.SuccessResponse(c, mapped)
}

// UploadAttachment 上传附件
// @Summary 上传附件
// @Description 上传邮件附件
// @Tags 邮件发送
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "附件文件"
// @Success 200 {object} dto.Response{data=response.AttachmentUploadResponse}
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

	// 落盘到临时存储；temp_path 由发送请求回传，发送结束后清理。
	// 顺带清理超过 24 小时的孤儿文件（上传后从未发送），避免无后台任务下的无限增长。
	h.tempStore.RemoveExpired(24 * time.Hour)
	tempPath, err := h.tempStore.Save(content)
	if err != nil {
		dto.ErrorResponse(c, http.StatusInternalServerError, "保存附件失败: "+err.Error())
		return
	}

	dto.SuccessResponse(c, response.AttachmentUploadResponse{
		Filename:    file.Filename,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
		TempPath:    tempPath,
	})
}
