package handler

import (
	"context"
	"log"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// PublicBatchImportRequest 公共 API 批量导入请求（不支持同步配置，统一使用默认值）
type PublicBatchImportRequest struct {
	Accounts []string            `json:"accounts" binding:"required"`
	Format   *ImportFormatConfig `json:"format,omitempty"`
	GroupID  *int64              `json:"group_id,omitempty"` // 分组 ID
}

// BatchImportAccounts 通过 API Key 批量导入邮箱账户
// @Summary      批量导入邮箱账户
// @Description  通过 API Key 批量导入 Outlook 邮箱账户，支持自定义分隔符和字段顺序。同步配置使用固定默认值（启用同步、间隔 2 分钟、首次同步 7 天）
// @Tags         public
// @Accept       json
// @Produce      json
// @Param        body body PublicBatchImportRequest true "批量导入请求"
// @Security     ApiKeyAuth
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} dto.Response
// @Failure      401 {object} dto.Response
// @Router       /api/v1/public/mail/import-accounts [post]
func (h *PublicHandler) BatchImportAccounts(c *gin.Context) {
	var req PublicBatchImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求格式错误: "+err.Error())
		return
	}

	if len(req.Accounts) == 0 {
		dto.BadRequestResponse(c, "账户列表不能为空")
		return
	}

	if len(req.Accounts) > 50 {
		dto.BadRequestResponse(c, "单次最多导入 50 个账户")
		return
	}

	response := BatchImportResponse{
		Success: 0,
		Failed:  0,
		Results: make([]BatchImportResult, 0, len(req.Accounts)),
	}

	for _, accountString := range req.Accounts {
		result := h.importSingleAccountPublic(c.Request.Context(), accountString, req.Format, req.GroupID)
		response.Results = append(response.Results, result)
		if result.Status == "success" {
			response.Success++
		} else {
			response.Failed++
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    response,
	})
}

// importSingleAccountPublic 公共 API 版本的单个账户导入逻辑（同步配置使用固定默认值）
func (h *PublicHandler) importSingleAccountPublic(ctx context.Context, accountString string, format *ImportFormatConfig, groupID *int64) BatchImportResult {
	var config *adapter.Config
	var err error
	delimiter := adapter.QuickAccountSeparator
	if format != nil && format.Delimiter != "" {
		delimiter = format.Delimiter
	}

	if format != nil && len(format.Fields) > 0 {
		config, err = adapter.ParseAccountStringWithFormat(accountString, format.Delimiter, format.Fields)
	} else {
		config, err = adapter.ParseQuickAccountString(accountString)
	}
	if err != nil {
		return BatchImportResult{
			Email:  extractEmailFromString(accountString, delimiter),
			Status: "failed",
			Error:  "账户格式错误: " + err.Error(),
		}
	}

	if config.Provider == "outlook" {
		if config.Credentials.RefreshToken == "" {
			return BatchImportResult{
				Email:  config.Email,
				Status: "failed",
				Error:  "Outlook 账户缺少刷新令牌",
			}
		}

		err = h.oauth2Service.ValidateMicrosoftAccount(ctx, config.Credentials.RefreshToken, config.Credentials.ClientID)
		if err != nil {
			return BatchImportResult{
				Email:  config.Email,
				Status: "failed",
				Error:  "Outlook 账户验证失败: " + err.Error(),
			}
		}
	}

	syncEnabledVal := true
	syncIntervalVal := 2
	firstSyncDaysVal := 7

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

// ---------------------------------------------------------------------------
// 检查邮箱账户是否存在
// ---------------------------------------------------------------------------

// CheckAccountRequest 检查邮箱账户请求
type CheckAccountRequest struct {
	Email string `form:"email" binding:"required,email"` // 邮箱地址
}

// AccountBasicInfo 邮箱账户基础信息（不含凭证等敏感字段）
type AccountBasicInfo struct {
	UID        string     `json:"uid"`
	Email      string     `json:"email"`
	Status     string     `json:"status"`
	LastSyncAt *time.Time `json:"last_sync_at,omitempty"` // 最近同步时间
	GroupID    *int64     `json:"group_id,omitempty"`     // 分组 ID
	GroupName  string     `json:"group_name,omitempty"`   // 分组名称
	CreatedAt  time.Time  `json:"created_at"`
}

// CheckAccountResponse 检查邮箱账户响应（账户不存在时 account 为 null）
type CheckAccountResponse struct {
	Account *AccountBasicInfo `json:"account"` // 存在时返回基础信息，不存在时为 null
}

// CheckAccount 检查邮箱账户是否存在
// @Summary      检查邮箱账户是否存在
// @Description  通过 API Key 检查指定邮箱是否已存在于系统中，存在时仅返回账户基础信息（不含凭证等敏感字段）
// @Tags         public
// @Accept       json
// @Produce      json
// @Param        email query string true "邮箱地址"
// @Security     ApiKeyAuth
// @Success      200 {object} CheckAccountResponse
// @Failure      400 {object} dto.Response
// @Failure      401 {object} dto.Response
// @Router       /api/v1/public/mail/check-account [get]
func (h *PublicHandler) CheckAccount(c *gin.Context) {
	var req CheckAccountRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	account, err := h.accountService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("[ERROR] Failed to get account by email: %s, error: %v", req.Email, err)
		dto.HandleServiceError(c, err)
		return
	}

	resp := CheckAccountResponse{}
	if account != nil {
		info := &AccountBasicInfo{
			UID:        account.UID,
			Email:      account.Email,
			Status:     account.Status,
			LastSyncAt: account.LastSyncAt,
			GroupID:    account.GroupID,
			CreatedAt:  account.CreatedAt,
		}

		// 查询分组名称
		if account.GroupID != nil {
			group, gerr := h.groupService.GetGroupByID(c.Request.Context(), *account.GroupID)
			if gerr != nil {
				log.Printf("[WARN] Failed to get group %d for account %s: %v", *account.GroupID, account.UID, gerr)
			} else if group != nil {
				info.GroupName = group.Name
			}
		}
		resp.Account = info
	}

	dto.SuccessResponse(c, resp)
}

// ListAccountsRequest 账户列表请求（公开 API）
type ListAccountsRequest struct {
	GroupID  *int64 `form:"group_id"`  // 分组 ID：-1=所有，0=未分组，>0=具体分组
	Page     int    `form:"page"`      // 页码，从 1 开始
	PageSize int    `form:"page_size"` // 每页数量，最大 100
}

// ListAccounts 按分组列出邮箱账户（公开 API，API Key 鉴权）
// @Summary      按分组列出邮箱账户
// @Description  通过 API Key 列出邮箱账户，支持按分组筛选。仅返回账户基础信息（不含凭证等敏感字段）
// @Tags         public
// @Accept       json
// @Produce      json
// @Param        group_id query int false "分组 ID：-1=所有，0=未分组，>0=具体分组"
// @Param        page query int false "页码，从 1 开始"
// @Param        page_size query int false "每页数量，最大 100"
// @Security     ApiKeyAuth
// @Success      200 {object} service.AccountListResponse
// @Failure      400 {object} dto.Response
// @Failure      401 {object} dto.Response
// @Router       /api/v1/public/mail/accounts [get]
func (h *PublicHandler) ListAccounts(c *gin.Context) {
	var req ListAccountsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	filter := &service.AccountListFilter{Page: 1, PageSize: 100}
	if req.Page > 0 {
		filter.Page = req.Page
	}
	if req.PageSize > 0 {
		filter.PageSize = req.PageSize
	}
	if req.GroupID != nil {
		filter.GroupID = req.GroupID
	}

	resp, err := h.accountService.ListWithFilter(c.Request.Context(), filter)
	if err != nil {
		log.Printf("[ERROR] Failed to list accounts (group_id=%v): %v", req.GroupID, err)
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, resp)
}
