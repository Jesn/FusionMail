package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// AttachmentHandler 附件处理器
type AttachmentHandler struct {
	attachmentService *service.AttachmentService
}

// NewAttachmentHandler 创建附件处理器
func NewAttachmentHandler(attachmentService *service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentService: attachmentService,
	}
}

// GetAttachment 获取附件信息
// GET /api/v1/attachments/:id
func (h *AttachmentHandler) GetAttachment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的附件 ID",
		})
		return
	}

	attachment, err := h.attachmentService.GetAttachment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "附件不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    attachment,
	})
}

// DownloadAttachment 下载附件
// GET /api/v1/attachments/:id/download
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的附件 ID",
		})
		return
	}

	reader, attachment, err := h.attachmentService.DownloadAttachment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "附件不存在或下载失败",
		})
		return
	}
	defer reader.Close()

	// 设置响应头
	c.Header("Content-Type", attachment.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
	c.Header("Content-Length", fmt.Sprintf("%d", attachment.SizeBytes))

	// 流式传输文件内容
	if _, err := io.Copy(c.Writer, reader); err != nil {
		// 如果传输失败，记录错误（但此时已经开始发送响应，无法返回 JSON 错误）
		fmt.Printf("Error streaming attachment: %v\n", err)
	}
}

// DeleteAttachment 删除附件
// DELETE /api/v1/attachments/:id
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的附件 ID",
		})
		return
	}

	if err := h.attachmentService.DeleteAttachment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "删除附件失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "附件已删除",
	})
}

// GetEmailAttachments 获取邮件的所有附件
// GET /api/v1/emails/:id/attachments
func (h *AttachmentHandler) GetEmailAttachments(c *gin.Context) {
	emailID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的邮件 ID",
		})
		return
	}

	attachments, err := h.attachmentService.GetAttachmentsByEmailID(c.Request.Context(), emailID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取附件列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    attachments,
	})
}
