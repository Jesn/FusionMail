type EventCallback = (...args: any[]) => void

class EventBus {
  private events: Map<string, EventCallback[]> = new Map()

  /**
   * 订阅事件
   */
  on(event: string, callback: EventCallback): () => void {
    if (!this.events.has(event)) {
      this.events.set(event, [])
    }

    const callbacks = this.events.get(event)!
    callbacks.push(callback)

    // 返回取消订阅函数
    return () => {
      const index = callbacks.indexOf(callback)
      if (index > -1) {
        callbacks.splice(index, 1)
      }
    }
  }

  /**
   * 触发事件
   */
  emit(event: string, ...args: any[]): void {
    const callbacks = this.events.get(event)
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(...args)
        } catch (error) {
          console.error(`Error in event callback for "${event}":`, error)
        }
      })
    }
  }

  /**
   * 取消订阅事件
   */
  off(event: string, callback?: EventCallback): void {
    if (!callback) {
      // 如果没有指定回调函数，移除该事件的所有监听器
      this.events.delete(event)
      return
    }

    const callbacks = this.events.get(event)
    if (callbacks) {
      const index = callbacks.indexOf(callback)
      if (index > -1) {
        callbacks.splice(index, 1)
      }
    }
  }

  /**
   * 清除所有事件监听器
   */
  clear(): void {
    this.events.clear()
  }
}

// 导出全局事件总线实例
export const eventBus = new EventBus()

// 预定义的事件常量
export const AUTH_EVENTS = {
  UNAUTHORIZED: 'auth:unauthorized',
  LOGIN: 'auth:login',
  LOGOUT: 'auth:logout',
} as const