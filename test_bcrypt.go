package main

import (
	"fmt"
	"os"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 获取环境变量中的哈希值
	hashFromEnv := os.Getenv("ADMIN_PASSWORD_HASH")
	if hashFromEnv == "" {
		fmt.Println("ADMIN_PASSWORD_HASH 环境变量未设置")
		return
	}

	fmt.Printf("环境变量中的哈希值: %s\n", hashFromEnv)

	// 测试密码
	password := "admin123"
	fmt.Printf("测试密码: %s\n", password)

	// 验证密码
	err := bcrypt.CompareHashAndPassword([]byte(hashFromEnv), []byte(password))
	if err != nil {
		fmt.Printf("密码验证失败: %v\n", err)
	} else {
		fmt.Println("密码验证成功!")
	}
}