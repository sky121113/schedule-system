package db

import (
	"log"
	// "schedule-system/config"
	"schedule-system/models"
)

// RunMigrations 專門跑一次建表
func RunMigrations() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Shift{},
		// &models.Schedule{},
	)
	if err != nil {
		log.Fatal("資料庫建表失敗:", err)
	}
	log.Println("✅ 資料表已建立/更新完成")
}
