package models

import (
	"time"

	"gorm.io/gorm"
)

// MonthlyPreScheduledLeave 月度班表專用的預假（按具體日期）
type MonthlyPreScheduledLeave struct {
	gorm.Model
	EmployeeID uint      `gorm:"not null;uniqueIndex:idx_monthly_pre_leave" json:"employee_id"`
	Date       time.Time `gorm:"not null;uniqueIndex:idx_monthly_pre_leave" json:"date"`
	Reason     string    `gorm:"type:varchar(200)" json:"reason"`
}
