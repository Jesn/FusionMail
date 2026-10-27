package handler

import (
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/dto/request"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// RuleHandler 规则 API 处理器
type RuleHandler struct {
	ruleService service.RuleService
}

// NewRuleHandler 创建规则处理器实例
func NewRuleHandler(ruleService service.RuleService) *RuleHandler {
	return &RuleHandler{
		ruleService: ruleService,
	}
}

// CreateRule 创建规则
// @Summary 创建规则
// @Description 创建新的邮件处理规则
// @Tags rules
// @Accept json
// @Produce json
// @Param body body request.CreateRuleRequest true "规则信息"
// @Success 200 {object} dto.Response{data=response.EmailRuleResponse}
// @Router /api/v1/rules [post]
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req request.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	rule := &model.EmailRule{
		Name:           req.Name,
		AccountUID:     req.AccountUID,
		Description:    req.Description,
		Enabled:        req.Enabled,
		Priority:       req.Priority,
		MatchMode:      req.MatchMode,
		Conditions:     req.Conditions,
		Actions:        req.Actions,
		StopProcessing: req.StopProcessing,
	}

	if err := h.ruleService.Create(c.Request.Context(), rule); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, response.NewEmailRuleResponse(rule))
}

// GetRuleByID 获取规则详情
// @Summary 获取规则详情
// @Description 根据 ID 获取规则的详细信息
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} dto.Response{data=response.EmailRuleResponse}
// @Router /api/v1/rules/{id} [get]
func (h *RuleHandler) GetRuleByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "规则 ID 格式无效")
		return
	}

	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, response.NewEmailRuleResponse(rule))
}

// ListRules 获取规则列表
// @Summary 获取规则列表
// @Description 获取指定账户的所有规则
// @Tags rules
// @Accept json
// @Produce json
// @Param account_uid query string false "账户 UID"
// @Success 200 {object} dto.Response{data=[]response.EmailRuleResponse}
// @Router /api/v1/rules [get]
func (h *RuleHandler) ListRules(c *gin.Context) {
	accountUID := c.Query("account_uid")

	rules, err := h.ruleService.ListByAccount(c.Request.Context(), accountUID)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, response.NewEmailRuleResponses(rules))
}

// UpdateRule 更新规则
// @Summary 更新规则
// @Description 更新规则信息
// @Tags 规则管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则 ID"
// @Param body body request.UpdateRuleRequest true "规则信息"
// @Success 200 {object} dto.Response{data=response.EmailRuleResponse}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /rules/{id} [put]
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "规则 ID 格式无效")
		return
	}

	var req request.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 获取现有规则
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 更新字段
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.MatchMode != nil {
		rule.MatchMode = *req.MatchMode
	}
	if req.Conditions != nil {
		rule.Conditions = req.Conditions
	}
	if req.Actions != nil {
		rule.Actions = req.Actions
	}
	if req.StopProcessing != nil {
		rule.StopProcessing = *req.StopProcessing
	}

	if err := h.ruleService.Update(c.Request.Context(), rule); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, response.NewEmailRuleResponse(rule))
}

// DeleteRule 删除规则
// @Summary 删除规则
// @Description 删除指定的规则
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/rules/{id} [delete]
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "规则 ID 格式无效")
		return
	}

	if err := h.ruleService.Delete(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "规则删除成功")
}

// ToggleRule 切换规则启用状态
// @Summary 切换规则启用状态
// @Description 启用或禁用规则
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} dto.Response{data=response.EmailRuleResponse}
// @Router /api/v1/rules/{id}/toggle [post]
func (h *RuleHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "规则 ID 格式无效")
		return
	}

	// 获取规则
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 切换状态
	rule.Enabled = !rule.Enabled
	if err := h.ruleService.Update(c.Request.Context(), rule); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, response.NewEmailRuleResponse(rule))
}

// TestRule 测试规则
// @Summary 测试规则是否匹配邮件
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param request body request.TestRuleRequest true "测试请求"
// @Success 200 {object} dto.Response{data=response.TestRuleResponse}
// @Router /api/v1/rules/{id}/test [post]
func (h *RuleHandler) TestRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "规则 ID 格式无效")
		return
	}

	var req request.TestRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 获取规则
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 构建测试邮件
	email := &model.Email{
		FromAddress:   req.FromAddress,
		ToAddress:     req.ToAddress,
		Subject:       req.Subject,
		TextBody:      req.Body,
		HasAttachment: req.HasAttachment,
	}

	// 测试规则
	matched, err := h.ruleService.TestRule(c.Request.Context(), rule, email)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 返回 RuleSummary 而不是 *model.EmailRule：后者带 gorm tag，
	// 会让 created_at / updated_at 等存储层字段进入 API 契约。
	dto.SuccessResponse(c, response.TestRuleResponse{
		Matched: matched,
		Rule:    response.NewRuleSummary(rule),
	})
}

// GetEmailRuleMatches 查询邮件的规则命中记录
// @Summary 查询邮件的规则命中记录
// @Description 返回某封邮件被哪些规则命中、执行了什么动作（时间升序）
// @Tags rules
// @Produce json
// @Param id path int true "邮件 ID"
// @Success 200 {object} dto.Response{data=response.RuleMatchLogListResponse}
// @Router /api/v1/emails/{id}/rule-matches [get]
func (h *RuleHandler) GetEmailRuleMatches(c *gin.Context) {
	emailID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || emailID <= 0 {
		dto.BadRequestResponse(c, "邮件 ID 无效")
		return
	}

	logs, err := h.ruleService.GetEmailRuleMatches(c.Request.Context(), emailID)
	if err != nil {
		dto.InternalServerErrorResponse(c, "查询规则命中记录失败")
		return
	}

	items := make([]response.RuleMatchLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, response.RuleMatchLogItem{
			ID:        l.ID,
			RuleID:    l.RuleID,
			RuleName:  l.RuleName,
			Actions:   l.Actions,
			MatchedAt: l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	dto.SuccessResponse(c, response.RuleMatchLogListResponse{Matches: items})
}
