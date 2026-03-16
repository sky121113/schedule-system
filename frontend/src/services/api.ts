import axios from 'axios';
import type {
  Employee,
  ShiftRestriction,
  StaffingRequirement,
  CycleTemplate,
  TemplateSlot,
  AutoScheduleResult,
  ValidationResult,
  CalendarEntry,
} from '../types';

// Axios 實例
const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

// --- 員工 ---
export const getEmployees = () => api.get<Employee[]>('/employees/');
export const getEmployee = (id: number) => api.get<Employee>(`/employees/${id}`);
export const createEmployee = (data: Partial<Employee>) => api.post('/employees/', data);
export const updateEmployee = (id: number, data: Partial<Employee>) => api.put(`/employees/${id}`, data);
export const deleteEmployee = (id: number) => api.delete(`/employees/${id}`);

// --- 班別限制 ---
export const getEmployeeRestrictions = (id: number, templateId?: number) =>
  api.get<ShiftRestriction[]>(`/employees/${id}/restrictions`, {
    params: templateId ? { template_id: templateId } : {},
  });
export const createRestriction = (data: Partial<ShiftRestriction>) => api.post('/restrictions/', data);
export const deleteRestriction = (id: number) => api.delete(`/restrictions/${id}`);
export const validateRestrictions = (templateId?: number) =>
  api.get<ValidationResult>('/restrictions/validate', {
    params: templateId ? { template_id: templateId } : {},
  });

// --- 人力需求 ---
export const getStaffingRequirements = () => api.get<StaffingRequirement[]>('/staffing/');
export const upsertStaffingRequirement = (data: Partial<StaffingRequirement>) => api.post('/staffing/', data);
export const batchUpsertStaffingRequirements = (data: Partial<StaffingRequirement>[]) =>
  api.post('/staffing/batch', data);

// --- 循環模板 ---
export const getTemplates = () => api.get<CycleTemplate[]>('/templates/');
export const getTemplate = (id: number) =>
  api.get<{ template: CycleTemplate; slots: TemplateSlot[] }>(`/templates/${id}`);
export const createTemplate = (data: { start_date: string; cycle_weeks?: number }) =>
  api.post('/templates/', data);
export const deleteTemplate = (id: number) => api.delete(`/templates/${id}`);

// --- 排班格 ---
export const setSlot = (data: Partial<TemplateSlot>) => api.post('/templates/slots', data);
export const removeSlot = (id: number) => api.delete(`/templates/slots/${id}`);
export const clearTemplateSlots = (templateId: number) => api.delete(`/templates/${templateId}/slots`);

// --- 自動排班 ---
export const autoSchedule = (templateId: number) =>
  api.post<AutoScheduleResult>(`/templates/${templateId}/auto-schedule`);

// --- 日曆展開 ---
export const getTemplateCalendar = (templateId: number) =>
  api.get<{ template: CycleTemplate; calendar: CalendarEntry[] }>(`/templates/${templateId}/calendar`);

// --- 預假與假期配額 ---
export interface PreScheduledLeave {
  ID: number;
  employee_id: number;
  template_id: number;
  day_offset: number;
  reason: string;
}

export const getPreLeaves = (templateId: number) =>
  api.get<PreScheduledLeave[]>(`/templates/${templateId}/pre-leaves`);

export const setPreLeave = (templateId: number, data: { employee_id: number; day_offset: number; reason?: string }) =>
  api.post(`/templates/${templateId}/pre-leaves`, { ...data, template_id: templateId });

export const deletePreLeave = (templateId: number, leaveId: number) =>
  api.delete(`/templates/${templateId}/pre-leaves/${leaveId}`);

export interface LeaveQuotaStats {
  total_available: number;
  total_required: number;
  total_leave: number;
  per_person_leave: number;
  active_employees: number;
  total_days: number;
}

export const getLeaveQuota = (templateId: number) =>
  api.get<LeaveQuotaStats>(`/templates/${templateId}/leave-quota`);

// --- 月度班表 ---
export interface MonthlySlot {
  ID: number;
  schedule_id: number;
  date: string;
  shift_type: string;
  employee_id: number;
  cycle_index: number;
  day_offset: number;
}

export interface CycleBoundary {
  cycle_index: number;
  start_date: string;
  end_date: string;
  days_in_month: number;
  total_days: number;
  default_total_leave: number;
}

export interface MonthlyScheduleResponse {
  schedule: { ID: number; year: number; month: number; status: string };
  slots: MonthlySlot[];
  employees: Record<number, string>;
  boundaries: CycleBoundary[];
}

export interface LeaveSummaryItem {
  employee_id: number;
  employee_name: string;
  cycle_index: number;
  total_leave: number;
  used_leave: number;
  remaining: number;
  current_month_quota: number;
}

export const getMonthlySchedule = (year: number, month: number) =>
  api.get<MonthlyScheduleResponse>(`/monthly/${year}/${month}`);

export const generateMonthlySchedule = (
  year: number,
  month: number,
  cycleBalances?: { cycle_index: number; employee_id: number; total_leave: number }[]
) => api.post(`/monthly/${year}/${month}/generate`, { cycle_balances: cycleBalances || [] });

export const getMonthlyLeaveSummary = (year: number, month: number) =>
  api.get<{ boundaries: CycleBoundary[]; summaries: LeaveSummaryItem[] }>(`/monthly/${year}/${month}/leave-summary`);

export const getCycleBoundaries = (year: number, month: number) =>
  api.get<{ boundaries: CycleBoundary[] }>(`/monthly/${year}/${month}/boundaries`);

export const updateCycleBalance = (data: {
  cycle_index: number;
  employee_id: number;
  total_leave: number;
  used_leave?: number;
}) => api.put('/monthly/cycle-balance', data);

export const updateMonthlySlot = (slotId: number, shiftType: string) =>
  api.put<{ message: string; slot: MonthlySlot; warnings?: string[]; summaries?: LeaveSummaryItem[]; boundaries?: CycleBoundary[] }>(
    `/monthly/slots/${slotId}`,
    { shift_type: shiftType }
  );

// --- 月度預假 ---
export interface MonthlyPreScheduledLeave {
  ID: number;
  employee_id: number;
  date: string;
  reason: string;
}

export const getMonthlyPreLeaves = (year: number, month: number) =>
  api.get<MonthlyPreScheduledLeave[]>(`/monthly/${year}/${month}/pre-leaves`);

export const createMonthlyPreLeave = (data: { employee_id: number; date: string; reason?: string }) =>
  api.post<MonthlyPreScheduledLeave>('/monthly/pre-leaves', data);

export const deleteMonthlyPreLeave = (id: number) =>
  api.delete(`/monthly/pre-leaves/${id}`);

// --- 班表版本管理 ---
export interface MonthlyScheduleVersion {
  ID: number;
  CreatedAt: string;
  year: number;
  month: number;
  version_name: string;
  creator: string;
}

export const listMonthlyVersions = (year: number, month: number) =>
  api.get<MonthlyScheduleVersion[]>(`/monthly/${year}/${month}/versions`);

export const saveMonthlyVersion = (year: number, month: number, data: { version_name: string; creator?: string }) =>
  api.post(`/monthly/${year}/${month}/versions`, data);

export const restoreMonthlyVersion = (versionId: number) =>
  api.post<{ message: string; slots: MonthlySlot[]; warnings: string[]; summaries: LeaveSummaryItem[]; boundaries: CycleBoundary[] }>(
    `/monthly/versions/${versionId}/restore`
  );

export const deleteMonthlyVersion = (versionId: number) =>
  api.delete(`/monthly/versions/${versionId}`);
