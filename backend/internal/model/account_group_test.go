package model

import (
	"strings"
	"testing"
	"unicode"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// **Feature: account-group, Property 1: Group creation with valid data**
// *For any* valid group name (non-empty, non-whitespace string), creating a group
// SHALL result in a new group record with the specified name.
// **Validates: Requirements 1.1, 1.4**
func TestProperty_GroupCreationWithValidData(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成有效的分组名称（1-100个字符的字母数字字符串）
	validNameGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default"
		}
		if len(s) > 100 {
			return s[:100]
		}
		return s
	}).SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= 100
	})

	properties.Property("valid group name creates group with correct name", prop.ForAll(
		func(name string) bool {
			group := &AccountGroup{
				Name: name,
			}

			err := group.Validate()
			if err != nil {
				return false
			}

			// 验证名称被正确处理（去除首尾空格）
			expectedName := strings.TrimSpace(name)
			return group.Name == expectedName
		},
		validNameGen,
	))

	properties.Property("valid group with description creates correctly", prop.ForAll(
		func(name string, desc string) bool {
			group := &AccountGroup{
				Name:        name,
				Description: desc,
			}

			err := group.Validate()
			if err != nil {
				return false
			}

			return group.Name == strings.TrimSpace(name) && group.Description == desc
		},
		validNameGen,
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// **Feature: account-group, Property 2: Group name validation**
// *For any* string composed entirely of whitespace or empty string, attempting to
// create a group SHALL be rejected with a validation error.
// **Validates: Requirements 1.2**
func TestProperty_GroupNameValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成空白字符串（空或仅包含空白字符）
	whitespaceGen := gen.OneGenOf(
		gen.Const(""),
		gen.SliceOf(gen.OneConstOf(' ', '\t', '\n', '\r')).Map(func(chars []rune) string {
			return string(chars)
		}),
	)

	properties.Property("empty or whitespace-only name is rejected", prop.ForAll(
		func(name string) bool {
			group := &AccountGroup{
				Name: name,
			}

			err := group.Validate()

			// 如果名称为空或仅包含空白，应该返回错误
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				return err == ErrGroupNameRequired
			}
			return true
		},
		whitespaceGen,
	))

	// 测试超长名称
	properties.Property("name exceeding 100 characters is rejected", prop.ForAll(
		func(baseStr string) bool {
			// 生成超过100字符的名称
			longName := strings.Repeat(baseStr+"a", 101)
			if len(longName) <= 100 {
				longName = strings.Repeat("a", 101)
			}

			group := &AccountGroup{
				Name: longName,
			}

			err := group.Validate()
			return err == ErrGroupNameTooLong
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// 单元测试：验证基本功能
func TestAccountGroup_Validate(t *testing.T) {
	tests := []struct {
		name    string
		group   *AccountGroup
		wantErr error
	}{
		{
			name:    "valid name",
			group:   &AccountGroup{Name: "工作邮箱"},
			wantErr: nil,
		},
		{
			name:    "valid name with description",
			group:   &AccountGroup{Name: "个人邮箱", Description: "个人使用的邮箱账号"},
			wantErr: nil,
		},
		{
			name:    "empty name",
			group:   &AccountGroup{Name: ""},
			wantErr: ErrGroupNameRequired,
		},
		{
			name:    "whitespace only name",
			group:   &AccountGroup{Name: "   "},
			wantErr: ErrGroupNameRequired,
		},
		{
			name:    "name with leading/trailing spaces",
			group:   &AccountGroup{Name: "  工作邮箱  "},
			wantErr: nil,
		},
		{
			name:    "name too long",
			group:   &AccountGroup{Name: strings.Repeat("a", 101)},
			wantErr: ErrGroupNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.group.Validate()
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccountGroup_TableName(t *testing.T) {
	group := &AccountGroup{}
	assert.Equal(t, "account_groups", group.TableName())
}

func TestAccountGroup_NameTrimming(t *testing.T) {
	group := &AccountGroup{Name: "  测试分组  "}
	err := group.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "测试分组", group.Name)
}

// isWhitespace 检查字符串是否仅包含空白字符
func isWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
