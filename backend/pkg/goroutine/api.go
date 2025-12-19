package goroutine

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RuntimeStatsResponse 运行时统计响应
type RuntimeStatsResponse struct {
	// Goroutine 统计
	Goroutine GoroutineStatsResponse `json:"goroutine"`
	// 内存统计
	Memory MemoryStatsResponse `json:"memory"`
	// CPU 信息
	CPU CPUStatsResponse `json:"cpu"`
}

// GoroutineStatsResponse Goroutine 统计响应
type GoroutineStatsResponse struct {
	// 当前数量
	Current int `json:"current"`
	// 峰值数量（如果有监控器）
	Peak int `json:"peak,omitempty"`
	// 平均数量（如果有监控器）
	Average float64 `json:"average,omitempty"`
	// 告警次数（如果有监控器）
	WarningCount int64 `json:"warning_count,omitempty"`
	// 疑似泄露次数（如果有监控器）
	LeakSuspectCount int64 `json:"leak_suspect_count,omitempty"`
}

// MemoryStatsResponse 内存统计响应
type MemoryStatsResponse struct {
	// 已分配内存（字节）
	Alloc uint64 `json:"alloc"`
	// 已分配内存（人类可读）
	AllocHuman string `json:"alloc_human"`
	// 系统内存（字节）
	Sys uint64 `json:"sys"`
	// 系统内存（人类可读）
	SysHuman string `json:"sys_human"`
	// 堆分配内存（字节）
	HeapAlloc uint64 `json:"heap_alloc"`
	// 堆分配内存（人类可读）
	HeapAllocHuman string `json:"heap_alloc_human"`
	// 堆对象数量
	HeapObjects uint64 `json:"heap_objects"`
	// GC 次数
	NumGC uint32 `json:"num_gc"`
	// GC 暂停总时间（纳秒）
	PauseTotalNs uint64 `json:"pause_total_ns"`
}

// CPUStatsResponse CPU 统计响应
type CPUStatsResponse struct {
	// CPU 数量
	NumCPU int `json:"num_cpu"`
	// GOMAXPROCS
	GOMAXPROCS int `json:"gomaxprocs"`
}

// RegisterRuntimeStatsAPI 注册运行时统计 API
func RegisterRuntimeStatsAPI(router *gin.Engine, prefix string, monitor *Monitor) {
	if prefix == "" {
		prefix = "/api/v1/system"
	}

	group := router.Group(prefix)
	{
		group.GET("/runtime", func(c *gin.Context) {
			stats := GetRuntimeStats()

			response := RuntimeStatsResponse{
				Goroutine: GoroutineStatsResponse{
					Current: stats.NumGoroutine,
				},
				Memory: MemoryStatsResponse{
					Alloc:          stats.MemStats.Alloc,
					AllocHuman:     FormatBytes(stats.MemStats.Alloc),
					Sys:            stats.MemStats.Sys,
					SysHuman:       FormatBytes(stats.MemStats.Sys),
					HeapAlloc:      stats.MemStats.HeapAlloc,
					HeapAllocHuman: FormatBytes(stats.MemStats.HeapAlloc),
					HeapObjects:    stats.MemStats.HeapObjects,
					NumGC:          stats.MemStats.NumGC,
					PauseTotalNs:   stats.MemStats.PauseTotalNs,
				},
				CPU: CPUStatsResponse{
					NumCPU:     stats.NumCPU,
					GOMAXPROCS: stats.GOMAXPROCS,
				},
			}

			// 如果有监控器，添加监控统计
			if monitor != nil {
				monitorStats := monitor.GetStats()
				response.Goroutine.Peak = monitorStats.PeakCount
				response.Goroutine.Average = monitorStats.AvgCount
				response.Goroutine.WarningCount = monitorStats.WarningCount
				response.Goroutine.LeakSuspectCount = monitorStats.LeakSuspectCount
			}

			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data":    response,
			})
		})

		// Goroutine 堆栈导出（仅开发环境）
		group.GET("/goroutine-stacks", func(c *gin.Context) {
			stacks := DumpGoroutineStacks()
			c.String(http.StatusOK, stacks)
		})
	}
}
