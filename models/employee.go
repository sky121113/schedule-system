package models

import "gorm.io/gorm"

// Employee 員工模型
type Employee struct {
	gorm.Model
	Name           string `gorm:"type:varchar(50);not null" json:"name"`                       // 姓名
	Email          string `gorm:"type:varchar(100);uniqueIndex" json:"email"`                   // 電子郵件
	IsDay88Primary bool   `gorm:"default:false" json:"is_day88_primary"`                        // 是否為 8-8 主力
	Status         int    `gorm:"default:1" json:"status"`                                      // 狀態 (1=在職, 0=停用, 2=長期請假)
}
