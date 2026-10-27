package adapter

// GraphQuickAdapter 连接测试与凭据验证：
// TestConnection / TestConnectionWithDetails / ValidateCredentials、
// Graph API 错误解析与用户友好错误消息。
// 自 graph_quick.go 同包机械拆分，逻辑保持不变。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TestConnection 测试连接
// TestConnection 测试连接
func (a *GraphQuickAdapter) TestConnection(ctx context.Context) error {
	a.logger.Info("开始测试连接", "email", a.config.Email)

	// 创建带超时的上下文
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 执行连接测试
	result, err := a.performConnectionTest(testCtx)
	if err != nil {
		a.logger.Error("连接测试失败", "email", a.config.Email, "error", err)
		return err
	}

	a.logger.Info("连接测试成功",
		"email", a.config.Email,
		"user_id", result.UserID,
		"display_name", result.DisplayName,
		"response_time_ms", result.ResponseTimeMs)

	return nil
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	UserID         string `json:"id"`
	DisplayName    string `json:"displayName"`
	Mail           string `json:"mail"`
	UserPrincipal  string `json:"userPrincipalName"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// performConnectionTest 执行实际的连接测试
func (a *GraphQuickAdapter) performConnectionTest(ctx context.Context) (*ConnectionTestResult, error) {
	startTime := time.Now()

	// 确保有有效的 access token
	if err := a.ensureValidToken(ctx); err != nil {
		return nil, &ConnectionTestError{
			Type:    "authentication",
			Message: "无法获取有效的访问令牌",
			Details: err.Error(),
		}
	}

	// 测试获取用户信息
	requestURL := fmt.Sprintf("%s/me", a.baseURL)

	// 使用新的认证请求方法
	resp, err := a.makeAuthenticatedRequest(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, &ConnectionTestError{
			Type:    "network",
			Message: "网络请求失败",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	// 计算响应时间
	responseTime := time.Since(startTime).Milliseconds()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, a.parseGraphAPIError(resp.StatusCode, body)
	}

	// 解析用户信息
	var userInfo ConnectionTestResult
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, &ConnectionTestError{
			Type:    "parsing",
			Message: "无法解析用户信息响应",
			Details: err.Error(),
		}
	}

	userInfo.ResponseTimeMs = responseTime

	// 验证关键字段
	if userInfo.UserID == "" {
		return nil, &ConnectionTestError{
			Type:    "validation",
			Message: "用户信息不完整",
			Details: "用户 ID 为空",
		}
	}

	return &userInfo, nil
}

// ConnectionTestError 连接测试错误
type ConnectionTestError struct {
	Type    string `json:"type"`             // authentication, network, parsing, validation, api
	Message string `json:"message"`          // 用户友好的错误消息
	Details string `json:"details"`          // 详细的技术错误信息
	Code    string `json:"code,omitempty"`   // API 错误码
	Status  int    `json:"status,omitempty"` // HTTP 状态码
}

func (e *ConnectionTestError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s (code: %s): %s", e.Type, e.Message, e.Code, e.Details)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Message, e.Details)
}

// parseGraphAPIError 解析 Graph API 错误响应
func (a *GraphQuickAdapter) parseGraphAPIError(statusCode int, body []byte) *ConnectionTestError {
	// 尝试解析 Graph API 错误格式
	var graphError struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if json.Unmarshal(body, &graphError) == nil && graphError.Error.Code != "" {
		return &ConnectionTestError{
			Type:    "api",
			Message: a.getErrorMessage(graphError.Error.Code, statusCode),
			Details: graphError.Error.Message,
			Code:    graphError.Error.Code,
			Status:  statusCode,
		}
	}

	// 如果无法解析，返回通用错误
	return &ConnectionTestError{
		Type:    "api",
		Message: a.getErrorMessage("", statusCode),
		Details: string(body),
		Status:  statusCode,
	}
}

// getErrorMessage 根据错误码和状态码返回用户友好的错误消息
func (a *GraphQuickAdapter) getErrorMessage(errorCode string, statusCode int) string {
	switch errorCode {
	case "InvalidAuthenticationToken":
		return "访问令牌无效或已过期"
	case "Forbidden":
		return "没有权限访问此资源"
	case "ThrottledRequest":
		return "请求过于频繁，请稍后重试"
	case "ServiceNotAvailable":
		return "Microsoft Graph 服务暂时不可用"
	case "TooManyRequests":
		return "请求次数超过限制"
	default:
		switch statusCode {
		case 401:
			return "身份验证失败"
		case 403:
			return "访问被拒绝"
		case 404:
			return "资源不存在"
		case 429:
			return "请求过于频繁"
		case 500:
			return "服务器内部错误"
		case 502:
			return "网关错误"
		case 503:
			return "服务不可用"
		case 504:
			return "网关超时"
		default:
			return fmt.Sprintf("请求失败 (HTTP %d)", statusCode)
		}
	}
}

// TestConnectionWithDetails 测试连接并返回详细信息
func (a *GraphQuickAdapter) TestConnectionWithDetails(ctx context.Context) (*ConnectionTestResult, error) {
	a.logger.Info("开始详细连接测试", "email", a.config.Email)

	// 创建带超时的上下文
	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 执行连接测试
	result, err := a.performConnectionTest(testCtx)
	if err != nil {
		a.logger.Error("详细连接测试失败", "email", a.config.Email, "error", err)
		return nil, err
	}

	a.logger.Info("详细连接测试成功",
		"email", a.config.Email,
		"user_id", result.UserID,
		"display_name", result.DisplayName,
		"mail", result.Mail,
		"response_time_ms", result.ResponseTimeMs)

	return result, nil
}

// ValidateCredentials 验证凭据有效性
func (a *GraphQuickAdapter) ValidateCredentials(ctx context.Context) error {
	a.logger.Info("开始验证凭据", "email", a.config.Email)

	// 检查必要的凭据字段
	if a.config.Credentials == nil {
		return &ConnectionTestError{
			Type:    "validation",
			Message: "凭据配置为空",
			Details: "Credentials 字段不能为 nil",
		}
	}

	if a.config.Credentials.ClientID == "" {
		return &ConnectionTestError{
			Type:    "validation",
			Message: "客户端 ID 不能为空",
			Details: "ClientID 字段是必需的",
		}
	}

	if a.config.Credentials.RefreshToken == "" {
		return &ConnectionTestError{
			Type:    "validation",
			Message: "刷新令牌不能为空",
			Details: "RefreshToken 字段是必需的",
		}
	}

	// 尝试刷新 token 来验证凭据
	if err := a.refreshAccessToken(ctx); err != nil {
		var tokenErr *TokenError
		if errors.As(err, &tokenErr) {
			return &ConnectionTestError{
				Type:    "authentication",
				Message: "凭据验证失败",
				Details: tokenErr.Description,
				Code:    tokenErr.Code,
				Status:  tokenErr.StatusCode,
			}
		}
		return &ConnectionTestError{
			Type:    "authentication",
			Message: "凭据验证失败",
			Details: err.Error(),
		}
	}

	a.logger.Info("凭据验证成功", "email", a.config.Email)
	return nil
}
