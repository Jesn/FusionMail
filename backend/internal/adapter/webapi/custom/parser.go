package custom

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/adapter/webapi"
	"fusionmail/internal/model"
)

// ResponseParser JSON 响应解析器
// 根据配置的字段映射从 JSON 响应中提取邮件数据
type ResponseParser struct {
	config *model.CustomWebAPIAuthData
}

// NewResponseParser 创建响应解析器
func NewResponseParser(config *model.CustomWebAPIAuthData) *ResponseParser {
	return &ResponseParser{config: config}
}

// ParseResponse 解析 JSON 响应，返回邮件列表
func (p *ResponseParser) ParseResponse(responseBody []byte) ([]*adapter.Email, error) {
	// 解析 JSON
	var rawData interface{}
	if err := json.Unmarshal(responseBody, &rawData); err != nil {
		return nil, webapi.WrapError(webapi.ErrCodeParseError, "解析 JSON 失败", err)
	}

	// 获取数据数组
	dataArray, err := p.extractDataArray(rawData)
	if err != nil {
		return nil, err
	}

	// 解析每封邮件
	emails := make([]*adapter.Email, 0, len(dataArray))
	for i, item := range dataArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		email, err := p.parseEmailItem(itemMap)
		if err != nil {
			// 记录错误但继续处理
			continue
		}

		// 设置目标邮箱（如果配置了）
		if p.config.TargetEmail != "" && len(email.ToAddresses) == 0 {
			email.ToAddresses = []string{p.config.TargetEmail}
		}

		emails = append(emails, email)

		// 防止解析过多
		if i >= 1000 {
			break
		}
	}

	return emails, nil
}

// extractDataArray 从响应中提取数据数组
func (p *ResponseParser) extractDataArray(data interface{}) ([]interface{}, error) {
	// 如果没有配置数据路径，假设响应本身就是数组
	if p.config.DataPath == "" {
		if arr, ok := data.([]interface{}); ok {
			return arr, nil
		}
		return nil, errors.New("响应不是数组格式")
	}

	// 按路径提取数据
	result := p.getValueByPath(data, p.config.DataPath)
	if result == nil {
		return nil, fmt.Errorf("数据路径 %s 不存在", p.config.DataPath)
	}

	arr, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("数据路径 %s 不是数组", p.config.DataPath)
	}

	return arr, nil
}

// parseEmailItem 解析单封邮件
func (p *ResponseParser) parseEmailItem(item map[string]interface{}) (*adapter.Email, error) {
	mapping := p.config.FieldMapping

	email := &adapter.Email{}

	// 必填字段
	email.ProviderID = p.getString(item, mapping.ID)
	if email.ProviderID == "" {
		return nil, errors.New("邮件 ID 为空")
	}

	email.Subject = p.getString(item, mapping.Subject)
	email.FromAddress = p.getString(item, mapping.From)

	// 收件人
	if mapping.To != "" {
		toValue := p.getValueByPath(item, mapping.To)
		email.ToAddresses = p.parseAddresses(toValue)
	}

	// 日期
	if mapping.Date != "" {
		dateValue := p.getString(item, mapping.Date)
		if dateValue != "" {
			email.ReceivedAt = p.parseDate(dateValue)
			email.SentAt = email.ReceivedAt
		}
	}

	// 可选字段
	if mapping.Body != "" {
		email.TextBody = p.getString(item, mapping.Body)
	}
	if mapping.HTMLBody != "" {
		email.HTMLBody = p.getString(item, mapping.HTMLBody)
	}

	// RFC822 原始内容
	if mapping.RawContent != "" {
		rawContent := p.getString(item, mapping.RawContent)
		if rawContent != "" {
			// 使用 RFC822 解析器解析
			parser := webapi.NewRFC822Parser()
			if parsed, err := parser.Parse(rawContent); err == nil {
				// 合并解析结果
				if email.Subject == "" {
					email.Subject = parsed.Subject
				}
				if email.FromAddress == "" {
					email.FromAddress = parsed.FromAddress
					email.FromName = parsed.FromName
				}
				if len(email.ToAddresses) == 0 {
					email.ToAddresses = parsed.ToAddresses
				}
				if email.TextBody == "" {
					email.TextBody = parsed.TextBody
				}
				if email.HTMLBody == "" {
					email.HTMLBody = parsed.HTMLBody
				}
				if email.ReceivedAt.IsZero() {
					email.ReceivedAt = parsed.ReceivedAt
					email.SentAt = parsed.SentAt
				}
				email.Attachments = parsed.Attachments
				email.HasAttachments = parsed.HasAttachments
				email.AttachmentsCount = parsed.AttachmentsCount
			}
		}
	}

	// 已读状态
	if mapping.IsRead != "" {
		isRead := p.getBool(item, mapping.IsRead)
		email.SourceIsRead = &isRead
	}

	// 目标邮箱（用于 Admin 模式分发）
	if mapping.TargetEmail != "" {
		targetEmail := p.getString(item, mapping.TargetEmail)
		if targetEmail != "" && len(email.ToAddresses) == 0 {
			email.ToAddresses = []string{targetEmail}
		}
	}

	// 生成摘要
	if email.Snippet == "" {
		email.Snippet = webapi.GenerateSnippet(email, 200)
	}

	return email, nil
}

// getValueByPath 按路径获取值（支持嵌套，如 data.list）
func (p *ResponseParser) getValueByPath(data interface{}, path string) interface{} {
	if path == "" {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if current == nil {
			return nil
		}

		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			// 尝试解析为索引
			if idx, err := strconv.Atoi(part); err == nil && idx >= 0 && idx < len(v) {
				current = v[idx]
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// getString 获取字符串值
func (p *ResponseParser) getString(item map[string]interface{}, path string) string {
	if path == "" {
		return ""
	}

	value := p.getValueByPath(item, path)
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getBool 获取布尔值
func (p *ResponseParser) getBool(item map[string]interface{}, path string) bool {
	if path == "" {
		return false
	}

	value := p.getValueByPath(item, path)
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1" || v == "yes"
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return false
	}
}

// parseAddresses 解析邮箱地址
func (p *ResponseParser) parseAddresses(value interface{}) []string {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		// 单个地址或逗号分隔
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			result := make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					result = append(result, part)
				}
			}
			return result
		}
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// parseDate 解析日期
func (p *ResponseParser) parseDate(dateStr string) time.Time {
	// 尝试多种日期格式
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02-Jan-2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// 尝试解析 Unix 时间戳
	if ts, err := strconv.ParseInt(dateStr, 10, 64); err == nil {
		if ts > 1e12 {
			// 毫秒时间戳
			return time.UnixMilli(ts)
		}
		return time.Unix(ts, 0)
	}

	return time.Time{}
}

// ParsePaginationInfo 解析分页信息
func (p *ResponseParser) ParsePaginationInfo(responseBody []byte) (*PaginationInfo, error) {
	var rawData map[string]interface{}
	if err := json.Unmarshal(responseBody, &rawData); err != nil {
		return nil, err
	}

	info := &PaginationInfo{}

	// 尝试提取常见的分页字段
	if total, ok := rawData["total"].(float64); ok {
		info.Total = int(total)
	}
	if hasMore, ok := rawData["has_more"].(bool); ok {
		info.HasMore = hasMore
	}
	if nextCursor, ok := rawData["next_cursor"].(string); ok {
		info.NextCursor = nextCursor
	}
	if page, ok := rawData["page"].(float64); ok {
		info.CurrentPage = int(page)
	}
	if pageSize, ok := rawData["page_size"].(float64); ok {
		info.PageSize = int(pageSize)
	}

	return info, nil
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Total       int    // 总数
	HasMore     bool   // 是否有更多
	NextCursor  string // 下一页游标
	CurrentPage int    // 当前页
	PageSize    int    // 每页数量
}
