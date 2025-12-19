package goroutine

import (
	"context"
	"sync"
)

// Semaphore 信号量，用于限制并发数
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore 创建新的信号量
func NewSemaphore(maxConcurrency int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, maxConcurrency),
	}
}

// Acquire 获取信号量（阻塞）
func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

// AcquireContext 获取信号量（支持上下文取消）
func (s *Semaphore) AcquireContext(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire 尝试获取信号量（非阻塞）
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release 释放信号量
func (s *Semaphore) Release() {
	<-s.ch
}

// Available 返回可用的信号量数量
func (s *Semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}

// WaitGroup 带超时的 WaitGroup
type WaitGroupWithTimeout struct {
	wg sync.WaitGroup
}

// NewWaitGroupWithTimeout 创建新的 WaitGroup
func NewWaitGroupWithTimeout() *WaitGroupWithTimeout {
	return &WaitGroupWithTimeout{}
}

// Add 添加计数
func (w *WaitGroupWithTimeout) Add(delta int) {
	w.wg.Add(delta)
}

// Done 完成一个计数
func (w *WaitGroupWithTimeout) Done() {
	w.wg.Done()
}

// Wait 等待所有任务完成
func (w *WaitGroupWithTimeout) Wait() {
	w.wg.Wait()
}

// WaitContext 等待所有任务完成（支持上下文取消）
func (w *WaitGroupWithTimeout) WaitContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SafeGo 安全地启动 goroutine，捕获 panic
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				poolLog.Error("SafeGo panic recovered: %v", r)
			}
		}()
		fn()
	}()
}

// SafeGoWithContext 安全地启动 goroutine，支持上下文和 panic 捕获
func SafeGoWithContext(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				poolLog.Error("SafeGoWithContext panic recovered: %v", r)
			}
		}()
		fn(ctx)
	}()
}

// RunWithRecovery 运行函数并捕获 panic
func RunWithRecovery(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{Value: r}
		}
	}()
	return fn()
}
