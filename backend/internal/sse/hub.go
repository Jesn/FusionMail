package sse

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Event 表示一个要推送到客户端的事件
// Type: 事件名，例如 "email_counts_maybe_changed"、"ping"
// Data: 事件负载（JSON 字符串即可）
type Event struct {
	Type string
	Data string
}

// Hub 负责管理所有 SSE 客户端与事件广播
// 设计为轻量单例：满足本项目的计数更新广播即可
// 如后续需要分 Topic/多类型事件，可扩展为 map[topic]Hub

type Hub struct {
	mu         sync.RWMutex
	clients    map[chan Event]struct{}
	register   chan chan Event
	unregister chan chan Event
	broadcast  chan Event

	// 简单监控指标（仅用于日志打印，不做强一致保证）
	totalConnections   int64 // 累计建立过的连接数
	activeConnections  int64 // 当前活跃连接数
	totalBroadcasts    int64 // 累计广播次数
	totalSlowConsumers int64 // 累计因“慢消费者”而被剔除的连接数
}

var (
	defaultHub *Hub
	once       sync.Once
)

// Start 确保单例 Hub 启动（幂等）
func Start() {
	once.Do(func() {
		defaultHub = &Hub{
			clients:    make(map[chan Event]struct{}),
			register:   make(chan chan Event),
			unregister: make(chan chan Event),
			broadcast:  make(chan Event, 128), // 简单缓冲，避免短抖动丢事件
		}
		go defaultHub.run()
	})
}

func (h *Hub) run() {
	for {
		select {
		case ch := <-h.register:
			h.mu.Lock()
			h.clients[ch] = struct{}{}
			active := len(h.clients)
			h.mu.Unlock()

			atomic.AddInt64(&h.totalConnections, 1)
			atomic.StoreInt64(&h.activeConnections, int64(active))
			fmt.Printf("[SSE Hub] 新连接注册，当前连接数: %d，总连接数: %d\n", active, atomic.LoadInt64(&h.totalConnections))

		case ch := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[ch]; ok {
				delete(h.clients, ch)
				close(ch)
			}
			active := len(h.clients)
			h.mu.Unlock()

			atomic.StoreInt64(&h.activeConnections, int64(active))
			fmt.Printf("[SSE Hub] 连接注销，当前连接数: %d\n", active)

		case ev := <-h.broadcast:
			atomic.AddInt64(&h.totalBroadcasts, 1)

			h.mu.RLock()
			clientCount := len(h.clients)
			for ch := range h.clients {
				select {
				case ch <- ev:
				default:
					// 慢消费者：丢弃该连接
					atomic.AddInt64(&h.totalSlowConsumers, 1)
					go func(c chan Event) { h.unregister <- c }(ch)
				}
			}
			h.mu.RUnlock()

			fmt.Printf(
				"[SSE Hub] 广播事件: %s, 连接数: %d, 累计广播: %d, 累计慢消费者: %d\n",
				ev.Type,
				clientCount,
				atomic.LoadInt64(&h.totalBroadcasts),
				atomic.LoadInt64(&h.totalSlowConsumers),
			)
		}
	}
}

// Subscribe 订阅事件，返回事件通道与取消函数
func Subscribe() (chan Event, func()) {
	Start()
	ch := make(chan Event, 16)
	defaultHub.register <- ch
	return ch, func() { defaultHub.unregister <- ch }
}

// Broadcast 广播事件给所有订阅者
func Broadcast(eventType string, data string) {
	Start()
	defaultHub.broadcast <- Event{Type: eventType, Data: data}
}
