package routes

import (
	"log"
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
		log.Println("🚀 路由設定中...")
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

		// (循環模板路由已移除，排班功能統一由月度排班處理)

		// 月度班表
		monthly := api.Group("/monthly")
		{
			// 班表版本管理 (放在較通用的路由之前)
			monthly.GET("/:year/:month/versions", controllers.ListMonthlyVersions)
			monthly.POST("/:year/:month/versions", controllers.SaveMonthlyVersion)
			monthly.POST("/versions/:versionId/restore", controllers.RestoreMonthlyVersion)
			monthly.DELETE("/versions/:versionId", controllers.DeleteMonthlyVersion)

			// 統計與摘要
			monthly.GET("/:year/:month/leave-summary", controllers.GetMonthlyLeaveSummary)
			monthly.GET("/:year/:month/boundaries", controllers.GetCycleBoundaries)
			monthly.PUT("/cycle-balance", controllers.UpdateCycleBalance)
			monthly.PUT("/slots/:id", controllers.UpdateMonthlySlot)

			// 核心操作
			monthly.POST("/:year/:month/generate", controllers.GenerateMonthlySchedule)
			monthly.GET("/:year/:month", controllers.GetMonthlySchedule)

			// 月度預假
			monthly.GET("/:year/:month/pre-leaves", controllers.GetMonthlyPreLeaves)
			monthly.POST("/pre-leaves", controllers.CreateMonthlyPreLeave)
			monthly.DELETE("/pre-leaves/:id", controllers.DeleteMonthlyPreLeave)
		}
	}

	return router
}
