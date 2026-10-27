package handler

import (
	"fusionmail/internal/service"
)

// OAuth2Handler OAuth2 认证处理器
type OAuth2Handler struct {
	oauth2Service *service.OAuth2Service
}

// NewOAuth2Handler 创建 OAuth2 处理器实例
func NewOAuth2Handler(oauth2Service *service.OAuth2Service) *OAuth2Handler {
	return &OAuth2Handler{
		oauth2Service: oauth2Service,
	}
}

// GoogleAuthorize 生成 Google OAuth2 授权 URL
// @Summary 生成 Google OAuth2 授权 URL
// @Description 生成 Google OAuth2 授权 URL，用户需要访问此 URL 进行授权
// @Tags OAuth2
// @Accept json
// @Produce json
// @Param email query string false "邮箱地址（可选，用于预填充）"
// @Success 200 {object} dto.Response{data=service.OAuth2AuthResponse}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /auth/google/authorize [get]
