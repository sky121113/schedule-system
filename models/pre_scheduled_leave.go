package models

import "gorm.io/gorm"

// PreScheduledLeave 預假（每人每循環固定 3 天）
type PreScheduledLeave struct {
	gorm.Model
	EmployeeID uint   `gorm:"not null;uniqueIndex:idx_pre_leave" json:"employee_id"` // 員工 ID
	TemplateID uint   `gorm:"not null;uniqueIndex:idx_pre_leave" json:"template_id"` // 循環模板 ID
	DayOffset  int    `gorm:"not null;uniqueIndex:idx_pre_leave" json:"day_offset"`  // 第幾天 (0~27)
	Reason     string `gorm:"type:varchar(200)" json:"reason"`                       // 原因 (選填)
}
