package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/logger"
)

// 模块日志记录器
var builtinRulesLog = logger.NewWithModule("BuiltinRules")

// BuiltinRulesInitializer 内置规则初始化器
type BuiltinRulesInitializer struct {
	repo repository.SpamRuleRepository
}

// NewBuiltinRulesInitializer 创建内置规则初始化器
func NewBuiltinRulesInitializer(repo repository.SpamRuleRepository) *BuiltinRulesInitializer {
	return &BuiltinRulesInitializer{repo: repo}
}

// Initialize 初始化内置规则
func (b *BuiltinRulesInitializer) Initialize(ctx context.Context) error {
	// 检查是否已经初始化
	count, err := b.repo.CountBuiltin(ctx)
	if err != nil {
		return fmt.Errorf("failed to check builtin rules: %w", err)
	}

	if count > 0 {
		builtinRulesLog.Info("内置规则已存在，跳过初始化: count=%d", count)
		return nil
	}

	builtinRulesLog.Info("开始初始化内置垃圾邮件规则...")

	// 获取所有内置规则
	rules := b.getBuiltinRules()

	// 批量插入
	for i, rule := range rules {
		if err := b.repo.Create(ctx, rule); err != nil {
			builtinRulesLog.Warn("创建规则失败: index=%d/%d, err=%v", i+1, len(rules), err)
			continue
		}
	}

	builtinRulesLog.Info("内置规则初始化完成: count=%d", len(rules))
	return nil
}

// getBuiltinRules 获取所有内置规则
func (b *BuiltinRulesInitializer) getBuiltinRules() []*model.SpamRule {
	rules := []*model.SpamRule{}

	// 1. 高风险关键词规则（中文）
	highRiskKeywordsCN := []string{
		"一夜暴富", "免费赚钱", "点击领取", "中奖通知", "紧急通知",
		"账户异常", "立即验证", "密码过期", "安全警告", "冻结账户",
		"免费试用", "限时抢购", "独家优惠", "赚钱秘籍", "投资回报",
		"贷款审批", "信用卡办理", "快速贷款", "无抵押贷款", "低息贷款",
		"增大增粗", "壮阳补肾", "减肥瘦身", "祛斑美白", "丰胸产品",
		"发票代开", "刻章办证", "代办文凭", "假证制作", "身份证复印",
		"博彩游戏", "在线赌场", "六合彩", "时时彩", "北京赛车",
		"色情服务", "上门服务", "特殊服务", "成人用品", "情趣用品",
		"法律诉讼", "法院传票", "律师函件", "诉讼通知", "强制执行",
	}

	for _, keyword := range highRiskKeywordsCN {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("高风险关键词: %s", keyword),
			Description: fmt.Sprintf("检测邮件中是否包含高风险关键词: %s", keyword),
			Category:    "keyword",
			Pattern:     keyword,
			Score:       20,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 2. 高风险关键词规则（英文）
	highRiskKeywordsEN := []string{
		"get rich quick", "make money fast", "click here now", "you won", "urgent action",
		"account suspended", "verify immediately", "password expired", "security alert", "frozen account",
		"free trial", "limited time", "exclusive offer", "investment opportunity", "guaranteed returns",
		"loan approved", "credit card offer", "quick loan", "no collateral", "low interest",
		"weight loss", "male enhancement", "viagra", "cialis", "pharmacy",
		"fake invoice", "diploma copy", "fake certificate", "identity theft", "social security",
		"online casino", "gambling", "lottery", "betting", "poker",
		"adult content", "dating service", "meet singles", "xxx", "porn",
		"legal action", "court notice", "lawsuit", "attorney", "legal proceedings",
	}

	for _, keyword := range highRiskKeywordsEN {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("High Risk Keyword: %s", keyword),
			Description: fmt.Sprintf("Detect high risk keyword: %s", keyword),
			Category:    "keyword",
			Pattern:     keyword,
			Score:       20,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 3. 中风险关键词规则（中文）
	mediumRiskKeywordsCN := []string{
		"限时优惠", "立即购买", "马上行动", "不容错过", "最后机会",
		"特价促销", "清仓大甩卖", "全场五折", "买一送一", "包邮到家",
		"健康养生", "美容护肤", "保健品", "营养补充", "中药调理",
		"投资理财", "股票推荐", "基金定投", "外汇交易", "数字货币",
		"兼职招聘", "在家办公", "轻松赚钱", "副业推荐", "创业项目",
		"学历提升", "职业培训", "技能认证", "考试包过", "快速拿证",
		"房产投资", "买房优惠", "租房信息", "二手房", "学区房",
		"汽车促销", "购车优惠", "二手车", "车辆保险", "违章代办",
	}

	for _, keyword := range mediumRiskKeywordsCN {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("中风险关键词: %s", keyword),
			Description: fmt.Sprintf("检测邮件中是否包含中风险关键词: %s", keyword),
			Category:    "keyword",
			Pattern:     keyword,
			Score:       10,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 4. 中风险关键词规则（英文）
	mediumRiskKeywordsEN := []string{
		"limited offer", "buy now", "act now", "don't miss", "last chance",
		"special promotion", "clearance sale", "50% off", "buy one get one", "free shipping",
		"health supplement", "beauty product", "skin care", "nutrition", "herbal medicine",
		"investment advice", "stock tips", "mutual fund", "forex trading", "cryptocurrency",
		"work from home", "part time job", "easy money", "side hustle", "business opportunity",
		"degree program", "training course", "certification", "exam guarantee", "quick diploma",
		"real estate", "property investment", "rental", "second hand", "school district",
		"car sale", "vehicle discount", "used car", "auto insurance", "traffic ticket",
	}

	for _, keyword := range mediumRiskKeywordsEN {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("Medium Risk Keyword: %s", keyword),
			Description: fmt.Sprintf("Detect medium risk keyword: %s", keyword),
			Category:    "keyword",
			Pattern:     keyword,
			Score:       10,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 5. 链接数量检测规则
	rules = append(rules, &model.SpamRule{
		Name:        "链接数量过多",
		Description: "检测邮件正文中链接数量是否超过 5 个",
		Category:    "url",
		Pattern:     "5",
		Score:       10,
		IsBuiltin:   true,
		Enabled:     true,
	})

	// 6. 短链接检测规则
	shortLinkDomains := []string{
		"bit.ly", "t.cn", "goo.gl", "ow.ly", "tinyurl.com",
		"is.gd", "buff.ly", "adf.ly", "bit.do", "lnkd.in",
		"db.tt", "qr.ae", "adf.ly", "cur.lv", "ity.im",
	}

	for _, domain := range shortLinkDomains {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("短链接检测: %s", domain),
			Description: fmt.Sprintf("检测邮件中是否包含短链接域名: %s", domain),
			Category:    "url",
			Pattern:     domain,
			Score:       15,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 7. 文本格式检测规则
	rules = append(rules, &model.SpamRule{
		Name:        "大写字母比例过高",
		Description: "检测邮件正文中大写字母比例是否超过 30%",
		Category:    "content",
		Pattern:     "0.3",
		Score:       10,
		IsBuiltin:   true,
		Enabled:     true,
	})

	rules = append(rules, &model.SpamRule{
		Name:        "特殊字符比例过高",
		Description: "检测邮件正文中特殊字符比例是否超过 20%",
		Category:    "content",
		Pattern:     "0.2",
		Score:       8,
		IsBuiltin:   true,
		Enabled:     true,
	})

	// 8. 可执行附件检测规则
	executableExtensions := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr",
		".vbs", ".js", ".jar", ".zip", ".rar", ".7z",
		".msi", ".dll", ".sys", ".drv", ".ocx",
	}

	for _, ext := range executableExtensions {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("可执行附件检测: %s", ext),
			Description: fmt.Sprintf("检测邮件是否包含可执行附件: %s", ext),
			Category:    "attachment",
			Pattern:     ext,
			Score:       25,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 9. 可疑发件人域名规则
	suspiciousDomains := []string{
		"@tempmail.com", "@guerrillamail.com", "@10minutemail.com",
		"@mailinator.com", "@throwaway.email", "@temp-mail.org",
		"@fakeinbox.com", "@trashmail.com", "@yopmail.com",
	}

	for _, domain := range suspiciousDomains {
		rules = append(rules, &model.SpamRule{
			Name:        fmt.Sprintf("可疑发件人域名: %s", domain),
			Description: fmt.Sprintf("检测发件人是否使用临时邮箱域名: %s", domain),
			Category:    "header",
			Pattern:     domain,
			Score:       15,
			IsBuiltin:   true,
			Enabled:     true,
		})
	}

	// 10. HTML 标签过多规则
	rules = append(rules, &model.SpamRule{
		Name:        "HTML 标签过多",
		Description: "检测邮件 HTML 正文中标签数量是否异常",
		Category:    "content",
		Pattern:     "100",
		Score:       8,
		IsBuiltin:   true,
		Enabled:     true,
	})

	// 11. 图片附件过多规则
	rules = append(rules, &model.SpamRule{
		Name:        "图片附件过多",
		Description: "检测邮件是否包含超过 10 个图片附件",
		Category:    "attachment",
		Pattern:     "10",
		Score:       8,
		IsBuiltin:   true,
		Enabled:     true,
	})

	return rules
}
