import { useEffect, useState } from 'react';
import { Table, InputNumber, Button, message, Tag } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useAppStore } from '../store/appStore';
import { WEEKDAY_NAMES, SHIFT_CONFIG } from '../types';
import type { StaffingRequirement, ShiftType } from '../types';
import * as api from '../services/api';

// 人力需求設定頁面
const StaffingSettings = () => {
  const { staffingRequirements, fetchStaffingRequirements } = useAppStore();
  const [editData, setEditData] = useState<Record<string, { min: number; min88: number }>>({});

  useEffect(() => {
    fetchStaffingRequirements();
  }, []);

  useEffect(() => {
    // 初始化編輯資料
    const data: Record<string, { min: number; min88: number }> = {};
    // 預設值
    for (let w = 0; w < 7; w++) {
      for (const st of ['day', 'evening', 'night']) {
        data[`${w}-${st}`] = { min: 0, min88: 0 };
      }
    }
    // 填入現有資料
    for (const r of staffingRequirements) {
      data[`${r.weekday}-${r.shift_type}`] = {
        min: r.min_count,
        min88: r.min_count_with_day88,
      };
    }
    setEditData(data);
  }, [staffingRequirements]);

  const handleSave = async () => {
    const batch: Partial<StaffingRequirement>[] = [];
    for (const [key, val] of Object.entries(editData)) {
      const [weekday, shiftType] = key.split('-');
      batch.push({
        weekday: parseInt(weekday),
        shift_type: shiftType as ShiftType,
        min_count: val.min,
        min_count_with_day88: val.min88,
      });
    }
    try {
      await api.batchUpsertStaffingRequirements(batch);
      message.success('儲存成功');
      fetchStaffingRequirements();
    } catch {
      message.error('儲存失敗');
    }
  };

  const updateVal = (key: string, field: 'min' | 'min88', value: number) => {
    setEditData((prev) => ({
      ...prev,
      [key]: { ...prev[key], [field]: value },
    }));
  };

  // 建立表格
  const shiftTypes = ['day', 'evening', 'night'] as const;
  const columns = [
    {
      title: '星期',
      key: 'weekday',
      width: 80,
      render: (_: unknown, __: unknown, index: number) => (
        <strong>星期{WEEKDAY_NAMES[index]}</strong>
      ),
    },
    ...shiftTypes.flatMap((st) => [
      {
        title: () => (
          <span>
            <Tag color={SHIFT_CONFIG[st].color}>{SHIFT_CONFIG[st].label}</Tag> 最少
          </span>
        ),
        key: `${st}-min`,
        width: 120,
        render: (_: unknown, __: unknown, index: number) => {
          const key = `${index}-${st}`;
          return (
            <InputNumber
              size="small"
              min={0}
              max={10}
              value={editData[key]?.min ?? 0}
              onChange={(v) => updateVal(key, 'min', v ?? 0)}
            />
          );
        },
      },
      {
        title: () => (
          <span>
            <Tag color={SHIFT_CONFIG[st].color}>{SHIFT_CONFIG[st].label}</Tag> 含8系列
          </span>
        ),
        key: `${st}-min88`,
        width: 120,
        render: (_: unknown, __: unknown, index: number) => {
          const key = `${index}-${st}`;
          return (
            <InputNumber
              size="small"
              min={0}
              max={10}
              value={editData[key]?.min88 ?? 0}
              onChange={(v) => updateVal(key, 'min88', v ?? 0)}
            />
          );
        },
      },
    ]),
  ];

  // 7 天陣列
  const dataSource = Array.from({ length: 7 }, (_, i) => ({ key: i }));

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>📊 人力需求設定</h2>
        <Button type="primary" icon={<SaveOutlined />} onClick={handleSave}>
          儲存
        </Button>
      </div>

      <Table
        dataSource={dataSource}
        columns={columns}
        pagination={false}
        size="small"
        bordered
      />
    </div>
  );
};

export default StaffingSettings;
