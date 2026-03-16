package controllers

import (
	"net/http"
	"schedule-system/db"
	"schedule-system/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetEmployees 取得所有員工
func GetEmployees(c *gin.Context) {
	var employees []models.Employee
	db.DB.Find(&employees)
	c.JSON(http.StatusOK, employees)
}

// GetEmployee 取得單一員工
func GetEmployee(c *gin.Context) {
	id := c.Param("id")
	var employee models.Employee
	if err := db.DB.First(&employee, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "員工不存在"})
		return
	}
	c.JSON(http.StatusOK, employee)
}

// CreateEmployeeRequest 建立員工請求
type CreateEmployeeRequest struct {
	Name           string `json:"name" binding:"required"`
	Email          string `json:"email" binding:"required"`
	IsDay88Primary bool   `json:"is_day88_primary"`
	Status         int    `json:"status"`
}

// CreateEmployee 建立員工
func CreateEmployee(c *gin.Context) {
	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	employee := models.Employee{
		Name:           req.Name,
		Email:          req.Email,
		IsDay88Primary: req.IsDay88Primary,
		Status:         req.Status,
	}
	if employee.Status == 0 {
		employee.Status = 1
	}

	if err := db.DB.Create(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立員工失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "建立成功", "data": employee})
}

// UpdateEmployee 更新員工
func UpdateEmployee(c *gin.Context) {
	id := c.Param("id")
	var employee models.Employee
	if err := db.DB.First(&employee, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "員工不存在"})
		return
	}

	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	employee.Name = req.Name
	employee.Email = req.Email
	employee.IsDay88Primary = req.IsDay88Primary
	employee.Status = req.Status

	if err := db.DB.Save(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功", "data": employee})
}

// DeleteEmployee 刪除員工
func DeleteEmployee(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.Employee{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刪除失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "刪除成功"})
}

// --- 班別限制 ---

// GetEmployeeRestrictions 取得某員工的限制
func GetEmployeeRestrictions(c *gin.Context) {
	employeeID := c.Param("id")
	templateID := c.Query("template_id") // 可選

	query := db.DB.Where("employee_id = ?", employeeID)
	if templateID != "" {
		query = query.Where("template_id = ? OR template_id IS NULL", templateID)
	}

	var restrictions []models.ShiftRestriction
	query.Find(&restrictions)
	c.JSON(http.StatusOK, restrictions)
}

// SetRestrictionRequest 設定限制請求
type SetRestrictionRequest struct {
	EmployeeID uint   `json:"employee_id" binding:"required"`
	TemplateID *uint  `json:"template_id"`
	ShiftType  string `json:"shift_type" binding:"required"`
	MaxDays    *int   `json:"max_days"`
	Note       string `json:"note"`
}

// CreateRestriction 建立限制
func CreateRestriction(c *gin.Context) {
	var req SetRestrictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	restriction := models.ShiftRestriction{
		EmployeeID: req.EmployeeID,
		TemplateID: req.TemplateID,
		ShiftType:  req.ShiftType,
		MaxDays:    req.MaxDays,
		Note:       req.Note,
	}

	if err := db.DB.Create(&restriction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立限制失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "設定成功", "data": restriction})
}

// DeleteRestriction 刪除限制
func DeleteRestriction(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.ShiftRestriction{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刪除失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "刪除成功"})
}

// ValidateRestrictions 驗證限制是否可行
func ValidateRestrictions(c *gin.Context) {
	// 取得所有在職員工
	var employees []models.Employee
	db.DB.Where("status = 1").Find(&employees)

	// 取得所有限制
	templateIDStr := c.Query("template_id")
	var restrictions []models.ShiftRestriction
	if templateIDStr != "" {
		db.DB.Where("template_id = ? OR template_id IS NULL", templateIDStr).Find(&restrictions)
	} else {
		db.DB.Where("template_id IS NULL").Find(&restrictions)
	}

	// 取得人力需求
	var requirements []models.StaffingRequirement
	db.DB.Find(&requirements)

	// 建立員工禁班 map
	restrictedMap := make(map[uint]map[string]bool) // employeeID -> shiftType -> banned
	for _, r := range restrictions {
		if r.MaxDays == nil { // nil = 完全禁止
			if restrictedMap[r.EmployeeID] == nil {
				restrictedMap[r.EmployeeID] = make(map[string]bool)
			}
			restrictedMap[r.EmployeeID][r.ShiftType] = true
		}
	}

	// 檢查各班別可用人數 vs. 需求
	shiftTypes := []string{"day", "evening", "night"}
	warnings := []string{}

	for _, st := range shiftTypes {
		availableCount := 0
		for _, emp := range employees {
			if restrictedMap[emp.ID] == nil || !restrictedMap[emp.ID][st] {
				availableCount++
			}
		}

		// 取最大需求
		maxRequired := 0
		for _, req := range requirements {
			if req.ShiftType == st && req.MinCount > maxRequired {
				maxRequired = req.MinCount
			}
		}

		if availableCount < maxRequired {
			warnings = append(warnings, st+"班可用人數("+strconv.Itoa(availableCount)+")不足最大需求("+strconv.Itoa(maxRequired)+")")
		}
	}

	if len(warnings) > 0 {
		c.JSON(http.StatusOK, gin.H{"valid": false, "warnings": warnings})
	} else {
		c.JSON(http.StatusOK, gin.H{"valid": true, "warnings": []string{}})
	}
}
