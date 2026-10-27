package service

import (
	"fmt"
	"testing"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
)

// ============ isAuthError 三层归因 ============
//
// 守卫要点（AI 分析文档 §1.9）：
//   - 裸 "401" / "auth" 子串不得再触发认证分类（历史误判源）
//   - 结构化错误（WebAPIError / TokenError）必须穿透 %w 包装链被识别

func TestIsAuthError_StructuredWebAPIError(t *testing.T) {
	policy := &SyncErrorPolicy{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "WebAPI 401 包装后仍识别（穿透 %w）",
			err:  fmt.Errorf("failed to connect: %w", webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败", 401, false, nil)),
			want: true,
		},
		{
			name: "WebAPI 403 同样视为认证失败",
			err:  webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败", 403, false, nil),
			want: true,
		},
		{
			name: "WebAPI 连接失败不是认证错误（不累计禁用）",
			err:  webapi.NewWebAPIError(webapi.ErrCodeConnectionFailed, "连接失败", 0, true, nil),
			want: false,
		},
		{
			name: "WebAPI 服务器错误不是认证错误",
			err:  webapi.NewWebAPIError(webapi.ErrCodeServerError, "服务器错误", 500, true, nil),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.isAuthError(tt.err); got != tt.want {
				t.Fatalf("isAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthError_TokenError(t *testing.T) {
	policy := &SyncErrorPolicy{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "invalid_grant 视为认证失效",
			err:  &adapter.TokenError{Code: "invalid_grant", Description: "token expired", StatusCode: 400},
			want: true,
		},
		{
			name: "HTTP 401 的 TokenError 视为认证失效",
			err:  &adapter.TokenError{Code: "unknown_code", Description: "denied", StatusCode: 401},
			want: true,
		},
		{
			name: "unsupported_grant_type 是配置问题，不禁用账户",
			err:  &adapter.TokenError{Code: "unsupported_grant_type", Description: "grant type not supported", StatusCode: 400},
			want: false,
		},
		{
			name: "包装后的 TokenError 仍识别",
			err:  fmt.Errorf("refresh failed: %w", &adapter.TokenError{Code: "invalid_grant", StatusCode: 400}),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.isAuthError(tt.err); got != tt.want {
				t.Fatalf("isAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthError_StringFallback_Narrowed(t *testing.T) {
	policy := &SyncErrorPolicy{}

	tests := []struct {
		name  string
		errMsg string
		want  bool
	}{
		// 历史误判源：裸 "401" / "auth" 已移除
		{"裸 401 不再误判（如字节数/行号）", "read 1401 bytes from conn", false},
		{"auth 子串不再误判（如 author）", "invalid author header", false},
		{"日志行号含 401 不再误判", "parse error at line 401", false},
		// 完整短语仍识别
		{"unauthorized 识别", "server returned unauthorized", true},
		{"token expired 识别", "token expired while refreshing", true},
		{"authentication failed 识别", "imap: authentication failed", true},
		{"invalid credentials 识别", "invalid credentials provided", true},
		{"access denied 识别", "access denied by server", true},
		// 普通网络错误不误判
		{"连接超时不是认证错误", "dial tcp: connection refused", false},
		{"EOF 不是认证错误", "unexpected EOF", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("%s", tt.errMsg)
			if got := policy.isAuthError(err); got != tt.want {
				t.Fatalf("isAuthError(%q) = %v, want %v", tt.errMsg, got, tt.want)
			}
		})
	}
}

func TestIsAuthError_NilAndWrapping(t *testing.T) {
	policy := &SyncErrorPolicy{}
	if policy.isAuthError(nil) {
		t.Fatal("nil 错误不应判为认证错误")
	}
	// 多层 %w 包装
	inner := webapi.NewWebAPIError(webapi.ErrCodeAuthFailed, "认证失败", 401, false, nil)
	wrapped := fmt.Errorf("sync failed: %w", fmt.Errorf("fetch: %w", inner))
	if !policy.isAuthError(wrapped) {
		t.Fatal("多层包装的认证错误应被识别")
	}
}
