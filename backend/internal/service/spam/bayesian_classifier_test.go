package spam

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/internal/testutil/pgtest"
)

func TestMain(m *testing.M) {
	pgtest.Init("spam")
	os.Exit(m.Run())
}

var (
	spamTokens = []string{"free", "winner", "prize", "claim", "lottery"}
	hamTokens  = []string{"meeting", "report", "project", "agenda", "deadline"}
)

// newClassifierTestEnv 构造接真实训练数据仓库的分类器。
func newClassifierTestEnv(t *testing.T) (*BayesianClassifier, repository.BayesianTrainingRepository) {
	t.Helper()

	db := pgtest.DB(t)
	pgtest.Truncate(t)
	repo := repository.NewBayesianTrainingRepository(db)
	return NewBayesianClassifier(repo), repo
}

// seedTraining 直接写训练样本（绕过 AddTrainingData 的自动训练 goroutine，保证可重复）。
func seedTraining(t *testing.T, repo repository.BayesianTrainingRepository, uid string, isSpam bool, count int, tokens []string) {
	t.Helper()

	for i := 0; i < count; i++ {
		raw, err := json.Marshal(tokens)
		if err != nil {
			t.Fatalf("序列化特征词失败: %v", err)
		}
		training := &model.BayesianTraining{
			UserUID: uid,
			EmailID: fmt.Sprintf("%s-%d", tokens[0], i),
			IsSpam:  isSpam,
			Tokens:  string(raw),
		}
		if err := repo.Create(context.Background(), training); err != nil {
			t.Fatalf("写入训练样本失败: %v", err)
		}
	}
}

func classifierTestEmail(subject, body, from string) *model.Email {
	return &model.Email{
		Subject:     subject,
		TextBody:    body,
		FromAddress: from,
	}
}

// TestClassifyWithoutTrainingDataIsNeutral 覆盖冷启动：
// 没有训练数据时必须返回中性结果（不可判定为垃圾），且不报错。
func TestClassifyWithoutTrainingDataIsNeutral(t *testing.T) {
	classifier, _ := newClassifierTestEnv(t)
	ctx := context.Background()

	result, err := classifier.Classify(ctx, "user-cold", classifierTestEmail("free winner prize", "claim your lottery", "spam@example.com"))
	if err != nil {
		t.Fatalf("冷启动分类不应报错: %v", err)
	}
	if result.IsSpam {
		t.Error("模型未经训练却把邮件判成垃圾邮件")
	}
	if result.ModelUsed {
		t.Error("冷启动应标记 ModelUsed=false")
	}
	if result.SpamProb != 0.5 || result.HamProb != 0.5 {
		t.Errorf("冷启动概率 = %.2f/%.2f，期望 0.5/0.5", result.SpamProb, result.HamProb)
	}
	if result.Score != 0 {
		t.Errorf("冷启动不应调整评分，实际 %d", result.Score)
	}
}

// TestClassifyWithInsufficientDataStaysNeutral 覆盖训练量不足：
// 样本数低于阈值时不训练模型，仍返回中性结果（避免小样本过拟合）。
func TestClassifyWithInsufficientDataStaysNeutral(t *testing.T) {
	classifier, repo := newClassifierTestEnv(t)
	ctx := context.Background()

	seedTraining(t, repo, "user-few", true, 5, spamTokens)
	seedTraining(t, repo, "user-few", false, 5, hamTokens)

	result, err := classifier.Classify(ctx, "user-few", classifierTestEmail("free winner", "claim prize", "spam@example.com"))
	if err != nil {
		t.Fatalf("数据不足时分类不应报错: %v", err)
	}
	if result.ModelUsed || result.IsSpam {
		t.Errorf("样本不足却启用了模型: %+v", result)
	}

	status, err := classifier.GetModelStatus(ctx, "user-few")
	if err != nil {
		t.Fatalf("读取模型状态失败: %v", err)
	}
	if status.IsTrained {
		t.Error("样本不足却被标记为已训练")
	}
	if status.TrainingCount != 10 || status.SpamCount != 5 || status.HamCount != 5 {
		t.Errorf("模型状态计数 = %+v，期望 10/5/5", status)
	}
}

// TestTrainRejectsInsufficientOrImbalancedData 覆盖训练前置条件：
// 总量不足、类别不平衡都必须拒绝，且不能留下「已训练」状态。
func TestTrainRejectsInsufficientOrImbalancedData(t *testing.T) {
	classifier, repo := newClassifierTestEnv(t)
	ctx := context.Background()

	if err := classifier.Train(ctx, "user-empty"); err == nil {
		t.Error("无训练数据时 Train 应报错")
	}

	seedTraining(t, repo, "user-imbalanced", true, 45, spamTokens)
	seedTraining(t, repo, "user-imbalanced", false, 5, hamTokens)

	err := classifier.Train(ctx, "user-imbalanced")
	if err == nil {
		t.Fatal("类别不平衡时 Train 应报错")
	}
	if want := "训练数据不平衡"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q，期望包含 %q", err.Error(), want)
	}

	status, err := classifier.GetModelStatus(ctx, "user-imbalanced")
	if err != nil {
		t.Fatalf("读取模型状态失败: %v", err)
	}
	if status.IsTrained {
		t.Error("训练失败后模型被标记为已训练")
	}
}

// TestTrainAndClassifyDistinguishesSpamFromHam 覆盖端到端判定：
// 训练达标后，垃圾特征邮件应判为垃圾、正常特征邮件应判为正常。
func TestTrainAndClassifyDistinguishesSpamFromHam(t *testing.T) {
	classifier, repo := newClassifierTestEnv(t)
	ctx := context.Background()
	uid := "user-trained"

	seedTraining(t, repo, uid, true, 25, spamTokens)
	seedTraining(t, repo, uid, false, 25, hamTokens)

	if err := classifier.Train(ctx, uid); err != nil {
		t.Fatalf("训练失败: %v", err)
	}

	status, err := classifier.GetModelStatus(ctx, uid)
	if err != nil {
		t.Fatalf("读取模型状态失败: %v", err)
	}
	if !status.IsTrained {
		t.Fatal("训练成功后模型状态仍为未训练")
	}
	if status.TrainingCount != 50 || status.SpamCount != 25 || status.HamCount != 25 {
		t.Errorf("训练状态计数 = %+v，期望 50/25/25", status)
	}
	if status.LastTrainedAt.IsZero() {
		t.Error("训练时间未记录")
	}

	spamResult, err := classifier.Classify(ctx, uid, classifierTestEmail("free winner prize", "claim your lottery prize", "promo@example.com"))
	if err != nil {
		t.Fatalf("分类垃圾邮件失败: %v", err)
	}
	if !spamResult.IsSpam {
		t.Errorf("垃圾特征邮件未被判定为垃圾: %+v", spamResult)
	}
	if !spamResult.ModelUsed || spamResult.SpamProb <= SpamProbThreshold {
		t.Errorf("垃圾邮件概率 = %.2f，期望超过阈值 %.2f", spamResult.SpamProb, SpamProbThreshold)
	}
	if spamResult.Score <= 0 {
		t.Errorf("判为垃圾时评分调整 = %d，期望为正", spamResult.Score)
	}

	hamResult, err := classifier.Classify(ctx, uid, classifierTestEmail("project meeting agenda", "report deadline for the project", "colleague@example.com"))
	if err != nil {
		t.Fatalf("分类正常邮件失败: %v", err)
	}
	if hamResult.IsSpam {
		t.Errorf("正常特征邮件被误判为垃圾: %+v", hamResult)
	}
	if !hamResult.ModelUsed || hamResult.HamProb <= HamProbThreshold {
		t.Errorf("正常邮件概率 = %.2f，期望超过阈值 %.2f", hamResult.HamProb, HamProbThreshold)
	}
	if hamResult.Score >= 0 {
		t.Errorf("判为正常时评分调整 = %d，期望为负", hamResult.Score)
	}
}

// TestClassifyEdgeCases 覆盖分类边界：无特征词、纯 HTML、只有 URL。
func TestClassifyEdgeCases(t *testing.T) {
	classifier, repo := newClassifierTestEnv(t)
	ctx := context.Background()
	uid := "user-edge"

	seedTraining(t, repo, uid, true, 25, spamTokens)
	seedTraining(t, repo, uid, false, 25, hamTokens)
	if err := classifier.Train(ctx, uid); err != nil {
		t.Fatalf("训练失败: %v", err)
	}

	cases := []struct {
		name  string
		email *model.Email
	}{
		{name: "空邮件", email: &model.Email{}},
		{name: "纯 HTML", email: classifierTestEmail("<div></div>", "<p></p>", "")},
		{name: "只有 URL", email: classifierTestEmail("http://example.com/claim", "", "")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := classifier.Classify(ctx, uid, c.email)
			if err != nil {
				t.Fatalf("边界输入不应报错: %v", err)
			}
			if result == nil {
				t.Fatal("边界输入返回了 nil 结果")
			}
			if result.IsSpam {
				t.Errorf("特征不足时不应判为垃圾: %+v", result)
			}
		})
	}
}

// TestResetClearsTrainingDataAndModel 覆盖重置：
// 训练数据与内存模型都要清掉，否则「重置」后仍会用旧模型判定。
func TestResetClearsTrainingDataAndModel(t *testing.T) {
	classifier, repo := newClassifierTestEnv(t)
	ctx := context.Background()
	uid := "user-reset"

	seedTraining(t, repo, uid, true, 25, spamTokens)
	seedTraining(t, repo, uid, false, 25, hamTokens)
	if err := classifier.Train(ctx, uid); err != nil {
		t.Fatalf("训练失败: %v", err)
	}

	if err := classifier.Reset(ctx, uid); err != nil {
		t.Fatalf("重置失败: %v", err)
	}

	count, err := repo.CountByUser(ctx, uid)
	if err != nil {
		t.Fatalf("统计训练数据失败: %v", err)
	}
	if count != 0 {
		t.Errorf("重置后训练数据仍有 %d 条", count)
	}

	result, err := classifier.Classify(ctx, uid, classifierTestEmail("free winner prize", "claim lottery", "promo@example.com"))
	if err != nil {
		t.Fatalf("重置后分类不应报错: %v", err)
	}
	if result.ModelUsed || result.IsSpam {
		t.Errorf("重置后仍在用旧模型判定: %+v", result)
	}
}

// TestGetTrainingStatsProgress 覆盖训练进度：进度取各项最小值，避免「总量够了就显示可训练」。
func TestGetTrainingStatsProgress(t *testing.T) {
	_, repo := newClassifierTestEnv(t)
	classifier := NewBayesianClassifier(repo)
	ctx := context.Background()
	uid := "user-progress"

	seedTraining(t, repo, uid, true, 3, spamTokens)
	seedTraining(t, repo, uid, false, 2, hamTokens)

	stats, err := classifier.GetTrainingStats(ctx, uid)
	if err != nil {
		t.Fatalf("读取训练统计失败: %v", err)
	}
	if stats.TotalCount != 5 || stats.SpamCount != 3 || stats.HamCount != 2 {
		t.Errorf("统计计数 = %+v，期望 5/3/2", stats)
	}
	if stats.CanTrain {
		t.Error("样本远低于阈值却显示可训练")
	}
	if stats.IsTrained {
		t.Error("未训练却标记 IsTrained")
	}
	// 进度 = min(5/50, 3/10, 2/10) * 100 = 10%
	if math.Abs(stats.TrainingProgress-10) > 0.001 {
		t.Errorf("TrainingProgress = %.3f，期望 10", stats.TrainingProgress)
	}
	if stats.MinRequired != MinTrainingDataCount || stats.MinSpamRequired != MinSpamCount || stats.MinHamRequired != MinHamCount {
		t.Errorf("最小要求暴露错误: %+v", stats)
	}
}

// TestAddTrainingDataValidatesAndPersists 覆盖训练样本写入：
// 无特征词的邮件拒绝写入；正常邮件写入后可读回且特征词为合法 JSON。
func TestAddTrainingDataValidatesAndPersists(t *testing.T) {
	classifier, repo := newClassifierTestEnv(t)
	ctx := context.Background()
	uid := "user-add"

	empty := &model.Email{ID: 1}
	if err := classifier.AddTrainingData(ctx, uid, empty, true); err == nil {
		t.Error("无特征词的邮件应拒绝写入训练数据")
	}

	email := classifierTestEmail("free winner", "claim prize", "promo@example.com")
	email.ID = 42
	if err := classifier.AddTrainingData(ctx, uid, email, true); err != nil {
		t.Fatalf("写入训练数据失败: %v", err)
	}

	rows, err := repo.FindByUser(ctx, uid)
	if err != nil {
		t.Fatalf("读取训练数据失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("训练数据行数 = %d，期望 1", len(rows))
	}
	row := rows[0]
	if !row.IsSpam || row.EmailID != "42" {
		t.Errorf("训练数据字段错误: %+v", row)
	}

	var tokens []string
	if err := json.Unmarshal([]byte(row.Tokens), &tokens); err != nil {
		t.Fatalf("特征词不是合法 JSON: %q (%v)", row.Tokens, err)
	}
	if len(tokens) == 0 {
		t.Error("特征词为空")
	}
	if tokens[len(tokens)-1] == "" {
		t.Error("发件人域名特征缺失")
	}
}
