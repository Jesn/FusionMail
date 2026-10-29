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
