package models

import "gorm.io/gorm"

// ShiftRestriction 員工班別限制
// 記錄某位員工在某個循環模板中的排班限制
type ShiftRestriction struct {
	gorm.Model
	EmployeeID uint   `gorm:"not null;index" json:"employee_id"`                              // 員工 ID
	TemplateID *uint  `gorm:"index" json:"template_id"`                                       // 所屬循環模板 ID (null=全域預設)
	ShiftType  string `gorm:"type:varchar(20);not null" json:"shift_type"`                    // 限制的班別 (day/day88/evening/night)
	MaxDays    *int   `json:"max_days"`                                                        // 最多排幾天 (null=完全禁止)
	Note       string `gorm:"type:varchar(200)" json:"note"`                                   // 備註
}
