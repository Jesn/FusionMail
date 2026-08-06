package handler

import (
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 配置管理处理器
type SettingHandler struct {
	settingService *service.SettingService
}

// NewSettingHandler 创建配置管理处理器
func NewSettingHandler(settingService *service.SettingService) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
	}
}

// GetSettingsByCategory 按分类获取配置
// GET /api/v1/settings/:category
func (h *SettingHandler) GetSettingsByCategory(c *gin.Context) {
	category := c.Param("category")

	// 获取用户ID（从认证中间件获取）
	userID := h.getUserID(c)

	// 解析查询参数
	opts := &service.GetOptions{
		IncludeSensitive: h.isAdmin(c), // 只有管理员才能获取敏感配置
		OnlyPublic:       false,
	}

	settings, err := h.settingService.GetByCategory(c.Request.Context(), userID, category, opts)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{
		"category": category,
		"settings": settings,
	})
}

// GetPublicSettings 获取公开配置
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	// 获取用户ID（可能有也可能没有）
	userID := h.getUserID(c)

	// 获取所有公开配置
	categories := service.CommonCategories
	allSettings := make(map[string]map[string]string)

	for _, category := range categories {
		settings, err := h.settingService.GetPublic(c.Request.Context(), userID, category)
		if err != nil {
			// 跳过获取失败的分类
			continue
		}
		allSettings[category] = settings
	}

	dto.SuccessResponse(c, gin.H{
		"settings": allSettings,
	})
}

// SetSettings 批量设置配置
// POST /api/v1/settings/:category
func (h *SettingHandler) SetSettings(c *gin.Context) {
	category := c.Param("category")

	// 获取用户ID
	userID := h.getUserID(c)

	// 绑定请求体
	var req struct {
		Settings map[string]string `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	if len(req.Settings) == 0 {
		dto.BadRequestResponse(c, "至少需要一个配置项")
		return
	}

	// 检查权限：
	// 1. 用户级配置（ui, sync, notification）：普通用户可以修改自己的配置
	// 2. 系统级配置（security, api, oauth, smtp）：只有管理员可以修改
	userCategories := []string{"ui", "sync", "notification"}
	isUserCategory := false
	for _, cat := range userCategories {
		if category == cat {
			isUserCategory = true
			break
		}
	}

	// 如果是系统级配置，必须是管理员
	if !isUserCategory && !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// 如果是用户级配置，必须有用户ID（已登录）
	if isUserCategory && userID == nil {
		dto.ForbiddenResponse(c, "请先登录")
		return
	}

	// 批量设置配置
	opts := &service.GetOptions{
		IncludeSensitive: h.isAdmin(c),
	}
	if err := h.settingService.BatchSet(c.Request.Context(), userID, category, req.Settings, opts); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "配置设置成功")
}

// SetSetting 设置单个配置项
// PUT /api/v1/settings/:category/:key
func (h *SettingHandler) SetSetting(c *gin.Context) {
	category := c.Param("category")
	key := c.Param("key")

	// 获取用户ID
	userID := h.getUserID(c)

	// 绑定请求体
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	if req.Value == "" {
		dto.BadRequestResponse(c, "配置值不能为空")
		return
	}

	// 检查权限：
	// 1. 用户级配置（ui, sync, notification）：普通用户可以修改自己的配置
	// 2. 系统级配置（security, api, oauth, smtp）：只有管理员可以修改
	userCategories := []string{"ui", "sync", "notification"}
	isUserCategory := false
	for _, cat := range userCategories {
		if category == cat {
			isUserCategory = true
			break
		}
	}

	// 如果是系统级配置，必须是管理员
	if !isUserCategory && !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// 如果是用户级配置，必须有用户ID（已登录）
	if isUserCategory && userID == nil {
		dto.ForbiddenResponse(c, "请先登录")
		return
	}

	// 设置配置
	opts := &service.GetOptions{
		IncludeSensitive: h.isAdmin(c),
	}
	if err := h.settingService.Set(c.Request.Context(), userID, category, key, req.Value, opts); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "配置设置成功")
}

// GetSetting 获取单个配置项
// GET /api/v1/settings/:category/:key
func (h *SettingHandler) GetSetting(c *gin.Context) {
	category := c.Param("category")
	key := c.Param("key")

	// 获取用户ID
	userID := h.getUserID(c)

	// 解析查询参数
	opts := &service.GetOptions{
		IncludeSensitive: h.isAdmin(c),
	}

	value, err := h.settingService.Get(c.Request.Context(), userID, category, key, opts)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{
		"category": category,
		"key":      key,
		"value":    value,
	})
}

// DeleteSetting 删除配置项
// DELETE /api/v1/settings/:category/:key
func (h *SettingHandler) DeleteSetting(c *gin.Context) {
	category := c.Param("category")
	key := c.Param("key")

	// 获取用户ID
	userID := h.getUserID(c)

	// 检查权限：只有管理员才能删除系统级配置
	if userID == nil && !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// 删除配置
	if err := h.settingService.Delete(c.Request.Context(), userID, category, key); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "配置删除成功")
}

// ResetSetting 重置配置为默认值
// POST /api/v1/settings/:category/:key/reset
func (h *SettingHandler) ResetSetting(c *gin.Context) {
	category := c.Param("category")
	key := c.Param("key")

	// 获取用户ID
	userID := h.getUserID(c)

	// 检查权限：
	// 1. 用户级配置（ui, sync, notification）：普通用户可以重置自己的配置
	// 2. 系统级配置（security, api, oauth, smtp）：只有管理员可以重置
	userCategories := []string{"ui", "sync", "notification"}
	isUserCategory := false
	for _, cat := range userCategories {
		if category == cat {
			isUserCategory = true
			break
		}
	}

	// 如果是系统级配置，必须是管理员
	if !isUserCategory && !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// 如果是用户级配置，必须有用户ID（已登录）
	if isUserCategory && userID == nil {
		dto.ForbiddenResponse(c, "请先登录")
		return
	}

	// 重置配置
	if err := h.settingService.Reset(c.Request.Context(), userID, category, key); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "配置重置成功")
}

// SearchSettings 搜索配置
// GET /api/v1/settings/search
func (h *SettingHandler) SearchSettings(c *gin.Context) {
	// 获取查询参数
	query := c.Query("q")
	if query == "" {
		dto.BadRequestResponse(c, "搜索关键词不能为空")
		return
	}

	category := c.Query("category")
	onlyPublic := c.Query("only_public") == "true"

	// 获取用户ID
	userID := h.getUserID(c)

	// 搜索配置
	settings, err := h.settingService.Search(c.Request.Context(), userID, query, category, onlyPublic)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{
		"query":    query,
		"category": category,
		"results":  settings,
	})
}

// GetSystem 获取系统级配置（仅管理员）
// GET /api/v1/settings/system/:category/:key
func (h *SettingHandler) GetSystem(c *gin.Context) {
	category := c.Param("category")
	key := c.Param("key")

	// 检查管理员权限
	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	value, err := h.settingService.GetSystem(c.Request.Context(), category, key)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{
		"category": category,
		"key":      key,
		"value":    value,
	})
}

// SetSystem 设置系统级配置（仅管理员）
// POST /api/v1/settings/system/:category/:key
func (h *SettingHandler) SetSystem(c *gin.Context) {
	category := c.Param("category")
	key := c.Param("key")

	// 检查管理员权限
	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// 绑定请求体
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	if req.Value == "" {
		dto.BadRequestResponse(c, "配置值不能为空")
		return
	}

	// 设置系统级配置
	if err := h.settingService.SetSystem(c.Request.Context(), category, key, req.Value); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "系统配置设置成功")
}

// GetSystemByCategory 批量获取系统级配置（仅管理员）
// GET /api/v1/settings/system/:category
func (h *SettingHandler) GetSystemByCategory(c *gin.Context) {
	category := c.Param("category")

	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	settings, err := h.settingService.GetSystemByCategory(c.Request.Context(), category)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, gin.H{
		"category": category,
		"settings": settings,
	})
}

// BatchSetSystem 批量设置系统级配置（仅管理员）
// POST /api/v1/settings/system/:category
func (h *SettingHandler) BatchSetSystem(c *gin.Context) {
	category := c.Param("category")

	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	var req struct {
		Settings map[string]string `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	if len(req.Settings) == 0 {
		dto.BadRequestResponse(c, "至少需要一个配置项")
		return
	}

	if err := h.settingService.BatchSetSystem(c.Request.Context(), category, req.Settings); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "系统配置设置成功")
}

// GetStats 获取缓存统计信息
// GET /api/v1/settings/stats
func (h *SettingHandler) GetStats(c *gin.Context) {
	// 检查管理员权限
	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	stats := h.settingService.GetStats()
	dto.SuccessResponse(c, stats)
}

// WarmUp 预热缓存
// POST /api/v1/settings/warmup
func (h *SettingHandler) WarmUp(c *gin.Context) {
	// 检查管理员权限
	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// 绑定请求体
	var req struct {
		Categories []string `json:"categories"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 获取用户ID（可选）
	userID := h.getUserID(c)

	// 预热缓存
	if err := h.settingService.WarmUp(c.Request.Context(), userID, req.Categories); err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessWithMessage(c, nil, "缓存预热成功")
}

// ExportSettings 导出配置
// GET /api/v1/settings/export
func (h *SettingHandler) ExportSettings(c *gin.Context) {
	// 检查管理员权限
	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// TODO: 实现导出功能
	// 这里需要实现从数据库读取配置并导出为JSON或ENV格式
	// 目前先返回未实现的响应

	dto.NotImplementedResponse(c, "导出功能尚未实现")
}

// ImportSettings 导入配置
// POST /api/v1/settings/import
func (h *SettingHandler) ImportSettings(c *gin.Context) {
	// 检查管理员权限
	if !h.isAdmin(c) {
		dto.ForbiddenResponse(c, "无权限执行此操作")
		return
	}

	// TODO: 实现导入功能
	// 这里需要实现从文件读取配置并写入数据库
	// 目前先返回未实现的响应

	dto.NotImplementedResponse(c, "导入功能尚未实现")
}

// getUserID 从上下文中获取用户ID
// 注意：当前系统中 JWT 的 sub 字段存储的是用户名（如 "admin"），而非数字ID
// 对于用户级配置，我们使用用户名作为标识符
func (h *SettingHandler) getUserID(c *gin.Context) *int64 {
	userIDValue, exists := c.Get("userID") // 修复：使用驼峰命名 "userID"
	if !exists {
		return nil
	}

	// 尝试将用户名转换为固定的用户ID
	// 对于 admin 用户，使用 ID 1
	username, ok := userIDValue.(string)
	if !ok {
		return nil
	}

	// 如果是 admin 用户，返回固定 ID 1
	if username == "admin" {
		id := int64(1)
		return &id
	}

	// 尝试解析为数字ID（兼容未来可能的数字ID）
	userID, err := strconv.ParseInt(username, 10, 64)
	if err != nil {
		// 如果不是数字，为其他用户名生成一个哈希ID
		// 这里简单返回 nil，表示非管理员用户
		return nil
	}

	return &userID
}

// isAdmin 检查是否为管理员
func (h *SettingHandler) isAdmin(c *gin.Context) bool {
	// 检查 role 字段
	role, exists := c.Get("role")
	if exists && role == "admin" {
		return true
	}

	// 检查 userID 是否为 "admin"
	userIDValue, exists := c.Get("userID") // 修复：使用驼峰命名 "userID"
	if !exists {
		return false
	}

	username, ok := userIDValue.(string)
	if !ok {
		return false
	}

	// 如果用户名是 "admin"，则认为是管理员
	return username == "admin"
}
