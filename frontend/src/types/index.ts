export type ShiftType = 'morning' | 'afternoon' | 'evening';

export interface User {
    ID: number;
    name: string;
    email: string;
    role: string;
    status: number;
}

export interface ShiftRequirement {
    ID: number;
    date: string;
    shift_type: ShiftType;
    required_count: number;
}

export interface UserSchedule {
    ID: number;
    user_id: number;
    date: string;
    shift_type: ShiftType;
}

export interface DailyShiftStatus {
    required: number;
    booked: number;
}

export interface MonthlySchedule {
    [date: string]: {
        [key in ShiftType]?: DailyShiftStatus;
    };
}
