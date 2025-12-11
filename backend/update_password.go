package main

import (
"fmt"
"os"

"golang.org/x/crypto/bcrypt"
"gorm.io/driver/postgres"
"gorm.io/gorm"
)

func main() {
	dsn := "host=192.168.2.200 port=5432 user=postgres password=8QMZn3yfrbkVG7 dbname=fusionmail-dev sslmode=disable"
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}
	
	password := "8158793720f789bb"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	
	result := db.Exec("UPDATE users SET password_hash = ?, failed_login_attempts = 0 WHERE username = 'admin'", string(hash))
	if result.Error != nil {
		fmt.Println("更新失败:", result.Error)
		os.Exit(1)
	}
	
	fmt.Printf("更新成功，影响 %d 行\n", result.RowsAffected)
	fmt.Println("新密码哈希:", string(hash))
}
