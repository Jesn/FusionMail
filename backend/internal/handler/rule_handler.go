package handler

import (
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

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name           string `json:"name" binding:"required"`
	AccountUID     string `json:"account_uid" binding:"required"`
	Description    string `json:"description"`
	Conditions     string `json:"conditions" binding:"required"` // JSON 字符串
	Actions        string `json:"actions" binding:"required"`    // JSON 字符串
	Priority       int    `json:"priority"`
	StopProcessing bool   `json:"stop_processing"`
	Enabled        bool   `json:"enabled"`
}

// CreateRule 创建规则
// @Summary 创建规则
// @Description 创建新的邮件处理规则
// @Tags rules
// @Accept json
// @Produce json
// @Param body body CreateRuleRequest true "规则信息"
// @Success 201 {object} model.Rule
// @Router /api/v1/rules [post]
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	rule := &model.Rule{
		Name:           req.Name,
		AccountUID:     req.AccountUID,
		Description:    req.Description,
		Conditions:     req.Conditions,
		Actions:        req.Actions,
		Priority:       req.Priority,
		StopProcessing: req.StopProcessing,
		Enabled:        req.Enabled,
	}

	if err := h.ruleService.CreateRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRuleByID 获取规则详情
// @Summary 获取规则详情
// @Description 根据 ID 获取规则的详细信息
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} model.Rule
// @Router /api/v1/rules/{id} [get]
func (h *RuleHandler) GetRuleByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid rule id",
		})
		return
	}

	rule, err := h.ruleService.GetRuleByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "rule not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "rule not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// ListRules 获取规则列表
// @Summary 获取规则列表
// @Description 获取指定账户的所有规则
// @Tags rules
// @Accept json
// @Produce json
// @Param account_uid query string false "账户 UID"
// @Success 200 {array} model.Rule
// @Router /api/v1/rules [get]
func (h *RuleHandler) ListRules(c *gin.Context) {
	accountUID := c.Query("account_uid")

	rules, err := h.ruleService.ListRules(c.Request.Context(), accountUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// UpdateRule 更新规则
// @Summary 更新规则
// @Description 更新规则信息
// @Tags rules
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param body body CreateRuleRequest true "规则信息"
// @Success 200 {object} model.Rule
// @Router /api/v1/rules/{id} [put]
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid rule id",
		})
		return
	}

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	rule := &model.Rule{
		ID:             id,
		Name:           req.Name,
		AccountUID:     req.AccountUID,
		Description:    req.Description,
		Conditions:     req.Conditions,
		Actions:        req.Actions,
		Priority:       req.Priority,
		StopProcessing: req.StopProcessing,
		Enabled:        req.Enabled,
	}

	if err := h.ruleService.UpdateRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rule)
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid rule id",
		})
		return
	}

	if err := h.ruleService.DeleteRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "rule deleted",
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid rule id",
		})
		return
	}

	if err := h.ruleService.ToggleRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "rule status toggled",
	})
}

// ApplyRulesToAccount 对账户应用规则
// @Summary 对账户应用规则
// @Description 对指定账户的所有邮件应用规则
// @Tags rules
// @Accept json
// @Produce json
// @Param account_uid path string true "账户 UID"
// @Success 200 {object} map[string]string
// @Router /api/v1/rules/apply/{account_uid} [post]
func (h *RuleHandler) ApplyRulesToAccount(c *gin.Context) {
	accountUID := c.Param("account_uid")
	if accountUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "account_uid is required",
		})
		return
	}

	if err := h.ruleService.ApplyRulesToAccount(c.Request.Context(), accountUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "rules applied to account",
	})
}
