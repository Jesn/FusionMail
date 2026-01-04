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
	logFile := c.DefaultQuery("log_file", "backend")
	logPath := filepath.Join(h.logDir, logFile+".log")

	stats := gin.H{
		"debug": 0,
		"info":  0,
		"warn":  0,
		"error": 0,
		"fatal": 0,
		"total": 0,
	}

	file, err := os.Open(logPath)
	if err != nil {
		dto.SuccessResponse(c, stats)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		stats["total"] = stats["total"].(int) + 1

		if strings.Contains(line, "[DEBUG]") {
			stats["debug"] = stats["debug"].(int) + 1
		} else if strings.Contains(line, "[INFO]") {
			stats["info"] = stats["info"].(int) + 1
		} else if strings.Contains(line, "[WARN]") {
			stats["warn"] = stats["warn"].(int) + 1
		} else if strings.Contains(line, "[ERROR]") {
			stats["error"] = stats["error"].(int) + 1
		} else if strings.Contains(line, "[FATAL]") {
			stats["fatal"] = stats["fatal"].(int) + 1
		}
	}

	dto.SuccessResponse(c, stats)
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
	logFile := c.DefaultQuery("log_file", "backend")
	logPath := filepath.Join(h.logDir, logFile+".log")

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		dto.NotFoundResponse(c, "日志文件不存在")
		return
	}

	fileName := logFile + "_" + time.Now().Format("20060102_150405") + ".log"
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.File(logPath)
}

// readAndFilterLogs 读取并过滤日志
func (h *LogHandler) readAndFilterLogs(logPath string, params LogQueryParams) ([]LogEntry, int, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var allEntries []LogEntry
	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// 日志格式正则: 2006/01/02 15:04:05 [LEVEL][Module] message | fields (file:line)
	logPattern := regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)\](?:\[([^\]]*)\])? (.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := h.parseLine(line, logPattern)

		// 应用过滤条件
		if !h.matchFilters(entry, params) {
			continue
		}

		allEntries = append(allEntries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	// 倒序排列（最新的在前）
	for i, j := 0, len(allEntries)-1; i < j; i, j = i+1, j-1 {
		allEntries[i], allEntries[j] = allEntries[j], allEntries[i]
	}

	total := len(allEntries)

	// 分页
	start := (params.Page - 1) * params.PageSize
	end := start + params.PageSize
	if start >= total {
		return []LogEntry{}, total, nil
	}
	if end > total {
		end = total
	}

	return allEntries[start:end], total, nil
}

// stripAnsiCodes 移除 ANSI 颜色代码
func stripAnsiCodes(s string) string {
	// ANSI 转义序列正则: \x1b[...m 或 \033[...m
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiPattern.ReplaceAllString(s, "")
}

// parseLine 解析单行日志
func (h *LogHandler) parseLine(line string, pattern *regexp.Regexp) LogEntry {
	// 先移除 ANSI 颜色代码
	cleanLine := stripAnsiCodes(line)
	entry := LogEntry{Raw: cleanLine}

	matches := pattern.FindStringSubmatch(cleanLine)
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
	logFile := c.DefaultQuery("log_file", "backend")
	logPath := filepath.Join(h.logDir, logFile+".log")

	// 清空文件内容
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
	logFile := c.DefaultQuery("log_file", "backend")
	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "100"))
	if lines <= 0 || lines > 1000 {
		lines = 100
	}

	logPath := filepath.Join(h.logDir, logFile+".log")

	file, err := os.Open(logPath)
	if err != nil {
		dto.SuccessResponse(c, gin.H{"lines": []string{}})
		return
	}
	defer file.Close()

	// 读取所有行
	var allLines []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	// 取最后N行
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	dto.SuccessResponse(c, gin.H{
		"lines": allLines[start:],
		"total": len(allLines),
	})
}
