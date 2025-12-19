package goroutine

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	// ErrPoolClosed 协程池已关闭
	ErrPoolClosed = errors.New("goroutine pool is closed")
	// ErrTaskTimeout 任务超时
	ErrTaskTimeout = errors.New("task execution timeout")
	// ErrQueueFull 任务队列已满
	ErrQueueFull = errors.New("task queue is full")
)

// PanicError 任务 panic 错误
type PanicError struct {
	Value interface{}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("task panicked: %v", e.Value)
}
