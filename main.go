package main

import (
	"schedule-system/db"
	"schedule-system/routes"
)

func main() {
	// 連接資料庫
	db.ConnectDB()

	// 自動建立資料表
	db.RunMigrations()

	// 設定路由
	router := routes.SetupRouter()

	// 啟動伺服器
	router.Run(":8080")
}
