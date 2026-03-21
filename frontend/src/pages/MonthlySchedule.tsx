import { useState, useEffect, useCallback } from 'react';
import {
  Button, Card, Select, Space, Table, Tag, message, Modal,
  InputNumber, Form, Dropdown, Spin, Row, Col, Divider,
  Drawer, List, Input, Popconfirm, Tabs, Badge
} from 'antd';
import {
  LeftOutlined, RightOutlined, ThunderboltOutlined, EditOutlined,
  HistoryOutlined, SaveOutlined, DeleteOutlined, InfoCircleOutlined,
  BellOutlined
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import {
  getMonthlySchedule, generateMonthlySchedule, updateMonthlySlot,
  getMonthlyLeaveSummary, getCycleBoundaries, getEmployees,
  getMonthlyPreLeaves, createMonthlyPreLeave, deleteMonthlyPreLeave,
  listMonthlyVersions, saveMonthlyVersion, restoreMonthlyVersion, deleteMonthlyVersion,
  type MonthlySlot, type CycleBoundary, type LeaveSummaryItem,
  type MonthlyPreScheduledLeave, type MonthlyScheduleVersion,
} from '../services/api';
import type { Employee, StaffingRequirement } from '../types';
import { SHIFT_CONFIG, type ShiftType } from '../types';

// 月度班表頁面
export default function MonthlySchedule() {
  const [year, setYear] = useState(2026);
  const [month, setMonth] = useState(4);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [slots, setSlots] = useState<MonthlySlot[]>([]);
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [empMap, setEmpMap] = useState<Record<number, string>>({});
  const [boundaries, setBoundaries] = useState<CycleBoundary[]>([]);
  const [leaveSummaries, setLeaveSummaries] = useState<LeaveSummaryItem[]>([]);
  const [hasSchedule, setHasSchedule] = useState(false);
  const [monthlyPreLeaves, setMonthlyPreLeaves] = useState<MonthlyPreScheduledLeave[]>([]);
  const [staffingRequirements, setStaffingRequirements] = useState<StaffingRequirement[]>([]);

  // 初始假期設定彈窗
  const [initModalOpen, setInitModalOpen] = useState(false);
  const [initLeaveValues, setInitLeaveValues] = useState<Record<string, number>>({});

  // 版本管理
  const [versionModalOpen, setVersionModalOpen] = useState(false);
  const [versionName, setVersionName] = useState('');
  const [versions, setVersions] = useState<MonthlyScheduleVersion[]>([]);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // 載入班表
  const loadSchedule = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getMonthlySchedule(year, month);
      setSlots(res.data.slots || []);
      setEmpMap(res.data.employees || {});
      setBoundaries(res.data.boundaries || []);
      setStaffingRequirements(res.data.requirements || []);
      setHasSchedule(true);
      setWarnings(res.data.warnings || []);

      // 載入假期摘要
      const leaveRes = await getMonthlyLeaveSummary(year, month);
      setLeaveSummaries(leaveRes.data.summaries || []);

      // 載入預假
      const preRes = await getMonthlyPreLeaves(year, month);
      setMonthlyPreLeaves(preRes.data || []);
    } catch {
      setHasSchedule(false);
      setSlots([]);
      setWarnings([]); // Clear warnings if no schedule
      // 載入分界資訊
      try {
        const bRes = await getCycleBoundaries(year, month);
        setBoundaries(bRes.data.boundaries || []);

        // 即使沒班表也要載入預假
        const preRes = await getMonthlyPreLeaves(year, month);
        setMonthlyPreLeaves(preRes.data || []);
      } catch { /* ignore */ }
    }
    setLoading(false);
  }, [year, month]);

  // 載入員工
  useEffect(() => {
    getEmployees().then((res) => {
      const emps = res.data;
      setEmployees(emps);
      const map: Record<number, string> = {};
      emps.forEach((e: Employee) => { map[e.ID] = e.name; });
      setEmpMap(map);
    });
  }, []);

  useEffect(() => { loadSchedule(); }, [loadSchedule]);

  // 產出班表
  const handleGenerateClick = () => {
    // 每次點擊都跳出確認窗，但先載入目前的餘額
    const defaults: Record<string, number> = {};
    for (const b of boundaries) {
      for (const emp of employees) {
        const key = `${b.cycle_index}_${emp.ID}`;
        const existing = leaveSummaries.find(s => s.employee_id === emp.ID && s.cycle_index === b.cycle_index);

        if (b.cycle_index === 1) {
          // C1 優先使用現有設定，未設定時才預設 3 天
          defaults[key] = (existing && existing.current_month_quota !== undefined) ? existing.current_month_quota : 3;
        } else {
          // 其他循環預設顯示「總假期」
          defaults[key] = (existing && existing.total_leave !== undefined) ? existing.total_leave : b.default_total_leave;
        }
      }
    }
    setInitLeaveValues(defaults);
    setInitModalOpen(true);
  };

  const doGenerate = async (
    cycleBalances: { cycle_index: number; employee_id: number; total_leave: number }[]
  ) => {
    setGenerating(true);
    try {
      const res = await generateMonthlySchedule(year, month, cycleBalances);
      setSlots(res.data.slots);
      setWarnings(res.data.warnings || []);
      setHasSchedule(true);
      setInitModalOpen(false);
      message.success(res.data.message);
      if (res.data.warnings && res.data.warnings.length > 0) {
        message.warning(`班表已產出，但有 ${res.data.warnings.length} 處人力不足警示`);
      }
      // Reload leave summaries after generation
      const leaveRes = await getMonthlyLeaveSummary(year, month);
      setLeaveSummaries(leaveRes.data.summaries || []);
    } catch (err: unknown) {
      const errMsg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '產出失敗';
      message.error(errMsg);
    }
    setGenerating(false);
  };

  // 提交初始假期
  const handleSubmitInitLeave = async () => {
    const balances: { cycle_index: number; employee_id: number; total_leave: number }[] = [];
    for (const key in initLeaveValues) {
      const [ci, eid] = key.split('_').map(Number);
      balances.push({ cycle_index: ci, employee_id: eid, total_leave: initLeaveValues[key] });
    }
    setInitModalOpen(false);
    await doGenerate(balances);
  };

  // 載入版本清單
  const loadVersions = useCallback(async () => {
    try {
      const res = await listMonthlyVersions(year, month);
      setVersions(res.data);
    } catch { /* ignore */ }
  }, [year, month]);

  useEffect(() => {
    if (hasSchedule) loadVersions();
  }, [hasSchedule, loadVersions]);

  // 儲存版本
  const handleSaveVersion = async () => {
    if (!versionName.trim()) {
      message.error('請輸入版本名稱');
      return;
    }
    try {
      await saveMonthlyVersion(year, month, { version_name: versionName, creator: '系統管理員' });
      message.success('版本儲存成功');
      setVersionModalOpen(false);
      setVersionName('');
      loadVersions();
    } catch {
      message.error('儲存版本失敗');
    }
  };

  // 恢復版本
  const handleRestoreVersion = async (vId: number) => {
    setLoading(true);
    try {
      const res = await restoreMonthlyVersion(vId);
      const { slots: newSlots, warnings: newWarnings, summaries: newSummaries, boundaries: newBoundaries } = res.data;

      if (newSlots) setSlots(newSlots);
      if (newWarnings) setWarnings(newWarnings);
      if (newSummaries) setLeaveSummaries(newSummaries);
      if (newBoundaries) setBoundaries(newBoundaries);

      message.success(res.data.message);
      setHistoryOpen(false);
    } catch {
      message.error('恢復失敗');
    }
    setLoading(false);
  };

  // 刪除版本
  const handleDeleteVersion = async (vId: number) => {
    try {
      await deleteMonthlyVersion(vId);
      message.success('版本已刪除');
      loadVersions();
    } catch {
      message.error('刪除失敗');
    }
  };

  // 手動修改格子
  const handleSlotChange = async (slotId: number, newShift: string) => {
    try {
      const res = await updateMonthlySlot(slotId, newShift);
      setSlots((prev) =>
        prev.map((s) => (s.ID === slotId ? { ...s, shift_type: newShift } : s))
      );

      // 動態更新警告與假期計算結果
      if (res.data.warnings) {
        setWarnings(res.data.warnings);
      }
      if (res.data.summaries) {
        setLeaveSummaries(res.data.summaries);
      }
      if (res.data.boundaries) {
        setBoundaries(res.data.boundaries);
      }

      message.success('已更新');
    } catch {
      message.error('更新失敗');
    }
  };

  // 處理預假設定
  const handlePreLeaveToggle = async (empId: number, day: number, existingId?: number) => {
    try {
      if (existingId) {
        await deleteMonthlyPreLeave(existingId);
        setMonthlyPreLeaves(prev => prev.filter(p => p.ID !== existingId));
        message.success('已取消預定休假');
      } else {
        const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
        const res = await createMonthlyPreLeave({ employee_id: empId, date: dateStr });
        setMonthlyPreLeaves(prev => [...prev, res.data]);
        message.success('已設定為預定休假（產出時生效）');
      }
    } catch {
      message.error('操作失敗');
    }
  };

  // 月份切換
  const changeMonth = (delta: number) => {
    let m = month + delta;
    let y = year;
    if (m > 12) { m = 1; y++; }
    if (m < 1) { m = 12; y--; }
    setYear(y);
    setMonth(m);
  };

  // 構建表格
  const daysInMonth = new Date(year, month, 0).getDate();
  const dates = Array.from({ length: daysInMonth }, (_, i) => i + 1);

  // 分界日（紅線位置）
  const boundaryDates = new Set<number>();
  for (const b of boundaries) {
    const endDate = new Date(b.end_date);
    if (endDate.getDate() < daysInMonth) {
      boundaryDates.add(endDate.getDate());
    }
  }

  // 用來取得某天某員工的 slot
  const getSlot = (empId: number, day: number): MonthlySlot | undefined => {
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
    return (slots || []).find((s) => s.employee_id === empId && s.date.startsWith(dateStr));
  };

  // 班別下拉選單
  const shiftMenuItems: MenuProps['items'] = [
    { key: 'day', label: '☀️ 白班' },
    { key: 'day88', label: '🌅 白8' },
    { key: 'evening', label: '🌙 小' },
    { key: 'night', label: '🌑 大' },
    { key: 'night88', label: '🌑 大8' },
    { key: 'off', label: '🟢 休假' },
  ];

  // 表格列
  const columns = [
    {
      title: '員工',
      dataIndex: 'name',
      key: 'name',
      fixed: 'left' as const,
      width: 80,
      render: (name: string) => <strong>{name}</strong>,
    },
    ...dates.map((day) => {
      const dateObj = new Date(year, month - 1, day);
      const weekday = dateObj.getDay();
      const weekdayNames = ['日', '一', '二', '三', '四', '五', '六'];
      const isWeekend = weekday === 0 || weekday === 6;
      const isBoundary = boundaryDates.has(day);

      return {
        title: (
          <div style={{ textAlign: 'center' as const, lineHeight: 1.2 }}>
            <div style={{
              fontSize: 14,
              fontWeight: 600,
              color: 'inherit',
              textDecoration: 'none'
            }}>
              {day}
            </div>
            <div style={{ fontSize: 12, color: isWeekend ? '#ff4d4f' : '#858585' }}>
              {weekdayNames[weekday]}
            </div>
          </div>
        ),

        dataIndex: `day_${day}`,
        key: `day_${day}`,
        width: 50,
        onHeaderCell: () => ({
          style: {
            borderRight: isBoundary ? '3px solid #ff4d4f' : undefined,
            background: isWeekend ? '#fff1f0' : undefined,
          },
        }),
        onCell: () => ({
          style: {
            borderRight: isBoundary ? '3px solid #ff4d4f' : undefined,
            padding: 2,
            background: isWeekend ? '#fff1f0' : undefined,
          },
        }),
        render: (_: unknown, record: { empId: number }) => {
          const slot = getSlot(record.empId, day);
          const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
          const preLeave = monthlyPreLeaves.find(p => p.employee_id === record.empId && p.date.startsWith(dateStr));

          // 構建選單
          const menuItems: MenuProps['items'] = [...shiftMenuItems];
          menuItems.push({ type: 'divider' });
          if (preLeave) {
            menuItems.push({ key: 'unset_pre', label: '❌ 取消預留休假', danger: true });
          } else {
            menuItems.push({ key: 'set_pre', label: '⭐ 設定預留休假', theme: 'light' } as any);
          }

          if (!slot) {
            // 如果還沒產出班表，只顯示預留休假的 UI
            return (
              <Dropdown
                menu={{
                  items: menuItems,
                  onClick: ({ key }) => {
                    if (key === 'set_pre') handlePreLeaveToggle(record.empId, day);
                    if (key === 'unset_pre') handlePreLeaveToggle(record.empId, day, preLeave?.ID);
                  }
                }}
                trigger={['click']}
              >
                <div style={{
                  cursor: 'pointer', height: 22, display: 'flex', alignItems: 'center', justifyContent: 'center',
                  background: preLeave ? '#f6ffed' : '#f0f0f0',
                  border: preLeave ? '1px dashed #52c41a' : 'none',
                  borderRadius: 4, fontSize: 10
                }}>
                  {preLeave ? '⭐ 預假' : ''}
                </div>
              </Dropdown>
            );
          }

          const shiftType = slot.shift_type as ShiftType;
          const isOff = shiftType === 'off';
          const config = SHIFT_CONFIG[shiftType] || { label: shiftType, color: '#999', bgColor: '#f5f5f5' };

          return (
            <Dropdown
              menu={{
                items: menuItems,
                onClick: ({ key }) => {
                  if (key === 'set_pre') handlePreLeaveToggle(record.empId, day);
                  else if (key === 'unset_pre') handlePreLeaveToggle(record.empId, day, preLeave?.ID);
                  else handleSlotChange(slot.ID, key);
                },
              }}
              trigger={['click']}
            >
              <Tag
                style={{
                  cursor: 'pointer',
                  fontSize: 12,
                  padding: '2px 6px',
                  margin: 0,
                  width: '100%',
                  height: 22,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: isOff ? 'transparent' : config.bgColor,
                  color: config.color,
                  border: isOff ? (preLeave ? '2px dashed #52c41a' : 'none') : `1px solid ${config.color}`,
                  borderColor: preLeave ? '#52c41a' : (isOff ? 'transparent' : config.color),
                  borderWidth: preLeave ? 2 : (isOff ? 0 : 1),
                  boxShadow: preLeave ? '0 0 4px rgba(82, 196, 26, 0.3)' : 'none',
                }}
              >
                {preLeave && !isOff ? '⭐ ' : ''}
                {isOff ? (preLeave ? '⭐' : '') : config.label}
              </Tag>
            </Dropdown>
          );
        },
      };
    }),
    {
      title: '總休',
      key: 'total_off',
      fixed: 'right' as const,
      width: 50,
      align: 'center' as const,
      render: (_: unknown, record: { empId: number }) => {
        const mySlots = (slots || []).filter(s => s.employee_id === record.empId && s.shift_type === 'off');
        return <strong style={{ color: '#52c41a' }}>{mySlots.length}</strong>;
      }
    }
  ];

  // 表格資料
  const dataSource = employees
    .filter((e) => e.status === 1)
    .map((emp) => ({
      key: emp.ID,
      name: empMap[emp.ID] || emp.name,
      empId: emp.ID,
    }));

  // 假期統計
  const cycleLeaveStats = (boundaries || []).map((b) => {
    const cycleLeave = (leaveSummaries || []).filter((ls) => ls && ls.cycle_index === b.cycle_index);
    const totalUsed = cycleLeave.reduce((sum, ls) => sum + (ls.used_leave || 0), 0);
    const totalRemaining = cycleLeave.reduce((sum, ls) => sum + (ls.remaining || 0), 0);
    return { ...b, totalUsed, totalRemaining, details: cycleLeave };
  });

  return (
    <div>
      {/* 月份選擇器 */}
      <Space style={{ marginBottom: 16 }} size="middle">
        <Button icon={<LeftOutlined />} onClick={() => changeMonth(-1)} />
        <Select value={year} onChange={setYear} style={{ width: 100 }}
          options={[2025, 2026, 2027].map((y) => ({ value: y, label: `${y} 年` }))}
        />
        <Select value={month} onChange={setMonth} style={{ width: 80 }}
          options={Array.from({ length: 12 }, (_, i) => ({ value: i + 1, label: `${i + 1} 月` }))}
        />
        <Button icon={<RightOutlined />} onClick={() => changeMonth(1)} />
        <Dropdown.Button
          type="primary"
          icon={<ThunderboltOutlined />}
          loading={generating}
          onClick={() => doGenerate([])}
          menu={{
            items: [
              {
                key: 'set_quota',
                label: '設定配額並產出',
                icon: <EditOutlined />,
                onClick: () => handleGenerateClick()
              }
            ]
          }}
        >
          {hasSchedule ? '快速重新產出' : '立即自動排班'}
        </Dropdown.Button>
        {hasSchedule && (
          <Space>
            <Button
              icon={<SaveOutlined />}
              onClick={() => setVersionModalOpen(true)}
            >
              儲存版本
            </Button>
            <Button
              icon={<HistoryOutlined />}
              onClick={() => setHistoryOpen(true)}
            >
              版本紀錄 ({versions.length})
            </Button>
            <Badge count={(warnings || []).length} offset={[-2, 0]}>
              <Button
                icon={<BellOutlined />}
                onClick={() => setDrawerOpen(true)}
              >
                班表資訊
              </Button>
            </Badge>
          </Space>
        )}
      </Space>

      {/* 循環分界資訊 */}
      {boundaries.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          <Space wrap>
            {boundaries.map((b, i) => (
              <Tag key={i} color={i === 0 ? 'blue' : 'green'}>
                C{b.cycle_index}：{b.start_date} ~ {b.end_date}（{b.days_in_month} 天）
              </Tag>
            ))}
          </Space>
          {boundaries.length > 1 && (
            <span style={{ marginLeft: 8, color: '#ff4d4f', fontSize: 12 }}>
              ┃ 紅色粗線 = 循環分界
            </span>
          )}
        </div>
      )}

      {/* 資訊中心 Drawer */}
      <Drawer
        title={<Space><BellOutlined /><span>班表資訊中心</span></Space>}
        placement="right"
        onClose={() => setDrawerOpen(false)}
        open={drawerOpen}
        width={450}
      >
        <Tabs
          defaultActiveKey="warnings"
          items={[
            {
              key: 'warnings',
              label: (
                <Badge count={(warnings || []).length} offset={[12, -2]} size="small">
                  <span>人力警示</span>
                </Badge>
              ),
              children: (
                <List
                  dataSource={warnings || []}
                  locale={{ emptyText: '暫無人力警示' }}
                  renderItem={(item, index) => (
                    <List.Item key={index}>
                      <div style={{ color: '#856404', fontSize: 13 }}>
                        <InfoCircleOutlined style={{ marginRight: 8, color: '#faad14' }} />
                        {item}
                      </div>
                    </List.Item>
                  )}
                />
              )
            },
            {
              key: 'restrictions',
              label: '員工班別限制',
              children: (
                <List
                  dataSource={employees.filter(e => e.status === 1 && e.restrictions && e.restrictions.length > 0)}
                  locale={{ emptyText: '所有員工皆無特殊限制' }}
                  renderItem={emp => (
                    <List.Item key={emp.ID}>
                      <div style={{ width: '100%' }}>
                        <div style={{ fontWeight: 'bold', marginBottom: 8 }}>{emp.name}</div>
                        <Space wrap>
                          {emp.restrictions?.map(r => (
                            <Tag key={r.ID} color={r.max_days === null ? 'red' : 'blue'}>
                              {SHIFT_CONFIG[r.shift_type as ShiftType]?.label || r.shift_type}: 
                              {r.max_days === null ? ' 禁止' : ` ≤${r.max_days}天`}
                            </Tag>
                          ))}
                        </Space>
                      </div>
                    </List.Item>
                  )}
                />
              )
            }
          ]}
        />
      </Drawer>

      {/* 班表 */}
      <Spin spinning={loading}>
        {hasSchedule ? (
          <Table
            columns={columns}
            dataSource={dataSource}
            pagination={false}
            size="small"
            bordered
            scroll={{ x: 80 + daysInMonth * 50 + 50 }}
            style={{ marginBottom: 16 }}
            summary={() => (
              <Table.Summary fixed>
                {['☀️', '🌙', '🌑'].map((emoji, idx) => {
                  const shiftType = ['day', 'evening', 'night'][idx];
                  return (
                    <Table.Summary.Row key={shiftType} style={{ background: '#fafafa' }}>
                      <Table.Summary.Cell index={0} align="right">
                        <div style={{ fontSize: 12, fontWeight: 'bold' }}>
                          {emoji} {shiftType === 'day' ? '白' : shiftType === 'evening' ? '小' : '大'}
                        </div>
                        <div style={{ fontSize: 10, color: '#8c8c8c' }}>(應/實)</div>
                      </Table.Summary.Cell>
                      {dates.map((day, dIdx) => {
                        const dateObj = new Date(year, month - 1, day);
                        const weekday = dateObj.getDay();
                        const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
                        const daySlots = (slots || []).filter(s => s.date.startsWith(dateStr));

                        let actual = 0;
                        let hasDay88 = false;
                        daySlots.forEach(s => {
                          if (shiftType === 'day' && (s.shift_type === 'day' || s.shift_type === 'day88')) actual++;
                          else if (shiftType === 'night' && (s.shift_type === 'night' || s.shift_type === 'night88')) actual++;
                          else if (s.shift_type === shiftType) actual++;
                          if (s.shift_type === 'day88') hasDay88 = true;
                        });

                        const req = staffingRequirements.find(r => r.weekday === weekday && r.shift_type === shiftType);
                        const needed = req ? (hasDay88 ? req.min_count_with_day88 : req.min_count) : 0;
                        const isLess = actual < needed;
                        const isBoundary = boundaryDates.has(day);

                        return (
                          <Table.Summary.Cell
                            key={dIdx}
                            index={dIdx + 1}
                          >
                            <div style={{
                              textAlign: 'center',
                              color: isLess ? '#ff4d4f' : '#8c8c8c',
                              background: isLess ? '#fff1f0' : 'transparent',
                              borderRight: isBoundary ? '3px solid #ff4d4f' : undefined,
                              fontSize: 11,
                              padding: '4px 0',
                              margin: '-16px -8px',
                              height: '100%',
                              display: 'flex',
                              flexDirection: 'column',
                              justifyContent: 'center'
                            }}>
                              <div style={{ fontWeight: isLess ? 'bold' : 'normal', opacity: 0.8 }}>
                                {needed}
                              </div>
                              <div style={{
                                fontSize: 10,
                                fontWeight: isLess ? 'bold' : 'normal',
                                borderTop: '1px solid #eee',
                                marginTop: 2,
                                paddingTop: 2
                              }}>
                                {actual}
                              </div>
                            </div>
                          </Table.Summary.Cell>
                        );
                      })}
                      <Table.Summary.Cell index={dates.length + 1}>
                        {/* 總休列對應的空白格 */}
                      </Table.Summary.Cell>
                    </Table.Summary.Row>
                  );
                })}
              </Table.Summary>
            )}
          />
        ) : (
          <Card style={{ textAlign: 'center', padding: 40 }}>
            <EditOutlined style={{ fontSize: 48, color: '#bfbfbf' }} />
            <p style={{ color: '#999', marginTop: 16 }}>尚未建立 {year}/{month} 月度班表</p>
            <Button type="primary" onClick={() => handleGenerateClick()}>
              立即產出
            </Button>
          </Card>
        )}
      </Spin>

      {/* 假期餘額面板 */}
      {hasSchedule && cycleLeaveStats.length > 0 && (
        <>
          <Divider>各循環假期使用詳情 (逐人)</Divider>
          <Tabs
            type="card"
            items={cycleLeaveStats.map((stat, i) => ({
              key: String(stat.cycle_index),
              label: (
                <Space>
                  <Tag color={i === 0 ? 'blue' : 'green'}>C{stat.cycle_index}</Tag>
                  <span>{stat.start_date} ~ {stat.end_date}</span>
                </Space>
              ),
              children: (
                <div style={{ padding: '8px 0' }}>
                  <Table
                    dataSource={stat.details}
                    pagination={false}
                    size="small"
                    rowKey="employee_id"
                    bordered
                    columns={[
                      {
                        title: '員工姓名',
                        dataIndex: 'employee_name',
                        key: 'name',
                        width: 120,
                        fixed: 'left',
                        render: (text) => <strong>{text}</strong>
                      },
                      {
                        title: '循環原始總假',
                        dataIndex: 'total_leave',
                        key: 'total',
                        align: 'center',
                        width: 110,
                        render: (val) => `${val} 天`
                      },
                      {
                        title: '循環累計已用',
                        dataIndex: 'used_leave',
                        key: 'used',
                        align: 'center',
                        width: 110,
                        render: (val) => `${val} 天`
                      },
                      {
                        title: '本月應休 (目標)',
                        key: 'monthly_quota',
                        align: 'center',
                        width: 130,
                        render: (_, record: any) => {
                          const isEnding = record.is_ending;
                          const label = isEnding ? (record.month_quota_manually_set ? '(手動輸入額度)' : '(循環結算剩餘)') : '(系統比例分配)';
                          return (
                            <div>
                              <strong>{record.current_month_quota} 天</strong>
                              <div style={{ fontSize: 10, color: '#8c8c8c' }}>{label}</div>
                            </div>
                          );
                        }
                      },
                      {
                        title: '本月已排 (休)',
                        key: 'month_used',
                        align: 'center',
                        width: 110,
                        render: (_, record) => {
                          const mySlots = (slots || []).filter(s =>
                            s.employee_id === record.employee_id &&
                            s.cycle_index === stat.cycle_index &&
                            s.shift_type === 'off'
                          );
                          return <Tag color="orange">{mySlots.length} 天</Tag>;
                        }
                      },
                      {
                        title: '最終剩餘',
                        key: 'final_remaining',
                        align: 'center',
                        width: 100,
                        render: (_: any, record: any) => {
                          const mySlots = (slots || []).filter(s =>
                            s.employee_id === record.employee_id &&
                            s.cycle_index === stat.cycle_index &&
                            s.shift_type === 'off'
                          );
                          const monthUsed = mySlots.length;
                          // C1 特別邏輯：本月應休 - 本月已排 (因為 C1 是系統啟始，前半部假已用掉，使用者手動 key 剩餘配額)
                          // 其他循環：總假 - 累計已用(包含本月前) - 本月已排
                          let res = 0;
                          if (stat.cycle_index === 1) {
                            res = record.current_month_quota - monthUsed;
                          } else {
                            res = record.total_leave - record.used_leave - monthUsed;
                          }

                          const color = res < 0 ? 'red' : (res === 0 ? 'blue' : 'green');
                          return <Tag color={color} style={{ fontWeight: 'bold' }}>{res} 天</Tag>;
                        }
                      },
                    ]}
                  />
                </div>
              )
            }))}
          />
        </>
      )}

      {/* 初始假期設定 Modal */}
      <Modal
        title="設定/確認循環假期總額（逐人）"
        open={initModalOpen}
        onOk={handleSubmitInitLeave}
        confirmLoading={generating}
        onCancel={() => setInitModalOpen(false)}
        width={600}
        okText="確認並產出班表"
      >
        <p style={{ color: '#666', marginBottom: 16 }}>
          請確認每位員工在該循環「可排休假總天數」。<br />
          <span style={{ color: '#ff4d4f', fontWeight: 'bold' }}>※ 此循環在本月結束，請「直接輸入在本月剩餘應休天數」即可。</span><br />
          <span style={{ color: '#52c41a', fontWeight: 'bold' }}>※ 其餘循環由系統依比例分配或下月結算，此處暫不供調整。</span>
        </p>
        <Form layout="vertical">
          {boundaries.map((b) => (
            <div key={b.cycle_index}>
              <Divider orientation="left" plain>
                C{b.cycle_index}（{b.start_date} ~ {b.end_date}）
              </Divider>
              <Row gutter={[8, 8]}>
                {employees
                  .filter((e) => e.status === 1)
                  .map((emp) => {
                    const key = `${b.cycle_index}_${emp.ID}`;
                    return (
                      <Col key={key} span={8}>
                        <Form.Item label={emp.name} style={{ marginBottom: 8 }}>
                          {b.is_ending ? (
                            <InputNumber
                              min={0}
                              max={28}
                              value={initLeaveValues[key] ?? 3}
                              onChange={(v) =>
                                setInitLeaveValues((prev) => ({ ...prev, [key]: v ?? 0 }))
                              }
                              style={{ width: '100%' }}
                              placeholder="剩餘假"
                              addonAfter="天"
                            />
                          ) : (
                            <div style={{
                              padding: '4px 11px', background: '#f5f5f5', border: '1px solid #d9d9d9',
                              borderRadius: 4, color: '#aaabbb'
                            }}>
                              {initLeaveValues[key] ?? 0} 天 (系統算好不給改)
                            </div>
                          )}
                        </Form.Item>
                      </Col>
                    );
                  })}
              </Row>
            </div>
          ))}
        </Form>
      </Modal>

      {/* 儲存版本 Modal */}
      <Modal
        title="儲存班表版本"
        open={versionModalOpen}
        onOk={handleSaveVersion}
        onCancel={() => setVersionModalOpen(false)}
        okText="儲存"
        cancelText="取消"
      >
        <p>請輸入一個好辨識的名稱（例如：初稿、修訂 V1、最終確認版）</p>
        <Input
          placeholder="版本名稱"
          value={versionName}
          onChange={(e) => setVersionName(e.target.value)}
          onPressEnter={handleSaveVersion}
        />
      </Modal>

      {/* 版本紀錄 Drawer */}
      <Drawer
        title="版本歷史紀錄"
        placement="right"
        onClose={() => setHistoryOpen(false)}
        open={historyOpen}
        width={400}
      >
        <List
          dataSource={versions}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button type="link" onClick={() => handleRestoreVersion(item.ID)}>載入</Button>,
                <Popconfirm
                  title="確定要刪除此版本嗎？"
                  onConfirm={() => handleDeleteVersion(item.ID)}
                  okText="確定"
                  cancelText="取消"
                >
                  <Button type="link" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              ]}
            >
              <List.Item.Meta
                title={item.version_name}
                description={
                  <div style={{ fontSize: 12 }}>
                    <div>建立時間: {new Date(item.CreatedAt).toLocaleString()}</div>
                    <div>建立者: {item.creator}</div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Drawer>
    </div>
  );
}
