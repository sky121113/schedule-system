package controllers

import (
	"net/http"
	"strconv"

	"schedule-system/config"
	"schedule-system/models"

	"github.com/gin-gonic/gin"
)

// 新增 User
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 預設狀態啟用
	if user.Status != 0 && user.Status != 1 {
		user.Status = 1
	}

	if result := config.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// 更新 User
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	// 查找 User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User 不存在"})
		return
	}

	// 更新資料
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if result := config.DB.Save(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ✅ 查找單一 User
func GetUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User 不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// 查找全部 Users (可選擇只查啟用)
func GetUsers(c *gin.Context) {
	var users []models.User

	// 檢查是否有 status query param
	statusQuery := c.Query("status")
	if statusQuery != "" {
		status, err := strconv.Atoi(statusQuery)
		if err == nil {
			config.DB.Where("status = ?", status).Find(&users)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status 必須是數字"})
			return
		}
	} else {
		config.DB.Find(&users)
	}

	c.JSON(http.StatusOK, users)
}

// 刪除 User
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if result := config.DB.Delete(&models.User{}, id); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User 刪除成功"})
}
