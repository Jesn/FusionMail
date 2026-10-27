package response

// LogEntry 日志条目（API 契约视图）。
//
// 自 handler 包迁移至此：GetLogs 的响应 data 里直接包含 []LogEntry，
// dto/response 需要引用它来声明 Swagger 契约，而 handler 包依赖 dto/response，
// 因此类型必须放在本包。字段与 json tag 与原 handler.LogEntry 完全一致。
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Module    string `json:"module"`
	Message   string `json:"message"`
	Fields    string `json:"fields"`
	Location  string `json:"location"`
	Raw       string `json:"raw"`
}

// LogListResponse 日志列表响应。
//
// 替换 handler 里的 gin.H（键稳定：items/page/page_size/total）。
// 字段按字母序排列，与 map 序列化（encoding/json 对 map key 排序）的
// 输出逐键一致。
type LogListResponse struct {
	Items    []LogEntry `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

// LogFileResponse 单个日志文件信息。
//
// 替换 handler 里的 gin.H（键稳定：display_name/modified_at/name/size），
// 字段按字母序排列以保持输出一致。
type LogFileResponse struct {
	DisplayName string `json:"display_name"`
	ModifiedAt  string `json:"modified_at"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
}

// LogStatsResponse 日志级别分布统计。
//
// 替换 handler 里的 gin.H（键稳定：debug/error/fatal/info/total/warn），
// 字段按字母序排列以保持输出一致。
type LogStatsResponse struct {
	Debug int `json:"debug"`
	Error int `json:"error"`
	Fatal int `json:"fatal"`
	Info  int `json:"info"`
	Total int `json:"total"`
	Warn  int `json:"warn"`
}

// LogTailResponse 日志尾部内容。
//
// 替换 handler 里的 gin.H（键稳定：lines/total），
// 字段按字母序排列以保持输出一致。
type LogTailResponse struct {
	Lines []string `json:"lines"`
	Total int      `json:"total"`
}
