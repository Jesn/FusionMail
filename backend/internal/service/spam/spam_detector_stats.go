package spam

import (
	"context"
	"fmt"
	"time"

	"fusionmail/internal/model"
)

// BatchDetect 批量检测邮件
func (s *SpamDetector) BatchDetect(ctx context.Context, emails []*model.Email) ([]*SpamDetectionResult, error) {
	results := make([]*SpamDetectionResult, len(emails))

	// 简单的串行处理，后续可以优化为并发处理
	for i, email := range emails {
		result, err := s.Detect(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to detect email %d: %w", email.ID, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetDetectionStats 获取检测统计信息
func (s *SpamDetector) GetDetectionStats(ctx context.Context, startTime, endTime time.Time) (*DetectionStats, error) {
	// 从日志表查询统计信息
	logs, err := s.logRepo.FindByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	stats := &DetectionStats{
		TotalChecked: int64(len(logs)),
		SpamCount:    0,
		HamCount:     0,
		AvgScore:     0,
		AvgCheckTime: 0,
	}

	totalScore := 0.0
	for _, log := range logs {
		if log.IsSpam {
			stats.SpamCount++
		} else {
			stats.HamCount++
		}
		totalScore += log.FinalScore
	}

	if stats.TotalChecked > 0 {
		stats.AvgScore = totalScore / float64(stats.TotalChecked)
		stats.SpamRate = float64(stats.SpamCount) / float64(stats.TotalChecked) * 100
	}

	return stats, nil
}

// DetectionStats 检测统计信息
type DetectionStats struct {
	TotalChecked int64   `json:"total_checked"`  // 总检测数
	SpamCount    int64   `json:"spam_count"`     // 垃圾邮件数
	HamCount     int64   `json:"ham_count"`      // 正常邮件数
	SpamRate     float64 `json:"spam_rate"`      // 垃圾邮件率
	AvgScore     float64 `json:"avg_score"`      // 平均评分
	AvgCheckTime float64 `json:"avg_check_time"` // 平均检测耗时（毫秒）
}
