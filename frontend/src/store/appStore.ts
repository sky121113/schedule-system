import { create } from 'zustand';
import type { Employee, CycleTemplate, TemplateSlot, StaffingRequirement, ShiftRestriction } from '../types';
import * as api from '../services/api';

interface AppState {
  // 員工
  employees: Employee[];
  loadingEmployees: boolean;
  fetchEmployees: () => Promise<void>;

  // 人力需求
  staffingRequirements: StaffingRequirement[];
  fetchStaffingRequirements: () => Promise<void>;

  // 模板
  templates: CycleTemplate[];
  currentTemplate: CycleTemplate | null;
  currentSlots: TemplateSlot[];
  fetchTemplates: () => Promise<void>;
  fetchTemplate: (id: number) => Promise<void>;
  setCurrentTemplate: (t: CycleTemplate | null) => void;

  // 限制
  restrictions: Record<number, ShiftRestriction[]>; // employeeID -> restrictions
  fetchRestrictions: (employeeId: number) => Promise<void>;
}

export const useAppStore = create<AppState>((set) => ({
  employees: [],
  loadingEmployees: false,
  fetchEmployees: async () => {
    set({ loadingEmployees: true });
    try {
      const res = await api.getEmployees();
      set({ employees: res.data, loadingEmployees: false });
    } catch {
      set({ loadingEmployees: false });
    }
  },

  staffingRequirements: [],
  fetchStaffingRequirements: async () => {
    const res = await api.getStaffingRequirements();
    set({ staffingRequirements: res.data });
  },

  templates: [],
  currentTemplate: null,
  currentSlots: [],
  fetchTemplates: async () => {
    const res = await api.getTemplates();
    set({ templates: res.data });
  },
  fetchTemplate: async (id: number) => {
    const res = await api.getTemplate(id);
    set({ currentTemplate: res.data.template, currentSlots: res.data.slots });
  },
  setCurrentTemplate: (t) => set({ currentTemplate: t }),

  restrictions: {},
  fetchRestrictions: async (employeeId: number) => {
    const res = await api.getEmployeeRestrictions(employeeId);
    set((state) => ({
      restrictions: { ...state.restrictions, [employeeId]: res.data },
    }));
  },
}));
