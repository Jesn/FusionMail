package model

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// jsonTagAllowlist 列出**允许**携带 json tag 的类型，以及允许的理由。
//
// 背景：父任务验收标准写的是「model 不含 json tag」，但审计后发现该标准
// 按字面执行会破坏线上数据 —— internal/model 里混着三类东西：
//
//  1. gorm 表模型（映射数据库表）
//  2. 以 JSON 列持久化的值对象（json tag 就是存储格式）
//  3. 尚未迁到 internal/dto 的请求/响应 DTO
//
// 第 2 类的 json tag 是**承重**的：删掉会改变已存数据的键名，
// 例如 RuleCondition 的 "field" 变成 "Field"，直接读不出历史规则。
//
// 因此本测试固化的是**精化后的规则**：
//
//	model 层不得定义 API 线格式 —— gorm 表模型一律不得携带 json tag；
//	只有「自身以 JSON 形式持久化」的值对象可以例外，且必须在此登记。
//
// API 线格式由 internal/dto/response 定义，另有
// TestNoGORMModelLeaksIntoAPIContract 守卫（断言 swagger.json 中无 gorm 模型）。
var jsonTagAllowlist = map[string]string{
	// —— 以 JSON 列持久化的值对象（json tag 即存储格式，不可删）——
	"SyncProgress":            "经 ToJSON/FromJSON 持久化到 email_accounts.sync_progress_json",
	"RuleCondition":           "经 RuleConditions.Scan/Value 持久化到 email_rules.conditions",
	"RuleAction":              "经 RuleActions.Scan/Value 持久化到 email_rules.actions",
	"SentEmailAttachmentInfo": "json.Marshal 后写入 sent_emails 的附件信息列",
	"SyncConfig":              "同步配置值对象，由 EmailAccount.GetSyncConfig 构造",

	// —— 已知债务：下列 DTO 应当迁到 internal/dto，尚未迁 ——
	"OAuth2ClientCreateRequest": "待迁至 internal/dto/request",
	"OAuth2ClientUpdateRequest": "待迁至 internal/dto/request",
	"OAuth2ClientResponse":      "待迁至 internal/dto/response",
	"ProviderResponse":          "待迁至 internal/dto/response",
	"ProviderAdapterResponse":   "待迁至 internal/dto/response",
	"AdapterResponse":           "待迁至 internal/dto/response",
	"AccountGroupWithCount":     "待迁至 internal/dto/response",
	"GroupListResponse":         "待迁至 internal/dto/response",
	"AccountGroupWithAccounts":  "待迁至 internal/dto/response",
}

// structInfo 描述一个 struct 类型声明中观察到的 tag 情况。
type structInfo struct {
	name     string
	hasGorm  bool
	hasJSON  bool
	jsonKeys []string
}

// scanModelStructs 解析 internal/model 下所有非测试文件里的 struct 声明。
func scanModelStructs(t *testing.T) []structInfo {
	t.Helper()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob 失败: %v", err)
	}

	fset := token.NewFileSet()
	var out []structInfo

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("解析 %s 失败: %v", file, err)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			info := structInfo{name: ts.Name.Name}
			for _, field := range st.Fields.List {
				if field.Tag == nil {
					continue
				}
				tag := strings.Trim(field.Tag.Value, "`")
				if strings.Contains(tag, "gorm:\"") {
					info.hasGorm = true
				}
				if idx := strings.Index(tag, "json:\""); idx >= 0 {
					info.hasJSON = true
					rest := tag[idx+len("json:\""):]
					if end := strings.Index(rest, "\""); end >= 0 {
						key := strings.Split(rest[:end], ",")[0]
						info.jsonKeys = append(info.jsonKeys, key)
					}
				}
			}
			if info.hasGorm || info.hasJSON {
				out = append(out, info)
			}
			return true
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// TestNoJSONTagsOnGormTableModels 守卫「gorm 表模型不得定义 API 线格式」。
//
// 这是 P6「分层边界强制」在 model 层的可执行契约：一旦有人给表模型加回
// json tag（等于让持久化模型重新定义线格式），本测试立刻失败。
func TestNoJSONTagsOnGormTableModels(t *testing.T) {
	for _, info := range scanModelStructs(t) {
		if !info.hasGorm || !info.hasJSON {
			continue
		}
		if _, allowed := jsonTagAllowlist[info.name]; allowed {
			continue
		}
		t.Errorf("gorm 表模型 %s 携带了 json tag（键: %s）——\n"+
			"表模型不得定义 API 线格式：请删除这些 json tag，让响应体走 internal/dto/response。\n"+
			"若该类型确实以 JSON 形式持久化，请把它的名字与理由加入 jsonTagAllowlist。",
			info.name, strings.Join(info.jsonKeys, ", "))
	}
}

// TestJSONTagAllowlistIsCurrent 防止白名单腐化。
//
// 白名单只应包含「确实带 json tag」的类型；若某条目的 json tag 已被删除
// （例如 DTO 终于迁到了 internal/dto），必须同步移除该条目，
// 否则白名单会变成掩盖问题的垃圾场。
func TestJSONTagAllowlistIsCurrent(t *testing.T) {
	actual := make(map[string]bool)
	for _, info := range scanModelStructs(t) {
		if info.hasJSON {
			actual[info.name] = true
		}
	}

	for name, reason := range jsonTagAllowlist {
		if !actual[name] {
			t.Errorf("jsonTagAllowlist 中的 %q 已不再携带 json tag（理由: %s）——\n"+
				"请从白名单中删除该条目，保持白名单与实际状态一致。", name, reason)
		}
	}
}

// TestModelPackageHasNoStrayJSONTags 断言 model 层不再有「未登记」的 json tag。
//
// 与 TestNoJSONTagsOnGormTableModels 互补：后者只盯表模型，
// 本测试盯住所有非表模型 —— 新增 DTO 若直接落在 model 包会立刻被发现。
func TestModelPackageHasNoStrayJSONTags(t *testing.T) {
	for _, info := range scanModelStructs(t) {
		if !info.hasJSON || info.hasGorm {
			continue
		}
		if _, allowed := jsonTagAllowlist[info.name]; allowed {
			continue
		}
		t.Errorf("model 包中的 %s 携带 json tag 但未登记（键: %s）——\n"+
			"请求/响应 DTO 应放在 internal/dto/request 或 internal/dto/response；\n"+
			"若确实需要留在 model，请把它的名字与理由加入 jsonTagAllowlist。",
			info.name, strings.Join(info.jsonKeys, ", "))
	}
}

// TestModelDirIsScannable 兜底：确保扫描确实读到了文件，
// 避免 glob 写错导致上面三个测试在空集合上「假通过」。
func TestModelDirIsScannable(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob 失败: %v", err)
	}
	nonTest := 0
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			nonTest++
		}
	}
	if nonTest < 10 {
		t.Fatalf("只扫到 %d 个非测试文件，疑似工作目录不对（期望在 internal/model 下运行）", nonTest)
	}
	if _, err := os.Stat("email.go"); err != nil {
		t.Fatalf("未找到 email.go，测试工作目录可能不对: %v", err)
	}
}
