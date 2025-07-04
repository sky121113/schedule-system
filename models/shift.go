package models

import "gorm.io/gorm"

type Shift struct {
	gorm.Model
	Name      string `json:"name"`       // 班別名稱
	StartTime string `json:"start_time"` // 格式: "HH:MM:SS"
	EndTime   string `json:"end_time"`   // 格式: "HH:MM:SS"
}
