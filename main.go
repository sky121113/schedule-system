package main

import (
	"schedule-system/config"
	"schedule-system/models"
	"schedule-system/routes"
)

func main() {
	// 連接資料庫
	config.ConnectDB()

	// 自動建立資料表
	config.DB.AutoMigrate(&models.User{})

	// 設定路由
	router := routes.SetupRouter()

	// 啟動伺服器
	router.Run(":8080")
}
