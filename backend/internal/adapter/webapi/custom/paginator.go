package custom

import (
	"fmt"
	"net/url"
	"strconv"

	"fusionmail/internal/model"
)

// Paginator 分页器接口
type Paginator interface {
	// GetFirstPageParams 获取第一页的参数
	GetFirstPageParams() url.Values

	// GetNextPageParams 根据当前响应获取下一页参数
	// 返回 nil 表示没有下一页
	GetNextPageParams(info *PaginationInfo, currentParams url.Values) url.Values

	// HasNextPage 检查是否有下一页
	HasNextPage(info *PaginationInfo) bool
}

// PaginatorFactory 分页器工厂
type PaginatorFactory struct{}

// NewPaginatorFactory 创建分页器工厂
func NewPaginatorFactory() *PaginatorFactory {
	return &PaginatorFactory{}
}

// CreatePaginator 根据配置创建分页器
func (f *PaginatorFactory) CreatePaginator(config *model.CustomWebAPIPagination) Paginator {
	if config == nil {
		// 默认使用 offset 分页
		return NewOffsetPaginator("offset", "limit", 50)
	}

	switch config.Type {
	case "cursor":
		return NewCursorPaginator(config.PageParam, config.LimitParam, config.PageSize)
	case "id_based":
		return NewIDBasedPaginator(config.PageParam, config.LimitParam, config.PageSize)
	case "page":
		return NewPagePaginator(config.PageParam, config.LimitParam, config.PageSize)
	default:
		// 默认 offset 分页
		return NewOffsetPaginator(config.PageParam, config.LimitParam, config.PageSize)
	}
}

// ============================================
// Offset 分页器
// ============================================

// OffsetPaginator offset/limit 分页器
type OffsetPaginator struct {
	offsetParam string
	limitParam  string
	pageSize    int
	offset      int
}

// NewOffsetPaginator 创建 offset 分页器
func NewOffsetPaginator(offsetParam, limitParam string, pageSize int) *OffsetPaginator {
	if offsetParam == "" {
		offsetParam = "offset"
	}
	if limitParam == "" {
		limitParam = "limit"
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	return &OffsetPaginator{
		offsetParam: offsetParam,
		limitParam:  limitParam,
		pageSize:    pageSize,
		offset:      0,
	}
}

// GetFirstPageParams 获取第一页参数
func (p *OffsetPaginator) GetFirstPageParams() url.Values {
	params := url.Values{}
	params.Set(p.offsetParam, "0")
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// GetNextPageParams 获取下一页参数
func (p *OffsetPaginator) GetNextPageParams(info *PaginationInfo, currentParams url.Values) url.Values {
	if !p.HasNextPage(info) {
		return nil
	}

	// 计算下一个 offset
	currentOffset := 0
	if offsetStr := currentParams.Get(p.offsetParam); offsetStr != "" {
		currentOffset, _ = strconv.Atoi(offsetStr)
	}

	nextOffset := currentOffset + p.pageSize

	params := url.Values{}
	params.Set(p.offsetParam, strconv.Itoa(nextOffset))
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// HasNextPage 检查是否有下一页
func (p *OffsetPaginator) HasNextPage(info *PaginationInfo) bool {
	if info == nil {
		return false
	}
	return info.HasMore
}

// ============================================
// Cursor 分页器
// ============================================

// CursorPaginator 游标分页器
type CursorPaginator struct {
	cursorParam string
	limitParam  string
	pageSize    int
}

// NewCursorPaginator 创建游标分页器
func NewCursorPaginator(cursorParam, limitParam string, pageSize int) *CursorPaginator {
	if cursorParam == "" {
		cursorParam = "cursor"
	}
	if limitParam == "" {
		limitParam = "limit"
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	return &CursorPaginator{
		cursorParam: cursorParam,
		limitParam:  limitParam,
		pageSize:    pageSize,
	}
}

// GetFirstPageParams 获取第一页参数
func (p *CursorPaginator) GetFirstPageParams() url.Values {
	params := url.Values{}
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// GetNextPageParams 获取下一页参数
func (p *CursorPaginator) GetNextPageParams(info *PaginationInfo, currentParams url.Values) url.Values {
	if !p.HasNextPage(info) {
		return nil
	}

	params := url.Values{}
	params.Set(p.cursorParam, info.NextCursor)
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// HasNextPage 检查是否有下一页
func (p *CursorPaginator) HasNextPage(info *PaginationInfo) bool {
	if info == nil {
		return false
	}
	return info.NextCursor != "" && info.HasMore
}

// ============================================
// ID-Based 分页器
// ============================================

// IDBasedPaginator 基于 ID 的分页器
type IDBasedPaginator struct {
	afterIDParam string
	limitParam   string
	pageSize     int
	lastID       string
}

// NewIDBasedPaginator 创建 ID 分页器
func NewIDBasedPaginator(afterIDParam, limitParam string, pageSize int) *IDBasedPaginator {
	if afterIDParam == "" {
		afterIDParam = "after_id"
	}
	if limitParam == "" {
		limitParam = "limit"
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	return &IDBasedPaginator{
		afterIDParam: afterIDParam,
		limitParam:   limitParam,
		pageSize:     pageSize,
	}
}

// GetFirstPageParams 获取第一页参数
func (p *IDBasedPaginator) GetFirstPageParams() url.Values {
	params := url.Values{}
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// GetNextPageParams 获取下一页参数
func (p *IDBasedPaginator) GetNextPageParams(info *PaginationInfo, currentParams url.Values) url.Values {
	if !p.HasNextPage(info) {
		return nil
	}

	// 使用 NextCursor 作为 lastID
	if info.NextCursor == "" {
		return nil
	}

	params := url.Values{}
	params.Set(p.afterIDParam, info.NextCursor)
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// HasNextPage 检查是否有下一页
func (p *IDBasedPaginator) HasNextPage(info *PaginationInfo) bool {
	if info == nil {
		return false
	}
	return info.HasMore && info.NextCursor != ""
}

// SetLastID 设置最后一个 ID（用于从邮件列表中提取）
func (p *IDBasedPaginator) SetLastID(lastID string) {
	p.lastID = lastID
}

// ============================================
// Page 分页器
// ============================================

// PagePaginator 页码分页器
type PagePaginator struct {
	pageParam  string
	limitParam string
	pageSize   int
}

// NewPagePaginator 创建页码分页器
func NewPagePaginator(pageParam, limitParam string, pageSize int) *PagePaginator {
	if pageParam == "" {
		pageParam = "page"
	}
	if limitParam == "" {
		limitParam = "page_size"
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	return &PagePaginator{
		pageParam:  pageParam,
		limitParam: limitParam,
		pageSize:   pageSize,
	}
}

// GetFirstPageParams 获取第一页参数
func (p *PagePaginator) GetFirstPageParams() url.Values {
	params := url.Values{}
	params.Set(p.pageParam, "1")
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// GetNextPageParams 获取下一页参数
func (p *PagePaginator) GetNextPageParams(info *PaginationInfo, currentParams url.Values) url.Values {
	if !p.HasNextPage(info) {
		return nil
	}

	// 计算下一页
	currentPage := 1
	if pageStr := currentParams.Get(p.pageParam); pageStr != "" {
		currentPage, _ = strconv.Atoi(pageStr)
	}

	params := url.Values{}
	params.Set(p.pageParam, strconv.Itoa(currentPage+1))
	params.Set(p.limitParam, strconv.Itoa(p.pageSize))
	return params
}

// HasNextPage 检查是否有下一页
func (p *PagePaginator) HasNextPage(info *PaginationInfo) bool {
	if info == nil {
		return false
	}

	// 如果有明确的 HasMore 标志
	if info.HasMore {
		return true
	}

	// 根据总数和当前页计算
	if info.Total > 0 && info.CurrentPage > 0 && info.PageSize > 0 {
		totalPages := (info.Total + info.PageSize - 1) / info.PageSize
		return info.CurrentPage < totalPages
	}

	return false
}

// ============================================
// 辅助函数
// ============================================

// BuildURL 构建带参数的 URL
func BuildURL(baseURL, endpoint string, params url.Values) string {
	u := baseURL + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return u
}

// MergeParams 合并参数
func MergeParams(base, override url.Values) url.Values {
	result := url.Values{}
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// GetPaginationType 获取分页类型描述
func GetPaginationType(paginationType string) string {
	switch paginationType {
	case "offset":
		return "Offset/Limit 分页"
	case "cursor":
		return "游标分页"
	case "id_based":
		return "ID 分页"
	case "page":
		return "页码分页"
	default:
		return fmt.Sprintf("未知分页类型: %s", paginationType)
	}
}
