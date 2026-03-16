package models

import (
	"time"

	"gorm.io/gorm"
)

// CycleTemplate 循環模板
type CycleTemplate struct {
	gorm.Model
	StartDate  time.Time `gorm:"type:date;not null" json:"start_date"`                         // 循環起始日
	CycleWeeks int       `gorm:"not null;default:4" json:"cycle_weeks"`                        // 循環週數
	Version    int       `gorm:"not null;default:1" json:"version"`                            // 版本號
	Status     string    `gorm:"type:varchar(20);default:'draft'" json:"status"`                // 狀態 (draft/active/archived)
}

// TemplateSlot 模板中的每一格排班
type TemplateSlot struct {
	gorm.Model
	TemplateID uint   `gorm:"not null;index;uniqueIndex:idx_template_day_emp" json:"template_id"` // 所屬模板
	DayOffset  int    `gorm:"not null;uniqueIndex:idx_template_day_emp" json:"day_offset"`        // 第幾天 (0~27)
	ShiftType  string `gorm:"type:varchar(20);not null" json:"shift_type"`                        // 班別
	EmployeeID uint   `gorm:"not null;uniqueIndex:idx_template_day_emp" json:"employee_id"`       // 員工 ID
}
