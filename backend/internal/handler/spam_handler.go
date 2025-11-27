package handler

import (
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// SpamHandler 垃圾邮件处理器
type SpamHandler struct {
	spamService service.SpamService
}

// NewSpamHandler 创建垃圾邮件处理器
func NewSpamHandler(spamService service.SpamService) *SpamHandler {
	return &SpamHandler{
		spamService: spamService,
	}
}

// MarkAsSpam 标记邮件为垃圾邮件
// POST /api/v1/spam/mark
func (h *SpamHandler) MarkAsSpam(c *gin.Context) {
	var req dto.MarkSpamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	// 验证请求
	if len(req.EmailIDs) == 0 {
		dto.BadRequestResponse(c, "邮件 ID 列表不能为空")
		return
	}

	// 标记为垃圾邮件
	if err := h.spamService.MarkAsSpam(c.Request.Context(), req.EmailIDs); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"marked_count": len(req.EmailIDs),
	})
}

// UnmarkAsSpam 取消垃圾邮件标记
// POST /api/v1/spam/unmark
func (h *SpamHandler) UnmarkAsSpam(c *gin.Context) {
	var req dto.MarkSpamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	// 验证请求
	if len(req.EmailIDs) == 0 {
		dto.BadRequestResponse(c, "邮件 ID 列表不能为空")
		return
	}

	// 取消垃圾邮件标记
	if err := h.spamService.UnmarkAsSpam(c.Request.Context(), req.EmailIDs); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"unmarked_count": len(req.EmailIDs),
	})
}

// BatchDeleteSpam 批量删除垃圾邮件
// DELETE /api/v1/spam/batch
func (h *SpamHandler) BatchDeleteSpam(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	// 验证请求
	if len(req.EmailIDs) == 0 {
		dto.BadRequestResponse(c, "邮件 ID 列表不能为空")
		return
	}

	// 批量删除
	deletedCount, err := h.spamService.BatchDeleteSpam(c.Request.Context(), req.EmailIDs)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"deleted_count": deletedCount,
	})
}

// EmptySpamFolder 清空垃圾箱
// POST /api/v1/spam/empty
func (h *SpamHandler) EmptySpamFolder(c *gin.Context) {
	// 可选：指定账户 UID
	accountUID := c.Query("account_uid")

	// 清空垃圾箱
	deletedCount, err := h.spamService.EmptySpamFolder(c.Request.Context(), accountUID)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"deleted_count": deletedCount,
	})
}

// GetSpamEmails 获取垃圾邮件列表
// GET /api/v1/spam/emails
func (h *SpamHandler) GetSpamEmails(c *gin.Context) {
	// 解析查询参数
	accountUID := c.Query("account_uid")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 获取垃圾邮件列表
	emails, total, err := h.spamService.GetSpamEmails(c.Request.Context(), accountUID, page, pageSize)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 使用分页响应
	dto.PaginatedSuccessResponse(c, emails, total, page, pageSize)
}

// GetSpamStats 获取垃圾邮件统计
// GET /api/v1/spam/stats
func (h *SpamHandler) GetSpamStats(c *gin.Context) {
	accountUID := c.Query("account_uid")

	stats, err := h.spamService.GetSpamStats(c.Request.Context(), accountUID)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, stats)
}

// GetBayesianStatus 获取贝叶斯模型状态
// GET /api/v1/spam/bayesian/status
func (h *SpamHandler) GetBayesianStatus(c *gin.Context) {
	// 从 JWT 中获取用户 UID（这里简化处理，使用 account_uid 参数）
	userUID := c.Query("user_uid")
	if userUID == "" {
		dto.BadRequestResponse(c, "用户 UID 不能为空")
		return
	}

	status, err := h.spamService.GetBayesianStatus(c.Request.Context(), userUID)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, status)
}

// TrainBayesianModel 手动训练贝叶斯模型
// POST /api/v1/spam/bayesian/train
func (h *SpamHandler) TrainBayesianModel(c *gin.Context) {
	var req struct {
		UserUID string `json:"user_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	if err := h.spamService.TrainBayesianModel(c.Request.Context(), req.UserUID); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"message": "贝叶斯模型训练成功",
	})
}

// ResetBayesianModel 重置贝叶斯模型
// POST /api/v1/spam/bayesian/reset
func (h *SpamHandler) ResetBayesianModel(c *gin.Context) {
	var req struct {
		UserUID string `json:"user_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	if err := h.spamService.ResetBayesianModel(c.Request.Context(), req.UserUID); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"message": "贝叶斯模型已重置",
	})
}

// GetBayesianTrainingStats 获取贝叶斯训练统计
// GET /api/v1/spam/bayesian/stats
func (h *SpamHandler) GetBayesianTrainingStats(c *gin.Context) {
	userUID := c.Query("user_uid")
	if userUID == "" {
		dto.BadRequestResponse(c, "用户 UID 不能为空")
		return
	}

	stats, err := h.spamService.GetBayesianTrainingStats(c.Request.Context(), userUID)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, stats)
}
