package routes

import (
	"schedule-system/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// API 分組
	api := router.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.POST("/", controllers.CreateUser)   // 新增 User
			users.PUT("/:id", controllers.UpdateUser) // 更新 User
			// users.GET("/", controllers.GetUsers)         // 查詢所有 User
			// users.GET("/:id", controllers.GetUser)       // 查詢單一 User
			// users.DELETE("/:id", controllers.DeleteUser) // 刪除 User
		}
	}

	return router
}
