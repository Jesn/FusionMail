package goroutine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_BasicFunctionality(t *testing.T) {
	config := &PoolConfig{
		MaxConcurrency: 3,
		QueueSize:      10,
		TaskTimeout:    5 * time.Second,
		WaitOnClose:    true,
	}

	pool := NewPool(config)

	ctx := context.Background()
	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("启动协程池失败: %v", err)
	}

	// 提交任务
	var completed int64
	for i := 0; i < 5; i++ {
		taskID := "task-" + string(rune('0'+i))
		err := pool.Submit(ctx, taskID, func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
			return nil
		})
		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}
	}

	// 等待任务完成
	time.Sleep(500 * time.Millisecond)

	stats := pool.GetStats()
	if stats.SubmittedCount != 5 {
		t.Errorf("提交任务数不正确: got %d, want 5", stats.SubmittedCount)
	}
	if atomic.LoadInt64(&completed) != 5 {
		t.Errorf("完成任务数不正确: got %d, want 5", atomic.LoadInt64(&completed))
	}

	pool.Stop()
}

func TestPool_ConcurrencyLimit(t *testing.T) {
	config := &PoolConfig{
		MaxConcurrency: 2, // 限制并发数为 2
		QueueSize:      10,
		TaskTimeout:    5 * time.Second,
		WaitOnClose:    true,
	}

	pool := NewPool(config)

	ctx := context.Background()
	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("启动协程池失败: %v", err)
	}

	var maxConcurrent int64
	var currentConcurrent int64

	// 提交多个任务
	for i := 0; i < 5; i++ {
		err := pool.Submit(ctx, "task", func(ctx context.Context) error {
			current := atomic.AddInt64(&currentConcurrent, 1)
			// 更新最大并发数
			for {
				max := atomic.LoadInt64(&maxConcurrent)
				if current <= max || atomic.CompareAndSwapInt64(&maxConcurrent, max, current) {
					break
				}
			}
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt64(&currentConcurrent, -1)
			return nil
		})
		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}
	}

	// 等待任务完成
	time.Sleep(600 * time.Millisecond)

	if atomic.LoadInt64(&maxConcurrent) > 2 {
		t.Errorf("最大并发数超过限制: got %d, want <= 2", atomic.LoadInt64(&maxConcurrent))
	}

	pool.Stop()
}

func TestPool_TaskError(t *testing.T) {
	config := &PoolConfig{
		MaxConcurrency: 2,
		QueueSize:      10,
		TaskTimeout:    5 * time.Second,
		WaitOnClose:    true,
	}

	pool := NewPool(config)

	ctx := context.Background()
	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("启动协程池失败: %v", err)
	}

	expectedErr := errors.New("task error")

	// 提交会失败的任务
	result, err := pool.SubmitWithResult(ctx, "error-task", func(ctx context.Context) error {
		return expectedErr
	})
	if err != nil {
		t.Fatalf("提交任务失败: %v", err)
	}

	if result.Error == nil {
		t.Error("任务应该返回错误")
	}
	if result.Error.Error() != expectedErr.Error() {
		t.Errorf("错误不匹配: got %v, want %v", result.Error, expectedErr)
	}

	pool.Stop()

	stats := pool.GetStats()
	if stats.FailedCount != 1 {
		t.Errorf("失败任务数不正确: got %d, want 1", stats.FailedCount)
	}
}

func TestPool_TrySubmit(t *testing.T) {
	config := &PoolConfig{
		MaxConcurrency: 1,
		QueueSize:      5, // 足够大的队列
		TaskTimeout:    5 * time.Second,
		WaitOnClose:    true,
	}

	pool := NewPool(config)

	ctx := context.Background()
	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("启动协程池失败: %v", err)
	}

	// 第一个任务应该成功
	ok := pool.TrySubmit("task-1", func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if !ok {
		t.Error("第一个任务应该提交成功")
	}

	// 第二个任务应该成功（进入队列）
	ok = pool.TrySubmit("task-2", func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if !ok {
		t.Error("第二个任务应该提交成功")
	}

	pool.Stop()
}

func TestSemaphore(t *testing.T) {
	sem := NewSemaphore(2)

	// 获取两个信号量
	sem.Acquire()
	sem.Acquire()

	// 第三个应该无法立即获取
	if sem.TryAcquire() {
		t.Error("第三个信号量不应该能获取")
	}

	// 释放一个
	sem.Release()

	// 现在应该能获取
	if !sem.TryAcquire() {
		t.Error("释放后应该能获取信号量")
	}

	// 检查可用数量
	if sem.Available() != 0 {
		t.Errorf("可用信号量数量不正确: got %d, want 0", sem.Available())
	}

	// 释放所有
	sem.Release()
	sem.Release()

	if sem.Available() != 2 {
		t.Errorf("可用信号量数量不正确: got %d, want 2", sem.Available())
	}
}

func TestSafeGo(t *testing.T) {
	done := make(chan bool, 1)

	// 测试正常执行
	SafeGo(func() {
		done <- true
	})

	select {
	case <-done:
		// 成功
	case <-time.After(time.Second):
		t.Error("SafeGo 应该执行完成")
	}

	// 测试 panic 恢复
	SafeGo(func() {
		panic("test panic")
	})

	// 如果没有 panic 传播，测试通过
	time.Sleep(100 * time.Millisecond)
}

func TestRunWithRecovery(t *testing.T) {
	// 测试正常执行
	err := RunWithRecovery(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("正常执行不应该返回错误: %v", err)
	}

	// 测试返回错误
	expectedErr := errors.New("expected error")
	err = RunWithRecovery(func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("应该返回预期错误: got %v, want %v", err, expectedErr)
	}

	// 测试 panic 恢复
	err = RunWithRecovery(func() error {
		panic("test panic")
	})
	if err == nil {
		t.Error("panic 应该被转换为错误")
	}
	if _, ok := err.(*PanicError); !ok {
		t.Errorf("错误类型应该是 PanicError: got %T", err)
	}
}
