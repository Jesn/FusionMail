package response_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// gormTagPattern 匹配 model 源码里带 gorm tag 的结构体字段。
var gormTagPattern = regexp.MustCompile(`gorm:"`)

// structPattern 匹配 `type Xxx struct { ... }`（非贪婪到首个行首 }）。
var structPattern = regexp.MustCompile(`(?ms)^type (\w+) struct \{(.*?)^\}`)

// backendRoot 返回 backend/ 的绝对路径。
func backendRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法解析测试文件路径")
	}
	// internal/dto/response/boundary_test.go -> backend/
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
}

// collectGORMModels 扫描 internal/model 下所有带 gorm tag 的结构体名。
func collectGORMModels(t *testing.T, root string) map[string]bool {
	t.Helper()
	models := map[string]bool{}
	pattern := filepath.Join(root, "internal", "model", "*.go")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		t.Fatalf("未找到 model 源码（pattern=%s）: %v", pattern, err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		content, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatalf("读取 %s 失败: %v", f, readErr)
		}
		for _, m := range structPattern.FindAllStringSubmatch(string(content), -1) {
			if gormTagPattern.MatchString(m[2]) {
				models[m[1]] = true
			}
		}
	}
	if len(models) == 0 {
		t.Fatal("未从 model 源码解析出任何 GORM 模型，解析逻辑可能已失效")
	}
	return models
}

// TestNoGORMModelLeaksIntoAPIContract 是分层边界的守卫测试。
//
// 契约：**带 gorm tag 的持久化模型不得出现在 API 契约里**。
//
// 为什么需要它：model 同时承担 GORM 存储映射与 JSON 序列化时，存储���字段
// （created_at / updated_at / deleted_at / dedupe_key / 加密凭证）会静默进入
// 对外契约，而且新增字段会继续泄露——编译器不会拦，评审也容易漏。
//
// 2026-10 的体检发现 5 个模型泄露（Adapter / EmailRule / Provider /
// ProviderAdapter / SentEmail），已全部改为显式 DTO。本测试防止回归。
//
// 依赖 docs/swagger.json（由 `swag init` 生成）。若文件缺失会跳过并提示生成命令，
// 因此 CI 里必须先生成再跑测试（见 .cnb.yml 的 quality-gate）。
func TestNoGORMModelLeaksIntoAPIContract(t *testing.T) {
	root := backendRoot(t)
	swaggerPath := filepath.Join(root, "docs", "swagger.json")

	raw, err := os.ReadFile(swaggerPath)
	if err != nil {
		t.Skipf("未找到 %s，跳过边界检查。请先运行：swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal", swaggerPath)
	}

	var doc struct {
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析 swagger.json 失败: %v", err)
	}
	if len(doc.Definitions) == 0 {
		t.Fatal("swagger.json 里没有任何 definitions，文档可能未正确生成")
	}

	gormModels := collectGORMModels(t, root)

	var leaks []string
	for name := range doc.Definitions {
		if !strings.HasPrefix(name, "model.") {
			continue
		}
		typeName := strings.TrimPrefix(name, "model.")
		if gormModels[typeName] {
			leaks = append(leaks, name)
		}
	}
	sort.Strings(leaks)

	if len(leaks) > 0 {
		t.Errorf("检测到 %d 个 GORM 持久化模型泄露进 API 契约：\n  %s\n\n"+
			"修复方式：为它建一个显式 DTO（逐字段赋值，不要结构体嵌入），"+
			"在 handler 里做映射，并把 Swagger 注解指向 DTO。\n"+
			"参考 internal/dto/response/{public_email,rule_summary,sent_email}.go。",
			len(leaks), strings.Join(leaks, "\n  "))
	}
}

// TestAPIContractHasNoGhostEnvelopeType 校验 API 契约里的响应信封类型真实存在。
//
// 历史问题：210 处 Swagger 注解声明返回 response.Response，但没有任何 handler
// 文件 import internal/dto/response —— 实际返回的是 dto.Response（多一个 Code 字段）。
// 已提交的 docs/swagger.json 因此描述了一个运行时并不存在的信封。
func TestAPIContractHasNoGhostEnvelopeType(t *testing.T) {
	root := backendRoot(t)
	swaggerPath := filepath.Join(root, "docs", "swagger.json")

	raw, err := os.ReadFile(swaggerPath)
	if err != nil {
		t.Skipf("未找到 %s，跳过信封检查", swaggerPath)
	}

	var doc struct {
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析 swagger.json 失败: %v", err)
	}

	if _, exists := doc.Definitions["response.Response"]; exists {
		t.Error("API 契约里出现 response.Response —— handler 实际返回的是 dto.Response，" +
			"请把 Swagger 注解改为 dto.Response（参考 2026-10 的 P7 修复）")
	}
	if _, exists := doc.Definitions["dto.Response"]; !exists {
		t.Error("API 契���里缺少 dto.Response —— 统一响应信封未出现在文档中")
	}
}
