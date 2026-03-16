package controllers

import (
	"log"
	"math/rand"
	"net/http"
	"schedule-system/db"
	"schedule-system/models"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- 循環模板 CRUD ---

func GetTemplates(c *gin.Context) {
	var templates []models.CycleTemplate
	db.DB.Order("created_at DESC").Find(&templates)
	c.JSON(http.StatusOK, templates)
}

func GetTemplate(c *gin.Context) {
	id := c.Param("id")
	var template models.CycleTemplate
	if err := db.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}
	var slots []models.TemplateSlot
	db.DB.Where("template_id = ?", id).Find(&slots)
	c.JSON(http.StatusOK, gin.H{"template": template, "slots": slots})
}

type CreateTemplateRequest struct {
	StartDate  string `json:"start_date" binding:"required"`
	CycleWeeks int    `json:"cycle_weeks"`
}

func CreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	cycleWeeks := req.CycleWeeks
	if cycleWeeks <= 0 {
		cycleWeeks = 4
	}
	template := models.CycleTemplate{StartDate: startDate, CycleWeeks: cycleWeeks, Version: 1, Status: "draft"}
	if err := db.DB.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "建立成功", "data": template})
}

func DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	var template models.CycleTemplate
	if err := db.DB.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}
	db.DB.Unscoped().Where("template_id = ?", id).Delete(&models.TemplateSlot{})
	db.DB.Unscoped().Where("template_id = ?", id).Delete(&models.PreScheduledLeave{})
	db.DB.Unscoped().Delete(&template)
	c.JSON(http.StatusOK, gin.H{"message": "模板已刪除"})
}

// =============================================================================
// 自動排班 v6 — 假期優先 + C7 雙向檢查 + 預假
// =============================================================================

func AutoSchedule(c *gin.Context) {
	templateID := c.Param("id")
	var template models.CycleTemplate
	if err := db.DB.First(&template, templateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	totalDays := template.CycleWeeks * 7

	var employees []models.Employee
	db.DB.Where("status = 1").Find(&employees)

	var restrictions []models.ShiftRestriction
	db.DB.Where("template_id = ? OR template_id IS NULL", template.ID).Find(&restrictions)
	var requirements []models.StaffingRequirement
	db.DB.Find(&requirements)

	var preLeaves []models.PreScheduledLeave
	db.DB.Where("template_id = ?", template.ID).Find(&preLeaves)

	reqMap := buildRequirementMap(requirements)
	constraints := buildConstraints(employees, restrictions)

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// 清除舊排班 (物理刪除)
		if err := tx.Unscoped().Where("template_id = ?", template.ID).Delete(&models.TemplateSlot{}).Error; err != nil {
			return err
		}

		slots, stats := runV6Algorithm(totalDays, template.StartDate, employees, constraints, reqMap, preLeaves)

		for i := range slots {
			slots[i].TemplateID = template.ID
			if err := tx.Create(&slots[i]).Error; err != nil {
				return err
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "自動排班已優化完成 (v6 假期優先)",
			"slots":   slots,
			"stats":   stats,
		})
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "排班失敗: " + err.Error()})
	}
}

// =============================================================================
// v6 核心演算法
// =============================================================================

func runV6Algorithm(
	totalDays int,
	startDate time.Time,
	employees []models.Employee,
	constraints map[uint]*employeeConstraint,
	reqMap map[requirementKey]models.StaffingRequirement,
	preLeaves []models.PreScheduledLeave,
) ([]models.TemplateSlot, map[string]interface{}) {
	log.Printf("[AutoSchedule] 開始執行 runV6Algorithm: 天數=%d, 員工數=%d", totalDays, len(employees))

	// schedule[day][empID] = shiftType
	schedule := make(map[int]map[uint]string)
	shiftCount := make(map[uint]map[string]int)
	for d := 0; d < totalDays; d++ {
		schedule[d] = make(map[uint]string)
	}
	for _, emp := range employees {
		shiftCount[emp.ID] = make(map[string]int)
	}

	// ─── 步驟 0：計算假期配額 ───
	log.Println("[AutoSchedule] 步驟 0: 計算假期配額")
	totalRequired := 0
	for d := 0; d < totalDays; d++ {
		weekday := int(startDate.AddDate(0, 0, d).Weekday())
		// 假設有 8-8 的天數用 MinCountWithDay88
		for _, st := range []string{"day", "evening", "night"} {
			if req, ok := reqMap[requirementKey{weekday, st}]; ok {
				totalRequired += req.MinCountWithDay88
			}
		}
	}
	// 加上 8-8 本身的人次
	// 加上 8-8 本身的人次 (假設 8-8 主力每天都上班，除非預假)
	day88Count := 0
	var regularEmps []models.Employee
	for _, emp := range employees {
		if emp.IsDay88Primary {
			day88Count += totalDays
		} else {
			regularEmps = append(regularEmps, emp)
		}
	}
	totalRequired += day88Count

	totalAvailable := len(employees) * totalDays
	totalLeave := totalAvailable - totalRequired
	if totalLeave < 0 {
		totalLeave = 0
	}

	leaveQuota := make(map[uint]int)
	if len(regularEmps) > 0 {
		perPersonLeave := totalLeave / len(regularEmps)
		for _, emp := range regularEmps {
			leaveQuota[emp.ID] = perPersonLeave
		}
		// 分配餘數
		remainder := totalLeave - perPersonLeave*len(regularEmps)
		for i := 0; i < remainder; i++ {
			leaveQuota[regularEmps[i].ID]++
		}
	}

	// ─── 步驟 1：鎖定預假 ⭐ ───
	log.Println("[AutoSchedule] 步驟 1: 鎖定預假")
	for _, pl := range preLeaves {
		schedule[pl.DayOffset][pl.EmployeeID] = "off"
		leaveQuota[pl.EmployeeID]--
	}

	// ─── 步驟 2：智慧假期分配 ⭐ ───
	log.Println("[AutoSchedule] 步驟 2: 智慧假期分配")
	// 按人力需求從低到高排序日期，優先在需求低的日子放假
	type dayNeed struct {
		day  int
		need int
	}
	dayNeeds := make([]dayNeed, totalDays)
	for d := 0; d < totalDays; d++ {
		weekday := int(startDate.AddDate(0, 0, d).Weekday())
		need := 0
		for _, st := range []string{"day", "evening", "night"} {
			if req, ok := reqMap[requirementKey{weekday, st}]; ok {
				need += req.MinCountWithDay88
			}
		}
		dayNeeds[d] = dayNeed{d, need}
	}
	sort.Slice(dayNeeds, func(i, j int) bool { return dayNeeds[i].need < dayNeeds[j].need })

	// 為每位常規員工分配剩餘假期 (扣除已佔的預假)
	for _, emp := range employees {
		if emp.IsDay88Primary {
			continue // J 不參與智慧排休
		}
		remaining := leaveQuota[emp.ID]
		if remaining <= 0 {
			continue
		}

		// 盡量均勻散佈：計算理想間隔
		interval := totalDays / (remaining + 1)
		if interval < 1 {
			interval = 1
		}

		assigned := 0
		for _, dn := range dayNeeds {
			if assigned >= remaining {
				break
			}
			d := dn.day
			if _, exists := schedule[d][emp.ID]; exists {
				continue // 已有排班或預假
			}

			// 檢查該天放假是否會造成前後班次銜接問題
			// 簡單檢查：不要讓相鄰兩天都是假（避免過度集中）
			schedule[d][emp.ID] = "off"
			assigned++
		}
	}

	// ─── 步驟 3：排 J (8-8 主力) ───
	log.Printf("[AutoSchedule] 步驟 3: 排 J 主力")
	for _, emp := range employees {
		if !emp.IsDay88Primary {
			continue
		}
		for d := 0; d < totalDays; d++ {
			if schedule[d][emp.ID] == "off" {
				continue // 尊重已排的假期 (例如預假)
			}
			schedule[d][emp.ID] = "day88"
			shiftCount[emp.ID]["day88"]++
		}
	}

	// 計算大夜、小夜的專責人力需求
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

	// ─── 步驟 4：排大夜連續班段 ───
	log.Printf("[AutoSchedule] 步驟 4: 排大夜連續班段, 需要人數: %d", nightEmpsNeeded)
	fillConsecutiveV3(schedule, shiftCount, employees, constraints, "night", totalDays, 4, 2, nightEmpsNeeded)

	// ─── 步驟 5：排小夜連續班段 ───
	fillConsecutiveV3(schedule, shiftCount, employees, constraints, "evening", totalDays, 4, 1, eveningEmpsNeeded)

	// ─── 步驟 5.5：C7 強制 off 掃描 ⭐ ───
	// 遍歷每位員工，在連續工作達 6 天的位置強制插入 off
	// 這確保補位階段不會把兩個班段連在一起產生超長連班
	for _, emp := range employees {
		if emp.IsDay88Primary {
			continue // J 不需遵守 6 休 1 規範
		}
		streak := 0
		for d := 0; d < totalDays; d++ {
			s := schedule[d][emp.ID]
			if s != "" && s != "off" && s != "pre_off" {
				streak++
				if streak >= 6 {
					// 第 6 天工作後，下一天強制 off
					nextDay := d + 1
					if nextDay < totalDays {
						existing := schedule[nextDay][emp.ID]
						if existing != "off" && existing != "" {
							// 必須強制覆蓋為 off
							// 如果這裡有班，需要從 shiftCount 扣除
							shiftCount[emp.ID][existing]--
						}
						schedule[nextDay][emp.ID] = "off"
					}
					streak = 0
				}
			} else {
				streak = 0
			}
		}
	}

	// ─── 步驟 6：補位白班 ───
	log.Println("[AutoSchedule] 步驟 6: 補位白班")
	for d := 0; d < totalDays; d++ {
		weekday := int(startDate.AddDate(0, 0, d).Weekday())
		has88 := false
		for _, s := range schedule[d] {
			if s == "day88" {
				has88 = true
				break
			}
		}

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
				// 若目前正在計算 day 班的需求，且該員工上了 day88 班，也算作一個人頭
				if st == "day" && assigned == "day88" {
					current++
				}
			}

			for n := current; n < minNeeded; n++ {
				bestID := findBestCandidateV3(employees, schedule, shiftCount, constraints, d, st, totalDays, preLeaves)
				if bestID != 0 {
					schedule[d][bestID] = st
					shiftCount[bestID][st]++
				}
			}
		}
	}

	// ─── 步驟 7：驗證 & 轉換 ───
	var slots []models.TemplateSlot
	for d := 0; d < totalDays; d++ {
		for empID, st := range schedule[d] {
			if st != "off" && st != "" {
				slots = append(slots, models.TemplateSlot{DayOffset: d, ShiftType: st, EmployeeID: empID})
			}
		}
	}

	stats := computeStatsV6(slots, employees, totalDays, leaveQuota, preLeaves)
	return slots, stats
}

// =============================================================================
// fillConsecutiveV3 - 原子性分配 + 強制休假佔位
// =============================================================================

func fillConsecutiveV3(
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	employees []models.Employee,
	constraints map[uint]*employeeConstraint,
	shiftType string,
	totalDays int,
	runLen int,
	restLen int,
	maxPeople int,
) {
	// 篩選可排此班的員工
	eligible := []uint{}
	for _, e := range employees {
		if e.IsDay88Primary {
			continue
		}
		if ec, ok := constraints[e.ID]; ok && ec.Banned[shiftType] {
			continue
		}
		eligible = append(eligible, e.ID)
	}
	if len(eligible) == 0 {
		return
	}
	rand.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })

	if len(eligible) > maxPeople {
		// 根據需求量限制參與輪值的人數，避免過度占用白班人力池
		eligible = eligible[:maxPeople]
	}

	for i, empID := range eligible {
		// 修正 offset：利用間隔 2 天來錯開，而不是乘以 block size 造成往後疊加
		blockCycle := runLen + restLen
		offset := (i * 2) % blockCycle
		for d := offset; d < totalDays; {
			// 原子性預檢
			canDoAll := true
			actualRun := 0
			for r := 0; r < runLen && d+r < totalDays; r++ {
				if !canAssignV6(empID, schedule, shiftCount, constraints, d+r, shiftType, totalDays) {
					canDoAll = false
					break
				}
				actualRun++
			}

			if canDoAll && actualRun >= 2 {
				// 分配工作班段
				for r := 0; r < actualRun; r++ {
					schedule[d+r][empID] = shiftType
					shiftCount[empID][shiftType]++
				}
				// 強制標記休假佔位
				for r := 0; r < restLen && d+actualRun+r < totalDays; r++ {
					if schedule[d+actualRun+r][empID] == "" {
						schedule[d+actualRun+r][empID] = "off"
					}
				}
				d += actualRun + restLen
			} else {
				d++
			}
		}
	}
}

// =============================================================================
// findBestCandidateV3 - 補位時優先延續昨日班段
// =============================================================================

func findBestCandidateV3(
	employees []models.Employee,
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	constraints map[uint]*employeeConstraint,
	day int,
	shiftType string,
	totalDays int,
	preLeaves []models.PreScheduledLeave,
) uint {
	// 優先找昨天也是同班的人（延續連續性）
	if day > 0 {
		var candidates []uint
		for _, emp := range employees {
			if schedule[day-1][emp.ID] == shiftType && canAssignV6(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays) {
				candidates = append(candidates, emp.ID)
			}
		}
		if len(candidates) > 0 {
			return pickLowestWork(candidates, shiftCount)
		}
	}

	// 找符合約束且工時最少的人
	// 找符合嚴格約束且工時最少的人
	var eligible []uint
	for _, emp := range employees {
		// 如果原本已經有別的班（非 off），跳過
		if s, ok := schedule[day][emp.ID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
			continue
		}
		// 預假不可覆蓋
		if isPreLeave(emp.ID, day, preLeaves) || schedule[day][emp.ID] == "pre_off" {
			continue
		}
		// (嚴格模式下) 主秀 Day88 的 J 如果現在沒班，他通常是休假，儘量先不要動他，讓其他人先排
		if emp.IsDay88Primary {
			continue
		}
		if canAssignV6(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays) {
			eligible = append(eligible, emp.ID)
		}
	}
	if len(eligible) > 0 {
		return pickLowestWork(eligible, shiftCount)
	}

	// 找不到人，進入寬鬆模式 (Relaxed)
	var relaxedEligible []uint
	for _, emp := range employees {
		if s, ok := schedule[day][emp.ID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
			continue
		}
		if isPreLeave(emp.ID, day, preLeaves) {
			continue // 預假絕對不可動
		}
		if emp.IsDay88Primary && shiftType != "day88" {
			continue // 主秀 Day88 即使缺人也不去救其他班
		}
		if canAssignV6Relaxed(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays) {
			relaxedEligible = append(relaxedEligible, emp.ID)
		}
	}
	if len(relaxedEligible) > 0 {
		return pickLowestWork(relaxedEligible, shiftCount)
	}

	// 終極備案 (Force Mode)：連放寬條件都找不到人時，為了滿足最低人力需求，強制抓一個非預假、當天也沒排班的人來上班
	var forceEligible []uint
	for _, emp := range employees {
		if s, ok := schedule[day][emp.ID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
			continue // 當天已經有其他班或預假
		}
		if isPreLeave(emp.ID, day, preLeaves) {
			continue // 預假絕對不可動
		}
		if emp.IsDay88Primary && shiftType != "day88" {
			continue // 主秀 Day88 不去救其他班
		}
		// 無視任何 C1~C7 規定，只要他不是預假，就強制抓來上
		forceEligible = append(forceEligible, emp.ID)
	}
	if len(forceEligible) > 0 {
		bestForce := pickLowestWork(forceEligible, shiftCount)
		log.Printf("⚠️ [Force Mode] Day %d 缺人，強抓 Emp %d (原排休假) 來上 %s 班 (候選人數: %d)", day, bestForce, shiftType, len(forceEligible))
		return bestForce
	}

	log.Printf("❌ [CRITICAL] Day %d 連 Force Mode 都找不到人上 %s 班! (沒休假可抓)", day, shiftType)
	return 0
}

func isPreLeave(empID uint, day int, preLeaves []models.PreScheduledLeave) bool {
	for _, pl := range preLeaves {
		if pl.EmployeeID == empID && pl.DayOffset == day {
			return true
		}
	}
	return false
}

// =============================================================================
// canAssignV6 - 完整約束判斷 (含 C7 雙向檢查)
// =============================================================================

func canAssignV6(
	empID uint,
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	constraints map[uint]*employeeConstraint,
	day int,
	shiftType string,
	totalDays int,
) bool {
	ec := constraints[empID]

	if ec != nil && ec.IsDay88Primary && shiftType != "day88" {
		return false // 只能排 day88
	}

	// C4: 個人禁排班別
	if ec != nil && ec.Banned[shiftType] {
		return false
	}

	// C5: 個人班別天數上限
	if ec != nil {
		if max, ok := ec.MaxDays[shiftType]; ok && shiftCount[empID][shiftType] >= max {
			return false
		}
	}

	// C3: 每人每天最多一班 (已有班或已排假)
	if existing, ok := schedule[day][empID]; ok && existing != "" {
		return false
	}

	// C1: 大夜 → 隔天白班 ❌
	// C2: 小夜 → 隔天白班 ❌
	if shiftType == "day" || shiftType == "day88" {
		if day > 0 {
			prev := schedule[day-1][empID]
			if prev == "night" || prev == "evening" {
				return false
			}
		}
	}

	// 反向：當天排夜班，但明天已排白班 → 禁止
	if shiftType == "evening" || shiftType == "night" {
		if day+1 < totalDays {
			next := schedule[day+1][empID]
			if next == "day" || next == "day88" {
				return false
			}
		}
	}

	// C7: 做 6 休 1 (雙向檢查) ⭐
	// 如果是 Day88Primary，則不受 C7 規範限制
	if ec == nil || !ec.IsDay88Primary {
		backwardStreak := 0
		for d := day - 1; d >= 0; d-- {
			s := schedule[d][empID]
			if s != "" && s != "off" && s != "pre_off" {
				backwardStreak++
			} else {
				break
			}
		}

		forwardStreak := 0
		for d := day + 1; d < totalDays; d++ {
			s := schedule[d][empID]
			if s != "" && s != "off" && s != "pre_off" {
				forwardStreak++
			} else {
				break
			}
		}

		// 前方連班 + 今天(1) + 後方連班 > 6 → 禁止
		if backwardStreak+1+forwardStreak > 6 {
			return false
		}
	}

	return true
}

// =============================================================================
// canAssignV6Relaxed - 寬鬆約束判斷 (找不到人時的備案)
// =============================================================================

func canAssignV6Relaxed(
	empID uint,
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	constraints map[uint]*employeeConstraint,
	day int,
	shiftType string,
	totalDays int,
) bool {
	ec := constraints[empID]

	if ec != nil && ec.IsDay88Primary && shiftType != "day88" {
		return false // 只能排 day88
	}

	// C4: 個人禁排班別 (仍然必須遵守)
	if ec != nil && ec.Banned[shiftType] {
		return false
	}

	// C1, C2: 班別銜接規定 (大夜/小夜不可接白班)
	if shiftType == "day" || shiftType == "day88" {
		if day > 0 {
			prev := schedule[day-1][empID]
			if prev == "night" || prev == "evening" {
				return false
			}
		}
	}

	if shiftType == "evening" || shiftType == "night" {
		if day+1 < totalDays {
			next := schedule[day+1][empID]
			if next == "day" || next == "day88" {
				return false
			}
		}
	}

	// C3: 檢查是否有班或預假
	if s, ok := schedule[day][empID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
		return false
	}

	return true
}

// =============================================================================
// 輔助函式
// =============================================================================

func pickLowestWork(ids []uint, shiftCount map[uint]map[string]int) uint {
	var best uint
	minWork := 999
	for _, id := range ids {
		work := 0
		for _, c := range shiftCount[id] {
			work += c
		}
		if work < minWork {
			minWork = work
			best = id
		}
	}
	return best
}

func buildRequirementMap(reqs []models.StaffingRequirement) map[requirementKey]models.StaffingRequirement {
	m := make(map[requirementKey]models.StaffingRequirement)
	for _, r := range reqs {
		m[requirementKey{r.Weekday, r.ShiftType}] = r
	}
	return m
}

func buildConstraints(employees []models.Employee, restrictions []models.ShiftRestriction) map[uint]*employeeConstraint {
	m := make(map[uint]*employeeConstraint)
	for _, emp := range employees {
		m[emp.ID] = &employeeConstraint{
			ID:             emp.ID,
			Name:           emp.Name,
			IsDay88Primary: emp.IsDay88Primary,
			Banned:         make(map[string]bool),
			MaxDays:        make(map[string]int),
		}
	}
	for _, r := range restrictions {
		if ec, ok := m[r.EmployeeID]; ok {
			if r.MaxDays == nil {
				ec.Banned[r.ShiftType] = true
			} else {
				ec.MaxDays[r.ShiftType] = *r.MaxDays
			}
		}
	}
	return m
}

func computeStatsV6(
	slots []models.TemplateSlot,
	employees []models.Employee,
	totalDays int,
	leaveQuota map[uint]int,
	preLeaves []models.PreScheduledLeave,
) map[string]interface{} {
	stats := make(map[uint]map[string]interface{})
	for _, emp := range employees {
		preLeaveCount := 0
		for _, pl := range preLeaves {
			if pl.EmployeeID == emp.ID {
				preLeaveCount++
			}
		}
		stats[emp.ID] = map[string]interface{}{
			"name":        emp.Name,
			"shift_days":  make(map[string]int),
			"total_work":  0,
			"off_days":    totalDays,
			"leave_quota": leaveQuota[emp.ID],
			"pre_leaves":  preLeaveCount,
		}
	}
	for _, s := range slots {
		if st, ok := stats[s.EmployeeID]; ok {
			st["shift_days"].(map[string]int)[s.ShiftType]++
			st["total_work"] = st["total_work"].(int) + 1
			st["off_days"] = st["off_days"].(int) - 1
		}
	}
	return map[string]interface{}{"employees": stats}
}

// --- Slot CRUD ---

func SetSlot(c *gin.Context) {
	var req models.TemplateSlot
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "排班失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已排班", "data": req})
}

func RemoveSlot(c *gin.Context) {
	id := c.Param("id")
	db.DB.Unscoped().Delete(&models.TemplateSlot{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "已移除"})
}

func ClearTemplateSlots(c *gin.Context) {
	db.DB.Unscoped().Where("template_id = ?", c.Param("id")).Delete(&models.TemplateSlot{})
	c.JSON(http.StatusOK, gin.H{"message": "已清除"})
}

func GetTemplateCalendar(c *gin.Context) {
	templateID := c.Param("id")
	var template models.CycleTemplate
	if err := db.DB.First(&template, templateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}
	var slots []models.TemplateSlot
	db.DB.Where("template_id = ?", templateID).Find(&slots)

	totalDays := template.CycleWeeks * 7
	type calEntry struct {
		Date      string `json:"date"`
		DayOffset int    `json:"day_offset"`
		Weekday   string `json:"weekday"`
	}
	weekdayNames := []string{"日", "一", "二", "三", "四", "五", "六"}
	var calendar []calEntry
	for d := 0; d < totalDays; d++ {
		date := template.StartDate.AddDate(0, 0, d)
		calendar = append(calendar, calEntry{
			Date:      date.Format("2006-01-02"),
			DayOffset: d,
			Weekday:   weekdayNames[date.Weekday()],
		})
	}
	c.JSON(http.StatusOK, gin.H{"template": template, "calendar": calendar, "slots": slots})
}

// --- 型別定義 ---

type employeeConstraint struct {
	ID             uint
	Name           string
	IsDay88Primary bool
	Banned         map[string]bool
	MaxDays        map[string]int
}

type requirementKey struct {
	Weekday   int
	ShiftType string
}
