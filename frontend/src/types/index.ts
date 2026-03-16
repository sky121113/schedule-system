// 員工模型
export interface Employee {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  name: string;
  email: string;
  is_day88_primary: boolean;
  status: number; // 1=在職, 0=停用, 2=長期請假
}

// 班別限制
export interface ShiftRestriction {
  ID: number;
  employee_id: number;
  template_id: number | null;
  shift_type: ShiftType;
  max_days: number | null; // null=完全禁止
  note: string;
}

// 人力需求
export interface StaffingRequirement {
  ID: number;
  weekday: number; // 0=日~6=六
  shift_type: ShiftType;
  min_count: number;
  min_count_with_day88: number;
}

// 循環模板
export interface CycleTemplate {
  ID: number;
  start_date: string;
  cycle_weeks: number;
  version: number;
  status: 'draft' | 'active' | 'archived';
}

// 模板排班格
export interface TemplateSlot {
  ID: number;
  template_id: number;
  day_offset: number;
  shift_type: ShiftType;
  employee_id: number;
}

// 班別類型
export type ShiftType = 'day' | 'day88' | 'evening' | 'night' | 'off';

// 班別顏色與標籤
export const SHIFT_CONFIG: Record<ShiftType, { label: string; color: string; bgColor: string }> = {
  day: { label: '白', color: '#faad14', bgColor: '#fffbe6' },
  day88: { label: '8-8', color: '#fa8c16', bgColor: '#fff7e6' },
  evening: { label: '小夜', color: '#722ed1', bgColor: '#f9f0ff' },
  night: { label: '大夜', color: '#1890ff', bgColor: '#e6f7ff' },
  off: { label: '休', color: '#52c41a', bgColor: '#f6ffed' },
};

// 星期名稱
export const WEEKDAY_NAMES = ['日', '一', '二', '三', '四', '五', '六'];

// 員工統計
export interface EmployeeStats {
  name: string;
  shift_days: Record<string, number>;
  off_days: number;
  total_work: number;
}

// 自動排班結果
export interface AutoScheduleResult {
  message: string;
  slots: TemplateSlot[];
  stats: {
    employees: Record<number, EmployeeStats>;
    evening_fairness: number;
    night_fairness: number;
  };
}

// 日曆項目
export interface CalendarEntry {
  date: string;
  shift_type: ShiftType;
  employee_id: number;
}

// 驗證結果
export interface ValidationResult {
  valid: boolean;
  warnings: string[];
}
