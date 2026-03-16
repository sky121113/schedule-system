import { useEffect, useState } from 'react';
import { Button, Card, Select, Tooltip, Modal, message, Row, Col, Spin, Empty, Space, Popconfirm } from 'antd';
import { ThunderboltOutlined, PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { useAppStore } from '../store/appStore';
import { SHIFT_CONFIG, WEEKDAY_NAMES } from '../types';
import type { Employee, TemplateSlot, ShiftType } from '../types';
import * as api from '../services/api';
import dayjs from 'dayjs';
import { PreScheduledLeave, LeaveQuotaStats } from '../services/api';

const TemplateEditor = () => {
  const { employees, fetchEmployees, templates, fetchTemplates, currentTemplate, currentSlots, fetchTemplate } =
    useAppStore();
  const [loading, setLoading] = useState(false);
  const [localStats, setLocalStats] = useState<any>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);

  // 預假管理狀態
  const [leaveModalOpen, setLeaveModalOpen] = useState(false);
  const [leaveQuota, setLeaveQuota] = useState<LeaveQuotaStats | null>(null);
  const [preLeaves, setPreLeaves] = useState<PreScheduledLeave[]>([]);
  const [selectedEmp, setSelectedEmp] = useState<number | null>(null);
  const [selectedDay, setSelectedDay] = useState<number | null>(null);
  const [leaveReason, setLeaveReason] = useState('');

  useEffect(() => {
    fetchEmployees();
    fetchTemplates();
  }, []);

  const handleSelectTemplate = async (id: number) => {
    setLoading(true);
    await fetchTemplate(id);
    setLoading(false);
    setLocalStats(null);
  };

  const handleAutoSchedule = async () => {
    if (!currentTemplate) return;
    setLoading(true);
    try {
      const res = await api.autoSchedule(currentTemplate.ID);
      message.success(res.data.message);
      setLocalStats(res.data.stats.employees);
      await fetchTemplate(currentTemplate.ID);
    } catch (err: any) {
      message.error(err.response?.data?.error || '自動排班失敗');
    }
    setLoading(false);
  };

  const handleClear = async () => {
    if (!currentTemplate) return;
    await api.clearTemplateSlots(currentTemplate.ID);
    message.success('已清除排班');
    fetchTemplate(currentTemplate.ID);
    setLocalStats(null);
  };

  // 刪除模板
  const handleDeleteTemplate = async () => {
    if (!currentTemplate) return;
    try {
      await api.deleteTemplate(currentTemplate.ID);
      message.success('模板已刪除');
      await fetchTemplates();
      useAppStore.setState({ currentTemplate: null, currentSlots: [] });
      setLocalStats(null);
    } catch (err: any) {
      message.error(err.response?.data?.error || '刪除失敗');
    }
  };

  // 預假管理功能
  const fetchLeaveData = async () => {
    if (!currentTemplate) return;
    try {
      const [quotaRes, leaveRes] = await Promise.all([
        api.getLeaveQuota(currentTemplate.ID),
        api.getPreLeaves(currentTemplate.ID)
      ]);
      setLeaveQuota(quotaRes.data);
      setPreLeaves(leaveRes.data);
    } catch (err) {
      console.error(err);
    }
  };

  const handleOpenLeaveModal = () => {
    setLeaveModalOpen(true);
    fetchLeaveData();
  };

  const handleAddPreLeave = async () => {
    if (!currentTemplate || !selectedEmp || selectedDay === null) {
      message.warning('請選擇員工與日期');
      return;
    }
    try {
      await api.setPreLeave(currentTemplate.ID, {
        employee_id: selectedEmp,
        day_offset: selectedDay,
        reason: leaveReason
      });
      message.success('預假設定成功');
      setSelectedEmp(null);
      setSelectedDay(null);
      setLeaveReason('');
      fetchLeaveData();
      fetchTemplate(currentTemplate.ID); // 更新背景行事曆
    } catch (err: any) {
      message.error(err.response?.data?.error || '設定失敗');
    }
  };

  const handleDeletePreLeave = async (leaveId: number) => {
    if (!currentTemplate) return;
    try {
      await api.deletePreLeave(currentTemplate.ID, leaveId);
      message.success('已刪除預假');
      fetchLeaveData();
      fetchTemplate(currentTemplate.ID); // 更新背景行事曆
    } catch (err: any) {
      message.error('刪除失敗');
    }
  };

  const activeEmployees = employees.filter((e) => e.status === 1);
  const totalDays = currentTemplate ? currentTemplate.cycle_weeks * 7 : 0;

  const slotMap = new Map<string, TemplateSlot>();
  currentSlots.forEach(slot => {
    slotMap.set(`${slot.day_offset}-${slot.employee_id}`, slot);
  });

  const renderCell = (dayOffset: number, emp: Employee) => {
    const slot = slotMap.get(`${dayOffset}-${emp.ID}`);
    const shiftType = slot?.shift_type as ShiftType | undefined;

    if (shiftType) {
      const config = SHIFT_CONFIG[shiftType];
      return (
        <Tooltip title={`${emp.name} - ${config.label}`}>
          <div
            onClick={() => handleCellClick(dayOffset, emp, slot)}
            style={{
              background: config.bgColor,
              color: config.color,
              fontWeight: 'bold',
              textAlign: 'center',
              cursor: 'pointer',
              borderRadius: 4,
              padding: '2px 0',
              fontSize: 12,
              border: `1px solid ${config.color}`,
            }}
          >
            {config.label}
          </div>
        </Tooltip>
      );
    }

    return (
      <div
        onClick={() => handleCellClick(dayOffset, emp)}
        style={{ textAlign: 'center', cursor: 'pointer', color: '#d9d9d9', padding: '2px 0' }}
      >
        {preLeaves.some(l => l.employee_id === emp.ID && l.day_offset === dayOffset) ? (
          <Tooltip title="已設定預假">
            <span style={{ color: '#faad14', fontWeight: 'bold' }}>休</span>
          </Tooltip>
        ) : (
          '—'
        )}
      </div>
    );
  };

  const handleCellClick = (dayOffset: number, emp: Employee, slot?: TemplateSlot) => {
    if (slot) {
      Modal.confirm({
        title: '移除班次',
        content: `確定要移除 ${emp.name} 在 Day ${dayOffset + 1} 的班次嗎？`,
        onOk: async () => {
          await api.removeSlot(slot.ID);
          fetchTemplate(currentTemplate!.ID);
        }
      });
      return;
    }
    message.info('請使用自動排班或點選空白處加入 (實作中)');
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>📅 循環模板編輯器</h2>
        <Space>
          <Select
            placeholder="選擇模板"
            style={{ width: 220 }}
            value={currentTemplate?.ID}
            onChange={handleSelectTemplate}
            options={templates.map(t => ({ value: t.ID, label: `${dayjs(t.start_date).format('YYYY/MM/DD')} (v${t.version})` }))}
          />
          <Button icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>新增模板</Button>
          {currentTemplate && (
            <Popconfirm
              title="確定刪除此模板？"
              description="將同時刪除此模板下的所有排班資料，此操作無法復原。"
              onConfirm={handleDeleteTemplate}
              okText="確定刪除"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Button danger icon={<DeleteOutlined />}>刪除模板</Button>
            </Popconfirm>
          )}
          <Button onClick={handleOpenLeaveModal} disabled={!currentTemplate}>
            🏖️ 預假設定
          </Button>
          <Button type="primary" icon={<ThunderboltOutlined />} onClick={handleAutoSchedule} loading={loading} disabled={!currentTemplate}>
            自動排班
          </Button>
          <Button danger icon={<DeleteOutlined />} onClick={handleClear} disabled={!currentTemplate}>清除排班</Button>
        </Space>
      </div>

      {!currentTemplate ? <Empty description="請選擇或新增模板" /> : (
        <Spin spinning={loading}>
          {localStats && (
            <Card size="small" style={{ marginBottom: 16, background: '#f0f5ff' }}>
              <Row gutter={[8, 8]}>
                {Object.values(localStats).map((s: any) => (
                  <Col key={s.name} span={4}>
                    <div style={{ fontSize: 12 }}>
                      <strong>{s.name}</strong>: 工{s.total_work} 休{s.off_days}
                    </div>
                  </Col>
                ))}
              </Row>
            </Card>
          )}

          <div style={{ overflowX: 'auto', border: '1px solid #f0f0f0', borderRadius: 8 }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
              <thead style={{ background: '#fafafa' }}>
                <tr>
                  <th style={{ padding: 8, borderBottom: '1px solid #f0f0f0', position: 'sticky', left: 0, background: '#fafafa', zIndex: 2 }}>員工</th>
                  {Array.from({ length: totalDays }).map((_, d) => (
                    <th key={d} style={{ padding: 4, borderBottom: '1px solid #f0f0f0', minWidth: 35, textAlign: 'center' }}>
                      {WEEKDAY_NAMES[dayjs(currentTemplate.start_date).add(d, 'day').day()]}
                      <div style={{ fontSize: 9, color: '#999' }}>{d + 1}</div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {activeEmployees.map(emp => (
                  <tr key={emp.ID}>
                    <td style={{ padding: 8, borderBottom: '1px solid #f0f0f0', position: 'sticky', left: 0, background: '#fff', fontWeight: 'bold' }}>{emp.name}</td>
                    {Array.from({ length: totalDays }).map((_, d) => (
                      <td key={d} style={{ padding: '2px 1px', borderBottom: '1px solid #f0f0f0', borderLeft: d % 7 === 0 ? '1px solid #d9d9d9' : '1px solid #f0f0f0' }}>
                        {renderCell(d, emp)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Spin>
      )}

      <Modal title="新增模板" open={createModalOpen} onOk={async () => { await api.createTemplate({ start_date: '2026-03-15' }); fetchTemplates(); setCreateModalOpen(false); }} onCancel={() => setCreateModalOpen(false)}>
        建立 2026/03/15 起的 4 週循環模板。
      </Modal>

      <Modal
        title="🏖️ 預假設定 (最高優先排假)"
        open={leaveModalOpen}
        onCancel={() => setLeaveModalOpen(false)}
        footer={null}
        width={700}
      >
        {leaveQuota && (
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={8}>
              <Card size="small">
                <div>總假 / 每人配額</div>
                <h3 style={{ margin: 0 }}>{leaveQuota.total_leave} / {leaveQuota.per_person_leave} 天</h3>
              </Card>
            </Col>
            <Col span={16}>
              <Card size="small" style={{ background: '#f6ffed', borderColor: '#b7eb8f' }}>
                <div>每人每循環最多可設定 <strong>3 天</strong>預假。自動排班時會<strong>最高優先</strong>安排。</div>
              </Card>
            </Col>
          </Row>
        )}

        <div style={{ padding: 16, background: '#fafafa', borderRadius: 8, marginBottom: 16 }}>
          <h4>新增預假</h4>
          <Space wrap>
            <Select
              placeholder="選擇員工"
              style={{ width: 120 }}
              value={selectedEmp}
              onChange={setSelectedEmp}
              options={activeEmployees.map(e => ({ value: e.ID, label: e.name }))}
            />
            <Select
              placeholder="選擇日期"
              style={{ width: 160 }}
              value={selectedDay}
              onChange={setSelectedDay}
            >
              {currentTemplate && Array.from({ length: totalDays }).map((_, d) => {
                const date = dayjs(currentTemplate.start_date).add(d, 'day');
                return (
                  <Select.Option key={d} value={d}>
                    {date.format('MM/DD')} ({WEEKDAY_NAMES[date.day()]})
                  </Select.Option>
                );
              })}
            </Select>
            <input
              style={{ padding: '4px 11px', border: '1px solid #d9d9d9', borderRadius: 6, height: 32 }}
              placeholder="原因 (選填)"
              value={leaveReason}
              onChange={e => setLeaveReason(e.target.value)}
            />
            <Button type="primary" onClick={handleAddPreLeave}>設定</Button>
          </Space>
        </div>

        <h4>已設定的預假</h4>
        {preLeaves.length === 0 ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暫無預假記錄" /> : (
          <div style={{ maxHeight: 300, overflowY: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <th style={{ padding: 8 }}>員工</th>
                  <th style={{ padding: 8 }}>日期</th>
                  <th style={{ padding: 8 }}>原因</th>
                  <th style={{ padding: 8 }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {activeEmployees.map(emp => {
                  const empLeaves = preLeaves.filter(l => l.employee_id === emp.ID);
                  if (empLeaves.length === 0) return null;
                  return empLeaves.map(l => {
                    const date = dayjs(currentTemplate?.start_date).add(l.day_offset, 'day');
                    return (
                      <tr key={l.ID} style={{ borderBottom: '1px solid #f0f0f0' }}>
                        <td style={{ padding: 8, fontWeight: 'bold' }}>{emp.name}</td>
                        <td style={{ padding: 8 }}>{date.format('MM/DD')} ({WEEKDAY_NAMES[date.day()]})</td>
                        <td style={{ padding: 8, color: '#888' }}>{l.reason || '-'}</td>
                        <td style={{ padding: 8 }}>
                          <Button 
                            danger 
                            type="text" 
                            icon={<DeleteOutlined />} 
                            size="small"
                            onClick={() => handleDeletePreLeave(l.ID)}
                          />
                        </td>
                      </tr>
                    );
                  });
                })}
              </tbody>
            </table>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default TemplateEditor;
