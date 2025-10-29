package handler

import (
	"fusionmail/internal/dto/request"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/service"
	"net/http"
	"strconv"

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
// @Success 201 {object} model.EmailRule
// @Router /api/v1/rules [post]
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req request.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
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
		response.Error(c, http.StatusInternalServerError, "failed to create rule: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// GetRuleByID 获取规则详情
// @Summary 获取规则详情
// @Description 根据 ID 获取规则的详细信息
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} model.EmailRule
// @Router /api/v1/rules/{id} [get]
func (h *RuleHandler) GetRuleByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
		return
	}

	response.Success(c, rule)
}

// ListRules 获取规则列表
// @Summary 获取规则列表
// @Description 获取指定账户的所有规则
// @Tags rules
// @Accept json
// @Produce json
// @Param account_uid query string false "账户 UID"
// @Success 200 {array} model.EmailRule
// @Router /api/v1/rules [get]
func (h *RuleHandler) ListRules(c *gin.Context) {
	accountUID := c.Query("account_uid")

	rules, err := h.ruleService.ListByAccount(c.Request.Context(), accountUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list rules: "+err.Error())
		return
	}

	response.Success(c, rules)
}

// UpdateRule 更新规则
// @Summary 更新规则
// @Description 更新规则信息
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param body body request.UpdateRuleRequest true "规则信息"
// @Success 200 {object} model.EmailRule
// @Router /api/v1/rules/{id} [put]
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req request.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// 获取现有规则
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
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
		response.Error(c, http.StatusInternalServerError, "failed to update rule: "+err.Error())
		return
	}

	response.Success(c, rule)
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
		response.Error(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	if err := h.ruleService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete rule: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ToggleRule 切换规则启用状态
// @Summary 切换规则启用状态
// @Description 启用或禁用规则
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/rules/{id}/toggle [post]
func (h *RuleHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	// 获取规则
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
		return
	}

	// 切换状态
	rule.Enabled = !rule.Enabled
	if err := h.ruleService.Update(c.Request.Context(), rule); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to toggle rule: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// TestRule 测试规则
// @Summary 测试规则是否匹配邮件
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param request body request.TestRuleRequest true "测试请求"
// @Success 200 {object} response.TestRuleResponse
// @Router /api/v1/rules/{id}/test [post]
func (h *RuleHandler) TestRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req request.TestRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// 获取规则
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
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
		response.Error(c, http.StatusInternalServerError, "failed to test rule: "+err.Error())
		return
	}

	response.Success(c, response.TestRuleResponse{
		Matched: matched,
		Rule:    rule,
	})
}
