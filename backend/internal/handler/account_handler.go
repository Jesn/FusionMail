package handler

import (
	"context"
	"net/http"
	"strings"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountHandler 账户管理处理器
type AccountHandler struct {
	accountService service.AccountService
}

// NewAccountHandler 创建账户管理处理器
func NewAccountHandler(accountService service.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// Create 创建账户
// POST /api/v1/accounts
func (h *AccountHandler) Create(c *gin.Context) {
	var req service.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.Create(c.Request.Context(), &req)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, account, "账户创建成功")
}

// GetByUID 获取账户详情
// GET /api/v1/accounts/:uid
func (h *AccountHandler) GetByUID(c *gin.Context) {
	uid := c.Param("uid")

	account, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, account)
}

// List 获取账户列表
// GET /api/v1/accounts
func (h *AccountHandler) List(c *gin.Context) {
	accounts, err := h.accountService.List(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, accounts)
}

// Update 更新账户
// PUT /api/v1/accounts/:uid
func (h *AccountHandler) Update(c *gin.Context) {
	uid := c.Param("uid")

	var req service.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.Update(c.Request.Context(), uid, &req)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, account, "账户更新成功")
}

// Delete 删除账户
// DELETE /api/v1/accounts/:uid
func (h *AccountHandler) Delete(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.Delete(c.Request.Context(), uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "账户删除成功")
}

// TestConnection 测试账户连接
// POST /api/v1/accounts/:uid/test
func (h *AccountHandler) TestConnection(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.TestConnection(c.Request.Context(), uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "连接测试成功")
}

// SyncAccount 手动同步账户
// POST /api/v1/accounts/:uid/sync
func (h *AccountHandler) SyncAccount(c *gin.Context) {
	uid := c.Param("uid")

	// 验证账户是否存在
	_, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 调用同步管理器进行同步
	// 注意：这里需要从依赖注入中获取 syncManager
	// 暂时返回提示信息，建议使用 /api/v1/sync/accounts/:uid 接口
	dto.BadRequestResponse(c, "请使用 /api/v1/sync/accounts/"+uid+" 接口进行同步")
}

// DisableAccount 禁用账户
// POST /api/v1/accounts/:uid/disable
func (h *AccountHandler) DisableAccount(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.DisableAccount(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "账户已禁用",
	})
}

// EnableAccount 启用账户
// POST /api/v1/accounts/:uid/enable
func (h *AccountHandler) EnableAccount(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.EnableAccount(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "账户已启用",
	})
}

// ClearSyncError 清除同步错误状态
// POST /api/v1/accounts/:uid/clear-error
func (h *AccountHandler) ClearSyncError(c *gin.Context) {
	uid := c.Param("uid")

	// 验证账户是否存在
	_, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "账户不存在",
		})
		return
	}

	// 清除同步错误状态
	if err := h.accountService.ClearSyncError(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "同步错误状态已清除",
	})
}

// BatchImportRequest 批量导入请求
type BatchImportRequest struct {
	Accounts []string `json:"accounts" binding:"required"`
}

// BatchImportResponse 批量导入响应
type BatchImportResponse struct {
	Success int                 `json:"success"`
	Failed  int                 `json:"failed"`
	Results []BatchImportResult `json:"results"`
}

// BatchImportResult 单个账户导入结果
type BatchImportResult struct {
	Email  string `json:"email"`
	Status string `json:"status"` // "success" 或 "failed"
	Error  string `json:"error,omitempty"`
}

// BatchImport 批量导入短效邮箱账户
// POST /api/v1/accounts/batch-import
func (h *AccountHandler) BatchImport(c *gin.Context) {
	var req BatchImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求格式错误: " + err.Error(),
		})
		return
	}

	if len(req.Accounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "账户列表不能为空",
		})
		return
	}

	// 限制批量导入数量
	if len(req.Accounts) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "单次最多导入 50 个账户",
		})
		return
	}

	response := BatchImportResponse{
		Success: 0,
		Failed:  0,
		Results: make([]BatchImportResult, 0, len(req.Accounts)),
	}

	// 逐个处理账户
	for _, accountString := range req.Accounts {
		result := h.importSingleAccount(c.Request.Context(), accountString)
		response.Results = append(response.Results, result)

		if result.Status == "success" {
			response.Success++
		} else {
			response.Failed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// importSingleAccount 导入单个账户
func (h *AccountHandler) importSingleAccount(ctx context.Context, accountString string) BatchImportResult {
	// 解析账户字符串
	config, err := adapter.ParseQuickAccountString(accountString)
	if err != nil {
		return BatchImportResult{
			Email:  extractEmailFromString(accountString),
			Status: "failed",
			Error:  "账户格式错误: " + err.Error(),
		}
	}

	// 创建账户请求
	createReq := &service.CreateAccountRequest{
		Email:        config.Email,
		Provider:     config.Provider,
		Protocol:     "graph_quick",
		AuthType:     "quick",
		RefreshToken: config.Credentials.RefreshToken,
		ClientID:     config.Credentials.ClientID,
		Password:     config.Credentials.Password,
		SyncEnabled:  true,
		SyncInterval: 5,
	}

	// 创建账户
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

// extractEmailFromString 从账户字符串中提取邮箱地址
func extractEmailFromString(accountString string) string {
	parts := strings.Split(accountString, "----")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return "unknown"
}
