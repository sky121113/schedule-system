package controllers

import (
	"net/http"
	"schedule-system/db"
	"schedule-system/models"

	"github.com/gin-gonic/gin"
)

// GetStaffingRequirements 取得所有人力需求
func GetStaffingRequirements(c *gin.Context) {
	var requirements []models.StaffingRequirement
	db.DB.Order("weekday, shift_type").Find(&requirements)
	c.JSON(http.StatusOK, requirements)
}

// SetStaffingRequirementRequest 設定人力需求請求
type SetStaffingRequirementRequest struct {
	Weekday           int    `json:"weekday" binding:"min=0,max=6"`
	ShiftType         string `json:"shift_type" binding:"required"`
	MinCount          int    `json:"min_count" binding:"min=0"`
	MinCountWithDay88 int    `json:"min_count_with_day88" binding:"min=0"`
}

// UpsertStaffingRequirement 新增或更新人力需求
func UpsertStaffingRequirement(c *gin.Context) {
	var req SetStaffingRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.StaffingRequirement
	result := db.DB.Where("weekday = ? AND shift_type = ?", req.Weekday, req.ShiftType).First(&existing)

	if result.Error != nil {
		// 新建
		sr := models.StaffingRequirement{
			Weekday:           req.Weekday,
			ShiftType:         req.ShiftType,
			MinCount:          req.MinCount,
			MinCountWithDay88: req.MinCountWithDay88,
		}
		if err := db.DB.Create(&sr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "建立失敗"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "建立成功", "data": sr})
	} else {
		// 更新
		existing.MinCount = req.MinCount
		existing.MinCountWithDay88 = req.MinCountWithDay88
		if err := db.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失敗"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "更新成功", "data": existing})
	}
}

// BatchUpsertStaffingRequirements 批次設定人力需求
func BatchUpsertStaffingRequirements(c *gin.Context) {
	var reqs []SetStaffingRequirementRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, req := range reqs {
		var existing models.StaffingRequirement
		result := db.DB.Where("weekday = ? AND shift_type = ?", req.Weekday, req.ShiftType).First(&existing)

		if result.Error != nil {
			sr := models.StaffingRequirement{
				Weekday:           req.Weekday,
				ShiftType:         req.ShiftType,
				MinCount:          req.MinCount,
				MinCountWithDay88: req.MinCountWithDay88,
			}
			db.DB.Create(&sr)
		} else {
			existing.MinCount = req.MinCount
			existing.MinCountWithDay88 = req.MinCountWithDay88
			db.DB.Save(&existing)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "批次設定成功"})
}
