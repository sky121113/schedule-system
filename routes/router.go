package routes

import (
	"schedule-system/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

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

		shifts := api.Group("/shifts")
		{
			shifts.POST("/requirements", controllers.SetRequirement)
			shifts.GET("/schedule", controllers.GetMonthlySchedule)
			shifts.POST("/book", controllers.BookShift)
		}
	}

	return router
}
