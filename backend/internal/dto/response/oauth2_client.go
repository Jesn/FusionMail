package response

import (
	"fusionmail/internal/model"
)

// OAuth2ClientListResponse OAuth2 客户端分页列表响应。
//
// 替换 handler 里的 gin.H（键稳定：data/page/page_size/total/total_page），
// 字段按字母序排列，与 map 序列化的输出逐键一致。
type OAuth2ClientListResponse struct {
	Data      []model.OAuth2ClientResponse `json:"data"`
	Page      int                          `json:"page"`
	PageSize  int                          `json:"page_size"`
	Total     int64                        `json:"total"`
	TotalPage int64                        `json:"total_page"`
}
