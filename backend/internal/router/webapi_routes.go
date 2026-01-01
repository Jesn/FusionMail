package router

import (
	"fusionmail/internal/handler"
	"fusionmail/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterWebAPIRoutes 注册 WebAPI Provider 相关路由
// 包括 Provider 管理和服务模板 API
func RegisterWebAPIRoutes(
	router *gin.Engine,
	providerHandler *handler.WebAPIProviderHandler,
	servicesHandler *handler.WebAPIServicesHandler,
	jwtSecret string,
) {
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())

	// ============================================
	// WebAPI Provider 管理接口
	// ============================================
	webapi := protected.Group("/webapi")
	{
		// Provider CRUD
		providers := webapi.Group("/providers")
		{
			// 注意：具体路径必须在参数路径之前注册，否则 /test 会被 /:uid 匹配
			providers.POST("", providerHandler.Create) // 创建 Provider
			providers.GET("", providerHandler.List)    // 获取列表

			// 连接测试（必须在 /:uid 之前注册）
			providers.POST("/test", providerHandler.TestConnection) // 测试连接（使用配置）

			// 带参数的路由
			providers.GET("/:uid", providerHandler.GetByUID)                                // 获取详情
			providers.PUT("/:uid", providerHandler.Update)                                  // 更新
			providers.DELETE("/:uid", providerHandler.Delete)                               // 删除
			providers.POST("/:uid/test", providerHandler.TestConnectionByUID)               // 测试已存在 Provider
			providers.POST("/:uid/sync", providerHandler.TriggerSync)                       // 手动触发同步
			providers.GET("/:uid/sync/status", providerHandler.GetSyncStatus)               // 获取同步状态
			providers.GET("/:uid/children", providerHandler.GetChildAccounts)               // 获取子邮箱列表
			providers.GET("/:uid/cloudmail-accounts", providerHandler.GetCloudMailAccounts) // 获取 Cloud Mail 服务端账户列表
		}

		// ============================================
		// WebAPI 账户配置接口（用于编辑 WebAPI 账户）
		// ============================================
		accounts := webapi.Group("/accounts")
		{
			accounts.GET("/:account_uid/config", providerHandler.GetAccountConfig)    // 获取账户配置
			accounts.PUT("/:account_uid/config", providerHandler.UpdateAccountConfig) // 更新账户配置
		}

		// ============================================
		// WebAPI 服务模板接口
		// ============================================
		services := webapi.Group("/services")
		{
			services.GET("", servicesHandler.ListServices)                   // 获取服务列表
			services.GET("/types", servicesHandler.GetSupportedTypes)        // 获取支持的服务类型
			services.GET("/:service_type", servicesHandler.GetServiceDetail) // 获取服务详情
			services.POST("/validate", servicesHandler.ValidateConfig)       // 验证配置
		}

		// ============================================
		// Cloudflare Temp Email 专用接口
		// ============================================
		cloudflare := webapi.Group("/cloudflare")
		{
			cloudflare.POST("/settings", providerHandler.FetchCloudflareTempEmailSettings) // 获取设置信息
		}
	}
}
