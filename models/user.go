package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name   string `json:"name"`
	Email  string `json:"email" gorm:"unique"`
	Role   string `json:"role"`
	Status int    `json:"status" gorm:"default:1"` // 1=啟用, 0=停用
}
