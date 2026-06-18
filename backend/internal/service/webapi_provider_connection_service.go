package service

import (
	"context"
	"fmt"
)

// TestConnectionResult 连接测试结果
type TestConnectionResult struct {
	Success     bool   `json:"success"`               // 是否成功
	Message     string `json:"message"`               // 消息
	ServiceName string `json:"service_name"`          // 服务名称
	EmailCount  int    `json:"email_count,omitempty"` // 邮件数量（如果成功）
	Error       string `json:"error,omitempty"`       // 错误信息
}

// TestConnection 测试连接（使用配置）
func (s *WebAPIProviderService) TestConnection(ctx context.Context, serviceType, authDataJSON string) (*TestConnectionResult, error) {
	// 1. 验证服务类型
	if !s.factory.IsServiceTypeSupported(serviceType) {
		return &TestConnectionResult{
			Success: false,
			Message: "不支持的服务类型",
			Error:   fmt.Sprintf("服务类型 %s 不支持", serviceType),
		}, nil
	}

	// 2. 验证配置
	if err := s.factory.ValidateConfig(serviceType, authDataJSON); err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "配置验证失败",
			Error:   err.Error(),
		}, nil
	}

	// 3. 创建适配器
	adapter, err := s.factory.CreateAdapter(serviceType, authDataJSON)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "创建适配器失败",
			Error:   err.Error(),
		}, nil
	}

	// 4. 测试连接（直接调用适配器的 TestConnection 方法）
	if err := adapter.TestConnection(ctx); err != nil {
		return &TestConnectionResult{
			Success:     false,
			Message:     "连接测试失败",
			ServiceName: adapter.GetServiceName(),
			Error:       err.Error(),
		}, nil
	}

	return &TestConnectionResult{
		Success:     true,
		Message:     "连接测试成功",
		ServiceName: adapter.GetServiceName(),
	}, nil
}

// TestConnectionByUID 测试已存在 Provider 的连接
func (s *WebAPIProviderService) TestConnectionByUID(ctx context.Context, uid string) (*TestConnectionResult, error) {
	// 1. 获取账户
	account, err := s.GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// 2. 解密认证数据
	authDataBytes, err := s.cryptoSvc.Decrypt(account.EncryptedCredentials)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "解密认证数据失败",
			Error:   err.Error(),
		}, nil
	}

	// 3. 获取服务类型
	serviceType, err := s.getServiceTypeFromAccount(ctx, account)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "获取服务类型失败",
			Error:   err.Error(),
		}, nil
	}

	// 4. 测试连接
	return s.TestConnection(ctx, serviceType, string(authDataBytes))
}
