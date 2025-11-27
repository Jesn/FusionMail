package spam

import (
	"context"
	"encoding/json"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/jbrukh/bayesian"
)

// 贝叶斯分类类别
const (
	Spam bayesian.Class = "Spam"
	Ham  bayesian.Class = "Ham"
)

// 训练触发阈值
const (
	MinTrainingDataCount = 50  // 最小训练数据数量
	MinSpamCount         = 10  // 最小垃圾邮件数量
	MinHamCount          = 10  // 最小正常邮件数量
	SpamProbThreshold    = 0.8 // 垃圾邮件概率阈值
	HamProbThreshold     = 0.8 // 正常邮件概率阈值
)

// BayesianClassifier 贝叶斯分类器
type BayesianClassifier struct {
	trainingRepo repository.BayesianTrainingRepository
	classifiers  map[string]*bayesian.Classifier // 用户 UID -> 分类器
	modelStatus  map[string]*ModelStatus         // 用户 UID -> 模型状态
	mu           sync.RWMutex
}

// ModelStatus 模型状态
type ModelStatus struct {
	UserUID       string    `json:"user_uid"`
	IsTrained     bool      `json:"is_trained"`      // 是否已训练
	TrainingCount int64     `json:"training_count"`  // 训练数据数量
	SpamCount     int64     `json:"spam_count"`      // 垃圾邮件数量
	HamCount      int64     `json:"ham_count"`       // 正常邮件数量
	LastTrainedAt time.Time `json:"last_trained_at"` // 最后训练时间
	Accuracy      float64   `json:"accuracy"`        // 准确率（估计值）
}

// ClassificationResult 分类结果
type ClassificationResult struct {
	IsSpam      bool    `json:"is_spam"`     // 是否为垃圾邮件
	SpamProb    float64 `json:"spam_prob"`   // 垃圾邮件概率
	HamProb     float64 `json:"ham_prob"`    // 正常邮件概率
	Score       int     `json:"score"`       // 评分调整值
	Confidence  float64 `json:"confidence"`  // 置信度
	ModelUsed   bool    `json:"model_used"`  // 是否使用了模型
	Description string  `json:"description"` // 描述
}

// NewBayesianClassifier 创建贝叶斯分类器
func NewBayesianClassifier(trainingRepo repository.BayesianTrainingRepository) *BayesianClassifier {
	return &BayesianClassifier{
		trainingRepo: trainingRepo,
		classifiers:  make(map[string]*bayesian.Classifier),
		modelStatus:  make(map[string]*ModelStatus),
	}
}

// Classify 对邮件进行贝叶斯分类
func (b *BayesianClassifier) Classify(ctx context.Context, userUID string, email *model.Email) (*ClassificationResult, error) {
	result := &ClassificationResult{
		IsSpam:      false,
		SpamProb:    0.5,
		HamProb:     0.5,
		Score:       0,
		Confidence:  0,
		ModelUsed:   false,
		Description: "贝叶斯模型未训练",
	}

	// 获取用户的分类器
	classifier, exists := b.getClassifier(userUID)
	if !exists {
		// 尝试加载或训练模型
		if err := b.loadOrTrainModel(ctx, userUID); err != nil {
			return result, nil // 模型不可用，返回默认结果
		}
		classifier, exists = b.getClassifier(userUID)
		if !exists {
			return result, nil
		}
	}

	// 检查模型是否已训练
	status := b.getModelStatus(userUID)
	if status == nil || !status.IsTrained {
		result.Description = "贝叶斯模型训练数据不足"
		return result, nil
	}

	// 提取邮件特征词
	tokens := b.extractTokens(email)
	if len(tokens) == 0 {
		result.Description = "无法提取邮件特征词"
		return result, nil
	}

	// 执行分类
	scores, _, _ := classifier.LogScores(tokens)
	probs, _, _ := classifier.ProbScores(tokens)

	// 获取概率
	if len(probs) >= 2 {
		result.SpamProb = probs[0] // Spam 类别的概率
		result.HamProb = probs[1]  // Ham 类别的概率
	}

	result.ModelUsed = true

	// 根据概率计算评分调整
	if result.SpamProb > SpamProbThreshold {
		// 高概率垃圾邮件，增加评分
		result.IsSpam = true
		result.Score = 20
		result.Confidence = result.SpamProb
		result.Description = fmt.Sprintf("贝叶斯分类: 垃圾邮件概率 %.2f%%", result.SpamProb*100)
	} else if result.HamProb > HamProbThreshold {
		// 高概率正常邮件，降低评分
		result.IsSpam = false
		result.Score = -10
		result.Confidence = result.HamProb
		result.Description = fmt.Sprintf("贝叶斯分类: 正常邮件概率 %.2f%%", result.HamProb*100)
	} else {
		// 不确定，不调整评分
		result.Confidence = 0.5
		result.Description = fmt.Sprintf("贝叶斯分类: 不确定 (垃圾: %.2f%%, 正常: %.2f%%)", result.SpamProb*100, result.HamProb*100)
	}

	_ = scores // 避免未使用变量警告

	return result, nil
}

// Train 训练贝叶斯模型
func (b *BayesianClassifier) Train(ctx context.Context, userUID string) error {
	// 获取用户的训练数据
	trainings, err := b.trainingRepo.FindByUser(ctx, userUID)
	if err != nil {
		return fmt.Errorf("获取训练数据失败: %w", err)
	}

	// 检查训练数据数量
	if len(trainings) < MinTrainingDataCount {
		return fmt.Errorf("训练数据不足，需要至少 %d 条，当前 %d 条", MinTrainingDataCount, len(trainings))
	}

	// 统计垃圾邮件和正常邮件数量
	var spamCount, hamCount int64
	for _, t := range trainings {
		if t.IsSpam {
			spamCount++
		} else {
			hamCount++
		}
	}

	if spamCount < MinSpamCount || hamCount < MinHamCount {
		return fmt.Errorf("训练数据不平衡，需要至少 %d 封垃圾邮件和 %d 封正常邮件", MinSpamCount, MinHamCount)
	}

	// 创建新的分类器
	classifier := bayesian.NewClassifier(Spam, Ham)

	// 训练模型
	for _, training := range trainings {
		var tokens []string
		if err := json.Unmarshal([]byte(training.Tokens), &tokens); err != nil {
			continue // 跳过无效数据
		}

		if training.IsSpam {
			classifier.Learn(tokens, Spam)
		} else {
			classifier.Learn(tokens, Ham)
		}
	}

	// 保存分类器
	b.mu.Lock()
	b.classifiers[userUID] = classifier
	b.modelStatus[userUID] = &ModelStatus{
		UserUID:       userUID,
		IsTrained:     true,
		TrainingCount: int64(len(trainings)),
		SpamCount:     spamCount,
		HamCount:      hamCount,
		LastTrainedAt: time.Now(),
		Accuracy:      0.85, // 估计值，实际需要通过交叉验证计算
	}
	b.mu.Unlock()

	return nil
}

// AddTrainingData 添加训练数据
func (b *BayesianClassifier) AddTrainingData(ctx context.Context, userUID string, email *model.Email, isSpam bool) error {
	// 提取特征词
	tokens := b.extractTokens(email)
	if len(tokens) == 0 {
		return fmt.Errorf("无法提取邮件特征词")
	}

	// 序列化特征词
	tokensJSON, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("序列化特征词失败: %w", err)
	}

	// 创建训练数据记录
	training := &model.BayesianTraining{
		UserUID: userUID,
		EmailID: fmt.Sprintf("%d", email.ID),
		IsSpam:  isSpam,
		Tokens:  string(tokensJSON),
	}

	if err := b.trainingRepo.Create(ctx, training); err != nil {
		return fmt.Errorf("保存训练数据失败: %w", err)
	}

	// 检查是否需要自动训练
	count, err := b.trainingRepo.CountByUser(ctx, userUID)
	if err == nil && count >= MinTrainingDataCount {
		// 检查是否满足训练条件
		spamCount, _ := b.trainingRepo.CountByUserAndType(ctx, userUID, true)
		hamCount, _ := b.trainingRepo.CountByUserAndType(ctx, userUID, false)

		if spamCount >= MinSpamCount && hamCount >= MinHamCount {
			// 异步触发训练
			go func() {
				if err := b.Train(context.Background(), userUID); err != nil {
					fmt.Printf("自动训练贝叶斯模型失败 [%s]: %v\n", userUID, err)
				} else {
					fmt.Printf("自动训练贝叶斯模型成功 [%s]\n", userUID)
				}
			}()
		}
	}

	return nil
}

// Reset 重置用户的贝叶斯模型
func (b *BayesianClassifier) Reset(ctx context.Context, userUID string) error {
	// 删除训练数据
	if err := b.trainingRepo.DeleteByUser(ctx, userUID); err != nil {
		return fmt.Errorf("删除训练数据失败: %w", err)
	}

	// 清除内存中的分类器
	b.mu.Lock()
	delete(b.classifiers, userUID)
	delete(b.modelStatus, userUID)
	b.mu.Unlock()

	return nil
}

// GetModelStatus 获取模型状态
func (b *BayesianClassifier) GetModelStatus(ctx context.Context, userUID string) (*ModelStatus, error) {
	// 先检查内存中的状态
	status := b.getModelStatus(userUID)
	if status != nil {
		return status, nil
	}

	// 从数据库获取训练数据统计
	count, err := b.trainingRepo.CountByUser(ctx, userUID)
	if err != nil {
		return nil, fmt.Errorf("获取训练数据统计失败: %w", err)
	}

	spamCount, _ := b.trainingRepo.CountByUserAndType(ctx, userUID, true)
	hamCount, _ := b.trainingRepo.CountByUserAndType(ctx, userUID, false)

	status = &ModelStatus{
		UserUID:       userUID,
		IsTrained:     false,
		TrainingCount: count,
		SpamCount:     spamCount,
		HamCount:      hamCount,
	}

	// 检查是否满足训练条件
	if count >= MinTrainingDataCount && spamCount >= MinSpamCount && hamCount >= MinHamCount {
		// 尝试训练模型
		if err := b.Train(ctx, userUID); err == nil {
			status.IsTrained = true
			status.LastTrainedAt = time.Now()
		}
	}

	return status, nil
}

// loadOrTrainModel 加载或训练模型
func (b *BayesianClassifier) loadOrTrainModel(ctx context.Context, userUID string) error {
	// 检查训练数据数量
	count, err := b.trainingRepo.CountByUser(ctx, userUID)
	if err != nil {
		return err
	}

	if count < MinTrainingDataCount {
		return fmt.Errorf("训练数据不足")
	}

	// 尝试训练模型
	return b.Train(ctx, userUID)
}

// getClassifier 获取用户的分类器
func (b *BayesianClassifier) getClassifier(userUID string) (*bayesian.Classifier, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	classifier, exists := b.classifiers[userUID]
	return classifier, exists
}

// getModelStatus 获取模型状态（内存）
func (b *BayesianClassifier) getModelStatus(userUID string) *ModelStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.modelStatus[userUID]
}

// extractTokens 从邮件中提取特征词
func (b *BayesianClassifier) extractTokens(email *model.Email) []string {
	var tokens []string
	seen := make(map[string]bool)

	// 提取主题中的词
	subjectTokens := b.tokenize(email.Subject)
	for _, token := range subjectTokens {
		if !seen[token] {
			tokens = append(tokens, token)
			seen[token] = true
		}
	}

	// 提取正文中的词
	bodyTokens := b.tokenize(email.TextBody)
	for _, token := range bodyTokens {
		if !seen[token] {
			tokens = append(tokens, token)
			seen[token] = true
		}
	}

	// 提取发件人特征
	if email.FromAddress != "" {
		// 提取发件人域名
		parts := strings.Split(email.FromAddress, "@")
		if len(parts) == 2 {
			tokens = append(tokens, "FROM_DOMAIN:"+strings.ToLower(parts[1]))
		}
	}

	return tokens
}

// tokenize 分词
func (b *BayesianClassifier) tokenize(text string) []string {
	if text == "" {
		return nil
	}

	var tokens []string

	// 转换为小写
	text = strings.ToLower(text)

	// 移除 HTML 标签
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	text = htmlTagRegex.ReplaceAllString(text, " ")

	// 移除 URL
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlRegex.FindAllString(text, -1)
	for _, url := range urls {
		// 提取 URL 域名作为特征
		if strings.Contains(url, "://") {
			parts := strings.Split(url, "://")
			if len(parts) > 1 {
				domainParts := strings.Split(parts[1], "/")
				if len(domainParts) > 0 {
					tokens = append(tokens, "URL_DOMAIN:"+domainParts[0])
				}
			}
		}
	}
	text = urlRegex.ReplaceAllString(text, " ")

	// 分词（支持中英文）
	words := b.splitWords(text)

	// 过滤停用词和短词
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) >= 2 && !b.isStopWord(word) {
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// splitWords 分词（支持中英文混合）
func (b *BayesianClassifier) splitWords(text string) []string {
	var words []string
	var currentWord strings.Builder
	var currentChinese strings.Builder

	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			// 中文字符，每个字作为一个词
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			currentChinese.WriteRune(r)
			// 中文采用 bigram
			if currentChinese.Len() >= 6 { // 2个中文字符 = 6 bytes
				words = append(words, currentChinese.String())
				// 保留最后一个字符用于下一个 bigram
				str := currentChinese.String()
				runes := []rune(str)
				currentChinese.Reset()
				if len(runes) > 0 {
					currentChinese.WriteRune(runes[len(runes)-1])
				}
			}
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// 英文字母或数字
			if currentChinese.Len() > 0 {
				words = append(words, currentChinese.String())
				currentChinese.Reset()
			}
			currentWord.WriteRune(r)
		} else {
			// 其他字符（空格、标点等）
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			if currentChinese.Len() > 0 {
				words = append(words, currentChinese.String())
				currentChinese.Reset()
			}
		}
	}

	// 处理剩余的词
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}
	if currentChinese.Len() > 0 {
		words = append(words, currentChinese.String())
	}

	return words
}

// isStopWord 检查是否为停用词
func (b *BayesianClassifier) isStopWord(word string) bool {
	// 英文停用词
	englishStopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"this": true, "that": true, "these": true, "those": true,
		"it": true, "its": true, "i": true, "you": true, "he": true,
		"she": true, "we": true, "they": true, "me": true, "him": true,
		"her": true, "us": true, "them": true, "my": true, "your": true,
		"his": true, "our": true, "their": true, "what": true, "which": true,
		"who": true, "whom": true, "when": true, "where": true, "why": true,
		"how": true, "all": true, "each": true, "every": true, "both": true,
		"few": true, "more": true, "most": true, "other": true, "some": true,
		"such": true, "no": true, "not": true, "only": true, "same": true,
		"so": true, "than": true, "too": true, "very": true, "just": true,
		"can": true, "now": true, "also": true, "as": true, "if": true,
	}

	// 中文停用词
	chineseStopWords := map[string]bool{
		"的": true, "了": true, "是": true, "在": true, "和": true,
		"有": true, "我": true, "你": true, "他": true, "她": true,
		"它": true, "们": true, "这": true, "那": true, "就": true,
		"也": true, "都": true, "而": true, "及": true, "与": true,
		"或": true, "但": true, "如": true, "果": true, "因": true,
		"为": true, "所": true, "以": true, "于": true, "从": true,
		"到": true, "把": true, "被": true, "让": true, "给": true,
		"对": true, "向": true, "着": true, "过": true, "等": true,
		"很": true, "更": true, "最": true, "非": true, "不": true,
		"没": true, "无": true, "只": true, "还": true, "又": true,
		"再": true, "已": true, "将": true, "会": true, "能": true,
		"可": true, "要": true, "应": true, "该": true, "请": true,
		"您": true, "吗": true, "呢": true, "吧": true, "啊": true,
		"哦": true, "嗯": true, "哈": true, "呵": true, "嘿": true,
	}

	return englishStopWords[word] || chineseStopWords[word]
}

// GetTrainingStats 获取训练统计信息
func (b *BayesianClassifier) GetTrainingStats(ctx context.Context, userUID string) (*TrainingStats, error) {
	count, err := b.trainingRepo.CountByUser(ctx, userUID)
	if err != nil {
		return nil, err
	}

	spamCount, _ := b.trainingRepo.CountByUserAndType(ctx, userUID, true)
	hamCount, _ := b.trainingRepo.CountByUserAndType(ctx, userUID, false)

	status := b.getModelStatus(userUID)
	isTrained := status != nil && status.IsTrained

	return &TrainingStats{
		TotalCount:       count,
		SpamCount:        spamCount,
		HamCount:         hamCount,
		IsTrained:        isTrained,
		MinRequired:      MinTrainingDataCount,
		MinSpamRequired:  MinSpamCount,
		MinHamRequired:   MinHamCount,
		CanTrain:         count >= MinTrainingDataCount && spamCount >= MinSpamCount && hamCount >= MinHamCount,
		TrainingProgress: b.calculateTrainingProgress(count, spamCount, hamCount),
	}, nil
}

// TrainingStats 训练统计信息
type TrainingStats struct {
	TotalCount       int64   `json:"total_count"`
	SpamCount        int64   `json:"spam_count"`
	HamCount         int64   `json:"ham_count"`
	IsTrained        bool    `json:"is_trained"`
	MinRequired      int64   `json:"min_required"`
	MinSpamRequired  int64   `json:"min_spam_required"`
	MinHamRequired   int64   `json:"min_ham_required"`
	CanTrain         bool    `json:"can_train"`
	TrainingProgress float64 `json:"training_progress"` // 0-100
}

// calculateTrainingProgress 计算训练进度
func (b *BayesianClassifier) calculateTrainingProgress(total, spam, ham int64) float64 {
	// 计算各项进度
	totalProgress := float64(total) / float64(MinTrainingDataCount) * 100
	spamProgress := float64(spam) / float64(MinSpamCount) * 100
	hamProgress := float64(ham) / float64(MinHamCount) * 100

	// 取最小进度作为整体进度
	progress := totalProgress
	if spamProgress < progress {
		progress = spamProgress
	}
	if hamProgress < progress {
		progress = hamProgress
	}

	if progress > 100 {
		progress = 100
	}

	return progress
}
