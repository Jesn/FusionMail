# Gmail、Hotmail/Outlook 邮箱推送功能分析

## 概述

分析 Gmail、Hotmail 和 Outlook 邮箱是否支持主动推送功能，以及如何在 FusionMail 中实现实时邮件接收。

## 1. Gmail 推送功能

### 1.1 IMAP IDLE（传统方式）

**支持情况**: ✅ 支持

Gmail 的 IMAP 服务器支持 IMAP IDLE 扩展（RFC 2177）。

**工作原理**:
- 客户端发送 IDLE 命令后，保持连接打开
- 服务器在有新邮件时主动推送通知
- 客户端收到通知后立即获取新邮件

**优点**:
- 实时性好（几乎即时）
- 标准协议，兼容性好
- 无需额外配置

**缺点**:
- 需要保持长连接
- 每 29 分钟需要重新发送 IDLE 命令（Gmail 限制）
- 连接可能被防火墙或代理中断
- 资源占用相对较高

**实现示例**:
```go
// 使用 go-imap 库实现 IDLE
import "github.com/emersion/go-imap/client"

// 开启 IDLE 模式
idleClient := idle.NewClient(c)
updates := make(chan client.Update)
c.Updates = updates

// 启动 IDLE
done := make(chan error, 1)
go func() {
    done <- idleClient.IdleWithFallback(nil, 0)
}()

// 监听更新
for {
    select {
    case update := <-updates:
        // 处理邮件更新
        if mailboxUpdate, ok := update.(*client.MailboxUpdate); ok {
            // 有新邮件
            fetchNewEmails()
        }
    case err := <-done:
        // IDLE 结束，重新启动
        if err != nil {
            log.Printf("IDLE error: %v", err)
        }
    }
}
```

### 1.2 Gmail API Push Notifications（推荐方式）

**支持情况**: ✅ 支持（推荐）

Gmail API 提供了基于 Cloud Pub/Sub 的推送通知机制。

**工作原理**:
1. 应用订阅用户的邮箱变更事件
2. Gmail 通过 Google Cloud Pub/Sub 推送通知
3. 应用接收通知后调用 API 获取变更详情

**优点**:
- 真正的服务器推送，无需保持连接
- 可靠性高，由 Google 基础设施保证
- 支持批量处理
- 资源占用低

**缺点**:
- 需要配置 Google Cloud Pub/Sub
- 需要公网可访问的 Webhook 端点
- 配置相对复杂
- 可能产生额外费用（Pub/Sub）

**实现步骤**:
```go
// 1. 创建 Pub/Sub Topic 和 Subscription
// 2. 订阅邮箱变更
watchRequest := &gmail.WatchRequest{
    TopicName: "projects/YOUR_PROJECT/topics/gmail-push",
    LabelIds:  []string{"INBOX"},
}
watchResponse, err := gmailService.Users.Watch("me", watchRequest).Do()

// 3. 接收 Pub/Sub 推送
// 在 HTTP 端点接收推送通知
func handlePubSubPush(w http.ResponseWriter, r *http.Request) {
    var message PubSubMessage
    json.NewDecoder(r.Body).Decode(&message)
    
    // 解析通知
    historyId := message.Data.HistoryId
    
    // 调用 API 获取变更
    history, err := gmailService.Users.History.List("me").
        StartHistoryId(historyId).Do()
    
    // 处理新邮件
    for _, h := range history.History {
        for _, msg := range h.MessagesAdded {
            // 处理新邮件
        }
    }
}
```

**配置要求**:
- Google Cloud Project
- 启用 Gmail API 和 Cloud Pub/Sub API
- 创建 Pub/Sub Topic
- 配置 Webhook 端点（需要 HTTPS）

## 2. Outlook/Hotmail 推送功能

### 2.1 IMAP IDLE

**支持情况**: ⚠️ 有限支持

Outlook.com 的 IMAP 服务器对 IDLE 的支持不够稳定。

**问题**:
- IDLE 连接经常被服务器主动断开
- 超时时间不稳定（可能 5-15 分钟）
- 推送延迟较大
- 不推荐用于生产环境

**建议**: 不推荐使用 IMAP IDLE 方式

### 2.2 Microsoft Graph API Webhooks（推荐方式）

**支持情况**: ✅ 支持（推荐）

Microsoft Graph API 提供了 Webhook 订阅机制。

**工作原理**:
1. 应用创建订阅，指定要监听的资源和 Webhook URL
2. 当资源发生变化时，Microsoft Graph 发送通知到 Webhook
3. 应用接收通知后调用 API 获取详细信息

**优点**:
- 真正的服务器推送
- 实时性好（通常 1-3 秒延迟）
- 可靠性高
- 无需保持连接

**缺点**:
- 需要公网可访问的 HTTPS 端点
- 订阅有效期最长 3 天，需要定期续订
- 需要验证 Webhook 端点

**实现示例**:
```go
// 1. 创建订阅
subscription := &msgraph.Subscription{
    ChangeType:         "created,updated",
    NotificationURL:    "https://your-domain.com/api/webhooks/outlook",
    Resource:           "/me/mailFolders('Inbox')/messages",
    ExpirationDateTime: time.Now().Add(3 * 24 * time.Hour),
    ClientState:        "secret-validation-token",
}

// 发送订阅请求
resp, err := graphClient.Subscriptions().Request().Add(context.Background(), subscription)

// 2. 验证 Webhook 端点
func handleWebhookValidation(w http.ResponseWriter, r *http.Request) {
    validationToken := r.URL.Query().Get("validationToken")
    if validationToken != "" {
        // 返回验证令牌
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte(validationToken))
        return
    }
}

// 3. 接收推送通知
func handleOutlookWebhook(w http.ResponseWriter, r *http.Request) {
    var notification GraphNotification
    json.NewDecoder(r.Body).Decode(&notification)
    
    // 验证 clientState
    if notification.Value[0].ClientState != "secret-validation-token" {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }
    
    // 处理通知
    for _, change := range notification.Value {
        // 获取邮件详情
        messageID := extractMessageID(change.ResourceData.Id)
        message, err := graphClient.Me().Messages().ByID(messageID).Request().Get(context.Background())
        
        // 处理新邮件
        processNewEmail(message)
    }
    
    w.WriteHeader(http.StatusAccepted)
}

// 4. 定期续订（每 2 天）
func renewSubscription(subscriptionID string) {
    update := &msgraph.Subscription{
        ExpirationDateTime: time.Now().Add(3 * 24 * time.Hour),
    }
    
    graphClient.Subscriptions().ByID(subscriptionID).Request().Update(context.Background(), update)
}
```

**配置要求**:
- Azure AD 应用注册
- 配置 Webhook URL（必须是 HTTPS）
- 订阅权限：`Mail.Read`
- 定期续订机制

### 2.3 Exchange Web Services (EWS) Push Notifications

**支持情况**: ⚠️ 仅企业版

EWS 推送通知仅适用于 Exchange Server（企业版），不适用于 Outlook.com。

## 3. 推送功能对比

| 功能 | Gmail IMAP IDLE | Gmail API Push | Outlook IMAP IDLE | Graph API Webhooks |
|------|----------------|----------------|-------------------|-------------------|
| **实时性** | 优秀（即时） | 优秀（1-2秒） | 一般（不稳定） | 优秀（1-3秒） |
| **可靠性** | 良好 | 优秀 | 较差 | 优秀 |
| **资源占用** | 中等（长连接） | 低 | 中等 | 低 |
| **配置复杂度** | 简单 | 复杂 | 简单 | 中等 |
| **网络要求** | 出站连接 | 入站 HTTPS | 出站连接 | 入站 HTTPS |
| **额外费用** | 无 | 可能有（Pub/Sub） | 无 | 无 |
| **推荐程度** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐⭐⭐⭐ |

## 4. FusionMail 实现建议

### 4.1 短期方案（MVP）

**使用轮询机制**（当前实现）

**优点**:
- 实现简单
- 无需额外配置
- 适合个人用户和小团队

**缺点**:
- 实时性较差（取决于轮询间隔）
- 资源占用相对较高

**建议配置**:
- 默认轮询间隔：5 分钟
- 最小轮询间隔：1 分钟
- 用户可自定义

### 4.2 中期方案（推荐）

**实现 IMAP IDLE 支持**

**适用场景**:
- Gmail IMAP 账户
- 自建邮件服务器
- 支持 IDLE 的其他提供商

**实现要点**:
```go
// 1. 检测 IMAP 服务器是否支持 IDLE
caps, err := client.Capability()
if !caps["IDLE"] {
    // 降级到轮询模式
    return usePollingMode()
}

// 2. 实现 IDLE 监听
type IMAPIdleListener struct {
    client      *client.Client
    idleClient  *idle.Client
    accountUID  string
    stopChan    chan struct{}
}

func (l *IMAPIdleListener) Start() {
    for {
        select {
        case <-l.stopChan:
            return
        default:
            // 启动 IDLE，29 分钟超时
            err := l.idleClient.IdleWithFallback(l.stopChan, 29*time.Minute)
            if err != nil {
                log.Printf("IDLE error: %v", err)
                time.Sleep(5 * time.Second)
                continue
            }
            
            // 收到通知，获取新邮件
            l.fetchNewEmails()
        }
    }
}

// 3. 优雅关闭
func (l *IMAPIdleListener) Stop() {
    close(l.stopChan)
}
```

**优点**:
- 实时性好
- 无需额外配置
- 适合 Gmail IMAP

**缺点**:
- Outlook IMAP 支持不佳
- 需要管理长连接

### 4.3 长期方案（企业级）

**实现 API Push Notifications**

#### Gmail API Push

**实现步骤**:

1. **配置 Google Cloud Pub/Sub**
```go
// 创建 Pub/Sub Topic
func setupGmailPush() error {
    // 1. 创建 Topic
    topic := pubsubClient.Topic("gmail-push-notifications")
    exists, err := topic.Exists(ctx)
    if !exists {
        topic, err = pubsubClient.CreateTopic(ctx, "gmail-push-notifications")
    }
    
    // 2. 创建 Subscription
    sub := pubsubClient.Subscription("gmail-push-sub")
    exists, err = sub.Exists(ctx)
    if !exists {
        sub, err = pubsubClient.CreateSubscription(ctx, "gmail-push-sub", pubsub.SubscriptionConfig{
            Topic:       topic,
            AckDeadline: 10 * time.Second,
        })
    }
    
    return nil
}

// 订阅邮箱变更
func subscribeGmailAccount(accountUID string, gmailService *gmail.Service) error {
    watchRequest := &gmail.WatchRequest{
        TopicName: "projects/YOUR_PROJECT/topics/gmail-push-notifications",
        LabelIds:  []string{"INBOX"},
    }
    
    watchResponse, err := gmailService.Users.Watch("me", watchRequest).Do()
    if err != nil {
        return err
    }
    
    // 保存 historyId
    saveHistoryID(accountUID, watchResponse.HistoryId)
    
    return nil
}

// 处理 Pub/Sub 消息
func processPubSubMessages() {
    sub := pubsubClient.Subscription("gmail-push-sub")
    
    err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
        var notification GmailPushNotification
        json.Unmarshal(msg.Data, &notification)
        
        // 获取账户信息
        account := getAccountByEmail(notification.EmailAddress)
        
        // 获取历史变更
        historyID := getLastHistoryID(account.UID)
        history, err := gmailService.Users.History.List("me").
            StartHistoryId(historyID).Do()
        
        // 处理新邮件
        for _, h := range history.History {
            for _, msgAdded := range h.MessagesAdded {
                fetchAndSaveEmail(account.UID, msgAdded.Message.Id)
            }
        }
        
        // 更新 historyId
        saveHistoryID(account.UID, history.HistoryId)
        
        msg.Ack()
    })
}
```

#### Microsoft Graph Webhooks

**实现步骤**:

1. **创建订阅管理器**
```go
type WebhookSubscriptionManager struct {
    graphClient *msgraph.GraphServiceRequestBuilder
    db          *gorm.DB
    logger      *logger.Logger
}

// 创建订阅
func (m *WebhookSubscriptionManager) CreateSubscription(accountUID string) error {
    account, err := m.getAccount(accountUID)
    if err != nil {
        return err
    }
    
    subscription := &msgraph.Subscription{
        ChangeType:         "created",
        NotificationURL:    fmt.Sprintf("https://your-domain.com/api/webhooks/outlook/%s", accountUID),
        Resource:           "/me/mailFolders('Inbox')/messages",
        ExpirationDateTime: time.Now().Add(3 * 24 * time.Hour),
        ClientState:        generateClientState(accountUID),
    }
    
    resp, err := m.graphClient.Subscriptions().Request().Add(context.Background(), subscription)
    if err != nil {
        return err
    }
    
    // 保存订阅信息
    m.saveSubscription(accountUID, resp.ID, resp.ExpirationDateTime)
    
    return nil
}

// 续订订阅（定时任务，每 2 天执行）
func (m *WebhookSubscriptionManager) RenewSubscriptions() {
    subscriptions := m.getExpiringSubscriptions()
    
    for _, sub := range subscriptions {
        update := &msgraph.Subscription{
            ExpirationDateTime: time.Now().Add(3 * 24 * time.Hour),
        }
        
        _, err := m.graphClient.Subscriptions().ByID(sub.SubscriptionID).
            Request().Update(context.Background(), update)
        
        if err != nil {
            m.logger.Error("Failed to renew subscription", "account_uid", sub.AccountUID, "error", err)
            // 重新创建订阅
            m.CreateSubscription(sub.AccountUID)
        } else {
            m.updateSubscriptionExpiry(sub.AccountUID, time.Now().Add(3*24*time.Hour))
        }
    }
}

// 处理 Webhook 通知
func (m *WebhookSubscriptionManager) HandleWebhook(accountUID string, notification *GraphNotification) error {
    // 验证 clientState
    if notification.Value[0].ClientState != generateClientState(accountUID) {
        return fmt.Errorf("invalid client state")
    }
    
    account, err := m.getAccount(accountUID)
    if err != nil {
        return err
    }
    
    // 获取邮件详情
    for _, change := range notification.Value {
        messageID := extractMessageID(change.ResourceData.Id)
        
        // 调用 Graph API 获取邮件
        message, err := m.graphClient.Me().Messages().ByID(messageID).
            Request().Get(context.Background())
        
        if err != nil {
            m.logger.Error("Failed to fetch message", "message_id", messageID, "error", err)
            continue
        }
        
        // 保存邮件
        m.saveEmail(account.UID, message)
    }
    
    return nil
}
```

2. **Webhook 端点实现**
```go
// Webhook 验证和处理
func (h *WebhookHandler) HandleOutlookWebhook(c *gin.Context) {
    accountUID := c.Param("account_uid")
    
    // 1. 处理验证请求
    validationToken := c.Query("validationToken")
    if validationToken != "" {
        c.String(http.StatusOK, validationToken)
        return
    }
    
    // 2. 处理通知
    var notification GraphNotification
    if err := c.ShouldBindJSON(&notification); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification"})
        return
    }
    
    // 3. 异步处理（避免超时）
    go func() {
        err := h.subscriptionManager.HandleWebhook(accountUID, &notification)
        if err != nil {
            h.logger.Error("Failed to handle webhook", "account_uid", accountUID, "error", err)
        }
    }()
    
    // 4. 立即返回 202 Accepted
    c.Status(http.StatusAccepted)
}
```

**配置要求**:
- 公网可访问的 HTTPS 端点
- 域名和 SSL 证书
- 定时任务（续订订阅）

## 5. 推荐实现路线图

### Phase 1: MVP（当前）
- ✅ 轮询机制
- ✅ 可配置轮询间隔（1-60 分钟）
- ✅ 适合个人用户

### Phase 2: 增强实时性
- ⬜ 实现 IMAP IDLE 支持（Gmail）
- ⬜ 自动检测并启用 IDLE
- ⬜ IDLE 失败时降级到轮询
- ⬜ 适合中小团队

### Phase 3: 企业级推送
- ⬜ Gmail API Push Notifications
- ⬜ Microsoft Graph Webhooks
- ⬜ 订阅管理和自动续订
- ⬜ 需要公网 HTTPS 端点

## 6. 实现优先级建议

### 高优先级
1. **优化轮询机制**
   - 智能轮询间隔（根据邮件活跃度调整）
   - 错误重试和指数退避
   - 资源使用优化

2. **IMAP IDLE 支持**（Gmail）
   - 实现相对简单
   - 显著提升实时性
   - 无需额外配置

### 中优先级
3. **Microsoft Graph Webhooks**
   - Outlook 用户体验提升
   - 需要 HTTPS 端点
   - 订阅管理复杂度中等

### 低优先级
4. **Gmail API Push Notifications**
   - 需要 Google Cloud 配置
   - 可能产生额外费用
   - 配置复杂度高

## 7. 技术决策建议

### 对于 FusionMail 项目

**当前阶段（MVP）**:
- 保持轮询机制
- 优化轮询策略
- 提供用户可配置的轮询间隔

**下一阶段（v1.1）**:
- 实现 IMAP IDLE 支持
- 仅对 Gmail 启用
- 提供开关选项

**未来阶段（v2.0）**:
- 实现 Webhook 支持
- 需要用户配置公网域名
- 提供详细的配置文档

## 8. 总结

| 邮箱提供商 | IMAP IDLE | API Push | 推荐方案 |
|-----------|-----------|----------|---------|
| **Gmail** | ✅ 支持良好 | ✅ 支持（复杂） | IMAP IDLE |
| **Outlook/Hotmail** | ⚠️ 支持不佳 | ✅ 支持（推荐） | Graph Webhooks |

**关键结论**:
1. Gmail 的 IMAP IDLE 支持良好，是最简单的实时推送方案
2. Outlook 的 IMAP IDLE 不稳定，建议使用 Graph API Webhooks
3. API Push 方案需要公网 HTTPS 端点，适合企业部署
4. 对于个人用户和小团队，优化的轮询机制已经足够

**FusionMail 建议**:
- 短期：优化轮询机制
- 中期：实现 Gmail IMAP IDLE
- 长期：提供 Webhook 支持（可选功能）
