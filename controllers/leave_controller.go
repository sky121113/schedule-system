package controllers

import (
	"net/http"
	"schedule-system/db"
	"schedule-system/models"

	"github.com/gin-gonic/gin"
)

// GetPreScheduledLeaves 取得某模板的所有預假
func GetPreScheduledLeaves(c *gin.Context) {
	templateID := c.Param("id")
	var leaves []models.PreScheduledLeave
	db.DB.Where("template_id = ?", templateID).Find(&leaves)
	c.JSON(http.StatusOK, leaves)
}

// SetPreScheduledLeave 設定預假 (每人每循環最多 3 天)
func SetPreScheduledLeave(c *gin.Context) {
	var req struct {
		TemplateID uint   `json:"template_id" binding:"required"`
		EmployeeID uint   `json:"employee_id" binding:"required"`
		DayOffset  int    `json:"day_offset" binding:"required"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 檢查該員工在此模板是否已超過 3 天預假
	var count int64
	db.DB.Model(&models.PreScheduledLeave{}).
		Where("template_id = ? AND employee_id = ?", req.TemplateID, req.EmployeeID).
		Count(&count)
	if count >= 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "每人每循環最多 3 天預假"})
		return
	}

	leave := models.PreScheduledLeave{
		TemplateID: req.TemplateID,
		EmployeeID: req.EmployeeID,
		DayOffset:  req.DayOffset,
		Reason:     req.Reason,
	}
	if err := db.DB.Create(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "設定失敗 (可能重複)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "預假已設定", "data": leave})
}

// DeletePreScheduledLeave 刪除預假
func DeletePreScheduledLeave(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Unscoped().Delete(&models.PreScheduledLeave{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "預假不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "預假已刪除"})
}

// CalculateLeaveQuota 計算假期配額
func CalculateLeaveQuota(c *gin.Context) {
	templateID := c.Param("id")

	var template models.CycleTemplate
	if err := db.DB.First(&template, templateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	totalDays := template.CycleWeeks * 7

	// 可用員工數
	var activeCount int64
	db.DB.Model(&models.Employee{}).Where("status = 1").Count(&activeCount)

	// 計算總需求人次
	var requirements []models.StaffingRequirement
	db.DB.Find(&requirements)

	totalRequired := 0
	for d := 0; d < totalDays; d++ {
		weekday := d % 7 // 簡化：假設從某個星期幾開始
		for _, req := range requirements {
			if req.Weekday == weekday {
				// 使用有 8-8 的最低需求 (因為 J 幾乎每天上 8-8)
				totalRequired += req.MinCountWithDay88
			}
		}
	}

	// 加上 8-8 本身的人次 (J 每天上班，28天中約24天)
	day88Days := 0
	for d := 0; d < totalDays; d++ {
		if (d % 6) < 5 {
			day88Days++
		}
	}
	totalRequired += day88Days

	totalAvailable := int(activeCount) * totalDays
	totalLeave := totalAvailable - totalRequired
	if totalLeave < 0 {
		totalLeave = 0
	}
	perPerson := 0
	if activeCount > 0 {
		perPerson = totalLeave / int(activeCount)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_available":  totalAvailable,
		"total_required":   totalRequired,
		"total_leave":      totalLeave,
		"per_person_leave": perPerson,
		"active_employees": activeCount,
		"total_days":       totalDays,
	})
}
