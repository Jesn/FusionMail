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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.accountService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": account,
	})
}

// List 获取账户列表
// GET /api/v1/accounts
func (h *AccountHandler) List(c *gin.Context) {
	accounts, err := h.accountService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": accounts,
	})
}

// Update 更新账户
// PUT /api/v1/accounts/:uid
func (h *AccountHandler) Update(c *gin.Context) {
	uid := c.Param("uid")

	var req service.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.accountService.Update(c.Request.Context(), uid, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account updated successfully",
		"data":    account,
	})
}

// Delete 删除账户
// DELETE /api/v1/accounts/:uid
func (h *AccountHandler) Delete(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.Delete(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
