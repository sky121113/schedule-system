package main

import (
	"log"
	"schedule-system/db"
)

func main() {
	// 連接資料庫
	db.ConnectDB()

	// 刪除可能存在錯誤的表
	log.Println("正在清理舊表...")

	// 刪除 user_schedules 表（如果存在）
	if err := db.DB.Exec("DROP TABLE IF EXISTS user_schedules").Error; err != nil {
		log.Println("警告: 刪除 user_schedules 失敗:", err)
	} else {
		log.Println("✅ 已刪除 user_schedules 表")
	}

	// 刪除 shift_requirements 表（如果存在）
	if err := db.DB.Exec("DROP TABLE IF EXISTS shift_requirements").Error; err != nil {
		log.Println("警告: 刪除 shift_requirements 失敗:", err)
	} else {
		log.Println("✅ 已刪除 shift_requirements 表")
	}

	log.Println("✅ 資料庫清理完成！請重新執行 go run main.go")
}
