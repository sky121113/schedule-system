package controllers

import (
	"log"
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

// v6 核心演算法
func runV6Algorithm(
	totalDays int,
	startDate time.Time,
	employees []models.Employee,
	constraints map[uint]*employeeConstraint,
	reqMap map[requirementKey]models.StaffingRequirement,
	preLeaves []models.PreScheduledLeave,
) ([]models.TemplateSlot, map[string]interface{}) {
	log.Printf("[AutoSchedule] 開始執行 runV6Algorithm: 天數=%d, 員工數=%d", totalDays, len(employees))

	schedule := make(map[int]map[uint]string)
	shiftCount := make(map[uint]map[string]int)
	for d := 0; d < totalDays; d++ {
		schedule[d] = make(map[uint]string)
	}
	for _, emp := range employees {
		shiftCount[emp.ID] = make(map[string]int)
	}

	// ─── 步驟 0：計算假期配額 ───
	totalRequired := 0
	for d := 0; d < totalDays; d++ {
		weekday := int(startDate.AddDate(0, 0, d).Weekday())
		for _, st := range []string{"day", "evening", "night"} {
			if req, ok := reqMap[requirementKey{weekday, st}]; ok {
				totalRequired += req.MinCountWithDay88
			}
		}
	}
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
		remainder := totalLeave - perPersonLeave*len(regularEmps)
		for i := 0; i < remainder; i++ {
			leaveQuota[regularEmps[i].ID]++
		}
	}

	// ─── 步驟 1：鎖定預假 ⭐ ───
	for _, pl := range preLeaves {
		schedule[pl.DayOffset][pl.EmployeeID] = "off"
		leaveQuota[pl.EmployeeID]--
	}

	// ─── 步驟 2：智慧假期分配 ⭐ ───
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

	for _, emp := range employees {
		if emp.IsDay88Primary {
			continue
		}
		remaining := leaveQuota[emp.ID]
		if remaining <= 0 {
			continue
		}
		assigned := 0
		for _, dn := range dayNeeds {
			if assigned >= remaining {
				break
			}
			d := dn.day
			if _, exists := schedule[d][emp.ID]; exists {
				continue
			}
			schedule[d][emp.ID] = "off"
			assigned++
		}
	}

	// ─── 步驟 3：排 J (8-8 主力) ───
	for _, emp := range employees {
		if !emp.IsDay88Primary {
			continue
		}
		for d := 0; d < totalDays; d++ {
			if schedule[d][emp.ID] == "off" {
				continue
			}
			schedule[d][emp.ID] = "day88"
			shiftCount[emp.ID]["day88"]++
		}
	}

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
	fillConsecutiveV3(schedule, shiftCount, employees, constraints, "night", totalDays, 4, 2, nightEmpsNeeded, nil)
	// ─── 步驟 5：排小夜連續班段 ───
	fillConsecutiveV3(schedule, shiftCount, employees, constraints, "evening", totalDays, 4, 1, eveningEmpsNeeded, nil)

	// ─── 步驟 5.5：C7 強制 off 掃描 ⭐ ───
	for _, emp := range employees {
		if emp.IsDay88Primary {
			continue
		}
		streak := 0
		for d := 0; d < totalDays; d++ {
			s := schedule[d][emp.ID]
			if s != "" && s != "off" && s != "pre_off" {
				streak++
				if streak >= 6 {
					nextDay := d + 1
					if nextDay < totalDays {
						existing := schedule[nextDay][emp.ID]
						if existing != "off" && existing != "" {
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
				if st == "day" && assigned == "day88" {
					current++
				}
			}
			for n := current; n < minNeeded; n++ {
				bestID := findBestCandidateV3(employees, schedule, shiftCount, constraints, d, st, totalDays, preLeaves, nil)
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
	weekdayNames := []string{"日", "一", "二", "三", "四", "五", "六"}
	type calEntry struct {
		Date      string `json:"date"`
		DayOffset int    `json:"day_offset"`
		Weekday   string `json:"weekday"`
	}
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
