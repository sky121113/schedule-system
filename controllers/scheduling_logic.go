package controllers

import (
	"math/rand"
	"schedule-system/models"
	"sort"
)


// requirementKey 用於快速查找特定星期與班別的人力需求
type requirementKey struct {
	Weekday   int
	ShiftType string
}

// employeeConstraint 存放單一員工的所有排班約束內容
type employeeConstraint struct {
	ID             uint
	Name           string
	IsDay88Primary bool
	Banned         map[string]bool
	MaxDays        map[string]int
}

// buildRequirementMap 建立人力需求快速索引表
func buildRequirementMap(reqs []models.StaffingRequirement) map[requirementKey]models.StaffingRequirement {
	m := make(map[requirementKey]models.StaffingRequirement)
	for _, r := range reqs {
		m[requirementKey{r.Weekday, r.ShiftType}] = r
	}
	return m
}

// buildConstraints 建立員工約束快速索引表
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

// canAssignV6 - 完整約束判斷 (含 C7 雙向檢查)
func canAssignV6(
	empID uint,
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	constraints map[uint]*employeeConstraint,
	day int,
	shiftType string,
	totalDays int,
	externalPrevDayShift map[uint]string,
) bool {
	ec := constraints[empID]

	if ec != nil && ec.IsDay88Primary && shiftType != "day88" {
		return false // 只能排 day88
	}

	// R4: 個人禁排班別
	if ec != nil {
		if ec.Banned[shiftType] {
			return false
		}
		if shiftType == "night88" && ec.Banned["night"] {
			return false
		}
	}

	// R5: 個人班別天數上限
	if ec != nil {
		if shiftType == "night" || shiftType == "night88" {
			if maxD, ok := ec.MaxDays["night"]; ok {
				if shiftCount[empID]["night"]+shiftCount[empID]["night88"] >= maxD {
					return false
				}
			}
		} else {
			if maxD, ok := ec.MaxDays[shiftType]; ok {
				if shiftCount[empID][shiftType] >= maxD {
					return false
				}
			}
		}
	}

	// R3: 每人每天最多一班 (已有班或已排假)
	if existing, ok := schedule[day][empID]; ok && existing != "" {
		return false
	}

	// R1, R2: 夜班 → 隔天白班 ❌
	if shiftType == "day" || shiftType == "day88" {
		if day > 0 {
			prev := schedule[day-1][empID]
			if isNightShift(prev) {
				return false
			}
		} else if day == 0 && externalPrevDayShift != nil {
			prev := externalPrevDayShift[empID]
			if isNightShift(prev) {
				return false
			}
		}
	}

	// 反向檢查：當天排夜班，但明天已排白班 → 禁止
	if isNightShift(shiftType) {
		if day+1 < totalDays {
			next := schedule[day+1][empID]
			if next == "day" || next == "day88" {
				return false
			}
		}
	}

	// R8: night88 不可連續兩天 (往後檢查這天如果排 night88 下天不能是, 往前檢查前天不能是)
	if shiftType == "night88" {
		if day > 0 && schedule[day-1][empID] == "night88" {
			return false
		} else if day == 0 && externalPrevDayShift != nil && externalPrevDayShift[empID] == "night88" {
			return false
		}
		if day+1 < totalDays && schedule[day+1][empID] == "night88" {
			return false
		}
	}

	// R10: 大夜/小夜最多連續 4 天
	if isNightShift(shiftType) {
		if !checkNightEveningMaxConsecutive(empID, schedule, day, shiftType, totalDays, externalPrevDayShift) {
			return false
		}
	}

	// C7: 做 6 休 1 (雙向檢查) - 所有員工都適用（含 J）
	{
		backwardStreak := 0
		for d := day - 1; d >= 0; d-- {
			s := schedule[d][empID]
			if s != "" && s != "off" && s != "pre_off" {
				backwardStreak++
			} else {
				break
			}
		}

		// 跨月份邊界檢查：若一直連到第一天，需加上上個月末尾的連續天數
		if day == 0 && externalPrevDayShift != nil {
			prev := externalPrevDayShift[empID]
			if prev != "" && prev != "off" && prev != "pre_off" {
				backwardStreak++
			}
		} else if day > 0 && backwardStreak == day && externalPrevDayShift != nil {
			prev := externalPrevDayShift[empID]
			if prev != "" && prev != "off" && prev != "pre_off" {
				backwardStreak++
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
		if backwardStreak+1+forwardStreak > 6 {
			return false
		}
	}

	return true
}

// canAssignV6Relaxed - 寬鬆約束判斷
func canAssignV6Relaxed(
	empID uint,
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	constraints map[uint]*employeeConstraint,
	day int,
	shiftType string,
	totalDays int,
	externalPrevDayShift map[uint]string,
) bool {
	ec := constraints[empID]
	if ec != nil && ec.IsDay88Primary && shiftType != "day88" {
		return false
	}
	if ec != nil {
		if ec.Banned[shiftType] {
			return false
		}
		if shiftType == "night88" && ec.Banned["night"] {
			return false
		}
	}

	// R5: 個人班別天數上限
	if ec != nil {
		if shiftType == "night" || shiftType == "night88" {
			if maxD, ok := ec.MaxDays["night"]; ok {
				if shiftCount[empID]["night"]+shiftCount[empID]["night88"] >= maxD {
					return false
				}
			}
		} else {
			if maxD, ok := ec.MaxDays[shiftType]; ok {
				if shiftCount[empID][shiftType] >= maxD {
					return false
				}
			}
		}
	}
	if shiftType == "day" || shiftType == "day88" {
		if day > 0 {
			prev := schedule[day-1][empID]
			if isNightShift(prev) {
				return false
			}
		} else if day == 0 && externalPrevDayShift != nil {
			prev := externalPrevDayShift[empID]
			if isNightShift(prev) {
				return false
			}
		}
	}
	if isNightShift(shiftType) {
		if day+1 < totalDays {
			next := schedule[day+1][empID]
			if next == "day" || next == "day88" {
				return false
			}
		}
	}
	if s, ok := schedule[day][empID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
		return false
	}

	// R10: 大夜/小夜最多連續 4 天 — 寬鬆模式也必須遵守
	if isNightShift(shiftType) {
		if !checkNightEveningMaxConsecutive(empID, schedule, day, shiftType, totalDays, externalPrevDayShift) {
			return false
		}
	}

	// R6: 做 6 休 1 (雙向檢查) — 寬鬆模式也必須遵守此硬性約束
	{
		backwardStreak := 0
		for d := day - 1; d >= 0; d-- {
			s := schedule[d][empID]
			if s != "" && s != "off" && s != "pre_off" {
				backwardStreak++
			} else {
				break
			}
		}

		// 跨月份邊界檢查
		if day == 0 && externalPrevDayShift != nil {
			prev := externalPrevDayShift[empID]
			if prev != "" && prev != "off" && prev != "pre_off" {
				backwardStreak++
			}
		} else if day > 0 && backwardStreak == day && externalPrevDayShift != nil {
			prev := externalPrevDayShift[empID]
			if prev != "" && prev != "off" && prev != "pre_off" {
				backwardStreak++
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
		if backwardStreak+1+forwardStreak > 6 {
			return false
		}
	}

	return true
}

// isNightShift 判斷是否為會產生休息時間衝突的夜間班別
func isNightShift(st string) bool {
	return st == "night" || st == "evening" || st == "night88"
}

// isNightFamily 判斷是否為大夜家族 (night / night88 視為同類)
func isNightFamily(st string) bool {
	return st == "night" || st == "night88"
}

// checkNightEveningMaxConsecutive 檢查大夜/小夜是否超過最大連續天數 (4 天)
// night 和 night88 屬於同家族，互相視為連續
func checkNightEveningMaxConsecutive(
	empID uint,
	schedule map[int]map[uint]string,
	day int,
	shiftType string,
	totalDays int,
	externalPrevDayShift map[uint]string,
) bool {
	const maxConsecutive = 4

	// 判斷 s 是否與 shiftType 同家族
	isSameFamily := func(s string) bool {
		if isNightFamily(shiftType) {
			return isNightFamily(s)
		}
		return s == shiftType // evening 只與 evening 同家族
	}

	// 往前數連續同家族天數
	backward := 0
	for d := day - 1; d >= 0; d-- {
		if isSameFamily(schedule[d][empID]) {
			backward++
		} else {
			break
		}
	}
	// 跨月邊界
	if day == 0 && externalPrevDayShift != nil {
		if isSameFamily(externalPrevDayShift[empID]) {
			backward++
		}
	} else if day > 0 && backward == day && externalPrevDayShift != nil {
		if isSameFamily(externalPrevDayShift[empID]) {
			backward++
		}
	}

	// 往後數連續同家族天數
	forward := 0
	for d := day + 1; d < totalDays; d++ {
		if isSameFamily(schedule[d][empID]) {
			forward++
		} else {
			break
		}
	}

	return backward+1+forward <= maxConsecutive
}

// findBestCandidateV3 尋找最適合排入特定日期的員工
func findBestCandidateV3(
	employees []models.Employee,
	schedule map[int]map[uint]string,
	shiftCount map[uint]map[string]int,
	constraints map[uint]*employeeConstraint,
	day int,
	shiftType string,
	totalDays int,
	preLeaves []models.PreScheduledLeave,
	externalPrevDayShift map[uint]string,
) uint {
	// 1. 優先找昨天也是同班的人
	if day > 0 {
		var candidates []uint
		for _, emp := range employees {
			if schedule[day-1][emp.ID] == shiftType && canAssignV6(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays, externalPrevDayShift) {
				candidates = append(candidates, emp.ID)
			}
		}
		if len(candidates) > 0 {
			return pickLowestWork(candidates, shiftCount)
		}
	} else if day == 0 && externalPrevDayShift != nil {
		var candidates []uint
		for _, emp := range employees {
			if externalPrevDayShift[emp.ID] == shiftType && canAssignV6(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays, externalPrevDayShift) {
				candidates = append(candidates, emp.ID)
			}
		}
		if len(candidates) > 0 {
			return pickLowestWork(candidates, shiftCount)
		}
	}

	// 2. 找符合嚴格約束的人
	var eligible []uint
	for _, emp := range employees {
		if s, ok := schedule[day][emp.ID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
			continue
		}
		if isPreLeave(emp.ID, day, preLeaves) || schedule[day][emp.ID] == "pre_off" {
			continue
		}
		if emp.IsDay88Primary {
			continue
		}
		if canAssignV6(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays, externalPrevDayShift) {
			eligible = append(eligible, emp.ID)
		}
	}
	if len(eligible) > 0 {
		return pickLowestWork(eligible, shiftCount)
	}

	// 3. 寬鬆模式
	var relaxedEligible []uint
	for _, emp := range employees {
		if s, ok := schedule[day][emp.ID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
			continue
		}
		if isPreLeave(emp.ID, day, preLeaves) {
			continue
		}
		if emp.IsDay88Primary && shiftType != "day88" {
			continue
		}
		if canAssignV6Relaxed(emp.ID, schedule, shiftCount, constraints, day, shiftType, totalDays, externalPrevDayShift) {
			relaxedEligible = append(relaxedEligible, emp.ID)
		}
	}
	if len(relaxedEligible) > 0 {
		return pickLowestWork(relaxedEligible, shiftCount)
	}

	// 4. 強制模式 (Force Mode)
	var forceEligible []uint
	for _, emp := range employees {
		if s, ok := schedule[day][emp.ID]; ok && (s == "pre_off" || (s != "" && s != "off")) {
			continue
		}
		if isPreLeave(emp.ID, day, preLeaves) {
			continue
		}
		if shiftType == "day" || shiftType == "day88" {
			prev := ""
			if day > 0 {
				prev = schedule[day-1][emp.ID]
			} else if day == 0 && externalPrevDayShift != nil {
				prev = externalPrevDayShift[emp.ID]
			}
			if isNightShift(prev) {
				continue
			}
		}
		// R5: 強制模式也要檢查班別天數上限
		if ec, ok := constraints[emp.ID]; ok {
			if maxD, exists := ec.MaxDays[shiftType]; exists {
				if shiftCount[emp.ID][shiftType] >= maxD {
					continue
				}
			}
		}
		// R10: 大夜/小夜最多連續 4 天 — 強制模式也不可違反
		if isNightShift(shiftType) {
			if !checkNightEveningMaxConsecutive(emp.ID, schedule, day, shiftType, totalDays, externalPrevDayShift) {
				continue
			}
		}
		// R6: 做 6 休 1 — 即便強制模式也不可違反
		{
			backwardStreak := 0
			for d := day - 1; d >= 0; d-- {
				s := schedule[d][emp.ID]
				if s != "" && s != "off" && s != "pre_off" {
					backwardStreak++
				} else {
					break
				}
			}
			if day == 0 && externalPrevDayShift != nil {
				prev := externalPrevDayShift[emp.ID]
				if prev != "" && prev != "off" && prev != "pre_off" {
					backwardStreak++
				}
			} else if day > 0 && backwardStreak == day && externalPrevDayShift != nil {
				prev := externalPrevDayShift[emp.ID]
				if prev != "" && prev != "off" && prev != "pre_off" {
					backwardStreak++
				}
			}
			forwardStreak := 0
			for d := day + 1; d < totalDays; d++ {
				s := schedule[d][emp.ID]
				if s != "" && s != "off" && s != "pre_off" {
					forwardStreak++
				} else {
					break
				}
			}
			if backwardStreak+1+forwardStreak > 6 {
				continue
			}
		}
		forceEligible = append(forceEligible, emp.ID)
	}
	if len(forceEligible) > 0 {
		return pickLowestWork(forceEligible, shiftCount)
	}

	return 0
}

// fillConsecutiveV3 填充連續班次
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
	externalPrevDayShift map[uint]string,
) {
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

	// 1. 基於公平原則排序：本月工作量少的人優先參與夜選/小夜選 (人才庫輪替)
	// 先計算每人工作量，用於排序
	workCount := make(map[uint]int)
	for _, id := range eligible {
		w := 0
		for st, count := range shiftCount[id] {
			if st != "off" && st != "pre_off" && st != "" {
				w += count
			}
		}
		workCount[id] = w
	}

	// 先完全隨機打亂，消除原始陣列順序的影響
	rand.Shuffle(len(eligible), func(i, j int) {
		eligible[i], eligible[j] = eligible[j], eligible[i]
	})

	// 再按工作量穩定排序（工作量相同的人保持打亂後的隨機順序）
	sort.SliceStable(eligible, func(i, j int) bool {
		return workCount[eligible[i]] < workCount[eligible[j]]
	})

	// 2. 僅保留所需人數 (由排序後的頂部選取，確保公平性)
	if len(eligible) > maxPeople {
		eligible = eligible[:maxPeople]
	}

	// 3. 隨機打亂入選者的處理順序，避免固定順序產生的 pattern
	rand.Shuffle(len(eligible), func(i, j int) {
		eligible[i], eligible[j] = eligible[j], eligible[i]
	})

	for _, empID := range eligible {
		blockCycle := runLen + restLen
		// ⭐ 智慧多樣化：不再使用 (i*2)%block，改用隨機初始偏移量
		offset := rand.Intn(blockCycle)
		for d := offset; d < totalDays; {
			canDoAll := true
			actualRun := 0
			for r := 0; r < runLen && d+r < totalDays; r++ {
				if !canAssignV6(empID, schedule, shiftCount, constraints, d+r, shiftType, totalDays, externalPrevDayShift) {
					canDoAll = false
					break
				}
				actualRun++
			}

			if canDoAll && actualRun >= 2 {
				// R5: 檢查整段分配後是否會超過 MaxDays 上限
				if ec, ok := constraints[empID]; ok {
					if maxD, exists := ec.MaxDays[shiftType]; exists {
						allowed := maxD - shiftCount[empID][shiftType]
						if allowed <= 0 {
							d++
							continue
						}
						if actualRun > allowed {
							actualRun = allowed
						}
						if actualRun < 2 {
							d++
							continue
						}
					}
				}
				for r := 0; r < actualRun; r++ {
					schedule[d+r][empID] = shiftType
					shiftCount[empID][shiftType]++
				}
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

func pickLowestWork(ids []uint, shiftCount map[uint]map[string]int) uint {
	if len(ids) == 0 {
		return 0
	}
	// 先打亂候選人順序，確保同一水平的人選機會均等
	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})

	var best uint
	minWork := 9999
	for _, id := range ids {
		work := 0
		for shiftType, c := range shiftCount[id] {
			if shiftType != "off" && shiftType != "pre_off" && shiftType != "" {
				work += c
			}
		}
		if work < minWork {
			minWork = work
			best = id
		}
	}
	return best
}

func isPreLeave(empID uint, day int, preLeaves []models.PreScheduledLeave) bool {
	for _, pl := range preLeaves {
		if pl.EmployeeID == empID && pl.DayOffset == day {
			return true
		}
	}
	return false
}
