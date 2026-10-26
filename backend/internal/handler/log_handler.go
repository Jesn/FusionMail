package handler

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fusionmail/internal/dto"
	"fusionmail/pkg/logger"

	"github.com/gin-gonic/gin"
)

// 模块日志记录器
var logHandlerLog = logger.NewWithModule("LogHandler")

// 包级预编译正则，避免每次调用重新编译
var (
	// ansiPattern 匹配 ANSI 颜色转义序列
	ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	// logPattern 日志行格式正则: 2006/01/02 15:04:05 [LEVEL][Module] message | fields (file:line)
	logPattern = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)\](?:\[([^\]]*)\])? (.*)$`)
)

// LogHandler 日志查询处理器
type LogHandler struct {
	logDir string
}

// NewLogHandler 创建日志查询处理器
func NewLogHandler(logDir string) *LogHandler {
	return &LogHandler{
		logDir: logDir,
	}
}

// allowedLogFiles 是 log_file 查询参数允许的取值白名单。
//
// 必须做白名单校验：此前直接 filepath.Join(h.logDir, logFile+".log")，
// 而 filepath.Join 会 Clean 路径，?log_file=../../app/x 即可读到容器内
// 任意 *.log（含其他服务的日志）。白名单同时约束了读取、下载与清空三个动作。
var allowedLogFiles = map[string]bool{
	"backend":  true,
	"frontend": true,
}

// resolveLogPath 解析并校验 log_file 查询参数。
// 返回 (文件名, 绝对路径, 是否合法)；非法取值时 ok 为 false，调用方应回 400。
func (h *LogHandler) resolveLogPath(c *gin.Context) (string, string, bool) {
	name := c.DefaultQuery("log_file", "backend")
	if !allowedLogFiles[name] {
		return "", "", false
	}
	return name, filepath.Join(h.logDir, name+".log"), true
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Module    string `json:"module"`
	Message   string `json:"message"`
	Fields    string `json:"fields"`
	Location  string `json:"location"`
	Raw       string `json:"raw"`
}

// LogQueryParams 日志查询参数
type LogQueryParams struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=500"`
	Level    string `form:"level" binding:"omitempty,oneof=DEBUG INFO WARN ERROR FATAL"`
	Module   string `form:"module"`
	Keyword  string `form:"keyword"`
	Start    string `form:"start"` // 开始时间 2006/01/02 15:04:05
	End      string `form:"end"`   // 结束时间
	LogFile  string `form:"log_file" binding:"omitempty,oneof=backend frontend"`
}

// GetLogs 获取日志列表
// @Summary 获取日志列表
// @Description 获取系统日志，支持分页、级别筛选、关键词搜索
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(100)
// @Param level query string false "日志级别(DEBUG/INFO/WARN/ERROR/FATAL)"
// @Param module query string false "模块名称"
// @Param keyword query string false "关键词搜索"
// @Param start query string false "开始时间(2006/01/02 15:04:05)"
// @Param end query string false "结束时间"
// @Param log_file query string false "日志文件(backend/frontend)" default(backend)
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/logs [get]
func (h *LogHandler) GetLogs(c *gin.Context) {
	var params LogQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		dto.BadRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = 100
	}
	if params.LogFile == "" {
		params.LogFile = "backend"
	}

	// 构建日志文件路径
	logFile := filepath.Join(h.logDir, params.LogFile+".log")

	// 检查文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		dto.SuccessResponse(c, gin.H{
			"items":     []LogEntry{},
			"total":     0,
			"page":      params.Page,
			"page_size": params.PageSize,
		})
		return
	}

	// 读取并解析日志
	entries, total, err := h.readAndFilterLogs(logFile, params)
	if err != nil {
		logHandlerLog.Error("读取日志失败: %v", err)
		dto.InternalServerErrorResponse(c, "读取日志失败")
		return
	}

	dto.SuccessResponse(c, gin.H{
		"items":     entries,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
	})
}

// GetLogFiles 获取可用的日志文件列表
// @Summary 获取日志文件列表
// @Description 获取系统中可用的日志文件
// @Tags 日志管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/logs/files [get]
func (h *LogHandler) GetLogFiles(c *gin.Context) {
	files := []gin.H{}

	// 检查 backend.log
	backendLog := filepath.Join(h.logDir, "backend.log")
	if info, err := os.Stat(backendLog); err == nil {
		files = append(files, gin.H{
			"name":         "backend",
			"display_name": "后端日志",
			"size":         info.Size(),
			"modified_at":  info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// 检查 frontend.log
	frontendLog := filepath.Join(h.logDir, "frontend.log")
	if info, err := os.Stat(frontendLog); err == nil {
		files = append(files, gin.H{
			"name":         "frontend",
			"display_name": "前端日志",
			"size":         info.Size(),
			"modified_at":  info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	dto.SuccessResponse(c, files)
}

// GetLogStats 获取日志统计信息
// @Summary 获取日志统计
// @Description 获取日志级别分布统计
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param log_file query string false "日志文件(backend/frontend)" default(backend)
// @Success 200 {object} response.Response
// @Router /api/v1/logs/stats [get]
func (h *LogHandler) GetLogStats(c *gin.Context) {
	_, logPath, ok := h.resolveLogPath(c)
	if !ok {
		dto.BadRequestResponse(c, "非法的日志文件参数")
		return
	}

	// 使用类型化计数器，避免每行 gin.H 类型断言开销
	var counts struct {
		debug, info, warn, err, fatal, total int
	}

	file, err := os.Open(logPath)
	if err != nil {
		dto.SuccessResponse(c, gin.H{
			"debug": 0, "info": 0, "warn": 0, "error": 0, "fatal": 0, "total": 0,
		})
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		counts.total++

		if strings.Contains(line, "[DEBUG]") {
			counts.debug++
		} else if strings.Contains(line, "[INFO]") {
			counts.info++
		} else if strings.Contains(line, "[WARN]") {
			counts.warn++
		} else if strings.Contains(line, "[ERROR]") {
			counts.err++
		} else if strings.Contains(line, "[FATAL]") {
			counts.fatal++
		}
	}

	dto.SuccessResponse(c, gin.H{
		"debug": counts.debug,
		"info":  counts.info,
		"warn":  counts.warn,
		"error": counts.err,
		"fatal": counts.fatal,
		"total": counts.total,
	})
}

// DownloadLog 下载日志文件
// @Summary 下载日志文件
// @Description 下载指定的日志文件
// @Tags 日志管理
// @Produce octet-stream
// @Param log_file query string false "日志文件(backend/frontend)" default(backend)
// @Success 200 {file} file
// @Failure 404 {object} response.Response
// @Router /api/v1/logs/download [get]
func (h *LogHandler) DownloadLog(c *gin.Context) {
	logFile, logPath, ok := h.resolveLogPath(c)
	if !ok {
		dto.BadRequestResponse(c, "非法的日志文件参数")
		return
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		dto.NotFoundResponse(c, "日志文件不存在")
		return
	}

	fileName := logFile + "_" + time.Now().Format("20060102_150405") + ".log"
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.File(logPath)
}

// readAndFilterLogs 读取并过滤日志（流式 ring buffer 方案，内存 O(page*page_size)）
// 正向扫描日志文件，只保留最近 page*page_size 条匹配记录，避免全量加载到内存。
func (h *LogHandler) readAndFilterLogs(logPath string, params LogQueryParams) ([]LogEntry, int, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// ring buffer 容量：只保留到达目标页所需的条目数
	// page 1 需要最后 page_size 条，page N 需要最后 N*page_size 条
	capacity := params.Page * params.PageSize
	if capacity > 10000 {
		capacity = 10000 // 安全上限，防止恶意大页码
	}

	ring := make([]LogEntry, capacity)
	head := 0  // 下一次写入位置
	full := false
	total := 0 // 匹配的总条数

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := h.parseLine(line)

		if !h.matchFilters(entry, params) {
			continue
		}

		total++
		ring[head] = entry
		head++
		if head >= capacity {
			head = 0
			full = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	// 将 ring buffer 展开为有序切片（文件顺序：旧→新）
	var ordered []LogEntry
	if full {
		ordered = make([]LogEntry, capacity)
		// head 指向最旧的元素，从 head 开始读
		copy(ordered, ring[head:])
		copy(ordered[capacity-head:], ring[:head])
	} else {
		ordered = ring[:head]
	}

	orderedLen := len(ordered)

	// 计算目标页在 ordered 中的范围
	// page P, page_size S: 在 total 中的 forward 索引为 [total-P*S, total-(P-1)*S)
	startIdx := total - params.Page*params.PageSize
	endIdx := total - (params.Page-1)*params.PageSize

	// 转换为 ordered 切片索引
	offset := total - orderedLen
	orderedStart := startIdx - offset
	orderedEnd := endIdx - offset

	if orderedEnd < 0 || orderedStart >= orderedLen {
		return []LogEntry{}, total, nil
	}
	if orderedStart < 0 {
		orderedStart = 0
	}
	if orderedEnd > orderedLen {
		orderedEnd = orderedLen
	}

	// 提取目标页并倒序（最新在前）
	pageEntries := make([]LogEntry, 0, orderedEnd-orderedStart)
	for i := orderedEnd - 1; i >= orderedStart; i-- {
		pageEntries = append(pageEntries, ordered[i])
	}

	return pageEntries, total, nil
}

// stripAnsiCodes 移除 ANSI 颜色代码
func stripAnsiCodes(s string) string {
	// ansiPattern 已提升为包级变量，只编译一次
	return ansiPattern.ReplaceAllString(s, "")
}

// parseLine 解析单行日志
func (h *LogHandler) parseLine(line string) LogEntry {
	// 先移除 ANSI 颜色代码
	cleanLine := stripAnsiCodes(line)
	entry := LogEntry{Raw: cleanLine}

	matches := logPattern.FindStringSubmatch(cleanLine)
	if len(matches) >= 5 {
		entry.Timestamp = matches[1]
		entry.Level = matches[2]
		entry.Module = matches[3]

		// 解析消息和字段
		rest := matches[4]
		if idx := strings.Index(rest, " |"); idx != -1 {
			entry.Message = strings.TrimSpace(rest[:idx])
			remaining := rest[idx+2:]
			if locIdx := strings.LastIndex(remaining, " ("); locIdx != -1 {
				entry.Fields = strings.TrimSpace(remaining[:locIdx])
				entry.Location = strings.Trim(remaining[locIdx+2:], " ()")
			} else {
				entry.Fields = strings.TrimSpace(remaining)
			}
		} else if locIdx := strings.LastIndex(rest, " ("); locIdx != -1 {
			entry.Message = strings.TrimSpace(rest[:locIdx])
			entry.Location = strings.Trim(rest[locIdx+2:], " ()")
		} else {
			entry.Message = strings.TrimSpace(rest)
		}
	} else {
		// 无法解析的行，作为原始消息
		entry.Message = cleanLine
	}

	return entry
}

// matchFilters 检查日志条目是否匹配过滤条件
func (h *LogHandler) matchFilters(entry LogEntry, params LogQueryParams) bool {
	// 级别过滤
	if params.Level != "" && entry.Level != params.Level {
		return false
	}

	// 模块过滤
	if params.Module != "" && !strings.Contains(strings.ToLower(entry.Module), strings.ToLower(params.Module)) {
		return false
	}

	// 关键词搜索
	if params.Keyword != "" {
		keyword := strings.ToLower(params.Keyword)
		if !strings.Contains(strings.ToLower(entry.Message), keyword) &&
			!strings.Contains(strings.ToLower(entry.Fields), keyword) &&
			!strings.Contains(strings.ToLower(entry.Raw), keyword) {
			return false
		}
	}

	// 时间范围过滤
	if params.Start != "" || params.End != "" {
		entryTime, err := time.Parse("2006/01/02 15:04:05", entry.Timestamp)
		if err != nil {
			return true // 无法解析时间的条目默认包含
		}

		if params.Start != "" {
			startTime, err := time.Parse("2006/01/02 15:04:05", params.Start)
			if err == nil && entryTime.Before(startTime) {
				return false
			}
		}

		if params.End != "" {
			endTime, err := time.Parse("2006/01/02 15:04:05", params.End)
			if err == nil && entryTime.After(endTime) {
				return false
			}
		}
	}

	return true
}

// ClearLogs 清空日志文件
// @Summary 清空日志文件
// @Description 清空指定的日志文件内容
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param log_file query string false "日志文件(backend/frontend)" default(backend)
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/logs/clear [post]
func (h *LogHandler) ClearLogs(c *gin.Context) {
	logFile, logPath, ok := h.resolveLogPath(c)
	if !ok {
		dto.BadRequestResponse(c, "非法的日志文件参数")
		return
	}

	// backend 使用 lumberjack 管理句柄，需经 logger 安全清空
	if logFile == "backend" {
		if err := logger.ClearBackendLog(); err != nil {
			// 文件日志未启用时回退到路径 truncate
			if trErr := os.Truncate(logPath, 0); trErr != nil {
				if os.IsNotExist(trErr) {
					dto.SuccessResponse(c, gin.H{"message": "日志文件不存在"})
					return
				}
				logHandlerLog.Error("清空日志失败: %v (clear: %v)", trErr, err)
				dto.InternalServerErrorResponse(c, "清空日志失败")
				return
			}
		}
		logHandlerLog.Info("日志文件已清空: %s", logFile)
		dto.SuccessResponse(c, gin.H{"message": "日志已清空"})
		return
	}

	if err := os.Truncate(logPath, 0); err != nil {
		if os.IsNotExist(err) {
			dto.SuccessResponse(c, gin.H{"message": "日志文件不存在"})
			return
		}
		logHandlerLog.Error("清空日志失败: %v", err)
		dto.InternalServerErrorResponse(c, "清空日志失败")
		return
	}

	logHandlerLog.Info("日志文件已清空: %s", logFile)
	dto.SuccessResponse(c, gin.H{"message": "日志已清空"})
}

// GetLogTail 获取日志尾部（实时日志）
// @Summary 获取最新日志
// @Description 获取日志文件最后N行
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param log_file query string false "日志文件(backend/frontend)" default(backend)
// @Param lines query int false "行数" default(100)
// @Success 200 {object} response.Response
// @Router /api/v1/logs/tail [get]
func (h *LogHandler) GetLogTail(c *gin.Context) {
	_, logPath, ok := h.resolveLogPath(c)
	if !ok {
		dto.BadRequestResponse(c, "非法的日志文件参数")
		return
	}

	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "100"))
	if lines <= 0 || lines > 1000 {
		lines = 100
	}

	file, err := os.Open(logPath)
	if err != nil {
		dto.SuccessResponse(c, gin.H{"lines": []string{}})
		return
	}
	defer file.Close()

	// Ring buffer: 只保留最后 N 行，内存 O(lines) 而非 O(n)
	ring := make([]string, lines)
	head := 0
	full := false
	total := 0

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		ring[head] = scanner.Text()
		head++
		if head >= lines {
			head = 0
			full = true
		}
		total++
	}

	// 展开 ring buffer 为有序切片（旧→新）
	var result []string
	if full {
		result = make([]string, lines)
		copy(result, ring[head:])
		copy(result[lines-head:], ring[:head])
	} else {
		result = ring[:head]
	}

	dto.SuccessResponse(c, gin.H{
		"lines": result,
		"total": total,
	})
}
