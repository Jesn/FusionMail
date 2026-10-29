package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// routerAnnotationPattern 匹配 handler 源码里的 `// @Router <path> [<method>]`。
var routerAnnotationPattern = regexp.MustCompile(`(?m)^//\s*@Router\s+(\S+)\s+\[(\w+)\]`)

// TestSwaggerAnnotationsAreEmitted 是 swagger 注解与生成文档的同步守卫。
//
// 契约：**源码里每一条 @Router 注解都必须出现在 docs/swagger.json 里**。
//
// 为什么需要它：swag 只处理紧邻函数声明的注释组。文件拆分/重构时若把函数体搬走
// 而把注解留在原处，注解就成了「孤儿」——源码里看得见，生成的文档里没有。
// 2026-10 的 41de287（handler 大文件拆分）制造了两处：
//
//   - oauth2_handler.go 末尾的 GoogleAuthorize 注解：整条路径从文档消失
//     （提交的 swagger.json 里还有，重新生成就没了 —— 文档与代码不同步）
//   - oauth2_google.go 末尾的 MicrosoftAuthorize 注解：路径还在，但操作内容
//     被清空（summary/parameters/responses 全丢）
//
// 编译器不会拦、评审也容易漏，且 CI 只跑 swag init 覆盖、不做 diff，
// 所以这类漂移可以长期不可见。本测试把它变成红灯。
//
// 只做「注解 -> 文档」单向断言。反向（路由注册了但没注解）需要解析 router.go 的
// 嵌套 Group 前缀，脆弱且易误报 —— 大量端点（/metrics、/health 等）本就不该进契约。
//
// 依赖 docs/swagger.json（由 `swag init` 生成）。若文件缺失会跳过并提示生成命令，
// 因此 CI 里必须先生成再跑测试（见 scripts/ci-quality-gate.sh）。
func TestSwaggerAnnotationsAreEmitted(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法解析测试文件路径")
	}
	handlerDir := filepath.Dir(currentFile)
	root := filepath.Join(handlerDir, "..", "..")

	swaggerPath := filepath.Join(root, "docs", "swagger.json")
	raw, err := os.ReadFile(swaggerPath)
	if err != nil {
		t.Skipf("未找到 %s，跳过注解同步检查。请先运行：swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal", swaggerPath)
	}

	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析 swagger.json 失败: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("swagger.json 里没有任何 paths，文档可能未正确生成")
	}

	// 收集源码里的所有 @Router 注解。
	type annotation struct {
		file   string
		line   int
		path   string
		method string
	}
	var annotations []annotation

	sources, err := filepath.Glob(filepath.Join(handlerDir, "*.go"))
	if err != nil {
		t.Fatalf("扫描 handler 源码失败: %v", err)
	}
	for _, src := range sources {
		if strings.HasSuffix(src, "_test.go") {
			continue
		}
		content, readErr := os.ReadFile(src)
		if readErr != nil {
			t.Fatalf("读取 %s 失败: %v", src, readErr)
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			m := routerAnnotationPattern.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			annotations = append(annotations, annotation{
				file:   filepath.Base(src),
				line:   i + 1,
				path:   m[1],
				method: strings.ToLower(m[2]),
			})
		}
	}

	// 防止正则失效后测试空转通过。
	if len(annotations) == 0 {
		t.Fatal("未从 handler 源码解析出任何 @Router 注解，解析逻辑可能已失效")
	}

	var missing []string
	for _, a := range annotations {
		item, ok := doc.Paths[a.path]
		if !ok {
			missing = append(missing, a.file+":"+strconv.Itoa(a.line)+"  "+a.method+" "+a.path+"  —— 路径不在 swagger.json 里")
			continue
		}
		if _, ok := item[a.method]; !ok {
			missing = append(missing, a.file+":"+strconv.Itoa(a.line)+"  "+a.method+" "+a.path+"  —— 路径在，但该方法不在")
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		t.Errorf("检测到 %d 条 @Router 注解未被 swag 写进 docs/swagger.json：\n  %s\n\n"+
			"最常见原因：注解是「孤儿」—— 函数体被搬到了别的文件，注解留在了原处。\n"+
			"swag 只处理紧邻函数声明的注释组，孤儿注解会被静默丢弃。\n"+
			"修复方式：把注解块整体搬到函数声明的正上方，然后重新运行\n"+
			"  swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal",
			len(missing), strings.Join(missing, "\n  "))
	}
}
