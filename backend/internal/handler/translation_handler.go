package handler

import (
	"fusionmail/internal/dto"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

// TranslationHandler exposes the backend translation proxy API.
type TranslationHandler struct {
	translationService service.TranslationService
}

// NewTranslationHandler creates a translation handler.
func NewTranslationHandler(translationService service.TranslationService) *TranslationHandler {
	return &TranslationHandler{translationService: translationService}
}

// Translate proxies a translation request to the configured translation service.
func (h *TranslationHandler) Translate(c *gin.Context) {
	var req service.TranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数格式错误: "+err.Error())
		return
	}

	result, err := h.translationService.Translate(c.Request.Context(), req)
	if err != nil {
		dto.HandleServiceError(c, err)
		return
	}

	dto.SuccessResponse(c, result)
}
