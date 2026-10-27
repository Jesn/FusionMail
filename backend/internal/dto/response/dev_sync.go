package response

// DevSyncResult 开发环境单项配置同步结果。
//
// 与 dev_sync_handler 里原先反复内联的匿名结构体逐字段一致
// （字段名与 json tag 完全相同），仅命名化以便声明 Swagger 契约。
type DevSyncResult struct {
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	ClientID   string `json:"client_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	ClientIDDB int64  `json:"client_id_db"`
}

// DevSyncResponse 开发环境同步响应。
//
// 替换 handler 里的 gin.H（键稳定：message/results）。
type DevSyncResponse struct {
	Message string          `json:"message"`
	Results []DevSyncResult `json:"results"`
}
