package config

// import (
// 	"log"

// 	"gorm.io/driver/mysql"
// 	"gorm.io/gorm"
// )

// var DB *gorm.DB

// func ConnectDB() {
// 	dsn := "root:root1234@tcp(127.0.0.1:3306)/schedule_system?charset=utf8mb4&parseTime=True&loc=Local"
// 	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
// 	if err != nil {
// 		log.Fatal("❌ 資料庫連線失敗: ", err)
// 	}
// 	log.Println("✅ 資料庫連線成功")
// 	DB = db
// }
