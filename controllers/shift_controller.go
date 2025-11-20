package controllers

import (
	"net/http"
	"schedule-system/db"
	"schedule-system/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetRequirementRequest 設定班表需求的請求結構
type SetRequirementRequest struct {
	Date          string           `json:"date" binding:"required"` // YYYY-MM-DD
	ShiftType     models.ShiftType `json:"shift_type" binding:"required"`
	RequiredCount int              `json:"required_count" binding:"required,min=0"`
}

// SetRequirement 設定某日某班別的需求人數 (Admin)
func SetRequirement(c *gin.Context) {
	var req SetRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式錯誤，請使用 YYYY-MM-DD"})
		return
	}

	var requirement models.ShiftRequirement
	result := db.DB.Where("date = ? AND shift_type = ?", date, req.ShiftType).First(&requirement)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new
		requirement = models.ShiftRequirement{
			Date:          date,
			ShiftType:     req.ShiftType,
			RequiredCount: req.RequiredCount,
		}
		if err := db.DB.Create(&requirement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "建立需求失敗"})
			return
		}
	} else {
		// Update existing
		requirement.RequiredCount = req.RequiredCount
		if err := db.DB.Save(&requirement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新需求失敗"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "設定成功", "data": requirement})
}

// GetMonthlySchedule 取得月份班表 (包含需求與已排人數)
func GetMonthlySchedule(c *gin.Context) {
	yearMonth := c.Query("month") // YYYY-MM
	if yearMonth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 month 參數 (YYYY-MM)"})
		return
	}

	startDate, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式錯誤"})
		return
	}
	endDate := startDate.AddDate(0, 1, 0)

	// 1. 取得該月所有需求設定
	var requirements []models.ShiftRequirement
	db.DB.Where("date >= ? AND date < ?", startDate, endDate).Find(&requirements)

	// 2. 取得該月所有排班紀錄
	var schedules []models.UserSchedule
	db.DB.Where("date >= ? AND date < ?", startDate, endDate).Find(&schedules)

	// 3. 整合資料
	type DailyShiftStatus struct {
		Required int `json:"required"`
		Booked   int `json:"booked"`
	}
	// Map: Date -> ShiftType -> Status
	result := make(map[string]map[models.ShiftType]*DailyShiftStatus)

	// Fill requirements
	for _, r := range requirements {
		dateStr := r.Date.Format("2006-01-02")
		if result[dateStr] == nil {
			result[dateStr] = make(map[models.ShiftType]*DailyShiftStatus)
		}
		if result[dateStr][r.ShiftType] == nil {
			result[dateStr][r.ShiftType] = &DailyShiftStatus{}
		}
		result[dateStr][r.ShiftType].Required = r.RequiredCount
	}

	// Fill booked counts
	for _, s := range schedules {
		dateStr := s.Date.Format("2006-01-02")
		if result[dateStr] == nil {
			result[dateStr] = make(map[models.ShiftType]*DailyShiftStatus)
		}
		if result[dateStr][s.ShiftType] == nil {
			result[dateStr][s.ShiftType] = &DailyShiftStatus{}
		}
		result[dateStr][s.ShiftType].Booked++
	}

	c.JSON(http.StatusOK, result)
}

// BookShiftRequest 預約班別請求
type BookShiftRequest struct {
	UserID    uint             `json:"user_id" binding:"required"`
	Date      string           `json:"date" binding:"required"`
	ShiftType models.ShiftType `json:"shift_type" binding:"required"`
}

// BookShift 使用者預約班別
func BookShift(c *gin.Context) {
	var req BookShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式錯誤"})
		return
	}

	// Check if already booked
	var count int64
	db.DB.Model(&models.UserSchedule{}).Where("user_id = ? AND date = ? AND shift_type = ?", req.UserID, date, req.ShiftType).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您已預約該時段"})
		return
	}

	// Check requirement (Optional: enforce limit?)
	// For now, we just allow booking but maybe warn if full?
	// Let's just book it.

	schedule := models.UserSchedule{
		UserID:    req.UserID,
		Date:      date,
		ShiftType: req.ShiftType,
	}

	if err := db.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "預約失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "預約成功", "data": schedule})
}
