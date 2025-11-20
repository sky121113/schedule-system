package main

import (
	"fmt"
	"log"
	"schedule-system/db"
)

type TableInfo struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default *string
	Extra   string
}

func main() {
	// 連接資料庫
	db.ConnectDB()

	// 查詢 users 表結構
	var info []TableInfo
	result := db.DB.Raw("DESCRIBE users").Scan(&info)

	if result.Error != nil {
		log.Fatal("查詢失敗:", result.Error)
	}

	fmt.Println("====== users 表結構 ======")
	for _, field := range info {
		fmt.Printf("欄位: %-15s 型別: %-20s\n", field.Field, field.Type)
	}
}
