package routes

import (
	"schedule-system/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	api := router.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.POST("/", controllers.CreateUser)
			users.PUT("/:id", controllers.UpdateUser)
			users.GET("/", controllers.GetUsers)   // 查全部
			users.GET("/:id", controllers.GetUser) // 查單一
			users.DELETE("/:id", controllers.DeleteUser)
		}
	}

	return router
}
