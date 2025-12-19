package goroutine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"fusionmail/pkg/logger"
)

// 模块日志记录器
var poolLog = logger.NewWithModule("GoroutinePool")

// Task 任务定义
type Task func(ctx context.Context) error

// TaskResult 任务结果
type TaskResult struct {
	TaskID string
	Error  error
	// 执行耗时
	Duration time.Duration
}

// PoolConfig 协程池配置
type PoolConfig struct {
	// 最大并发数
	MaxConcurrency int
	// 任务队列大小
	QueueSize int
	// 任务超时时间（0 表示不超时）
	TaskTimeout time.Duration
	// 是否等待所有任务完成后再关闭
	WaitOnClose bool
}

// DefaultPoolConfig 返回默认配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConcurrency: 10,
		QueueSize:      100,
		TaskTimeout:    5 * time.Minute,
		WaitOnClose:    true,
	}
}

// PoolStats 协程池统计
type PoolStats struct {
	// 已提交任务数
	SubmittedCount int64
	// 已完成任务数
	CompletedCount int64
	// 成功任务数
	SuccessCount int64
	// 失败任务数
	FailedCount int64
	// 超时任务数
	TimeoutCount int64
	// 当前运行中任务数
	RunningCount int64
	// 队列中等待任务数
	QueuedCount int64
}

// Pool 协程池
type Pool struct {
	config *PoolConfig

	// 信号量（控制并发数）
	sem chan struct{}
	// 任务队列
	taskQueue chan taskWrapper
	// 统计
	stats PoolStats

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.RWMutex
	running bool
	closed  bool
}

// taskWrapper 任务包装器
type taskWrapper struct {
	id       string
	task     Task
	resultCh chan<- TaskResult
}

// NewPool 创建新的协程池
func NewPool(config *PoolConfig) *Pool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	return &Pool{
		config:    config,
		sem:       make(chan struct{}, config.MaxConcurrency),
		taskQueue: make(chan taskWrapper, config.QueueSize),
	}
}

// Start 启动协程池
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.closed = false
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.mu.Unlock()

	// 启动任务分发器
	p.wg.Add(1)
	go p.dispatcher()

	poolLog.Info("协程池已启动 (maxConcurrency=%d, queueSize=%d)",
		p.config.MaxConcurrency, p.config.QueueSize)

	return nil
}

// Stop 停止协程池
func (p *Pool) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.closed = true
	p.cancel()
	p.mu.Unlock()

	if p.config.WaitOnClose {
		p.wg.Wait()
	}

	poolLog.Info("协程池已停止")
}

// Submit 提交任务（阻塞直到任务被接受或上下文取消）
func (p *Pool) Submit(ctx context.Context, taskID string, task Task) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrPoolClosed
	}
	p.mu.RUnlock()

	atomic.AddInt64(&p.stats.SubmittedCount, 1)
	atomic.AddInt64(&p.stats.QueuedCount, 1)

	select {
	case p.taskQueue <- taskWrapper{id: taskID, task: task}:
		return nil
	case <-ctx.Done():
		atomic.AddInt64(&p.stats.QueuedCount, -1)
		return ctx.Err()
	case <-p.ctx.Done():
		atomic.AddInt64(&p.stats.QueuedCount, -1)
		return ErrPoolClosed
	}
}

// SubmitWithResult 提交任务并等待结果
func (p *Pool) SubmitWithResult(ctx context.Context, taskID string, task Task) (TaskResult, error) {
	resultCh := make(chan TaskResult, 1)

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return TaskResult{}, ErrPoolClosed
	}
	p.mu.RUnlock()

	atomic.AddInt64(&p.stats.SubmittedCount, 1)
	atomic.AddInt64(&p.stats.QueuedCount, 1)

	select {
	case p.taskQueue <- taskWrapper{id: taskID, task: task, resultCh: resultCh}:
		// 等待结果
		select {
		case result := <-resultCh:
			return result, nil
		case <-ctx.Done():
			return TaskResult{}, ctx.Err()
		}
	case <-ctx.Done():
		atomic.AddInt64(&p.stats.QueuedCount, -1)
		return TaskResult{}, ctx.Err()
	case <-p.ctx.Done():
		atomic.AddInt64(&p.stats.QueuedCount, -1)
		return TaskResult{}, ErrPoolClosed
	}
}

// TrySubmit 尝试提交任务（非阻塞）
func (p *Pool) TrySubmit(taskID string, task Task) bool {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return false
	}
	p.mu.RUnlock()

	select {
	case p.taskQueue <- taskWrapper{id: taskID, task: task}:
		atomic.AddInt64(&p.stats.SubmittedCount, 1)
		atomic.AddInt64(&p.stats.QueuedCount, 1)
		return true
	default:
		return false
	}
}

// GetStats 获取统计数据
func (p *Pool) GetStats() PoolStats {
	return PoolStats{
		SubmittedCount: atomic.LoadInt64(&p.stats.SubmittedCount),
		CompletedCount: atomic.LoadInt64(&p.stats.CompletedCount),
		SuccessCount:   atomic.LoadInt64(&p.stats.SuccessCount),
		FailedCount:    atomic.LoadInt64(&p.stats.FailedCount),
		TimeoutCount:   atomic.LoadInt64(&p.stats.TimeoutCount),
		RunningCount:   atomic.LoadInt64(&p.stats.RunningCount),
		QueuedCount:    atomic.LoadInt64(&p.stats.QueuedCount),
	}
}

// dispatcher 任务分发器
func (p *Pool) dispatcher() {
	defer p.wg.Done()

	for {
		select {
		case tw := <-p.taskQueue:
			atomic.AddInt64(&p.stats.QueuedCount, -1)

			// 获取信号量
			select {
			case p.sem <- struct{}{}:
				p.wg.Add(1)
				go p.executeTask(tw)
			case <-p.ctx.Done():
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// executeTask 执行任务
func (p *Pool) executeTask(tw taskWrapper) {
	defer func() {
		<-p.sem // 释放信号量
		p.wg.Done()
	}()

	atomic.AddInt64(&p.stats.RunningCount, 1)
	defer atomic.AddInt64(&p.stats.RunningCount, -1)

	startTime := time.Now()
	var err error

	// 创建任务上下文
	var taskCtx context.Context
	var taskCancel context.CancelFunc
	if p.config.TaskTimeout > 0 {
		taskCtx, taskCancel = context.WithTimeout(p.ctx, p.config.TaskTimeout)
	} else {
		taskCtx, taskCancel = context.WithCancel(p.ctx)
	}
	defer taskCancel()

	// 执行任务
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				err = &PanicError{Value: r}
				poolLog.Error("任务 panic: taskID=%s, panic=%v", tw.id, r)
			}
		}()
		err = tw.task(taskCtx)
	}()

	// 等待任务完成或超时
	select {
	case <-done:
		// 任务正常完成
	case <-taskCtx.Done():
		if taskCtx.Err() == context.DeadlineExceeded {
			err = ErrTaskTimeout
			atomic.AddInt64(&p.stats.TimeoutCount, 1)
		}
	}

	duration := time.Since(startTime)
	atomic.AddInt64(&p.stats.CompletedCount, 1)

	if err != nil {
		atomic.AddInt64(&p.stats.FailedCount, 1)
		poolLog.Debug("任务失败: taskID=%s, duration=%v, err=%v", tw.id, duration, err)
	} else {
		atomic.AddInt64(&p.stats.SuccessCount, 1)
	}

	// 发送结果
	if tw.resultCh != nil {
		tw.resultCh <- TaskResult{
			TaskID:   tw.id,
			Error:    err,
			Duration: duration,
		}
	}
}
