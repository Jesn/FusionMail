package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Task 任务结构
type Task struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	Priority   int                    `json:"priority"`
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	CreatedAt  time.Time              `json:"created_at"`
}

// RedisQueue Redis 任务队列
type RedisQueue struct {
	client      *redis.Client
	queueName   string
	lockPrefix  string
	lockTimeout time.Duration
}

// NewRedisQueue 创建 Redis 队列实例
func NewRedisQueue(client *redis.Client, queueName string) *RedisQueue {
	return &RedisQueue{
		client:      client,
		queueName:   queueName,
		lockPrefix:  "lock:" + queueName + ":",
		lockTimeout: 5 * time.Minute, // 默认锁超时 5 分钟
	}
}

// Enqueue 将任务加入队列
func (q *RedisQueue) Enqueue(ctx context.Context, task *Task) error {
	if task.ID == "" {
		task.ID = fmt.Sprintf("%s_%d", task.Type, time.Now().UnixNano())
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	if task.MaxRetries == 0 {
		task.MaxRetries = 3 // 默认最大重试 3 次
	}

	// 序列化任务
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 根据优先级选择队列
	queueKey := q.queueName
	if task.Priority > 0 {
		queueKey = fmt.Sprintf("%s:priority:%d", q.queueName, task.Priority)
	}

	// 加入队列（右侧推入）
	if err := q.client.RPush(ctx, queueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	return nil
}

// Dequeue 从队列中取出任务
func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (*Task, error) {
	// 尝试从高优先级队列获取
	for priority := 10; priority >= 0; priority-- {
		queueKey := q.queueName
		if priority > 0 {
			queueKey = fmt.Sprintf("%s:priority:%d", q.queueName, priority)
		}

		// 从左侧弹出（阻塞）
		result, err := q.client.BLPop(ctx, timeout, queueKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue // 队列为空，尝试下一个优先级
			}
			return nil, fmt.Errorf("failed to dequeue task: %w", err)
		}

		if len(result) < 2 {
			continue
		}

		// 反序列化任务
		var task Task
		if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
			return nil, fmt.Errorf("failed to unmarshal task: %w", err)
		}

		return &task, nil
	}

	// 所有队列都为空
	return nil, redis.Nil
}

// AcquireLock 获取分布式锁
func (q *RedisQueue) AcquireLock(ctx context.Context, key string) (bool, error) {
	lockKey := q.lockPrefix + key

	// 使用 SET NX EX 命令获取锁
	result, err := q.client.SetNX(ctx, lockKey, time.Now().Unix(), q.lockTimeout).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return result, nil
}

// ReleaseLock 释放分布式锁
func (q *RedisQueue) ReleaseLock(ctx context.Context, key string) error {
	lockKey := q.lockPrefix + key
	return q.client.Del(ctx, lockKey).Err()
}

// ExtendLock 延长锁的过期时间
func (q *RedisQueue) ExtendLock(ctx context.Context, key string) error {
	lockKey := q.lockPrefix + key
	return q.client.Expire(ctx, lockKey, q.lockTimeout).Err()
}

// Size 获取队列大小
func (q *RedisQueue) Size(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueName).Result()
}

// Clear 清空队列
func (q *RedisQueue) Clear(ctx context.Context) error {
	return q.client.Del(ctx, q.queueName).Err()
}

// Peek 查看队列头部的任务（不移除）
func (q *RedisQueue) Peek(ctx context.Context) (*Task, error) {
	result, err := q.client.LIndex(ctx, q.queueName, 0).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to peek task: %w", err)
	}

	var task Task
	if err := json.Unmarshal([]byte(result), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}
