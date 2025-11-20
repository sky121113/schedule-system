package models

import (
	"time"

	"gorm.io/gorm"
)

// ShiftType 定義班別類型
type ShiftType string

const (
	ShiftMorning   ShiftType = "morning"
	ShiftAfternoon ShiftType = "afternoon"
	ShiftEvening   ShiftType = "evening"
)

// ShiftRequirement 定義每日各班別的需求人數
type ShiftRequirement struct {
	gorm.Model
	Date          time.Time `gorm:"type:date;uniqueIndex:idx_date_shift"`        // 日期
	ShiftType     ShiftType `gorm:"type:varchar(20);uniqueIndex:idx_date_shift"` // 班別
	RequiredCount int       `gorm:"not null"`                                    // 需求人數
}

// UserSchedule 定義使用者的排班紀錄
type UserSchedule struct {
	gorm.Model
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_user_date_shift"`
	Date      time.Time `gorm:"type:date;uniqueIndex:idx_user_date_shift"` // 使用者在同一天同一班別只能排一次
	ShiftType ShiftType `gorm:"type:varchar(20);uniqueIndex:idx_user_date_shift"`
}
