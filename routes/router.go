package routes

import (
	"schedule-system/controllers"

	"github.com/gin-gonic/gin"
)

// SetupRouter 設定所有 API 路由
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// CORS 中介層
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
		// 員工管理
		employees := api.Group("/employees")
		{
			employees.GET("/", controllers.GetEmployees)
			employees.GET("/:id", controllers.GetEmployee)
			employees.POST("/", controllers.CreateEmployee)
			employees.PUT("/:id", controllers.UpdateEmployee)
			employees.DELETE("/:id", controllers.DeleteEmployee)

			// 員工班別限制
			employees.GET("/:id/restrictions", controllers.GetEmployeeRestrictions)
		}

		// 班別限制
		restrictions := api.Group("/restrictions")
		{
			restrictions.POST("/", controllers.CreateRestriction)
			restrictions.DELETE("/:id", controllers.DeleteRestriction)
			restrictions.GET("/validate", controllers.ValidateRestrictions)
		}

		// 人力需求
		staffing := api.Group("/staffing")
		{
			staffing.GET("/", controllers.GetStaffingRequirements)
			staffing.POST("/", controllers.UpsertStaffingRequirement)
			staffing.POST("/batch", controllers.BatchUpsertStaffingRequirements)
		}

		// 循環模板
		templates := api.Group("/templates")
		{
			templates.GET("/", controllers.GetTemplates)
			templates.GET("/:id", controllers.GetTemplate)
			templates.POST("/", controllers.CreateTemplate)
			templates.DELETE("/:id", controllers.DeleteTemplate)

			// 排班格
			templates.POST("/slots", controllers.SetSlot)
			templates.DELETE("/slots/:id", controllers.RemoveSlot)
			templates.DELETE("/:id/slots", controllers.ClearTemplateSlots)

			// 自動排班
			templates.POST("/:id/auto-schedule", controllers.AutoSchedule)

			// 日曆展開
			templates.GET("/:id/calendar", controllers.GetTemplateCalendar)

			// 預假
			templates.GET("/:id/pre-leaves", controllers.GetPreScheduledLeaves)
			templates.POST("/:id/pre-leaves", controllers.SetPreScheduledLeave)
			templates.DELETE("/:id/pre-leaves/:leaveId", controllers.DeletePreScheduledLeave)

			// 假期配額
			templates.GET("/:id/leave-quota", controllers.CalculateLeaveQuota)
		}

		// 月度班表
		monthly := api.Group("/monthly")
		{
			monthly.GET("/:year/:month", controllers.GetMonthlySchedule)
			monthly.POST("/:year/:month/generate", controllers.GenerateMonthlySchedule)
			monthly.GET("/:year/:month/leave-summary", controllers.GetMonthlyLeaveSummary)
			monthly.GET("/:year/:month/boundaries", controllers.GetCycleBoundaries)
			monthly.PUT("/cycle-balance", controllers.UpdateCycleBalance)
			monthly.PUT("/slots/:id", controllers.UpdateMonthlySlot)

			// 月度預假
			monthly.GET("/:year/:month/pre-leaves", controllers.GetMonthlyPreLeaves)
			monthly.POST("/pre-leaves", controllers.CreateMonthlyPreLeave)
			monthly.DELETE("/pre-leaves/:id", controllers.DeleteMonthlyPreLeave)
		}
	}

	return router
}
