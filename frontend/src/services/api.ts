import api from '../utils/api';
import { ShiftType, MonthlySchedule } from '../types';

export const shiftApi = {
    // 設定班表需求
    setRequirement: (date: string, shiftType: ShiftType, requiredCount: number) => {
        return api.post('/shifts/requirements', {
            date,
            shift_type: shiftType,
            required_count: requiredCount,
        });
    },

    // 取得月份班表
    getMonthlySchedule: (month: string): Promise<MonthlySchedule> => {
        return api.get('/shifts/schedule', {
            params: { month },
        });
    },

    // 預約班別
    bookShift: (userId: number, date: string, shiftType: ShiftType) => {
        return api.post('/shifts/book', {
            user_id: userId,
            date,
            shift_type: shiftType,
        });
    },
};

export const userApi = {
    // 取得所有使用者
    getUsers: () => {
        return api.get('/users/');
    },

    // 取得單一使用者
    getUser: (id: number) => {
        return api.get(`/users/${id}`);
    },
};
