package repository

import "testing"

// TestIsASCIIQuery 校验查询字符集判定。
//
// 这个判定决定搜索走哪条路径：
//   - 纯 ASCII  → 先试 'english' 全文搜索（有 idx_emails_fulltext_en 索引）
//   - 含非 ASCII → 直接走 ILIKE（由 GIN trigram 索引加速）
//
// 背景：'english' 文本搜索配置对中文整段只产生一个 token，
// 实测「财报」在 english 全文搜索下命中数为 0，每次都要白跑一次
// 全表 tsvector 计算再降级。这个判定就是为了省掉那次无用功。
func TestIsASCIIQuery(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{"纯英文", "quarterly report", true},
		{"纯数字", "2026", true},
		{"英文标点", "hello, world!", true},
		{"空串", "", true},
		{"邮箱地址", "alice@example.com", true},
		{"中文", "财报", false},
		{"中英混合", "Q3 财报", false},
		{"日文", "四半期報告", false},
		{"韩文", "분기 보고서", false},
		{"西里尔", "отчёт", false},
		{"emoji", "report 📊", false},
		{"全角标点", "报告（Q3）", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isASCIIQuery(c.query); got != c.want {
				t.Errorf("isASCIIQuery(%q) = %v，期望 %v", c.query, got, c.want)
			}
		})
	}
}
