package controllers

import (
	"net/http"
	"strconv"

	"schedule-system/db"
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

	if result := db.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// 更新 User
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	// 先找出原始資料
	var user models.User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User 不存在"})
		return
	}

	// 用 map 接收 JSON，避免覆蓋整個 struct
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新資料
	if result := db.DB.Model(&user).Updates(input); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ✅ 查找單一 User
func GetUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := db.DB.First(&user, id).Error; err != nil {
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
			db.DB.Where("status = ?", status).Find(&users)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status 必須是數字"})
			return
		}
	} else {
		db.DB.Find(&users)
	}

	c.JSON(http.StatusOK, users)
}

// 刪除 User
// ✅ 刪除 User (僅允許 status=0)
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	// 先查找 User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User 不存在"})
		return
	}

	// 檢查狀態
	if user.Status != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "僅允許刪除狀態為停用 (status=0) 的 User"})
		return
	}

	// 執行刪除
	if result := db.DB.Delete(&user).Error; result != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User 刪除成功"})
}
