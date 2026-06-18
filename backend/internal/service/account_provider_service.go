package service

import (
	"context"
	"fmt"

	"fusionmail/internal/model"
)

// MatchProviderByEmail 根据邮箱地址自动匹配 Provider
// 解析邮箱域名并查找匹配的 Provider
func (s *accountService) MatchProviderByEmail(ctx context.Context, email string) (*model.Provider, error) {
	// 解析邮箱域名
	domain := extractEmailDomain(email)
	if domain == "" {
		return nil, fmt.Errorf("invalid email address: %s", email)
	}

	// 根据域名查找 Provider
	provider, err := s.providerRepo.FindByDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider by domain: %w", err)
	}

	return provider, nil
}

// extractEmailDomain 从邮箱地址中提取域名
func extractEmailDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return ""
}
