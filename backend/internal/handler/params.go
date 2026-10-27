package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"fusionmail/internal/dto"
)

// parseIDParam 解析路径参数中的 int64 ID；失败时写入 400 响应并返回 false。
// 收敛自各 handler 重复的 "strconv.ParseInt + BadRequestResponse" 样板；
// errMsg 逐字保留各调用点原文案，不做统一改写。
func parseIDParam(c *gin.Context, param, errMsg string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, errMsg)
		return 0, false
	}
	return id, true
}

// parseIntParam 解析整数查询参数，无效或缺省时返回 defaultValue
// （自 system_handler.go 同位搬移）。
func parseIntParam(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// parseInt 解析整数（自 account_handler.go 同位搬移）。
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// parseInt64 解析 int64（自 account_handler.go 同位搬移）。
func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
