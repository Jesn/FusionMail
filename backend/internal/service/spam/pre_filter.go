package spam

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// PreFilter 预过滤器
type PreFilter struct {
	rblChecker       *RBLChecker
	behaviorAnalyzer *BehaviorAnalyzer
}

// PreFilterResult 预过滤结果
type PreFilterResult struct {
	TotalScore     int             // 总评分
	RBLResult      *RBLResult      // RBL 检查结果
	BehaviorResult *BehaviorResult // 行为分析结果
	ProcessingTime time.Duration   // 处理时间
	Details        []string        // 详细信息
}

// EmailData 邮件数据
type EmailData struct {
	From            string            // 发件人邮箱
	SenderIP        string            // 发件人 IP（可选）
	Size            int64             // 邮件大小
	AttachmentTypes []string          // 附件类型列表
	Headers         map[string]string // 邮件头（可选）
}

// NewPreFilter 创建预过滤器实例
func NewPreFilter(rblChecker *RBLChecker, behaviorAnalyzer *BehaviorAnalyzer) *PreFilter {
	return &PreFilter{
		rblChecker:       rblChecker,
		behaviorAnalyzer: behaviorAnalyzer,
	}
}

// Filter 执行预过滤检测
func (p *PreFilter) Filter(ctx context.Context, email *EmailData) (*PreFilterResult, error) {
	startTime := time.Now()

	result := &PreFilterResult{
		TotalScore:     0,
		RBLResult:      nil,
		BehaviorResult: nil,
		ProcessingTime: 0,
		Details:        []string{},
	}

	// 1. RBL 检查（IP 和域名）
	rblScore := 0
	if p.rblChecker != nil {
		rblResult, err := p.checkRBL(ctx, email)
		if err == nil && rblResult != nil {
			result.RBLResult = rblResult
			rblScore = rblResult.Score

			if rblResult.IsListed {
				detail := fmt.Sprintf("RBL 黑名单检测: 命中 %d 个列表 (+%d 分)",
					len(rblResult.Lists), rblResult.Score)
				result.Details = append(result.Details, detail)
			}
		}
	}

	// 2. 行为分析
	behaviorScore := 0
	if p.behaviorAnalyzer != nil {
		behaviorInfo := &EmailInfo{
			From:            email.From,
			Size:            email.Size,
			AttachmentTypes: email.AttachmentTypes,
		}

		behaviorResult, err := p.behaviorAnalyzer.Analyze(ctx, behaviorInfo)
		if err == nil && behaviorResult != nil {
			result.BehaviorResult = behaviorResult
			behaviorScore = behaviorResult.TotalScore

			// 添加详细信息
			if behaviorResult.IsFrequencyAbnormal {
				detail := fmt.Sprintf("发信频率异常: 5分钟内超过20封 (+%d 分)",
					behaviorResult.FrequencyScore)
				result.Details = append(result.Details, detail)
			}

			if behaviorResult.HasDangerousAttach {
				detail := fmt.Sprintf("危险附件检测: 包含可执行文件 (+%d 分)",
					behaviorResult.AttachmentScore)
				result.Details = append(result.Details, detail)
			}

			if behaviorResult.IsLargeEmail && behaviorResult.HasDangerousAttach {
				result.Details = append(result.Details, "大邮件 + 危险附件组合")
			}
		}
	}

	// 3. 计算总评分
	result.TotalScore = rblScore + behaviorScore

	// 4. 记录处理时间
	result.ProcessingTime = time.Since(startTime)

	// 5. 检查性能要求（应在 50ms 内完成）
	if result.ProcessingTime > 50*time.Millisecond {
		log.Printf("性能警告: 预过滤处理时间超过阈值 [耗时: %v, 阈值: 50ms, 发件人: %s]",
			result.ProcessingTime, email.From)
	}

	return result, nil
}

// checkRBL 执行 RBL 检查
func (p *PreFilter) checkRBL(ctx context.Context, email *EmailData) (*RBLResult, error) {
	var ipResult *RBLResult
	var domainResult *RBLResult

	// 1. 检查发件人 IP（如果提供）
	if email.SenderIP != "" {
		result, err := p.rblChecker.CheckIP(ctx, email.SenderIP)
		if err == nil {
			ipResult = result
		}
	} else if email.Headers != nil {
		// 尝试从邮件头提取 IP
		ip := ExtractIPFromEmail(email.Headers)
		if ip != "" {
			result, err := p.rblChecker.CheckIP(ctx, ip)
			if err == nil {
				ipResult = result
			}
		}
	}

	// 2. 检查发件人域名
	domain := extractDomainFromEmail(email.From)
	if domain != "" {
		result, err := p.rblChecker.CheckDomain(ctx, domain)
		if err == nil {
			domainResult = result
		}
	}

	// 3. 合并结果（取评分较高的）
	if ipResult != nil && domainResult != nil {
		if ipResult.Score >= domainResult.Score {
			return ipResult, nil
		}
		return domainResult, nil
	}

	if ipResult != nil {
		return ipResult, nil
	}

	if domainResult != nil {
		return domainResult, nil
	}

	// 没有结果，返回默认值
	return &RBLResult{
		IsListed:  false,
		Lists:     []string{},
		Score:     0,
		CheckedAt: time.Now(),
		FromCache: false,
	}, nil
}

// extractDomainFromEmail 从邮箱地址提取域名
func extractDomainFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parts[1]))
}

// GetPerformanceStats 获取性能统计（用于监控）
func (p *PreFilterResult) GetPerformanceStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_score":     p.TotalScore,
		"processing_time": p.ProcessingTime.Milliseconds(),
		"within_target":   p.ProcessingTime <= 50*time.Millisecond,
	}

	if p.RBLResult != nil {
		stats["rbl_score"] = p.RBLResult.Score
		stats["rbl_listed"] = p.RBLResult.IsListed
		stats["rbl_from_cache"] = p.RBLResult.FromCache
	}

	if p.BehaviorResult != nil {
		stats["behavior_score"] = p.BehaviorResult.TotalScore
		stats["frequency_abnormal"] = p.BehaviorResult.IsFrequencyAbnormal
		stats["has_dangerous_attach"] = p.BehaviorResult.HasDangerousAttach
	}

	return stats
}

// IsSpam 判断是否为垃圾邮件（基于预过滤评分）
// 注意：这只是预过滤层的判断，最终判断需要综合所有层级
func (p *PreFilterResult) IsSpam(threshold int) bool {
	return p.TotalScore >= threshold
}

// GetSummary 获取检测摘要
func (p *PreFilterResult) GetSummary() string {
	if len(p.Details) == 0 {
		return "预过滤检测: 未发现异常"
	}
	return fmt.Sprintf("预过滤检测: %s", strings.Join(p.Details, "; "))
}
