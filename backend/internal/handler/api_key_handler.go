package handler

import (
	"strconv"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyHandler API Key 管理处理器
type APIKeyHandler struct {
	authService *service.AuthService
}

// NewAPIKeyHandler 创建 API Key 管理处理器
func NewAPIKeyHandler(authService *service.AuthService) *APIKeyHandler {
	return &APIKeyHandler{
		authService: authService,
	}
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name        string    `json:"name" binding:"required,max=255"`
	Description string    `json:"description"`
	RateLimit   int       `json:"rate_limit" binding:"min=1,max=10000"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
	Name      string `json:"name" binding:"required,max=255"`
	Description string `json:"description"`
	RateLimit int    `json:"rate_limit" binding:"min=1,max=10000"`
}

// CreateAPIKeyResponse 创建 API Key 响应
type CreateAPIKeyResponse struct {
	APIKey  string      `json:"api_key"`  // 明文 Key，仅此一次返回
	KeyInfo interface{} `json:"key_info"` // API Key 信息
}

// Create 创建 API Key
// POST /api/v1/api-keys
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 设置默认速率限制
	if req.RateLimit == 0 {
		req.RateLimit = 100
	}

	// 创建 API Key
	apiKey, keyInfo, err := h.authService.CreateAPIKey(
		c.Request.Context(),
		req.Name,
		req.Description,
		[]string{}, // 暂时不支持权限参数
		req.RateLimit,
		req.ExpiresAt,
	)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 返回明文 Key 和 Key 信息
	response := CreateAPIKeyResponse{
		APIKey:  apiKey,
		KeyInfo: keyInfo,
	}

	dto.SuccessWithMessage(c, response, "API Key 创建成功，请妥善保管，此密钥仅显示一次")
}

// List 列出所有 API Key
// GET /api/v1/api-keys
func (h *APIKeyHandler) List(c *gin.Context) {
	keys, err := h.authService.ListAPIKeys(c.Request.Context())
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, keys)
}

// GetByID 获取 API Key 详情
// GET /api/v1/api-keys/:id
func (h *APIKeyHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "API Key ID 格式无效")
		return
	}

	key, err := h.authService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, key)
}

// Update 更新 API Key
// PUT /api/v1/api-keys/:id
func (h *APIKeyHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "API Key ID 格式无效")
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 更新 API Key
	if err := h.authService.UpdateAPIKey(
		c.Request.Context(),
		id,
		req.Name,
		req.Description,
		req.RateLimit,
	); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 获取更新后的 API Key 信息
	key, err := h.authService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, key, "API Key 更新成功")
}

// Delete 删除 API Key
// DELETE /api/v1/api-keys/:id
func (h *APIKeyHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "API Key ID 格式无效")
		return
	}

	if err := h.authService.DeleteAPIKey(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "API Key 删除成功")
}

// Enable 启用 API Key
// POST /api/v1/api-keys/:id/enable
func (h *APIKeyHandler) Enable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "API Key ID 格式无效")
		return
	}

	if err := h.authService.EnableAPIKey(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 获取启用后的 API Key 信息
	key, err := h.authService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, key, "API Key 已启用")
}

// Disable 禁用 API Key
// POST /api/v1/api-keys/:id/disable
func (h *APIKeyHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "API Key ID 格式无效")
		return
	}

	if err := h.authService.DisableAPIKey(c.Request.Context(), id); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	// 获取禁用后的 API Key 信息
	key, err := h.authService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, key, "API Key 已禁用")
}

