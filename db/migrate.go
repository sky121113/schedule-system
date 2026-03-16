package db

import (
	"log"
	"schedule-system/models"
)

// RunMigrations 自動遷移所有資料表
func RunMigrations() {
	err := DB.AutoMigrate(
		&models.Employee{},
		&models.ShiftRestriction{},
		&models.StaffingRequirement{},
		&models.CycleTemplate{},
		&models.TemplateSlot{},
		&models.PreScheduledLeave{},
		&models.MonthlySchedule{},
		&models.MonthlySlot{},
		&models.CycleLeaveBalance{},
		&models.MonthlyPreScheduledLeave{},
	)
	if err != nil {
		log.Fatal("❌ 資料庫建表失敗:", err)
	}
	log.Println("✅ 資料表已建立/更新完成")
}
