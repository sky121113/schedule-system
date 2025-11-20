import { useState, useEffect } from 'react';
import { Calendar, Badge, Modal, Select, Button, message, Card, Row, Col, Statistic } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { shiftApi } from '../services/api';
import { MonthlySchedule, ShiftType } from '../types';
import { useUserStore } from '../store/userStore';

const { Option } = Select;

const shiftTypeNames: Record<ShiftType, string> = {
    morning: '早班',
    afternoon: '中班',
    evening: '晚班',
};

const shiftTypeColors: Record<ShiftType, string> = {
    morning: '#faad14',
    afternoon: '#52c41a',
    evening: '#1890ff',
};

const ScheduleCalendar = () => {
    const [scheduleData, setScheduleData] = useState<MonthlySchedule>({});
    const [selectedDate, setSelectedDate] = useState<Dayjs | null>(null);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [selectedShift, setSelectedShift] = useState<ShiftType>('morning');
    const { currentUser } = useUserStore();

    const fetchSchedule = async (date: Dayjs) => {
        try {
            const month = date.format('YYYY-MM');
            const data = await shiftApi.getMonthlySchedule(month);
            setScheduleData(data);
        } catch (error) {
            message.error('載入班表失敗');
            console.error(error);
        }
    };

    useEffect(() => {
        fetchSchedule(dayjs());
    }, []);

    const onPanelChange = (date: Dayjs) => {
        fetchSchedule(date);
    };

    const onSelect = (date: Dayjs) => {
        setSelectedDate(date);
        setIsModalVisible(true);
    };

    const handleBookShift = async () => {
        if (!selectedDate || !currentUser) {
            message.warning('請先選擇日期並登入');
            return;
        }

        try {
            await shiftApi.bookShift(
                currentUser.ID,
                selectedDate.format('YYYY-MM-DD'),
                selectedShift
            );
            message.success('預約成功！');
            setIsModalVisible(false);
            fetchSchedule(selectedDate);
        } catch (error: any) {
            message.error(error?.response?.data?.error || '預約失敗');
        }
    };

    const dateCellRender = (date: Dayjs) => {
        const dateStr = date.format('YYYY-MM-DD');
        const daySchedule = scheduleData[dateStr];

        if (!daySchedule) return null;

        return (
            <div style={{ padding: '4px' }}>
                {Object.entries(daySchedule).map(([shiftType, status]) => {
                    const shift = shiftType as ShiftType;
                    const isFull = status.booked >= status.required;
                    return (
                        <div key={shift} style={{ marginBottom: '2px' }}>
                            <Badge
                                color={isFull ? 'red' : shiftTypeColors[shift]}
                                text={
                                    <span style={{ fontSize: '12px' }}>
                                        {shiftTypeNames[shift]}: {status.booked}/{status.required}
                                    </span>
                                }
                            />
                        </div>
                    );
                })}
            </div>
        );
    };

    const getTodayStats = () => {
        const today = dayjs().format('YYYY-MM-DD');
        const todaySchedule = scheduleData[today];

        if (!todaySchedule) return { total: 0, booked: 0 };

        let total = 0;
        let booked = 0;

        Object.values(todaySchedule).forEach(status => {
            total += status.required;
            booked += status.booked;
        });

        return { total, booked };
    };

    const stats = getTodayStats();

    return (
        <div>
            <Row gutter={16} style={{ marginBottom: 24 }}>
                <Col span={8}>
                    <Card>
                        <Statistic
                            title="今日需求總人數"
                            value={stats.total}
                            valueStyle={{ color: '#3f8600' }}
                        />
                    </Card>
                </Col>
                <Col span={8}>
                    <Card>
                        <Statistic
                            title="今日已排班人數"
                            value={stats.booked}
                            valueStyle={{ color: '#1890ff' }}
                        />
                    </Card>
                </Col>
                <Col span={8}>
                    <Card>
                        <Statistic
                            title="今日缺額"
                            value={stats.total - stats.booked}
                            valueStyle={{ color: stats.total - stats.booked > 0 ? '#cf1322' : '#3f8600' }}
                        />
                    </Card>
                </Col>
            </Row>

            <Calendar
                cellRender={dateCellRender}
                onPanelChange={onPanelChange}
                onSelect={onSelect}
            />

            <Modal
                title={`預約班別 - ${selectedDate?.format('YYYY-MM-DD')}`}
                open={isModalVisible}
                onOk={handleBookShift}
                onCancel={() => setIsModalVisible(false)}
            >
                <div style={{ marginBottom: 16 }}>
                    <label>選擇班別：</label>
                    <Select
                        style={{ width: '100%', marginTop: 8 }}
                        value={selectedShift}
                        onChange={setSelectedShift}
                    >
                        <Option value="morning">
                            <Badge color={shiftTypeColors.morning} text="早班" />
                        </Option>
                        <Option value="afternoon">
                            <Badge color={shiftTypeColors.afternoon} text="中班" />
                        </Option>
                        <Option value="evening">
                            <Badge color={shiftTypeColors.evening} text="晚班" />
                        </Option>
                    </Select>
                </div>

                {selectedDate && scheduleData[selectedDate.format('YYYY-MM-DD')] && (
                    <div style={{ marginTop: 16, padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
                        <h4>當日班表狀態</h4>
                        {Object.entries(scheduleData[selectedDate.format('YYYY-MM-DD')]).map(([shift, status]) => (
                            <div key={shift} style={{ marginTop: 8 }}>
                                <Badge
                                    color={shiftTypeColors[shift as ShiftType]}
                                    text={`${shiftTypeNames[shift as ShiftType]}: ${status.booked}/${status.required} 人`}
                                />
                            </div>
                        ))}
                    </div>
                )}
            </Modal>
        </div>
    );
};

export default ScheduleCalendar;
