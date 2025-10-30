package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"fusionmail/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// EventType 事件类型
type EventType string

const (
	// 邮件事件
	EventEmailReceived EventType = "email.received"
	EventEmailRead     EventType = "email.read"
	EventEmailUnread   EventType = "email.unread"
	EventEmailStarred  EventType = "email.starred"
	EventEmailArchived EventType = "email.archived"
	EventEmailDeleted  EventType = "email.deleted"

	// 账户事件
	EventAccountAdded   EventType = "account.added"
	EventAccountUpdated EventType = "account.updated"
	EventAccountDeleted EventType = "account.deleted"
	EventAccountSynced  EventType = "account.synced"

	// 同步事件
	EventSyncStarted   EventType = "sync.started"
	EventSyncCompleted EventType = "sync.completed"
	EventSyncFailed    EventType = "sync.failed"
)

// Event 事件结构
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Source    string                 `json:"source"`
}

// Handler 事件处理器
type Handler func(ctx context.Context, event *Event) error

// Bus 事件总线接口
type Bus interface {
	// 发布事件
	Publish(ctx context.Context, event *Event) error

	// 订阅事件
	Subscribe(ctx context.Context, eventType EventType, handler Handler) error

	// 取消订阅
	Unsubscribe(ctx context.Context, eventType EventType) error

	// 启动事件总线
	Start(ctx context.Context) error

	// 停止事件总线
	Stop() error
}

// redisBus Redis 事件总线实现
type redisBus struct {
	client   *redis.Client
	logger   *logger.Logger
	handlers map[EventType][]Handler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	started  bool
}

// NewRedisBus 创建 Redis 事件总线
func NewRedisBus(client *redis.Client, logger *logger.Logger) Bus {
	return &redisBus{
		client:   client,
		logger:   logger,
		handlers: make(map[EventType][]Handler),
	}
}

// Publish 发布事件
func (b *redisBus) Publish(ctx context.Context, event *Event) error {
	// 设置事件 ID 和时间戳
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 序列化事件
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 发布到 Redis
	channel := fmt.Sprintf("fusionmail:events:%s", event.Type)
	if err := b.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	b.logger.Debug("Event published", "type", event.Type, "id", event.ID)
	return nil
}

// Subscribe 订阅事件
func (b *redisBus) Subscribe(ctx context.Context, eventType EventType, handler Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 添加处理器
	b.handlers[eventType] = append(b.handlers[eventType], handler)

	b.logger.Info("Event handler subscribed", "type", eventType)
	return nil
}

// Unsubscribe 取消订阅
func (b *redisBus) Unsubscribe(ctx context.Context, eventType EventType) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 清除处理器
	delete(b.handlers, eventType)

	b.logger.Info("Event handler unsubscribed", "type", eventType)
	return nil
}

// Start 启动事件总线
func (b *redisBus) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return fmt.Errorf("event bus already started")
	}

	// 创建上下文
	b.ctx, b.cancel = context.WithCancel(ctx)

	// 启动订阅处理
	b.wg.Add(1)
	go b.subscribeLoop()

	b.started = true
	b.logger.Info("Event bus started")
	return nil
}

// Stop 停止事件总线
func (b *redisBus) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return nil
	}

	// 取消上下文
	if b.cancel != nil {
		b.cancel()
	}

	// 等待 goroutine 结束
	b.wg.Wait()

	b.started = false
	b.logger.Info("Event bus stopped")
	return nil
}

// subscribeLoop 订阅循环
func (b *redisBus) subscribeLoop() {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		// 获取当前订阅的事件类型
		b.mu.RLock()
		var channels []string
		for eventType := range b.handlers {
			channels = append(channels, fmt.Sprintf("fusionmail:events:%s", eventType))
		}
		b.mu.RUnlock()

		if len(channels) == 0 {
			// 没有订阅，等待一段时间后重试
			time.Sleep(1 * time.Second)
			continue
		}

		// 订阅 Redis 频道
		pubsub := b.client.Subscribe(b.ctx, channels...)
		defer pubsub.Close()

		// 处理消息
		ch := pubsub.Channel()
		for {
			select {
			case <-b.ctx.Done():
				return
			case msg := <-ch:
				if msg == nil {
					// 频道关闭，重新订阅
					goto reconnect
				}
				b.handleMessage(msg)
			}
		}

	reconnect:
		// 重新连接前等待一段时间
		time.Sleep(1 * time.Second)
	}
}

// handleMessage 处理消息
func (b *redisBus) handleMessage(msg *redis.Message) {
	// 解析事件
	var event Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		b.logger.Error("Failed to unmarshal event", "error", err, "payload", msg.Payload)
		return
	}

	// 获取处理器
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	// 执行处理器
	for _, handler := range handlers {
		go func(h Handler) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := h(ctx, &event); err != nil {
				b.logger.Error("Event handler failed",
					"type", event.Type,
					"id", event.ID,
					"error", err)
			} else {
				b.logger.Debug("Event handled successfully",
					"type", event.Type,
					"id", event.ID)
			}
		}(handler)
	}
}

// generateEventID 生成事件 ID
func generateEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// NewEvent 创建新事件
func NewEvent(eventType EventType, data map[string]interface{}, source string) *Event {
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Source:    source,
	}
}

// EmailReceivedEvent 创建邮件接收事件
func EmailReceivedEvent(emailID int64, accountUID string, subject string) *Event {
	return NewEvent(EventEmailReceived, map[string]interface{}{
		"email_id":    emailID,
		"account_uid": accountUID,
		"subject":     subject,
	}, "email_service")
}

// EmailReadEvent 创建邮件已读事件
func EmailReadEvent(emailID int64, accountUID string) *Event {
	return NewEvent(EventEmailRead, map[string]interface{}{
		"email_id":    emailID,
		"account_uid": accountUID,
	}, "email_service")
}

// EmailArchivedEvent 创建邮件归档事件
func EmailArchivedEvent(emailID int64, accountUID string) *Event {
	return NewEvent(EventEmailArchived, map[string]interface{}{
		"email_id":    emailID,
		"account_uid": accountUID,
	}, "email_service")
}

// EmailDeletedEvent 创建邮件删除事件
func EmailDeletedEvent(emailID int64, accountUID string) *Event {
	return NewEvent(EventEmailDeleted, map[string]interface{}{
		"email_id":    emailID,
		"account_uid": accountUID,
	}, "email_service")
}
