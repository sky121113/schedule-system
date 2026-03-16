package models

import (
	"time"

	"gorm.io/gorm"
)

// MonthlySchedule 月度班表
type MonthlySchedule struct {
	gorm.Model
	Year   int    `gorm:"not null;uniqueIndex:idx_year_month" json:"year"`  // 年份
	Month  int    `gorm:"not null;uniqueIndex:idx_year_month" json:"month"` // 月份 (1~12)
	Status string `gorm:"type:varchar(20);default:'draft'" json:"status"`   // 狀態 (draft/published)
}

// MonthlySlot 月度班表的每一格排班
type MonthlySlot struct {
	gorm.Model
	ScheduleID uint      `gorm:"not null;index" json:"schedule_id"`                                     // 所屬月度班表
	Date       time.Time `gorm:"type:date;not null;uniqueIndex:idx_schedule_date_emp" json:"date"`       // 實際日期
	ShiftType  string    `gorm:"type:varchar(20);not null" json:"shift_type"`                            // 班別 (day/evening/night/day88/off)
	EmployeeID uint      `gorm:"not null;uniqueIndex:idx_schedule_date_emp" json:"employee_id"`          // 員工 ID
	CycleIndex int       `gorm:"not null" json:"cycle_index"`                                            // 第幾個循環 (1,2,3...)
	DayOffset  int       `gorm:"not null" json:"day_offset"`                                             // 循環中的第幾天 (0~27)
}

// CycleLeaveBalance 循環假期餘額追蹤（逐人）
type CycleLeaveBalance struct {
	gorm.Model
	CycleIndex  int  `gorm:"not null;uniqueIndex:idx_cycle_emp" json:"cycle_index"` // 第幾個循環 (1,2,3...)
	EmployeeID  uint `gorm:"not null;uniqueIndex:idx_cycle_emp" json:"employee_id"` // 員工 ID
	TotalLeave  int  `gorm:"not null;default:0" json:"total_leave"`                 // 該循環總假期天數
	UsedLeave   int  `gorm:"not null;default:0" json:"used_leave"`                  // 已使用假期天數
	OfflineUsed int  `gorm:"not null;default:0" json:"offline_used"`                // 在使用本系統前已扣除或線下使用的假
	MonthQuota  int  `gorm:"not null;default:-1" json:"month_quota"`                // 使用者手動指定的本月應休天數 (-1 = 未設定，由系統按比例算)
}
