package controllers

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"schedule-system/db"
	"schedule-system/models"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// cycleStartDate 系統循環起算日
var cycleStartDate = time.Date(2026, 3, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))

// cycleQuota 循環假期配額
type cycleQuota struct {
	CycleIndex     int          `json:"cycle_index"`
	DaysInMonth    int          `json:"days_in_month"`
	PerPersonLeave map[uint]int `json:"per_person_leave"`
}

// 行政院 2026 紅字天數 (含週末、國平假日、補假)
var taiwanHolidays2026 = map[int]int{
	1:  10, // 03/15 - 04/11 (清明)
	2:  9,  // 04/12 - 05/09 (勞動)
	3:  8,  // 05/10 - 06/06
	4:  9,  // 06/07 - 07/04 (端午)
	5:  8,  // 07/05 - 08/01
	6:  8,  // 08/02 - 08/29
	7:  9,  // 08/30 - 09/26 (中秋)
	8:  10, // 09/27 - 10/24 (國慶)
	9:  9,  // 10/25 - 11/21
	10: 8,  // 11/22 - 12/19
	11: 10, // 12/20 - 01/16 (元旦)
}

// --- 月度班表 API ---

// GetMonthlySchedule 取得月度班表
func GetMonthlySchedule(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	var schedule models.MonthlySchedule
	if err := db.DB.Where("year = ? AND month = ?", year, month).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "尚未建立該月班表"})
		return
	}

	var slots []models.MonthlySlot
	db.DB.Where("schedule_id = ?", schedule.ID).Order("date ASC, employee_id ASC").Find(&slots)

	// 取得員工名稱
	var employees []models.Employee
	db.DB.Where("status = 1").Find(&employees)
	empMap := make(map[uint]string)
	for _, e := range employees {
		empMap[e.ID] = e.Name
	}

	// 計算循環分界資訊
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)
	boundaries := calcCycleBoundaries(firstDay, lastDay)

	// 取得人力需求與計算警告
	var requirements []models.StaffingRequirement
	db.DB.Find(&requirements)
	reqMap := buildRequirementMap(requirements)

	totalDays := lastDay.Day()
	loc := time.FixedZone("CST", 8*3600)
	scheduleMap := make(map[int]map[uint]string)
	for d := 0; d < totalDays; d++ {
		scheduleMap[d] = make(map[uint]string)
	}
	for _, s := range slots {
		dIdx := int(s.Date.In(loc).Sub(firstDay).Hours() / 24)
		if dIdx >= 0 && dIdx < totalDays {
			scheduleMap[dIdx][s.EmployeeID] = s.ShiftType
		}
	}

	// 抓取前一天的班別
	prevDay := firstDay.AddDate(0, 0, -1)
	var prevDaySlots []models.MonthlySlot
	db.DB.Where("date = ?", prevDay).Find(&prevDaySlots)
	prevDaySchedule := make(map[uint]string)
	for _, ps := range prevDaySlots {
		prevDaySchedule[ps.EmployeeID] = ps.ShiftType
	}

	warnings := calculateWarnings(firstDay, totalDays, employees, reqMap, scheduleMap, prevDaySchedule)
	if warnings == nil {
		warnings = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"schedule":     schedule,
		"slots":        slots,
		"employees":    empMap,
		"boundaries":   boundaries,
		"requirements": requirements,
		"warnings":     warnings,
	})
}

// CycleBoundary 循環分界資訊
type CycleBoundary struct {
	CycleIndex        int    `json:"cycle_index"`
	StartDate         string `json:"start_date"`          // 該循環在本月內的起始日
	EndDate           string `json:"end_date"`            // 該循環在本月內的結束日
	DaysInMonth       int    `json:"days_in_month"`       // 該循環在本月佔幾天
	TotalDays         int    `json:"total_days"`          // 該循環總天數 (28)
	DefaultTotalLeave int    `json:"default_total_leave"` // 該循環的預設總假日
}

// calcCycleBoundaries 計算一個月涉及的循環及其分界
func calcCycleBoundaries(firstDay, lastDay time.Time) []CycleBoundary {
	var boundaries []CycleBoundary
	current := firstDay

	for !current.After(lastDay) {
		daysSinceStart := int(current.Sub(cycleStartDate).Hours() / 24)
		cycleIndex := daysSinceStart/28 + 1
		dayOffset := daysSinceStart % 28
		if dayOffset < 0 {
			dayOffset += 28
			cycleIndex--
		}

		// 計算這個循環在本月的結束日
		remainingInCycle := 27 - dayOffset // 此循環剩餘天數
		cycleEndInMonth := current.AddDate(0, 0, remainingInCycle)
		if cycleEndInMonth.After(lastDay) {
			cycleEndInMonth = lastDay
		}

		daysInMonth := int(cycleEndInMonth.Sub(current).Hours()/24) + 1

		boundaries = append(boundaries, CycleBoundary{
			CycleIndex:        cycleIndex,
			StartDate:         current.Format("2006-01-02"),
			EndDate:           cycleEndInMonth.Format("2006-01-02"),
			DaysInMonth:       daysInMonth,
			TotalDays:         28,
			DefaultTotalLeave: taiwanHolidays2026[cycleIndex],
		})

		current = cycleEndInMonth.AddDate(0, 0, 1)
	}

	return boundaries
}

// GenerateMonthlySchedule 產出月度班表 (核心)
func GenerateMonthlySchedule(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	// 讀取初始假期設定 (逐人)
	type InitLeaveInput struct {
		CycleBalances []struct {
			CycleIndex int  `json:"cycle_index"`
			EmployeeID uint `json:"employee_id"`
			TotalLeave int  `json:"total_leave"`
		} `json:"cycle_balances"`
	}
	var input InitLeaveInput
	c.ShouldBindJSON(&input)

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)
	totalDaysInMonth := lastDay.Day()

	// 取得員工
	var employees []models.Employee
	db.DB.Where("status = 1").Find(&employees)

	// 取得人力需求
	var requirements []models.StaffingRequirement
	db.DB.Find(&requirements)
	reqMap := buildRequirementMap(requirements)

	// 取得限制
	var restrictions []models.ShiftRestriction
	db.DB.Find(&restrictions)
	constraints := buildConstraints(employees, restrictions)

	// 計算循環分界
	boundaries := calcCycleBoundaries(firstDay, lastDay)

	log.Printf("[Monthly] 產出 %d/%d 月度班表，共 %d 天，涉及 %d 個循環", year, month, totalDaysInMonth, len(boundaries))

	// 處理初始假期餘額 (通常僅針對 C1)
	log.Printf("[DEBUG] 收到初始假期設定，共 %d 筆", len(input.CycleBalances))

	for _, ib := range input.CycleBalances {
		// 永遠使用系統預設的循環總假
		defaultTotal := taiwanHolidays2026[ib.CycleIndex]
		if defaultTotal == 0 {
			defaultTotal = calcCycleLeaveForEmployee(employees, requirements)
		}

		cycleEndDate := cycleStartDate.AddDate(0, 0, (ib.CycleIndex*28)-1)
		isEndingThisMonth := !cycleEndDate.After(lastDay)

		log.Printf("[DEBUG] 處理平衡更新 - EmpID:%d, Cycle:%d, InputLeave:%d, DefaultTotal:%d, IsEndingThisMonth:%v",
			ib.EmployeeID, ib.CycleIndex, ib.TotalLeave, defaultTotal, isEndingThisMonth)

		var balance models.CycleLeaveBalance
		err := db.DB.Where("cycle_index = ? AND employee_id = ?", ib.CycleIndex, ib.EmployeeID).First(&balance).Error

		if err == nil {
			// 已有記錄：絕對尊重資料庫現有數值，不覆寫 TotalLeave
			if isEndingThisMonth {
				// C1: 直接存使用者的手動輸入值
				balance.MonthQuota = ib.TotalLeave
			}
			db.DB.Save(&balance)
		} else {
			// 新建記錄
			monthQuota := -1
			if isEndingThisMonth {
				monthQuota = ib.TotalLeave
			}
			balance = models.CycleLeaveBalance{
				CycleIndex: ib.CycleIndex,
				EmployeeID: ib.EmployeeID,
				TotalLeave: defaultTotal,
				UsedLeave:  0,
				MonthQuota: monthQuota,
			}
			db.DB.Create(&balance)
		}
	}

	// 為每個循環計算假期配額
	var quotas []cycleQuota
	for _, b := range boundaries {
		ratio := float64(b.DaysInMonth) / float64(b.TotalDays)
		q := cycleQuota{
			CycleIndex:     b.CycleIndex,
			DaysInMonth:    b.DaysInMonth,
			PerPersonLeave: make(map[uint]int),
		}

		for _, emp := range employees {
			if emp.IsDay88Primary {
				q.PerPersonLeave[emp.ID] = 0 // J 不放假
				continue
			}

			// 查詢目前的假期平衡
			var balance models.CycleLeaveBalance
			err := db.DB.Where("cycle_index = ? AND employee_id = ?", b.CycleIndex, emp.ID).First(&balance).Error

			if err != nil {
				// 完全沒記錄，建立一筆預設的
				totalCycleLeave := taiwanHolidays2026[b.CycleIndex]
				if totalCycleLeave == 0 {
					totalCycleLeave = calcCycleLeaveForEmployee(employees, requirements)
				}
				balance = models.CycleLeaveBalance{
					CycleIndex: b.CycleIndex,
					EmployeeID: emp.ID,
					TotalLeave: totalCycleLeave,
					UsedLeave:  0,
					MonthQuota: -1,
				}
				db.DB.Create(&balance)
			}

			// 使用 MonthQuota（使用者手動指定）或按比例分配
			thisMonthLeave := 0
			if balance.MonthQuota >= 0 {
				// 使用者有手動設定本月應休天數（例如 C1）
				thisMonthLeave = balance.MonthQuota
			} else {
				// 系統按比例分配：直接用 TotalLeave * ratio，每個循環的假期是獨立的
				cycleEndDate := cycleStartDate.AddDate(0, 0, (b.CycleIndex*28)-1)
				if !cycleEndDate.After(lastDay) {
					// 循環在本月結束，剩下的假期全部要在本月排完
					thisMonthLeave = balance.TotalLeave - balance.UsedLeave
				} else {
					// 循環跨到下個月，按本月佔比分配
					thisMonthLeave = int(math.Round(float64(balance.TotalLeave) * ratio))
				}
			}

			if thisMonthLeave < 0 {
				thisMonthLeave = 0
			}
			if thisMonthLeave > b.DaysInMonth {
				thisMonthLeave = b.DaysInMonth
			}
			q.PerPersonLeave[emp.ID] = thisMonthLeave
		}
		quotas = append(quotas, q)
	}

	// 預先抓取該月份的具體日期預假
	monthlyPreLeaves := []models.MonthlyPreScheduledLeave{}
	db.DB.Where("date BETWEEN ? AND ?", firstDay, lastDay).Find(&monthlyPreLeaves)

	// 也抓取對應模板中的循環預假，並轉換為具體日期
	templatePreLeaves := []models.PreScheduledLeave{}
	for _, b := range boundaries {
		// 尋找對應啟動日期的模板
		cycleStart := cycleStartDate.AddDate(0, 0, (b.CycleIndex-1)*28)
		var t models.CycleTemplate
		if db.DB.Where("start_date = ?", cycleStart).First(&t).Error == nil {
			var leaves []models.PreScheduledLeave
			db.DB.Where("template_id = ?", t.ID).Find(&leaves)
			templatePreLeaves = append(templatePreLeaves, leaves...)
		}
	}

	// 抓取前一天的班別 (處理跨月邊界)
	prevDay := firstDay.AddDate(0, 0, -1)
	var prevDaySlots []models.MonthlySlot
	db.DB.Where("date = ?", prevDay).Find(&prevDaySlots)
	prevDaySchedule := make(map[uint]string)
	for _, ps := range prevDaySlots {
		prevDaySchedule[ps.EmployeeID] = ps.ShiftType
	}

	// 執行排班 (傳入預假與前日資料)
	slots, warnings := runMonthlySchedule(firstDay, totalDaysInMonth, employees, constraints, reqMap, boundaries, quotas, monthlyPreLeaves, templatePreLeaves, prevDaySchedule)

	// 儲存到資料庫
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// 建立或覆蓋月度班表
		var schedule models.MonthlySchedule
		tx.Where("year = ? AND month = ?", year, month).FirstOrCreate(&schedule, models.MonthlySchedule{
			Year:  year,
			Month: month,
		})

		// 清除舊的 slots
		tx.Unscoped().Where("schedule_id = ?", schedule.ID).Delete(&models.MonthlySlot{})

		// 寫入新的 slots
		for i := range slots {
			slots[i].ScheduleID = schedule.ID
			if err := tx.Create(&slots[i]).Error; err != nil {
				return err
			}
		}

		// 更新循環假期已使用量 (重新計算該循環在所有月份的總量)
		for _, q := range quotas {
			for empID := range q.PerPersonLeave {
				var count int64
				tx.Model(&models.MonthlySlot{}).
					Where("cycle_index = ? AND employee_id = ? AND shift_type = 'off'", q.CycleIndex, empID).
					Count(&count)

				var bal models.CycleLeaveBalance
				tx.Where("cycle_index = ? AND employee_id = ?", q.CycleIndex, empID).First(&bal)

				tx.Model(&models.CycleLeaveBalance{}).
					Where("cycle_index = ? AND employee_id = ?", q.CycleIndex, empID).
					Update("used_leave", int(count)+bal.OfflineUsed)
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "儲存班表失敗: " + err.Error()})
		return
	}

	// 取得假期摘要
	summaries := generateLeaveSummaries(year, month)

	c.JSON(http.StatusOK, gin.H{
		"message":    fmt.Sprintf("%d/%d 月度班表產出成功", year, month),
		"slots":      slots,
		"warnings":   warnings,
		"boundaries": boundaries,
		"quotas":     quotas,
		"summaries":  summaries,
	})
}

// calcCycleLeaveForEmployee 計算單一員工在一個循環中的假期天數
func calcCycleLeaveForEmployee(employees []models.Employee, requirements []models.StaffingRequirement) int {
	// 計算一個循環 (28天) 的總工作需求
	weeklyRequired := 0
	for _, r := range requirements {
		weeklyRequired += r.MinCountWithDay88
	}
	cycleRequired := weeklyRequired * 4

	// 可用員工（排除 Day88 主力）
	activeCount := 0
	for _, e := range employees {
		if !e.IsDay88Primary {
			activeCount++
		}
	}

	if activeCount == 0 {
		return 0
	}

	totalAvailable := activeCount * 28
	totalLeave := totalAvailable - cycleRequired
	if totalLeave < 0 {
		totalLeave = 0
	}
	perPerson := int(math.Round(float64(totalLeave) / float64(activeCount)))
	return perPerson
}

// runMonthlySchedule 月度排班核心演算法
// ⭐ 直接復用 v6 核心函式（canAssignV6、fillConsecutiveV3、findBestCandidateV3）
func runMonthlySchedule(
	firstDay time.Time,
	totalDays int,
	employees []models.Employee,
	constraints map[uint]*employeeConstraint,
	reqMap map[requirementKey]models.StaffingRequirement,
	boundaries []CycleBoundary,
	quotas []cycleQuota,
	monthlyPreLeaves []models.MonthlyPreScheduledLeave,
	templatePreLeaves []models.PreScheduledLeave,
	prevDaySchedule map[uint]string,
) ([]models.MonthlySlot, []string) {

	log.Printf("[Monthly] runMonthlySchedule 開始: %d 天, %d 名員工, 預假數: %d", totalDays, len(employees), len(monthlyPreLeaves))

	// schedule[day][empID] = shiftType
	schedule := make(map[int]map[uint]string)
	shiftCount := make(map[uint]map[string]int)
	for d := 0; d < totalDays; d++ {
		schedule[d] = make(map[uint]string)
	}
	for _, emp := range employees {
		shiftCount[emp.ID] = make(map[string]int)
	}

	// ─── 步驟 0：鎖定具體日期預假 ⭐ (最高優先) ───
	log.Println("[Monthly] 步驟 0: 鎖定預假")
	preLeaveCount := make(map[uint]int) // 追蹤每人已使用的預假天數（用於從配額扣除）

	// 建立日期索引 Map
	dateToIndex := make(map[string]int)
	for d := 0; d < totalDays; d++ {
		dateStr := firstDay.AddDate(0, 0, d).Format("2006-01-02")
		dateToIndex[dateStr] = d
	}

	for _, pl := range monthlyPreLeaves {
		dateStr := pl.Date.Format("2006-01-02")
		if idx, ok := dateToIndex[dateStr]; ok {
			log.Printf("[DEBUG] 正在鎖定日期預假: Date=%s, Index=%d, EmpID=%d", dateStr, idx, pl.EmployeeID)
			schedule[idx][pl.EmployeeID] = "pre_off" // 標記為預先鎖定
			log.Printf("[DEBUG] 鎖定後狀態: schedule[%d][%d] = %s", idx, pl.EmployeeID, schedule[idx][pl.EmployeeID])
			shiftCount[pl.EmployeeID]["off"]++
			preLeaveCount[pl.EmployeeID]++
			log.Printf("[Monthly] 鎖定硬約束(日期預假): Day %d, Emp ID %d (%s)", idx, pl.EmployeeID, dateStr)
		} else {
			log.Printf("[Monthly] 警告: 預假日期 %s 超出範圍", dateStr)
		}
	}

	// 鎖定模板預假 (循環偏移量)
	// 在 GenerateMonthlySchedule 中我們已經按循環搜過模板了。
	// 我們可以直接把 templatePreLeaves 轉換為具體日期索引。

	// 修正：在 GenerateMonthlySchedule 中我們已經按循環搜過模板了。
	// 我們可以直接把 templatePreLeaves 轉換為具體日期索引。
	for _, pl := range templatePreLeaves {
		// 找出該模板對應的循環起始日
		var t models.CycleTemplate
		if db.DB.First(&t, pl.TemplateID).Error != nil {
			continue
		}
		targetDate := t.StartDate.AddDate(0, 0, pl.DayOffset)
		dateStr := targetDate.Format("2006-01-02")
		if idx, ok := dateToIndex[dateStr]; ok {
			if schedule[idx][pl.EmployeeID] == "" {
				schedule[idx][pl.EmployeeID] = "pre_off"
				shiftCount[pl.EmployeeID]["off"]++
				preLeaveCount[pl.EmployeeID]++
				log.Printf("[Monthly] 鎖定硬約束(模板預假): Day %d, Emp ID %d (%s)", idx, pl.EmployeeID, dateStr)
			}
		}
	}

	// ─── 步驟 1：分配其餘假期（Round-Robin 輪流分配）⭐ ───
	log.Println("[Monthly] 步驟 1: 分配假期 (Round-Robin)")
	dayOffset := 0
	for qi, q := range quotas {
		boundary := boundaries[qi]
		segmentDays := boundary.DaysInMonth

		// 計算按需求排序的日期清單（低需求優先放假）
		type dayNeed struct {
			day  int
			need int
		}
		var dayNeeds []dayNeed
		for d := dayOffset; d < dayOffset+segmentDays; d++ {
			date := firstDay.AddDate(0, 0, d)
			weekday := int(date.Weekday())
			totalNeed := 0
			for _, st := range []string{"day", "evening", "night"} {
				if req, ok := reqMap[requirementKey{weekday, st}]; ok {
					totalNeed += req.MinCountWithDay88
				}
			}
			dayNeeds = append(dayNeeds, dayNeed{d, totalNeed})
		}
		sort.Slice(dayNeeds, func(i, j int) bool {
			return dayNeeds[i].need < dayNeeds[j].need
		})

		// 計算每人的剩餘假期配額
		remaining := make(map[uint]int)
		for _, emp := range employees {
			segmentPreLeaveCount := 0
			for d := dayOffset; d < dayOffset+segmentDays; d++ {
				if schedule[d][emp.ID] == "pre_off" {
					segmentPreLeaveCount++
				}
			}
			leaveCount := q.PerPersonLeave[emp.ID] - segmentPreLeaveCount
			if leaveCount > 0 {
				remaining[emp.ID] = leaveCount
			}
		}

		// Round-Robin：每輪每人挑 1 天假，直到所有人配額用完
		for {
			anyAssigned := false

			// 隨機打亂員工順序，避免排序靠前的 A、B 永遠優先選到低需求日
			shuffledEmployees := make([]models.Employee, len(employees))
			copy(shuffledEmployees, employees)
			rand.Shuffle(len(shuffledEmployees), func(i, j int) {
				shuffledEmployees[i], shuffledEmployees[j] = shuffledEmployees[j], shuffledEmployees[i]
			})

			for _, emp := range shuffledEmployees {
				if remaining[emp.ID] <= 0 {
					continue
				}
				// 從低需求日開始，找一個可以放假的天
				assigned := false
				for _, dn := range dayNeeds {
					if schedule[dn.day][emp.ID] != "" {
						continue
					}
					// 避免連續兩天假
					if dn.day > 0 && schedule[dn.day-1][emp.ID] == "off" {
						continue
					}
					if dn.day+1 < dayOffset+segmentDays && schedule[dn.day+1][emp.ID] == "off" {
						continue
					}
					// 人力需求保護
					offCount := 0
					for _, otherEmp := range employees {
						if schedule[dn.day][otherEmp.ID] == "off" || schedule[dn.day][otherEmp.ID] == "pre_off" {
							offCount++
						}
					}
					activeCount := len(employees) - (offCount + 1)
					if activeCount < dn.need {
						continue
					}
					schedule[dn.day][emp.ID] = "off"
					shiftCount[emp.ID]["off"]++
					remaining[emp.ID]--
					assigned = true
					anyAssigned = true
					break // 本輪只挑 1 天
				}
				// 降級：取消間隔限制
				if !assigned && remaining[emp.ID] > 0 {
					for _, dn := range dayNeeds {
						if schedule[dn.day][emp.ID] != "" {
							continue
						}
						offCount := 0
						for _, otherEmp := range employees {
							if schedule[dn.day][otherEmp.ID] == "off" || schedule[dn.day][otherEmp.ID] == "pre_off" {
								offCount++
							}
						}
						activeCount := len(employees) - (offCount + 1)
						if activeCount < dn.need {
							continue
						}
						schedule[dn.day][emp.ID] = "off"
						shiftCount[emp.ID]["off"]++
						remaining[emp.ID]--
						anyAssigned = true
						break
					}
				}
			}
			if !anyAssigned {
				break // 所有人都分配完畢或無法再分配
			}
		}
		dayOffset += segmentDays
	}

	// ─── 步驟 2：排 J (8-8 主力) — 含 R7 做 6 休 1 ⭐ ───
	log.Println("[Monthly] 步驟 2: 排 J (含 R7)")
	for _, emp := range employees {
		if !emp.IsDay88Primary {
			continue
		}
		// 計算前月連續工作天數（跨月邊界）
		streak := 0
		if prevDaySchedule != nil {
			prev := prevDaySchedule[emp.ID]
			if prev != "" && prev != "off" && prev != "pre_off" {
				streak = 1 // 至少前一天有上班，保守起見算 1
			}
		}
		for d := 0; d < totalDays; d++ {
			if schedule[d][emp.ID] == "off" || schedule[d][emp.ID] == "pre_off" {
				streak = 0
				continue
			}
			if streak >= 6 {
				// R7: 做 6 休 1 — J 的休假直接是 off
				schedule[d][emp.ID] = "off"
				streak = 0
				continue
			}
			schedule[d][emp.ID] = "day88"
			shiftCount[emp.ID]["day88"]++
			streak++
		}
	}

	// 計算大夜、小夜的專責人力需求（復用 v6 邏輯）
	maxNightReq := 0
	maxEveningReq := 0
	for d := 0; d < 7; d++ {
		if r, ok := reqMap[requirementKey{d, "night"}]; ok {
			if r.MinCountWithDay88 > maxNightReq {
				maxNightReq = r.MinCountWithDay88
			}
		}
		if r, ok := reqMap[requirementKey{d, "evening"}]; ok {
			if r.MinCountWithDay88 > maxEveningReq {
				maxEveningReq = r.MinCountWithDay88
			}
		}
	}
	nightEmpsNeeded := (maxNightReq * 6) / 4
	if (maxNightReq*6)%4 != 0 {
		nightEmpsNeeded++
	}
	eveningEmpsNeeded := (maxEveningReq * 5) / 4
	if (maxEveningReq*5)%4 != 0 {
		eveningEmpsNeeded++
	}

	// ─── 步驟 3：排大夜連續班段（直接復用 v6 fillConsecutiveV3）───
	log.Printf("[Monthly] 步驟 3: 排大夜, 需要人數: %d", nightEmpsNeeded)
	fillConsecutiveV3(schedule, shiftCount, employees, constraints, "night", totalDays, 4, 2, nightEmpsNeeded, prevDaySchedule)

	// ─── 步驟 3.5：升級大夜為 night88（R8 處理）⭐ ───
	// 逐日檢查，若當日有 day88，則需一位 night88
	// 優先尋找已排定 night 且昨日非 night88 的員工升級
	log.Println("[Monthly] 步驟 3.5: 升級大夜為 night88")
	for d := 0; d < totalDays; d++ {
		// 檢查當天是否有 day88
		hasDay88 := false
		for _, emp := range employees {
			if schedule[d][emp.ID] == "day88" {
				hasDay88 = true
				break
			}
		}
		if !hasDay88 {
			continue // 當天沒有 day88，不需要 night88
		}

		// 當天已有 night88 則跳過
		hasNight88 := false
		for _, emp := range employees {
			if schedule[d][emp.ID] == "night88" {
				hasNight88 = true
				break
			}
		}
		if hasNight88 {
			continue
		}

		// 從已排 night 的員工中選一人升級為 night88
		upgraded := false
		var nightEmpIDs []uint
		for _, emp := range employees {
			if schedule[d][emp.ID] == "night" {
				nightEmpIDs = append(nightEmpIDs, emp.ID)
			}
		}

		for _, empID := range nightEmpIDs {
			// R8: 前一天不可以是 night88
			prevIsNight88 := false
			if d > 0 {
				prevIsNight88 = schedule[d-1][empID] == "night88"
			} else if d == 0 && prevDaySchedule != nil {
				prevIsNight88 = prevDaySchedule[empID] == "night88"
			}
			if prevIsNight88 {
				continue
			}
			// R8: 後一天也不可以是 night88
			if d+1 < totalDays && schedule[d+1][empID] == "night88" {
				continue
			}

			// 升級：night → night88
			schedule[d][empID] = "night88"
			shiftCount[empID]["night"]--
			shiftCount[empID]["night88"]++
			upgraded = true
			log.Printf("[R8-UPGRADE] Day %d: 員工 %d 從 night 升級為 night88", d, empID)
			break
		}

		// 若無法從已排 night 的人升級，嘗試補位一位新的 night88
		// 三層優先序：
		//   (a) 找前後已有 night/night88 的人（天然滿足 R7）
		//   (b) 配對分配：night88 + 相鄰天 night（確保 R7 連續 ≥ 2 天）
		//   (c) 最終退路：單獨分配 night88（可能被 step 6.5 移除，但至少嘗試）
		if !upgraded {
			var fallbackID uint
			// (a) 找前後已有 night/night88 的人
			for _, emp := range employees {
				if emp.IsDay88Primary {
					continue
				}
				hasAdjacentNight := false
				if d > 0 {
					prev := schedule[d-1][emp.ID]
					if prev == "night" || prev == "night88" {
						hasAdjacentNight = true
					}
				}
				if d+1 < totalDays {
					next := schedule[d+1][emp.ID]
					if next == "night" || next == "night88" {
						hasAdjacentNight = true
					}
				}
				if !hasAdjacentNight {
					continue
				}
				if canAssignV6(emp.ID, schedule, shiftCount, constraints, d, "night88", totalDays, prevDaySchedule) {
					fallbackID = emp.ID
					break
				}
			}

			// (b) 配對分配：同時排 night88 + 相鄰天 night，確保 R7
			if fallbackID == 0 {
				for _, emp := range employees {
					if emp.IsDay88Primary {
						continue
					}
					if !canAssignV6(emp.ID, schedule, shiftCount, constraints, d, "night88", totalDays, prevDaySchedule) {
						continue
					}
					// 嘗試在隔天補一個 night（優先後一天）
					pairDay := -1
					if d+1 < totalDays && canAssignV6(emp.ID, schedule, shiftCount, constraints, d+1, "night", totalDays, prevDaySchedule) {
						pairDay = d + 1
					} else if d > 0 && schedule[d-1][emp.ID] == "" && canAssignV6(emp.ID, schedule, shiftCount, constraints, d-1, "night", totalDays, prevDaySchedule) {
						pairDay = d - 1
					}
					if pairDay >= 0 {
						fallbackID = emp.ID
						// 分配配對的 night
						schedule[pairDay][emp.ID] = "night"
						shiftCount[emp.ID]["night"]++
						log.Printf("[R8-PAIR] Day %d: 員工 %d 配對排 night (搭配 day %d 的 night88)", pairDay, emp.ID, d)
						break
					}
				}
			}

			// (c) 最終退路
			if fallbackID == 0 {
				var emptyPreLeaves []models.PreScheduledLeave
				fallbackID = findBestCandidateV3(employees, schedule, shiftCount, constraints, d, "night88", totalDays, emptyPreLeaves, prevDaySchedule)
			}

			if fallbackID != 0 {
				schedule[d][fallbackID] = "night88"
				shiftCount[fallbackID]["night88"]++
				log.Printf("[R8-FALLBACK] Day %d: 補位員工 %d 排 night88", d, fallbackID)
			} else {
				log.Printf("[R8-WARNING] Day %d: 有 day88 但無法安排 night88", d)
			}
		}
	}

	// ─── 步驟 4：排小夜連續班段（直接復用 v6 fillConsecutiveV3）───
	log.Printf("[Monthly] 步驟 4: 排小夜, 需要人數: %d", eveningEmpsNeeded)
	fillConsecutiveV3(schedule, shiftCount, employees, constraints, "evening", totalDays, 4, 1, eveningEmpsNeeded, prevDaySchedule)

	// DEBUG: 檢查特定日期之後的狀態
	for _, id := range []uint{9, 4} { // I=9, D=4
		log.Printf("[DEBUG-AFTER-STEP4] Emp %d: d=4(Apr5)=%s, d=5(Apr6)=%s, d=8(Apr9)=%s, d=9(Apr10)=%s",
			id, schedule[4][id], schedule[5][id], schedule[8][id], schedule[9][id])
	}

	// ─── 步驟 5：C7 強制 off 掃描 ⭐ ───
	log.Println("[Monthly] 步驟 5: C7 強制 off 掃描")
	for _, emp := range employees {
		// J 也適用 C7（做 6 休 1）
		streak := 0
		for d := 0; d < totalDays; d++ {
			s := schedule[d][emp.ID]
			if s != "" && s != "off" && s != "pre_off" {
				streak++
				if streak >= 6 {
					nextDay := d + 1
					if nextDay < totalDays {
						existing := schedule[nextDay][emp.ID]
						if existing != "off" && existing != "pre_off" && existing != "" {
							shiftCount[emp.ID][existing]--
							schedule[nextDay][emp.ID] = "off"
						} else if existing == "" {
							schedule[nextDay][emp.ID] = "off"
						}
						// 如果是 pre_off，則保留 pre_off，不覆蓋也不重複扣假
					}
					streak = 0
				}
			} else {
				streak = 0
			}
		}
	}

	// ─── 步驟 6：補位班次（直接復用 v6 findBestCandidateV3）⭐ ───
	log.Println("[Monthly] 步驟 6: 補位班次")
	// 月度班表沒有預假（已藉由假期分配處理），傳入空切片
	var emptyPreLeaves []models.PreScheduledLeave

	for d := 0; d < totalDays; d++ {
		weekday := int(firstDay.AddDate(0, 0, d).Weekday())
		has88 := false
		for _, s := range schedule[d] {
			if s == "day88" {
				has88 = true
				break
			}
		}

		// 依序檢查每個班別的需求
		for _, st := range []string{"night", "evening", "day"} {
			req := reqMap[requirementKey{weekday, st}]
			minNeeded := req.MinCount
			if st != "day" && has88 {
				minNeeded = req.MinCountWithDay88
			}

			current := 0
			for _, assigned := range schedule[d] {
				if assigned == st {
					current++
				}
				if st == "day" && assigned == "day88" {
					current++
				}
			}

			for n := current; n < minNeeded; n++ {
				bestID := findBestCandidateV3(employees, schedule, shiftCount, constraints, d, st, totalDays, emptyPreLeaves, prevDaySchedule)
				if bestID != 0 {
					schedule[d][bestID] = st
					shiftCount[bestID][st]++
				}
			}
		}
	}
	// ─── 步驟 6.5：R8 後處理 — 大小夜至少連續 2 天 ⭐ ───
	log.Println("[Monthly] 步驟 6.5: R8 後處理（大小夜至少連續 2 天）")
	for _, emp := range employees {
		if emp.IsDay88Primary {
			continue
		}
		for _, st := range []string{"night", "night88", "evening"} {
			for d := 0; d < totalDays; d++ {
				if schedule[d][emp.ID] != st {
					continue
				}
				// 檢查是否為孤立的 1 天班段
				// night 與 night88 屬於同家族，互相視為「相鄰同類」
				isNightFamily := (st == "night" || st == "night88")
				prevSame := false
				if d > 0 {
					prev := schedule[d-1][emp.ID]
					if isNightFamily {
						prevSame = prev == "night" || prev == "night88"
					} else {
						prevSame = prev == st
					}
				} else if d == 0 && prevDaySchedule != nil {
					prev := prevDaySchedule[emp.ID]
					if isNightFamily {
						prevSame = prev == "night" || prev == "night88"
					} else {
						prevSame = prev == st
					}
				}
				nextSame := false
				if d+1 < totalDays {
					next := schedule[d+1][emp.ID]
					if isNightFamily {
						nextSame = next == "night" || next == "night88"
					} else {
						nextSame = next == st
					}
				}

				if prevSame || nextSame {
					continue // 至少有一邊相鄰，不是孤立 1 天
				}
				// 孤立 1 天 — 嘗試延伸到隔天（night88 延伸用 night，避免連續 night88）
				extendSt := st
				if st == "night88" {
					extendSt = "night" // 延伸時用普通大夜，避免違反 R8（不可連續 night88）
				}
				extended := false
				if d+1 < totalDays && canAssignV6(emp.ID, schedule, shiftCount, constraints, d+1, extendSt, totalDays, prevDaySchedule) {
					schedule[d+1][emp.ID] = extendSt
					shiftCount[emp.ID][extendSt]++
					extended = true
				}
				if !extended {
					// 無法延伸 → 移除此孤立班段，讓步驟 7 重新分配
					log.Printf("[R7] 移除孤立 %s 班段: Day %d, Emp %d (%s)", st, d, emp.ID, emp.Name)
					schedule[d][emp.ID] = ""
					shiftCount[emp.ID][st]--
				}
			}
		}
	}

	// ─── 步驟 7：填充剩餘空格 ───
	log.Println("[Monthly] 步驟 7: 填充剩餘空格")
	for d := 0; d < totalDays; d++ {
		for _, emp := range employees {
			if schedule[d][emp.ID] == "" {
				can := canAssignV6(emp.ID, schedule, shiftCount, constraints, d, "day", totalDays, prevDaySchedule)
				if (d == 5 || d == 9) && (emp.ID == 9 || emp.ID == 4) {
					log.Printf("[DEBUG-STEP7-CHECK] Day %d: Emp %d, canAssign=%v, Prev=%s", d, emp.ID, can, schedule[d-1][emp.ID])
				}
				if can {
					schedule[d][emp.ID] = "day"
					shiftCount[emp.ID]["day"]++
				} else {
					// 如果不能排白班（例如前天是夜班），就排休
					schedule[d][emp.ID] = "off"
					shiftCount[emp.ID]["off"]++
				}
			}
		}
	}

	// ─── 步驟 7.5：J 最終清理 — 確保 J 只有 day88 或 off ⭐ ───
	log.Println("[Monthly] 步驟 7.5: J 最終清理")
	for _, emp := range employees {
		if !emp.IsDay88Primary {
			continue
		}
		for d := 0; d < totalDays; d++ {
			s := schedule[d][emp.ID]
			if s != "day88" && s != "off" && s != "pre_off" {
				log.Printf("[J-CLEANUP] Day %d: 將 '%s' 改為 'off' (Emp %d: %s)", d, s, emp.ID, emp.Name)
				if s != "" {
					shiftCount[emp.ID][s]--
				}
				schedule[d][emp.ID] = "off"
			}
		}
	}

	// ─── 步驟 8：人力不足檢查 (Warnings) ───
	warnings := calculateWarnings(firstDay, totalDays, employees, reqMap, schedule, prevDaySchedule)

	// ─── 轉換為 MonthlySlot ───
	var result []models.MonthlySlot
	for d := 0; d < totalDays; d++ {
		date := firstDay.AddDate(0, 0, d)
		daysSinceStart := int(date.Sub(cycleStartDate).Hours() / 24)
		cycIdx := daysSinceStart/28 + 1
		dOffset := daysSinceStart % 28
		if dOffset < 0 {
			dOffset += 28
			cycIdx--
		}

		for _, emp := range employees {
			shiftType := schedule[d][emp.ID]
			if d == 19 && (emp.ID == 1 || emp.ID == 2) {
				log.Printf("[TRANSFORM] Day 19 (3/20), EmpID %d: 原班別=%s", emp.ID, shiftType)
			}
			if shiftType == "" {
				if emp.IsDay88Primary {
					shiftType = "off" // J 的空格一律為 off，不排白班
				} else {
					shiftType = "day"
				}
			}
			if shiftType == "pre_off" {
				shiftType = "off"
			}
			result = append(result, models.MonthlySlot{
				Date:       date,
				ShiftType:  shiftType,
				EmployeeID: emp.ID,
				CycleIndex: cycIdx,
				DayOffset:  dOffset,
			})
		}
	}

	log.Printf("[Monthly] runMonthlySchedule 完成: 產出 %d 個 slots, 警告數: %d", len(result), len(warnings))
	return result, warnings
}

// calculateWarnings 獨立出人力不足的檢查邏輯
func calculateWarnings(firstDay time.Time, totalDays int, employees []models.Employee, reqMap map[requirementKey]models.StaffingRequirement, schedule map[int]map[uint]string, prevDaySchedule map[uint]string) []string {
	var warnings []string
	for d := 0; d < totalDays; d++ {
		date := firstDay.AddDate(0, 0, d)
		weekday := int(date.Weekday())

		// --- 額外：檢查夜班接白班之限制 (C1, C2) ---
		for _, emp := range employees {
			shift := schedule[d][emp.ID]
			if shift == "day" || shift == "day88" {
				prev := ""
				if d > 0 {
					prev = schedule[d-1][emp.ID]
				} else if d == 0 && prevDaySchedule != nil {
					prev = prevDaySchedule[emp.ID]
				}
				if isNightShift(prev) {
					warnings = append(warnings, fmt.Sprintf("%d/%02d/%02d 違反排班約束: %s 班接續前日夜班 (%s)",
						date.Year(), date.Month(), date.Day(), shift, emp.Name))
				}
			}
		}

		// 檢查總在職人數與 R8 約束 (night88 數量)
		hasDay88 := false
		actualNight88 := 0
		for _, emp := range employees {
			if schedule[d][emp.ID] == "day88" {
				hasDay88 = true
			}
			if schedule[d][emp.ID] == "night88" {
				actualNight88++
			}
		}

		if hasDay88 && actualNight88 == 0 {
			warn := fmt.Sprintf("%d/%02d/%02d 違反 R8 約束: 有安排白班 8-8，但無人安排大夜 8-8 (night88)",
				date.Year(), date.Month(), date.Day())
			warnings = append(warnings, warn)
		}

		// 檢查每個班別的需求
		for _, st := range []string{"day", "evening", "night"} {
			req := reqMap[requirementKey{weekday, st}]
			minNeeded := req.MinCount

			// 判斷是否有 Day88，影響 minNeededWithDay88
			if st != "day" && hasDay88 {
				minNeeded = req.MinCountWithDay88
			} else if st == "day" && hasDay88 {
				// Day班需求，如果Day88在，則Day88也算Day班人力
				minNeeded = req.MinCountWithDay88
			}

			current := 0
			for _, emp := range employees {
				if schedule[d][emp.ID] == st {
					current++
				}
				if st == "day" && schedule[d][emp.ID] == "day88" {
					current++
				}
				if st == "night" && schedule[d][emp.ID] == "night88" {
					current++
				}
			}

			if current < minNeeded {
				warn := fmt.Sprintf("%d/%02d/%02d %s班人力不足: 需求 %d 人, 實際 %d 人",
					date.Year(), date.Month(), date.Day(), st, minNeeded, current)
				warnings = append(warnings, warn)
			}
		}

		// 檢查總在職人數
		activeCount := 0
		for _, emp := range employees {
			if schedule[d][emp.ID] != "off" && schedule[d][emp.ID] != "pre_off" {
				activeCount++
			}
		}
		if activeCount < 2 { // 假設最低總在職人數為 2
			warn := fmt.Sprintf("%d/%02d/%02d 當日總在職人數過低: 實際 %d 人 (建議至少 2 人)",
				date.Year(), date.Month(), date.Day(), activeCount)
			warnings = append(warnings, warn)
		}
	}
	return warnings
}

// GetMonthlyLeaveSummary 獲取月度假期摘要
func GetMonthlyLeaveSummary(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)
	boundaries := calcCycleBoundaries(firstDay, lastDay)

	summaries := generateLeaveSummaries(year, month)

	c.JSON(http.StatusOK, gin.H{
		"year":       year,
		"month":      month,
		"boundaries": boundaries,
		"summaries":  summaries,
	})
}

// generateLeaveSummaries 將假期計算核心獨立出
func generateLeaveSummaries(year int, month int) []gin.H {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)
	boundaries := calcCycleBoundaries(firstDay, lastDay)

	var employees []models.Employee
	db.DB.Where("status = 1").Find(&employees)

	var summaries []gin.H
	for _, b := range boundaries {
		for _, emp := range employees {
			var balance models.CycleLeaveBalance
			if err := db.DB.Where("cycle_index = ? AND employee_id = ?", b.CycleIndex, emp.ID).First(&balance).Error; err != nil {
				// 若尚未有 balance 紀錄，賦予預設的循環總假供前端顯示
				defaultTotal := taiwanHolidays2026[b.CycleIndex]
				if defaultTotal == 0 {
					var reqs []models.StaffingRequirement
					db.DB.Find(&reqs)
					defaultTotal = calcCycleLeaveForEmployee(employees, reqs)
				}
				balance = models.CycleLeaveBalance{
					CycleIndex:  b.CycleIndex,
					EmployeeID:  emp.ID,
					TotalLeave:  defaultTotal,
					UsedLeave:   0,
					OfflineUsed: 0,
				}
			}

			isEndingThisMonth := !cycleStartDate.AddDate(0, 0, (b.CycleIndex*28)-1).After(lastDay)
			var currentMonthQuota int
			if balance.MonthQuota >= 0 {
				currentMonthQuota = balance.MonthQuota
			} else if isEndingThisMonth {
				currentMonthQuota = balance.TotalLeave - balance.UsedLeave
				if currentMonthQuota < 0 {
					currentMonthQuota = 0
				}
			} else {
				ratio := float64(b.DaysInMonth) / 28.0
				currentMonthQuota = int(math.Round(float64(balance.TotalLeave) * ratio))
				if currentMonthQuota > b.DaysInMonth {
					currentMonthQuota = b.DaysInMonth
				}
			}

			// 動態計算「循環累計已用」(只計算該循環中，查詢月份「之前」的排休天數)
			var usedBeforeThisMonth int64
			db.DB.Model(&models.MonthlySlot{}).
				Where("cycle_index = ? AND employee_id = ? AND shift_type = 'off' AND date < ?", b.CycleIndex, emp.ID, firstDay).
				Count(&usedBeforeThisMonth)

			// 加上外部額外紀錄的已用天數 (若有)
			actualUsedLeave := int(usedBeforeThisMonth) + balance.OfflineUsed

			summaries = append(summaries, gin.H{
				"employee_id":         emp.ID,
				"employee_name":       emp.Name,
				"cycle_index":         b.CycleIndex,
				"total_leave":         balance.TotalLeave,
				"used_leave":          actualUsedLeave,
				"remaining":           balance.TotalLeave - actualUsedLeave,
				"current_month_quota": currentMonthQuota,
			})
		}
	}
	return summaries
}

// UpdateCycleBalance 手動調整循環假期（逐人）
func UpdateCycleBalance(c *gin.Context) {
	type UpdateInput struct {
		CycleIndex int  `json:"cycle_index" binding:"required"`
		EmployeeID uint `json:"employee_id" binding:"required"`
		TotalLeave int  `json:"total_leave"`
		UsedLeave  int  `json:"used_leave"`
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var balance models.CycleLeaveBalance
	db.DB.Where("cycle_index = ? AND employee_id = ?", input.CycleIndex, input.EmployeeID).
		Assign(models.CycleLeaveBalance{
			TotalLeave: input.TotalLeave,
			UsedLeave:  input.UsedLeave,
		}).FirstOrCreate(&balance, models.CycleLeaveBalance{
		CycleIndex: input.CycleIndex,
		EmployeeID: input.EmployeeID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "假期餘額已更新", "balance": balance})
}

// UpdateMonthlySlot 手動修改單一格子班別
func UpdateMonthlySlot(c *gin.Context) {
	slotID := c.Param("id")

	type UpdateSlotInput struct {
		ShiftType string `json:"shift_type" binding:"required"`
	}
	var input UpdateSlotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 驗證班別有效性
	validShifts := map[string]bool{"day": true, "evening": true, "night": true, "day88": true, "night88": true, "off": true}
	if !validShifts[input.ShiftType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的班別: " + input.ShiftType})
		return
	}

	var slot models.MonthlySlot
	if err := db.DB.First(&slot, slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到此排班格"})
		return
	}

	oldShift := slot.ShiftType
	slot.ShiftType = input.ShiftType
	db.DB.Save(&slot)

	// --- 重新計算該月份人力的 Warnings 與 假期 Summaries ---
	// 取出該月份所有的 slots，建構 schedule 矩陣
	var monthSchedule models.MonthlySchedule
	if err := db.DB.First(&monthSchedule, slot.ScheduleID).Error; err == nil {
		year := monthSchedule.Year
		month := monthSchedule.Month
		firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
		lastDay := firstDay.AddDate(0, 1, -1)
		totalDays := lastDay.Day()

		var allSlots []models.MonthlySlot
		db.DB.Where("schedule_id = ?", monthSchedule.ID).Find(&allSlots)

		var employees []models.Employee
		db.DB.Where("status = 1").Find(&employees)

		var requirements []models.StaffingRequirement
		db.DB.Find(&requirements)
		reqMap := buildRequirementMap(requirements)

		// 使用更穩定的索引方式 (考慮時區)
		loc := time.FixedZone("CST", 8*3600)
		scheduleMap := make(map[int]map[uint]string)
		for d := 0; d < totalDays; d++ {
			scheduleMap[d] = make(map[uint]string)
		}
		for _, s := range allSlots {
			// 計算該日期相對於月初的天數偏移
			dIdx := int(s.Date.In(loc).Sub(firstDay).Hours() / 24)
			if dIdx >= 0 && dIdx < totalDays {
				scheduleMap[dIdx][s.EmployeeID] = s.ShiftType
			}
		}

		// 抓取前一天的班別 (處理跨月邊界)
		prevDay := firstDay.AddDate(0, 0, -1)
		var prevDaySlots []models.MonthlySlot
		db.DB.Where("date = ?", prevDay).Find(&prevDaySlots)
		prevDaySchedule := make(map[uint]string)
		for _, ps := range prevDaySlots {
			prevDaySchedule[ps.EmployeeID] = ps.ShiftType
		}

		warnings := calculateWarnings(firstDay, totalDays, employees, reqMap, scheduleMap, prevDaySchedule)
		if warnings == nil {
			warnings = []string{}
		}
		summaries := generateLeaveSummaries(year, month)
		if summaries == nil {
			summaries = []gin.H{}
		}
		boundaries := calcCycleBoundaries(firstDay, lastDay)

		log.Printf("[UpdateSlot] ID:%s, Date:%s, NewShift:%s, Warnings:%d", slotID, slot.Date.Format("2006-01-02"), input.ShiftType, len(warnings))

		c.JSON(http.StatusOK, gin.H{
			"message":    "排班已更新: " + oldShift + " → " + input.ShiftType,
			"slot":       slot,
			"warnings":   warnings,
			"summaries":  summaries,
			"boundaries": boundaries,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "排班已更新: " + oldShift + " → " + input.ShiftType,
		"slot":     slot,
		"warnings": []string{},
	})
}

// GetCycleBoundaries 取得指定月份的循環分界資訊
func GetCycleBoundaries(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)
	boundaries := calcCycleBoundaries(firstDay, lastDay)

	c.JSON(http.StatusOK, gin.H{
		"year":       year,
		"month":      month,
		"boundaries": boundaries,
	})
}

// GetMonthlyPreLeaves 獲取指定月份的具體日期預假
func GetMonthlyPreLeaves(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)

	var preLeaves []models.MonthlyPreScheduledLeave
	db.DB.Where("date BETWEEN ? AND ?", firstDay, lastDay).Order("date ASC").Find(&preLeaves)

	c.JSON(http.StatusOK, preLeaves)
}

// CreateMonthlyPreLeave 建立月度預假
func CreateMonthlyPreLeave(c *gin.Context) {
	var input struct {
		EmployeeID uint   `json:"employee_id" binding:"required"`
		Date       string `json:"date" binding:"required"` // 改為字串以相容 YYYY-MM-DD
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedDate, err := time.ParseInLocation("2006-01-02", input.Date, time.FixedZone("CST", 8*3600))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式錯誤，請使用 YYYY-MM-DD"})
		return
	}

	// 檢查是否已存在 (包含軟刪除的紀錄)
	var existing models.MonthlyPreScheduledLeave
	result := db.DB.Unscoped().Where("employee_id = ? AND date = ?", input.EmployeeID, parsedDate).First(&existing)
	if result.Error == nil {
		// 若紀錄以前被標記為軟刪除，將其恢復 (revive)
		if existing.DeletedAt.Valid {
			existing.DeletedAt = gorm.DeletedAt{} // 清除刪除標記
		}
		// 更新原因
		existing.Reason = input.Reason
		db.DB.Save(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	preLeave := models.MonthlyPreScheduledLeave{
		EmployeeID: input.EmployeeID,
		Date:       parsedDate,
		Reason:     input.Reason,
	}
	if err := db.DB.Create(&preLeave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preLeave)
}

// DeleteMonthlyPreLeave 刪除月度預假
func DeleteMonthlyPreLeave(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Unscoped().Delete(&models.MonthlyPreScheduledLeave{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "預假已刪除"})
}

// SaveMonthlyVersion 儲存當前班表為一個新版本
func SaveMonthlyVersion(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	var input struct {
		VersionName string `json:"version_name" binding:"required"`
		Creator     string `json:"creator"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 取得目前的月度班表
	var schedule models.MonthlySchedule
	if err := db.DB.Where("year = ? AND month = ?", year, month).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "尚未建立該月班表，無法儲存版本"})
		return
	}

	// 2. 取得目前所有的 slots
	var slots []models.MonthlySlot
	db.DB.Where("schedule_id = ?", schedule.ID).Find(&slots)

	// 3. 開啟交易進行儲存
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// 建立版本主紀錄
		version := models.MonthlyScheduleVersion{
			Year:        year,
			Month:       month,
			VersionName: input.VersionName,
			Creator:     input.Creator,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		// 複製所有 slots 到版本紀錄中
		for _, s := range slots {
			vSlot := models.MonthlySlotVersion{
				VersionID:  version.ID,
				Date:       s.Date,
				ShiftType:  s.ShiftType,
				EmployeeID: s.EmployeeID,
				CycleIndex: s.CycleIndex,
				DayOffset:  s.DayOffset,
			}
			if err := tx.Create(&vSlot).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "儲存版本失敗: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "版本儲存成功", "version_name": input.VersionName})
}

// ListMonthlyVersions 列出指定月份的所有版本
func ListMonthlyVersions(c *gin.Context) {
	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	var versions []models.MonthlyScheduleVersion
	db.DB.Where("year = ? AND month = ?", year, month).Order("created_at DESC").Find(&versions)

	c.JSON(http.StatusOK, versions)
}

// RestoreMonthlyVersion 恢復指定版本到主班表
func RestoreMonthlyVersion(c *gin.Context) {
	versionID := c.Param("versionId")

	var version models.MonthlyScheduleVersion
	if err := db.DB.First(&version, versionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到該版本"})
		return
	}

	// 1. 取得對應的主排班紀錄 (若不存在則建立)
	var schedule models.MonthlySchedule
	db.DB.Where("year = ? AND month = ?", version.Year, version.Month).FirstOrCreate(&schedule, models.MonthlySchedule{
		Year:  version.Year,
		Month: version.Month,
	})

	// 2. 取得版本中的所有 slots
	var vSlots []models.MonthlySlotVersion
	db.DB.Where("version_id = ?", version.ID).Find(&vSlots)

	// 3. 執行覆蓋
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// 清除主班表舊的 slots
		tx.Unscoped().Where("schedule_id = ?", schedule.ID).Delete(&models.MonthlySlot{})

		// 寫入版本資料
		for _, vs := range vSlots {
			slot := models.MonthlySlot{
				ScheduleID: schedule.ID,
				Date:       vs.Date,
				ShiftType:  vs.ShiftType,
				EmployeeID: vs.EmployeeID,
				CycleIndex: vs.CycleIndex,
				DayOffset:  vs.DayOffset,
			}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢復版本失敗: " + err.Error()})
		return
	}

	// 恢復完畢後，回傳最新的狀態 (Warnings 與 Summaries)
	firstDay := time.Date(version.Year, time.Month(version.Month), 1, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	lastDay := firstDay.AddDate(0, 1, -1)
	totalDays := lastDay.Day()

	var employees []models.Employee
	db.DB.Where("status = 1").Find(&employees)

	var requirements []models.StaffingRequirement
	db.DB.Find(&requirements)
	reqMap := buildRequirementMap(requirements)

	var allSlots []models.MonthlySlot
	db.DB.Where("schedule_id = ?", schedule.ID).Find(&allSlots)

	loc := time.FixedZone("CST", 8*3600)
	scheduleMap := make(map[int]map[uint]string)
	for d := 0; d < totalDays; d++ {
		scheduleMap[d] = make(map[uint]string)
	}
	for _, s := range allSlots {
		dIdx := int(s.Date.In(loc).Sub(firstDay).Hours() / 24)
		if dIdx >= 0 && dIdx < totalDays {
			scheduleMap[dIdx][s.EmployeeID] = s.ShiftType
		}
	}

	// 抓取前一天的班別 (處理跨月邊界)
	prevDay := firstDay.AddDate(0, 0, -1)
	var prevDaySlots []models.MonthlySlot
	db.DB.Where("date = ?", prevDay).Find(&prevDaySlots)
	prevDaySchedule := make(map[uint]string)
	for _, ps := range prevDaySlots {
		prevDaySchedule[ps.EmployeeID] = ps.ShiftType
	}

	warnings := calculateWarnings(firstDay, totalDays, employees, reqMap, scheduleMap, prevDaySchedule)
	if warnings == nil {
		warnings = []string{}
	}
	summaries := generateLeaveSummaries(version.Year, version.Month)
	if summaries == nil {
		summaries = []gin.H{}
	}
	boundaries := calcCycleBoundaries(firstDay, lastDay)

	c.JSON(http.StatusOK, gin.H{
		"message":    "已恢復至版本: " + version.VersionName,
		"slots":      allSlots,
		"warnings":   warnings,
		"summaries":  summaries,
		"boundaries": boundaries,
	})
}

// DeleteMonthlyVersion 刪除版本
func DeleteMonthlyVersion(c *gin.Context) {
	versionID := c.Param("versionId")

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// 刪除 slots
		if err := tx.Unscoped().Where("version_id = ?", versionID).Delete(&models.MonthlySlotVersion{}).Error; err != nil {
			return err
		}
		// 刪除版本主紀錄
		if err := tx.Unscoped().Delete(&models.MonthlyScheduleVersion{}, versionID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刪除失敗: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "版本已徹底刪除"})
}
