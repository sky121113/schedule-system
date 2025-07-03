package main

import (
	"schedule-system/config"
	"schedule-system/controllers"
	"schedule-system/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// 連接資料庫
	config.ConnectDB()

	// 自動建立資料表
	config.DB.AutoMigrate(&models.User{})

	r := gin.Default()

	// User API 路由
	r.POST("/users", controllers.CreateUser)    // 新增 User
	r.PUT("/users/:id", controllers.UpdateUser) // 更新 User

	r.Run(":8080") // 啟動伺服器
}
