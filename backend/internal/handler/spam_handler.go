package handler

import (
	"strconv"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
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

// GetRules 获取规则列表
// GET /api/v1/spam/rules
func (h *SpamHandler) GetRules(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	rules, total, err := h.spamService.GetRules(c.Request.Context(), category, page, pageSize)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 转换为响应格式
	responses := make([]*dto.SpamRuleResponse, len(rules))
	for i, rule := range rules {
		responses[i] = &dto.SpamRuleResponse{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Description,
			Category:    rule.Category,
			Pattern:     rule.Pattern,
			Score:       rule.Score,
			Enabled:     rule.Enabled,
			IsBuiltin:   rule.IsBuiltin,
			HitCount:    rule.HitCount,
			CreatedAt:   rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   rule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	dto.PaginatedSuccessResponse(c, responses, total, page, pageSize)
}

// GetRule 获取单个规则
// GET /api/v1/spam/rules/:id
func (h *SpamHandler) GetRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的规则 ID")
		return
	}

	rule, err := h.spamService.GetRuleByID(c.Request.Context(), id)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}
	if rule == nil {
		dto.NotFoundResponse(c, "规则不存在")
		return
	}

	dto.SuccessResponse(c, &dto.SpamRuleResponse{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Category:    rule.Category,
		Pattern:     rule.Pattern,
		Score:       rule.Score,
		Enabled:     rule.Enabled,
		IsBuiltin:   rule.IsBuiltin,
		HitCount:    rule.HitCount,
		CreatedAt:   rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   rule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// CreateRule 创建规则
// POST /api/v1/spam/rules
func (h *SpamHandler) CreateRule(c *gin.Context) {
	var req dto.SpamRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	// 设置默认值
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	score := req.Score
	if score == 0 {
		score = 10 // 默认评分
	}

	rule := &model.SpamRule{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Pattern:     req.Pattern,
		Score:       score,
		Enabled:     enabled,
	}

	if err := h.spamService.CreateRule(c.Request.Context(), rule); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, &dto.SpamRuleResponse{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Category:    rule.Category,
		Pattern:     rule.Pattern,
		Score:       rule.Score,
		Enabled:     rule.Enabled,
		IsBuiltin:   rule.IsBuiltin,
		HitCount:    rule.HitCount,
		CreatedAt:   rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   rule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateRule 更新规则
// PUT /api/v1/spam/rules/:id
func (h *SpamHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的规则 ID")
		return
	}

	var req dto.SpamRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	// 设置默认值
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	score := req.Score
	if score == 0 {
		score = 10 // 默认评分
	}

	rule := &model.SpamRule{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Pattern:     req.Pattern,
		Score:       score,
		Enabled:     enabled,
	}

	if err := h.spamService.UpdateRule(c.Request.Context(), rule); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 重新获取更新后的规则
	updatedRule, _ := h.spamService.GetRuleByID(c.Request.Context(), id)
	if updatedRule != nil {
		dto.SuccessResponse(c, &dto.SpamRuleResponse{
			ID:          updatedRule.ID,
			Name:        updatedRule.Name,
			Description: updatedRule.Description,
			Category:    updatedRule.Category,
			Pattern:     updatedRule.Pattern,
			Score:       updatedRule.Score,
			Enabled:     updatedRule.Enabled,
			IsBuiltin:   updatedRule.IsBuiltin,
			HitCount:    updatedRule.HitCount,
			CreatedAt:   updatedRule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   updatedRule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	} else {
		dto.SuccessResponse(c, gin.H{"message": "规则更新成功"})
	}
}

// DeleteRule 删除规则
// DELETE /api/v1/spam/rules/:id
func (h *SpamHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的规则 ID")
		return
	}

	if err := h.spamService.DeleteRule(c.Request.Context(), id); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{"message": "规则删除成功"})
}

// ToggleRule 切换规则启用状态
// PUT /api/v1/spam/rules/:id/toggle
func (h *SpamHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的规则 ID")
		return
	}

	if err := h.spamService.ToggleRule(c.Request.Context(), id); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 获取更新后的规则状态
	rule, _ := h.spamService.GetRuleByID(c.Request.Context(), id)
	if rule != nil {
		dto.SuccessResponse(c, gin.H{
			"id":      rule.ID,
			"enabled": rule.Enabled,
			"message": "规则状态已切换",
		})
	} else {
		dto.SuccessResponse(c, gin.H{"message": "规则状态已切换"})
	}
}

// TestRule 测试规则
// POST /api/v1/spam/rules/test
func (h *SpamHandler) TestRule(c *gin.Context) {
	var req dto.SpamRuleTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	start := time.Now()
	matched, matches, err := h.spamService.TestRule(c.Request.Context(), req.Pattern, req.Category, req.Content)
	duration := time.Since(start)

	response := &dto.SpamRuleTestResponse{
		Matched:  matched,
		Matches:  matches,
		Duration: duration.String(),
	}

	if err != nil {
		response.Error = err.Error()
	}

	dto.SuccessResponse(c, response)
}

// GetRuleStats 获取规则统计
// GET /api/v1/spam/rules/stats
func (h *SpamHandler) GetRuleStats(c *gin.Context) {
	stats, err := h.spamService.GetRuleStats(c.Request.Context())
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, &dto.SpamRuleStatsResponse{
		TotalCount:    stats.TotalCount,
		EnabledCount:  stats.EnabledCount,
		DisabledCount: stats.DisabledCount,
		BuiltinCount:  stats.BuiltinCount,
		CustomCount:   stats.CustomCount,
		TotalHits:     stats.TotalHits,
	})
}
