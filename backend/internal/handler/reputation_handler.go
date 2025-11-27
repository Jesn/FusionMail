package handler

import (
	"strconv"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/service/spam"

	"github.com/gin-gonic/gin"
)

// ReputationHandler 发件人信誉处理器
type ReputationHandler struct {
	reputationManager *spam.ReputationManager
	reputationRepo    repository.SenderReputationRepository
}

// NewReputationHandler 创建发件人信誉处理器
func NewReputationHandler(reputationManager *spam.ReputationManager, reputationRepo repository.SenderReputationRepository) *ReputationHandler {
	return &ReputationHandler{
		reputationManager: reputationManager,
		reputationRepo:    reputationRepo,
	}
}

// GetSenderReputation 获取发件人信誉
// GET /api/v1/reputation/sender/:email
func (h *ReputationHandler) GetSenderReputation(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		dto.BadRequestResponse(c, "邮箱地址不能为空")
		return
	}

	// 获取发件人信誉统计
	stats, err := h.reputationManager.GetReputationStats(c.Request.Context(), email)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 转换为响应格式
	response := &dto.ReputationResponse{
		Email:       stats.Email,
		Domain:      stats.Domain,
		Score:       stats.Score,
		TrustLevel:  stats.TrustLevel,
		TotalEmails: stats.TotalEmails,
		SpamCount:   stats.SpamCount,
		HamCount:    stats.HamCount,
		SpamRate:    stats.SpamRate,
		RBLStatus:   stats.RBLStatus,
	}

	if stats.RBLCheckedAt != nil {
		checkedAt := stats.RBLCheckedAt.Format("2006-01-02T15:04:05Z07:00")
		response.RBLCheckedAt = &checkedAt
	}

	dto.SuccessResponse(c, response)
}

// UpdateReputation 更新发件人信誉
// POST /api/v1/reputation/update
func (h *ReputationHandler) UpdateReputation(c *gin.Context) {
	var req dto.UpdateReputationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, err.Error())
		return
	}

	// 根据用户反馈更新信誉
	if err := h.reputationManager.UpdateReputationByUserFeedback(c.Request.Context(), req.Email, req.IsSpam); err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 获取更新后的信誉
	stats, err := h.reputationManager.GetReputationStats(c.Request.Context(), req.Email)
	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.SuccessResponse(c, gin.H{
		"message":     "信誉更新成功",
		"email":       stats.Email,
		"new_score":   stats.Score,
		"trust_level": stats.TrustLevel,
	})
}

// GetReputationStats 获取发件人信誉统计
// GET /api/v1/reputation/stats
func (h *ReputationHandler) GetReputationStats(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	trustLevel := c.Query("trust_level")

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取信誉列表
	var reputations []*dto.ReputationResponse
	var total int64
	var err error

	if trustLevel != "" {
		// 按信任级别筛选
		reputations, total, err = h.getReputationsByTrustLevel(c, trustLevel, offset, pageSize)
	} else {
		// 获取所有信誉
		reputations, total, err = h.getAllReputations(c, offset, pageSize)
	}

	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	// 计算统计数据
	stats := h.calculateStats(c)

	dto.SuccessResponse(c, gin.H{
		"items":      reputations,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"statistics": stats,
	})
}

// getReputationsByTrustLevel 按信任级别获取信誉列表
func (h *ReputationHandler) getReputationsByTrustLevel(c *gin.Context, trustLevel string, offset, limit int) ([]*dto.ReputationResponse, int64, error) {
	reputations, total, err := h.reputationRepo.ListByTrustLevel(c.Request.Context(), trustLevel, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换为响应格式
	responses := make([]*dto.ReputationResponse, len(reputations))
	for i, rep := range reputations {
		responses[i] = h.convertToResponse(rep)
	}

	return responses, total, nil
}

// getAllReputations 获取所有信誉列表
func (h *ReputationHandler) getAllReputations(c *gin.Context, offset, limit int) ([]*dto.ReputationResponse, int64, error) {
	reputations, total, err := h.reputationRepo.List(c.Request.Context(), offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// 转换为响应格式
	responses := make([]*dto.ReputationResponse, len(reputations))
	for i, rep := range reputations {
		responses[i] = h.convertToResponse(rep)
	}

	return responses, total, nil
}

// convertToResponse 转换模型为响应格式
func (h *ReputationHandler) convertToResponse(rep *model.SenderReputation) *dto.ReputationResponse {
	response := &dto.ReputationResponse{
		Email:       rep.Email,
		Domain:      rep.Domain,
		Score:       rep.ReputationScore,
		TrustLevel:  rep.TrustLevel,
		TotalEmails: rep.TotalEmails,
		SpamCount:   rep.SpamCount,
		HamCount:    rep.HamCount,
		SpamRate:    0,
		RBLStatus:   rep.RBLStatus,
		CreatedAt:   rep.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   rep.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 计算垃圾邮件率
	if rep.TotalEmails > 0 {
		response.SpamRate = float64(rep.SpamCount) / float64(rep.TotalEmails) * 100
	}

	if rep.RBLCheckedAt != nil {
		checkedAt := rep.RBLCheckedAt.Format("2006-01-02T15:04:05Z07:00")
		response.RBLCheckedAt = &checkedAt
	}

	return response
}

// calculateStats 计算统计数据
func (h *ReputationHandler) calculateStats(c *gin.Context) *dto.ReputationStatsResponse {
	ctx := c.Request.Context()

	// 获取各信任级别的数量
	_, trustedCount, _ := h.reputationRepo.ListByTrustLevel(ctx, "trusted", 0, 1)
	_, neutralCount, _ := h.reputationRepo.ListByTrustLevel(ctx, "neutral", 0, 1)
	_, suspiciousCount, _ := h.reputationRepo.ListByTrustLevel(ctx, "suspicious", 0, 1)
	_, blockedCount, _ := h.reputationRepo.ListByTrustLevel(ctx, "blocked", 0, 1)

	totalSenders := trustedCount + neutralCount + suspiciousCount + blockedCount

	return &dto.ReputationStatsResponse{
		TotalSenders:    totalSenders,
		TrustedCount:    trustedCount,
		NeutralCount:    neutralCount,
		SuspiciousCount: suspiciousCount,
		BlockedCount:    blockedCount,
		AverageScore:    50.0, // 默认值
		RBLListedCount:  0,    // 需要额外查询
	}
}

// ListSenderReputations 获取发件人信誉列表
// GET /api/v1/reputation/list
func (h *ReputationHandler) ListSenderReputations(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	trustLevel := c.Query("trust_level")

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取信誉列表
	var reputations []*dto.ReputationResponse
	var total int64
	var err error

	if trustLevel != "" {
		reputations, total, err = h.getReputationsByTrustLevel(c, trustLevel, offset, pageSize)
	} else {
		reputations, total, err = h.getAllReputations(c, offset, pageSize)
	}

	if err != nil {
		dto.InternalServerErrorResponse(c, err.Error())
		return
	}

	dto.PaginatedSuccessResponse(c, reputations, total, page, pageSize)
}
