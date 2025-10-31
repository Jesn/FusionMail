package handler

import (
	"net/http"

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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	account, err := h.accountService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Account created successfully",
		"data":    account,
	})
}

// GetByUID 获取账户详情
// GET /api/v1/accounts/:uid
func (h *AccountHandler) GetByUID(c *gin.Context) {
	uid := c.Param("uid")

	account, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    account,
	})
}

// List 获取账户列表
// GET /api/v1/accounts
func (h *AccountHandler) List(c *gin.Context) {
	accounts, err := h.accountService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    accounts,
	})
}

// Update 更新账户
// PUT /api/v1/accounts/:uid
func (h *AccountHandler) Update(c *gin.Context) {
	uid := c.Param("uid")

	var req service.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	account, err := h.accountService.Update(c.Request.Context(), uid, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Account updated successfully",
		"data":    account,
	})
}

// Delete 删除账户
// DELETE /api/v1/accounts/:uid
func (h *AccountHandler) Delete(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.Delete(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Account deleted successfully",
	})
}

// TestConnection 测试账户连接
// POST /api/v1/accounts/:uid/test
func (h *AccountHandler) TestConnection(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.TestConnection(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection test successful",
	})
}

// SyncAccount 手动同步账户
// POST /api/v1/accounts/:uid/sync
func (h *AccountHandler) SyncAccount(c *gin.Context) {
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

	// 调用同步管理器进行同步
	// 注意：这里需要从依赖注入中获取 syncManager
	// 暂时返回提示信息，建议使用 /api/v1/sync/accounts/:uid 接口
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"error":   "请使用 /api/v1/sync/accounts/" + uid + " 接口进行同步",
		"message": "同步接口已迁移",
	})
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
