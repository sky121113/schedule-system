package db

import (
	"log"
	"schedule-system/models"
)

// SeedData 初始化種子資料 (員工、限制、人力需求)
func SeedData() {
	// 檢查是否已有資料
	var count int64
	DB.Model(&models.Employee{}).Count(&count)
	if count > 0 {
		log.Println("ℹ️ 資料已存在，跳過種子資料")
		return
	}

	log.Println("🌱 開始建立種子資料...")

	// 建立 10 位員工
	employees := []models.Employee{
		{Name: "A", Email: "a@example.com", Status: 1},
		{Name: "B", Email: "b@example.com", Status: 1},
		{Name: "C", Email: "c@example.com", Status: 1},
		{Name: "D", Email: "d@example.com", Status: 1},
		{Name: "E", Email: "e@example.com", Status: 1},
		{Name: "F", Email: "f@example.com", Status: 1},
		{Name: "G", Email: "g@example.com", Status: 2}, // 病假
		{Name: "H", Email: "h@example.com", Status: 1},
		{Name: "I", Email: "i@example.com", Status: 1},
		{Name: "J", Email: "j@example.com", IsDay88Primary: true, Status: 1}, // 8-8 主力
	}

	for i := range employees {
		DB.Create(&employees[i])
	}

	// 建立限制
	// A: 不能小夜, 白班≤5天
	fiveDays := 5
	sixDays := 6
	fiveNight := 5

	restrictions := []models.ShiftRestriction{
		// A: 禁小夜, 白班≤5
		{EmployeeID: employees[0].ID, ShiftType: "evening", Note: "A 不能排小夜"},
		{EmployeeID: employees[0].ID, ShiftType: "day", MaxDays: &fiveDays, Note: "A 白班≤5天"},
		// B: 禁小夜, 白班≤6
		{EmployeeID: employees[1].ID, ShiftType: "evening", Note: "B 不能排小夜"},
		{EmployeeID: employees[1].ID, ShiftType: "day", MaxDays: &sixDays, Note: "B 白班≤6天"},
		// C: 禁大夜
		{EmployeeID: employees[2].ID, ShiftType: "night", Note: "C 不能排大夜"},
		// D: 禁小夜, 白班≤6
		{EmployeeID: employees[3].ID, ShiftType: "evening", Note: "D 不能排小夜"},
		{EmployeeID: employees[3].ID, ShiftType: "day", MaxDays: &sixDays, Note: "D 白班≤6天"},
		// E: 禁大夜
		{EmployeeID: employees[4].ID, ShiftType: "night", Note: "E 不能排大夜"},
		// F: 禁大夜
		{EmployeeID: employees[5].ID, ShiftType: "night", Note: "F 不能排大夜"},
		// G 病假不需要限制 (status=2 即可)
		// H: 禁小夜 + 禁大夜
		{EmployeeID: employees[7].ID, ShiftType: "evening", Note: "H 不能排小夜"},
		{EmployeeID: employees[7].ID, ShiftType: "night", Note: "H 不能排大夜"},
		// I: 禁小夜, 大夜≤5
		{EmployeeID: employees[8].ID, ShiftType: "evening", Note: "I 不能排小夜"},
		{EmployeeID: employees[8].ID, ShiftType: "night", MaxDays: &fiveNight, Note: "I 大夜≤5天"},
		// J: 固定 8-8 (is_day88_primary 已設)
	}

	for i := range restrictions {
		DB.Create(&restrictions[i])
	}

	// 建立人力需求
	staffingReqs := []models.StaffingRequirement{
		// 日: 白3 小夜2(有8-8為1) 大夜2(有8-8為1)
		{Weekday: 0, ShiftType: "day", MinCount: 3, MinCountWithDay88: 3},
		{Weekday: 0, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 0, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
		// 一: 白4
		{Weekday: 1, ShiftType: "day", MinCount: 4, MinCountWithDay88: 4},
		{Weekday: 1, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 1, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
		// 二: 白5
		{Weekday: 2, ShiftType: "day", MinCount: 5, MinCountWithDay88: 5},
		{Weekday: 2, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 2, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
		// 三: 白4
		{Weekday: 3, ShiftType: "day", MinCount: 4, MinCountWithDay88: 4},
		{Weekday: 3, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 3, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
		// 四: 白4
		{Weekday: 4, ShiftType: "day", MinCount: 4, MinCountWithDay88: 4},
		{Weekday: 4, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 4, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
		// 五: 白5
		{Weekday: 5, ShiftType: "day", MinCount: 5, MinCountWithDay88: 5},
		{Weekday: 5, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 5, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
		// 六: 白4
		{Weekday: 6, ShiftType: "day", MinCount: 4, MinCountWithDay88: 4},
		{Weekday: 6, ShiftType: "evening", MinCount: 2, MinCountWithDay88: 1},
		{Weekday: 6, ShiftType: "night", MinCount: 2, MinCountWithDay88: 1},
	}

	for i := range staffingReqs {
		DB.Create(&staffingReqs[i])
	}

	log.Println("✅ 種子資料建立完成 (10位員工 + 限制 + 人力需求)")
}
