package goroutine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMonitor_BasicFunctionality(t *testing.T) {
	config := &MonitorConfig{
		CheckInterval:           100 * time.Millisecond,
		WarningThreshold:        10000, // 设置高阈值避免测试中触发告警
		CriticalThreshold:       50000,
		EnableLeakDetection:     false,
		LeakDetectionWindowSize: 5,
		LeakGrowthRateThreshold: 0.5,
	}

	monitor := NewMonitor(config)

	// 启动监控
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}

	// 等待几次检查
	time.Sleep(350 * time.Millisecond)

	// 获取统计
	stats := monitor.GetStats()
	if stats.CheckCount < 2 {
		t.Errorf("检查次数不足: got %d, want >= 2", stats.CheckCount)
	}
	if stats.CurrentCount <= 0 {
		t.Errorf("当前 Goroutine 数量应该大于 0: got %d", stats.CurrentCount)
	}
	if stats.PeakCount <= 0 {
		t.Errorf("峰值 Goroutine 数量应该大于 0: got %d", stats.PeakCount)
	}

	// 停止监控
	monitor.Stop()
}

func TestMonitor_WarningCallback(t *testing.T) {
	warningCalled := false
	var mu sync.Mutex

	config := &MonitorConfig{
		CheckInterval:       50 * time.Millisecond,
		WarningThreshold:    1, // 设置很低的阈值确保触发
		CriticalThreshold:   100000,
		EnableLeakDetection: false,
	}

	monitor := NewMonitor(config)
	monitor.SetWarningCallback(func(count int) {
		mu.Lock()
		warningCalled = true
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}

	// 等待回调被触发
	time.Sleep(150 * time.Millisecond)

	monitor.Stop()

	mu.Lock()
	if !warningCalled {
		t.Error("告警回调应该被触发")
	}
	mu.Unlock()
}

func TestGetCurrentCount(t *testing.T) {
	count := GetCurrentCount()
	if count <= 0 {
		t.Errorf("Goroutine 数量应该大于 0: got %d", count)
	}
}

func TestGetRuntimeStats(t *testing.T) {
	stats := GetRuntimeStats()

	if stats.NumGoroutine <= 0 {
		t.Errorf("Goroutine 数量应该大于 0: got %d", stats.NumGoroutine)
	}
	if stats.NumCPU <= 0 {
		t.Errorf("CPU 数量应该大于 0: got %d", stats.NumCPU)
	}
	if stats.GOMAXPROCS <= 0 {
		t.Errorf("GOMAXPROCS 应该大于 0: got %d", stats.GOMAXPROCS)
	}
	if stats.MemStats.Alloc == 0 {
		t.Error("已分配内存应该大于 0")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}
