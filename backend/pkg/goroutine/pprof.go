package goroutine

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"

	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
)

// 模块日志记录器
var pprofLog = logger.NewWithModule("Pprof")

// RegisterPprofRoutes 注册 pprof 路由到 Gin 引擎
// 仅在开发环境启用，生产环境应禁用
func RegisterPprofRoutes(router *gin.Engine, prefix string) {
	if prefix == "" {
		prefix = "/debug/pprof"
	}

	pprofGroup := router.Group(prefix)
	{
		pprofGroup.GET("/", gin.WrapF(pprof.Index))
		pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
		pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
		pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
		pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}

	pprofLog.Info("pprof 路由已注册: %s/*", prefix)
}

// RegisterPprofRoutesWithAuth 注册带认证的 pprof 路由
func RegisterPprofRoutesWithAuth(router *gin.Engine, prefix string, authMiddleware gin.HandlerFunc) {
	if prefix == "" {
		prefix = "/debug/pprof"
	}

	pprofGroup := router.Group(prefix)
	if authMiddleware != nil {
		pprofGroup.Use(authMiddleware)
	}

	{
		pprofGroup.GET("/", gin.WrapF(pprof.Index))
		pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
		pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
		pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
		pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}

	pprofLog.Info("pprof 路由已注册（带认证）: %s/*", prefix)
}

// StartPprofServer 启动独立的 pprof 服务器
// 用于生产环境，可以在不同端口启动，避免暴露到主服务
func StartPprofServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	pprofLog.Info("pprof 服务器启动: %s", addr)
	return http.ListenAndServe(addr, mux)
}

// RuntimeStats 运行时统计
type RuntimeStats struct {
	// Goroutine 数量
	NumGoroutine int
	// CPU 数量
	NumCPU int
	// GOMAXPROCS
	GOMAXPROCS int
	// 内存统计
	MemStats MemStats
}

// MemStats 内存统计
type MemStats struct {
	// 已分配内存（字节）
	Alloc uint64
	// 累计分配内存（字节）
	TotalAlloc uint64
	// 系统内存（字节）
	Sys uint64
	// 堆分配内存（字节）
	HeapAlloc uint64
	// 堆系统内存（字节）
	HeapSys uint64
	// 堆空闲内存（字节）
	HeapIdle uint64
	// 堆使用中内存（字节）
	HeapInuse uint64
	// 堆对象数量
	HeapObjects uint64
	// GC 次数
	NumGC uint32
	// 上次 GC 时间（纳秒）
	LastGC uint64
	// GC 暂停总时间（纳秒）
	PauseTotalNs uint64
}

// GetRuntimeStats 获取运行时统计
func GetRuntimeStats() RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return RuntimeStats{
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		GOMAXPROCS:   runtime.GOMAXPROCS(0),
		MemStats: MemStats{
			Alloc:        m.Alloc,
			TotalAlloc:   m.TotalAlloc,
			Sys:          m.Sys,
			HeapAlloc:    m.HeapAlloc,
			HeapSys:      m.HeapSys,
			HeapIdle:     m.HeapIdle,
			HeapInuse:    m.HeapInuse,
			HeapObjects:  m.HeapObjects,
			NumGC:        m.NumGC,
			LastGC:       m.LastGC,
			PauseTotalNs: m.PauseTotalNs,
		},
	}
}

// FormatBytes 格式化字节数
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
