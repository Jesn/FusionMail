package service

import (
	"context"
	"fmt"
	"time"
)

// GetSystemHealth 获取系统健康状态
func (s *SystemService) GetSystemHealth(ctx context.Context) (*SystemHealthResponse, error) {
	components := make(map[string]HealthCheck)
	overallStatus := "healthy"

	// 检查数据库连接
	dbCheck := s.checkDatabase(ctx)
	components["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// 检查 Redis 连接
	redisCheck := s.checkRedis(ctx)
	components["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// 检查存储
	storageCheck := s.checkStorage(ctx)
	components["storage"] = storageCheck
	if storageCheck.Status != "healthy" && overallStatus == "healthy" {
		overallStatus = "degraded"
	}

	return &SystemHealthResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    "1.0.0", // TODO: 从配置或构建信息获取
		Uptime:     int64(time.Since(s.startTime).Seconds()),
		Components: components,
	}, nil
}

// checkDatabase 检查数据库连接
func (s *SystemService) checkDatabase(ctx context.Context) HealthCheck {
	start := time.Now()

	sqlDB, err := s.db.DB()
	if err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("获取数据库连接失败: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("数据库连接失败: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "数据库连接正常",
		Latency: time.Since(start).Milliseconds(),
	}
}

// checkRedis 检查 Redis 连接
func (s *SystemService) checkRedis(ctx context.Context) HealthCheck {
	start := time.Now()

	if err := s.redis.Ping(ctx).Err(); err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Redis连接失败: %v", err),
			Latency: time.Since(start).Milliseconds(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Redis连接正常",
		Latency: time.Since(start).Milliseconds(),
	}
}

// checkStorage 检查存储
func (s *SystemService) checkStorage(ctx context.Context) HealthCheck {
	start := time.Now()

	// 这里可以添加存储健康检查逻辑
	// 例如检查存储目录是否可写，S3连接是否正常等

	return HealthCheck{
		Status:  "healthy",
		Message: "存储正常",
		Latency: time.Since(start).Milliseconds(),
	}
}
