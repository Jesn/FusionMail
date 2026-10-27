package webapi_test

import (
	"testing"

	"fusionmail/internal/adapter/webapi"
	cloudflareadapter "fusionmail/internal/adapter/webapi/cloudflare"
	cloudmailadapter "fusionmail/internal/adapter/webapi/cloudmail"
	customadapter "fusionmail/internal/adapter/webapi/custom"
	"fusionmail/internal/dto/webapiconfig"
)

// TestAdapterRegistrationContract 守卫「适配器必须经组合根显式注册」的契约。
//
// 背景：注册曾是 blank import + init() 副作用 —— 重构中删掉"未使用的 import"
// 会让适配器静默消失（无编译错误，仅运行时故障）。改为 container.go 显式调用
// Register() 后，本测试确保三个 Register() 确实把创建函数放进了注册表：
// 若未来有人新增适配器却忘了在组合根调用 Register()，这里不会直接失败
// （新适配器不在断言清单里），但删掉任何既有 Register() 调用会立刻失败。
func TestAdapterRegistrationContract(t *testing.T) {
	// Register() 幂等（map 覆盖写），重复调用安全
	cloudflareadapter.Register()
	cloudmailadapter.Register()
	customadapter.Register()

	factory := webapi.NewWebAPIAdapterFactory()
	registered := map[string]bool{}
	for _, st := range factory.GetSupportedServiceTypes() {
		registered[st] = true
	}

	required := []string{
		webapiconfig.WebAPIServiceTypeCloudflareTempEmail,
		webapiconfig.WebAPIServiceTypeCloudMail,
		webapiconfig.WebAPIServiceTypeCustom,
	}
	for _, st := range required {
		if !registered[st] {
			t.Errorf("服务类型 %s 未注册 —— 组合根必须调用对应适配器包的 Register()", st)
		}
	}
}

// TestRegisterIsIdempotent 验证重复调用 Register() 不产生副作用。
func TestRegisterIsIdempotent(t *testing.T) {
	factory := webapi.NewWebAPIAdapterFactory()

	customadapter.Register()
	first := len(factory.GetSupportedServiceTypes())
	customadapter.Register()
	second := len(factory.GetSupportedServiceTypes())

	if first != second {
		t.Errorf("重复 Register() 改变了注册表大小: %d -> %d", first, second)
	}
}
