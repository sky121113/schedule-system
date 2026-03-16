package models

import "gorm.io/gorm"

// StaffingRequirement 各星期各班別的最低人力需求
type StaffingRequirement struct {
	gorm.Model
	Weekday          int    `gorm:"not null;uniqueIndex:idx_weekday_shift" json:"weekday"`     // 星期 (0=日, 1=一, ..., 6=六)
	ShiftType        string `gorm:"type:varchar(20);not null;uniqueIndex:idx_weekday_shift" json:"shift_type"` // 班別
	MinCount         int    `gorm:"not null" json:"min_count"`                                  // 正常最少人數
	MinCountWithDay88 int   `gorm:"not null" json:"min_count_with_day88"`                       // 有 8-8 時的最少人數
}
