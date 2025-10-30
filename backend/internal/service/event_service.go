package service

import (
	"context"
	"fmt"

	"fusionmail/internal/model"
	"fusionmail/pkg/event"
	"fusionmail/pkg/logger"
)

// EventService 事件服务接口
type EventService interface {
	// 启动事件服务
	Start(ctx context.Context) error

	// 停止事件服务
	Stop() error

	// 发布事件
	PublishEvent(ctx context.Context, evt *event.Event) error
}

// eventService 事件服务实现
type eventService struct {
	eventBus       event.Bus
	ruleService    RuleService
	webhookService WebhookService
	logger         *logger.Logger
}

// NewEventService 创建事件服务实例
func NewEventService(
	eventBus event.Bus,
	ruleService RuleService,
	webhookService WebhookService,
	logger *logger.Logger,
) EventService {
	return &eventService{
		eventBus:       eventBus,
		ruleService:    ruleService,
		webhookService: webhookService,
		logger:         logger,
	}
}

// Start 启动事件服务
func (s *eventService) Start(ctx context.Context) error {
	// 启动事件总线
	if err := s.eventBus.Start(ctx); err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	// 订阅邮件事件以触发规则
	if err := s.subscribeToEmailEvents(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to email events: %w", err)
	}

	// 订阅所有事件以触发 Webhook
	if err := s.subscribeToWebhookEvents(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to webhook events: %w", err)
	}

	s.logger.Info("Event service started")
	return nil
}

// Stop 停止事件服务
func (s *eventService) Stop() error {
	if err := s.eventBus.Stop(); err != nil {
		return fmt.Errorf("failed to stop event bus: %w", err)
	}

	s.logger.Info("Event service stopped")
	return nil
}

// PublishEvent 发布事件
func (s *eventService) PublishEvent(ctx context.Context, evt *event.Event) error {
	return s.eventBus.Publish(ctx, evt)
}

// subscribeToEmailEvents 订阅邮件事件以触发规则
func (s *eventService) subscribeToEmailEvents(ctx context.Context) error {
	// 订阅邮件接收事件
	if err := s.eventBus.Subscribe(ctx, event.EventEmailReceived, s.handleEmailReceivedForRules); err != nil {
		return err
	}

	return nil
}

// subscribeToWebhookEvents 订阅所有事件以触发 Webhook
func (s *eventService) subscribeToWebhookEvents(ctx context.Context) error {
	// 邮件事件
	emailEvents := []event.EventType{
		event.EventEmailReceived,
		event.EventEmailRead,
		event.EventEmailUnread,
		event.EventEmailStarred,
		event.EventEmailArchived,
		event.EventEmailDeleted,
	}

	for _, eventType := range emailEvents {
		if err := s.eventBus.Subscribe(ctx, eventType, s.handleEventForWebhooks); err != nil {
			return err
		}
	}

	// 账户事件
	accountEvents := []event.EventType{
		event.EventAccountAdded,
		event.EventAccountUpdated,
		event.EventAccountDeleted,
		event.EventAccountSynced,
	}

	for _, eventType := range accountEvents {
		if err := s.eventBus.Subscribe(ctx, eventType, s.handleEventForWebhooks); err != nil {
			return err
		}
	}

	// 同步事件
	syncEvents := []event.EventType{
		event.EventSyncStarted,
		event.EventSyncCompleted,
		event.EventSyncFailed,
	}

	for _, eventType := range syncEvents {
		if err := s.eventBus.Subscribe(ctx, eventType, s.handleEventForWebhooks); err != nil {
			return err
		}
	}

	return nil
}

// handleEmailReceivedForRules 处理邮件接收事件以触发规则
func (s *eventService) handleEmailReceivedForRules(ctx context.Context, evt *event.Event) error {
	// 提取邮件信息
	emailID, ok := evt.Data["email_id"].(float64) // JSON 数字解析为 float64
	if !ok {
		s.logger.Error("Invalid email_id in event", "event_id", evt.ID)
		return fmt.Errorf("invalid email_id in event")
	}

	accountUID, ok := evt.Data["account_uid"].(string)
	if !ok {
		s.logger.Error("Invalid account_uid in event", "event_id", evt.ID)
		return fmt.Errorf("invalid account_uid in event")
	}

	// 应用规则到邮件
	if err := s.ruleService.ApplyRulesToEmail(ctx, int64(emailID), accountUID); err != nil {
		s.logger.Error("Failed to apply rules to email",
			"email_id", int64(emailID),
			"account_uid", accountUID,
			"error", err)
		return err
	}

	s.logger.Debug("Rules applied to email",
		"email_id", int64(emailID),
		"account_uid", accountUID)

	return nil
}

// handleEventForWebhooks 处理事件以触发 Webhook
func (s *eventService) handleEventForWebhooks(ctx context.Context, evt *event.Event) error {
	// 转换为 Webhook 事件格式
	webhookEvent := &WebhookEvent{
		Type:      string(evt.Type),
		Timestamp: evt.Timestamp,
		Data:      evt.Data,
	}

	// 触发 Webhook
	if err := s.webhookService.TriggerEvent(ctx, webhookEvent); err != nil {
		s.logger.Error("Failed to trigger webhooks",
			"event_type", evt.Type,
			"event_id", evt.ID,
			"error", err)
		return err
	}

	s.logger.Debug("Webhooks triggered for event",
		"event_type", evt.Type,
		"event_id", evt.ID)

	return nil
}

// EmailEventPublisher 邮件事件发布器
type EmailEventPublisher struct {
	eventService EventService
}

// NewEmailEventPublisher 创建邮件事件发布器
func NewEmailEventPublisher(eventService EventService) *EmailEventPublisher {
	return &EmailEventPublisher{
		eventService: eventService,
	}
}

// PublishEmailReceived 发布邮件接收事件
func (p *EmailEventPublisher) PublishEmailReceived(ctx context.Context, email *model.Email) error {
	evt := event.EmailReceivedEvent(email.ID, email.AccountUID, email.Subject)
	return p.eventService.PublishEvent(ctx, evt)
}

// PublishEmailRead 发布邮件已读事件
func (p *EmailEventPublisher) PublishEmailRead(ctx context.Context, emailID int64, accountUID string) error {
	evt := event.EmailReadEvent(emailID, accountUID)
	return p.eventService.PublishEvent(ctx, evt)
}

// PublishEmailArchived 发布邮件归档事件
func (p *EmailEventPublisher) PublishEmailArchived(ctx context.Context, emailID int64, accountUID string) error {
	evt := event.EmailArchivedEvent(emailID, accountUID)
	return p.eventService.PublishEvent(ctx, evt)
}

// PublishEmailDeleted 发布邮件删除事件
func (p *EmailEventPublisher) PublishEmailDeleted(ctx context.Context, emailID int64, accountUID string) error {
	evt := event.EmailDeletedEvent(emailID, accountUID)
	return p.eventService.PublishEvent(ctx, evt)
}
