package handler

import (
	"context"
	"fmt"
	"strings"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountHandler 账户管理处理器
type AccountHandler struct {
	accountService service.AccountService
	oauth2Service  *service.OAuth2Service
	syncService    service.SyncService // 用于取消同步和获取进度
}

// NewAccountHandler 创建账户管理处理器
func NewAccountHandler(accountService service.AccountService, oauth2Service *service.OAuth2Service, syncService service.SyncService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		oauth2Service:  oauth2Service,
		syncService:    syncService,
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

// ListWithFilter 带筛选条件的账户列表
// GET /api/v1/accounts/filter
// @Summary 获取账户列表（支持分页和筛选）
// @Tags accounts
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param group_id query int false "分组ID：-1=所有，0=未分组，>0=具体分组"
// @Param email query string false "邮箱搜索"
// @Param provider query string false "提供商筛选"
// @Param status query string false "状态筛选"
// @Success 200 {object} service.AccountListResponse
// @Router /api/v1/accounts/filter [get]
func (h *AccountHandler) ListWithFilter(c *gin.Context) {
	filter := &service.AccountListFilter{
		Page:     1,
		PageSize: 10,
	}

	// 解析分页参数
	if page := c.Query("page"); page != "" {
		if p, err := parseInt(page); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := parseInt(pageSize); err == nil && ps > 0 {
			filter.PageSize = ps
		}
	}

	// 解析筛选参数
	if groupID := c.Query("group_id"); groupID != "" {
		if gid, err := parseInt64(groupID); err == nil {
			filter.GroupID = &gid
		}
	}
	filter.Email = strings.TrimSpace(c.Query("email"))
	filter.Provider = strings.TrimSpace(c.Query("provider"))
	filter.Status = strings.TrimSpace(c.Query("status"))

	result, err := h.accountService.ListWithFilter(c.Request.Context(), filter)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, result)
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// parseInt64 解析 int64
func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
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
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "账户已禁用")
}

// EnableAccount 启用账户
// POST /api/v1/accounts/:uid/enable
func (h *AccountHandler) EnableAccount(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.EnableAccount(c.Request.Context(), uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "账户已启用")
}

// ClearSyncError 清除同步错误状态
// POST /api/v1/accounts/:uid/clear-error
func (h *AccountHandler) ClearSyncError(c *gin.Context) {
	uid := c.Param("uid")

	// 验证账户是否存在
	_, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 清除同步错误状态
	if err := h.accountService.ClearSyncError(c.Request.Context(), uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "同步错误状态已清除")
}

// BatchImportRequest 批量导入请求
type BatchImportRequest struct {
	Accounts      []string `json:"accounts" binding:"required"`
	SyncEnabled   *bool    `json:"sync_enabled,omitempty"`
	SyncInterval  *int     `json:"sync_interval,omitempty"`
	GroupID       *int64   `json:"group_id,omitempty"`        // 分组 ID
	FirstSyncDays *int     `json:"first_sync_days,omitempty"` // 首次同步天数
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
		dto.BadRequestResponse(c, "请求格式错误: "+err.Error())
		return
	}

	if len(req.Accounts) == 0 {
		dto.BadRequestResponse(c, "账户列表不能为空")
		return
	}

	// 限制批量导入数量
	if len(req.Accounts) > 50 {
		dto.BadRequestResponse(c, "单次最多导入 50 个账户")
		return
	}

	response := BatchImportResponse{
		Success: 0,
		Failed:  0,
		Results: make([]BatchImportResult, 0, len(req.Accounts)),
	}

	// 逐个处理账户
	for _, accountString := range req.Accounts {
		result := h.importSingleAccount(c.Request.Context(), accountString, req.SyncEnabled, req.SyncInterval, req.GroupID, req.FirstSyncDays)
		response.Results = append(response.Results, result)

		if result.Status == "success" {
			response.Success++
		} else {
			response.Failed++
		}
	}

	dto.SuccessResponse(c, response)
}

// importSingleAccount 导入单个账户
func (h *AccountHandler) importSingleAccount(ctx context.Context, accountString string, syncEnabled *bool, syncInterval *int, groupID *int64, firstSyncDays *int) BatchImportResult {
	// 解析账户字符串
	config, err := adapter.ParseQuickAccountString(accountString)
	if err != nil {
		return BatchImportResult{
			Email:  extractEmailFromString(accountString),
			Status: "failed",
			Error:  "账户格式错误: " + err.Error(),
		}
	}

	// 验证 Outlook 账户的有效性
	if config.Provider == "outlook" {
		if config.Credentials.RefreshToken == "" {
			return BatchImportResult{
				Email:  config.Email,
				Status: "failed",
				Error:  "Outlook 账户缺少刷新令牌",
			}
		}

		// 验证 Microsoft 账户有效性
		err = h.oauth2Service.ValidateMicrosoftAccount(ctx, config.Credentials.RefreshToken, config.Credentials.ClientID)
		if err != nil {
			return BatchImportResult{
				Email:  config.Email,
				Status: "failed",
				Error:  "Outlook 账户验证失败: " + err.Error(),
			}
		}
	}

	// 确定同步设置
	syncEnabledVal := true
	syncIntervalVal := 2
	firstSyncDaysVal := 7 // 默认首次同步 7 天
	if syncEnabled != nil {
		syncEnabledVal = *syncEnabled
	}
	if syncInterval != nil {
		syncIntervalVal = *syncInterval
	}
	if firstSyncDays != nil {
		firstSyncDaysVal = *firstSyncDays
	}

	// 创建账户请求
	createReq := &service.CreateAccountRequest{
		Email:         config.Email,
		Provider:      config.Provider,
		Protocol:      "graph_quick",
		AuthType:      "quick",
		RefreshToken:  config.Credentials.RefreshToken,
		ClientID:      config.Credentials.ClientID,
		Password:      config.Credentials.Password,
		SyncEnabled:   syncEnabledVal,
		SyncInterval:  syncIntervalVal,
		FirstSyncDays: firstSyncDaysVal,
		GroupID:       groupID,
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

// ListDeleted 获取回收站中的账号（仅软删除的）
// GET /api/v1/accounts/trash
func (h *AccountHandler) ListDeleted(c *gin.Context) {
	accounts, err := h.accountService.ListDeleted(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, accounts)
}

// Restore 恢复软删除的账号
// POST /api/v1/accounts/:uid/restore
func (h *AccountHandler) Restore(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.Restore(c.Request.Context(), uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "账号已恢复")
}

// ForceDelete 永久删除账号
// DELETE /api/v1/accounts/:uid/force
func (h *AccountHandler) ForceDelete(c *gin.Context) {
	uid := c.Param("uid")

	if err := h.accountService.ForceDelete(c.Request.Context(), uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "账号已永久删除")
}

// CancelSync 取消账户同步
// POST /api/v1/accounts/:uid/sync/cancel
// Requirements: 5.1 - 支持同步取消
func (h *AccountHandler) CancelSync(c *gin.Context) {
	uid := c.Param("uid")

	// 验证账户是否存在
	_, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 检查是否有 syncService
	if h.syncService == nil {
		dto.BadRequestResponse(c, "同步服务未初始化")
		return
	}

	// 取消同步
	if err := h.syncService.CancelSync(uid); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "同步已取消")
}

// GetSyncProgress 获取账户同步进度
// GET /api/v1/accounts/:uid/sync/progress
func (h *AccountHandler) GetSyncProgress(c *gin.Context) {
	uid := c.Param("uid")

	// 验证账户是否存在
	_, err := h.accountService.GetByUID(c.Request.Context(), uid)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 检查是否有 syncService
	if h.syncService == nil {
		dto.BadRequestResponse(c, "同步服务未初始化")
		return
	}

	// 获取同步进度
	progress := h.syncService.GetSyncProgress(uid)
	if progress == nil {
		dto.SuccessResponse(c, map[string]interface{}{
			"status":  "idle",
			"message": "当前没有进行中的同步",
		})
		return
	}

	dto.SuccessResponse(c, progress)
}
