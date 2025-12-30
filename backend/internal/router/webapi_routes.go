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
			providers.POST("", providerHandler.Create)        // 创建 Provider
			providers.GET("", providerHandler.List)           // 获取列表
			providers.GET("/:uid", providerHandler.GetByUID)  // 获取详情
			providers.PUT("/:uid", providerHandler.Update)    // 更新
			providers.DELETE("/:uid", providerHandler.Delete) // 删除

			// 连接测试
			providers.POST("/test", providerHandler.TestConnection)           // 测试连接（使用配置）
			providers.POST("/:uid/test", providerHandler.TestConnectionByUID) // 测试已存在 Provider

			// 同步操作
			providers.POST("/:uid/sync", providerHandler.TriggerSync)         // 手动触发同步
			providers.GET("/:uid/sync/status", providerHandler.GetSyncStatus) // 获取同步状态
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
	}
}
