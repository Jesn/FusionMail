package response

import "time"

// 垃圾邮件模块的响应 DTO（P16 swagger 注解补全）。
// 服务层统计结构（service.SpamStats / spam.ModelStatus / spam.TrainingStats）
// 不进 API 契约 —— 这里逐字段镜像，handler 构造时 JSON 输出一致。

// SpamMarkResponse 标记/取消标记垃圾邮件
type SpamMarkResponse struct {
	MarkedCount   int `json:"marked_count,omitempty"`
	UnmarkedCount int `json:"unmarked_count,omitempty"`
}

// SpamDeleteResponse 批量删除/清空垃圾箱
type SpamDeleteResponse struct {
	DeletedCount int64 `json:"deleted_count"`
}

// SpamStatsResponse 垃圾邮件统计（镜像 service.SpamStats）
type SpamStatsResponse struct {
	TotalCount   int64 `json:"total_count"`
	UnreadCount  int64 `json:"unread_count"`
	TodayCount   int64 `json:"today_count"`
	WeekCount    int64 `json:"week_count"`
	MonthCount   int64 `json:"month_count"`
	BlockedCount int64 `json:"blocked_count"`
}

// BayesianStatusResponse 贝叶斯模型状态（镜像 spam.ModelStatus）
type BayesianStatusResponse struct {
	UserUID       string    `json:"user_uid"`
	IsTrained     bool      `json:"is_trained"`
	TrainingCount int64     `json:"training_count"`
	SpamCount     int64     `json:"spam_count"`
	HamCount      int64     `json:"ham_count"`
	LastTrainedAt time.Time `json:"last_trained_at"`
	Accuracy      float64   `json:"accuracy"`
}

// BayesianTrainingStatsResponse 训练统计（镜像 spam.TrainingStats）
type BayesianTrainingStatsResponse struct {
	TotalCount       int64   `json:"total_count"`
	SpamCount        int64   `json:"spam_count"`
	HamCount         int64   `json:"ham_count"`
	IsTrained        bool    `json:"is_trained"`
	MinRequired      int64   `json:"min_required"`
	MinSpamRequired  int64   `json:"min_spam_required"`
	MinHamRequired   int64   `json:"min_ham_required"`
	CanTrain         bool    `json:"can_train"`
	TrainingProgress float64 `json:"training_progress"`
}

// RuleToggleResponse 规则切换结果
type RuleToggleResponse struct {
	ID      int64  `json:"id"`
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}
