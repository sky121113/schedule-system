import { useEffect, useState } from 'react';
import { Table, Tag, Button, Modal, Form, Input, Select, InputNumber, Space, message, Popconfirm } from 'antd';
import { PlusOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import { useAppStore } from '../store/appStore';
import { SHIFT_CONFIG } from '../types';
import type { Employee, ShiftRestriction, ShiftType } from '../types';
import * as api from '../services/api';

// 員工管理頁面
const EmployeeManagement = () => {
  const { employees, fetchEmployees } = useAppStore();
  const [modalOpen, setModalOpen] = useState(false);
  const [restrictionModalOpen, setRestrictionModalOpen] = useState(false);
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);
  const [restrictions, setRestrictions] = useState<ShiftRestriction[]>([]);
  const [form] = Form.useForm();
  const [restrictionForm] = Form.useForm();

  useEffect(() => {
    fetchEmployees();
  }, []);

  // 狀態文字
  const statusMap: Record<number, { text: string; color: string }> = {
    1: { text: '在職', color: 'green' },
    0: { text: '停用', color: 'default' },
    2: { text: '長期請假', color: 'orange' },
  };

  // 員工表格欄位
  const columns = [
    { title: '姓名', dataIndex: 'name', key: 'name', width: 100 },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    {
      title: '類型',
      key: 'type',
      width: 120,
      render: (_: unknown, record: Employee) =>
        record.is_day88_primary ? <Tag color="orange">8-8 主力</Tag> : <Tag>一般</Tag>,
    },
    {
      title: '狀態',
      key: 'status',
      width: 100,
      render: (_: unknown, record: Employee) => {
        const s = statusMap[record.status] || { text: '未知', color: 'default' };
        return <Tag color={s.color}>{s.text}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: unknown, record: Employee) => (
        <Space>
          <Button
            size="small"
            icon={<SettingOutlined />}
            onClick={() => openRestrictionModal(record)}
          >
            限制
          </Button>
          <Popconfirm title="確定刪除？" onConfirm={() => handleDelete(record.ID)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 新增/編輯員工
  const handleSubmit = async () => {
    const values = await form.validateFields();
    try {
      if (selectedEmployee) {
        await api.updateEmployee(selectedEmployee.ID, values);
        message.success('更新成功');
      } else {
        await api.createEmployee(values);
        message.success('新增成功');
      }
      setModalOpen(false);
      form.resetFields();
      setSelectedEmployee(null);
      fetchEmployees();
    } catch {
      message.error('操作失敗');
    }
  };

  // 刪除員工
  const handleDelete = async (id: number) => {
    await api.deleteEmployee(id);
    message.success('已刪除');
    fetchEmployees();
  };

  // 開啟限制設定
  const openRestrictionModal = async (emp: Employee) => {
    setSelectedEmployee(emp);
    try {
      const res = await api.getEmployeeRestrictions(emp.ID);
      setRestrictions(res.data);
    } catch {
      setRestrictions([]);
    }
    setRestrictionModalOpen(true);
  };

  // 新增限制
  const handleAddRestriction = async () => {
    const values = await restrictionForm.validateFields();
    try {
      await api.createRestriction({
        employee_id: selectedEmployee!.ID,
        shift_type: values.shift_type,
        max_days: values.restriction_type === 'ban' ? null : values.max_days,
        note: values.note || '',
      });
      message.success('限制已新增');
      restrictionForm.resetFields();
      // 重新載入
      const res = await api.getEmployeeRestrictions(selectedEmployee!.ID);
      setRestrictions(res.data);
    } catch {
      message.error('新增失敗');
    }
  };

  // 刪除限制
  const handleDeleteRestriction = async (id: number) => {
    await api.deleteRestriction(id);
    message.success('已刪除');
    const res = await api.getEmployeeRestrictions(selectedEmployee!.ID);
    setRestrictions(res.data);
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>👥 員工管理</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setSelectedEmployee(null);
            form.resetFields();
            setModalOpen(true);
          }}
        >
          新增員工
        </Button>
      </div>

      <Table
        dataSource={employees}
        columns={columns}
        rowKey="ID"
        pagination={false}
        size="middle"
        onRow={(record) => ({
          onDoubleClick: () => {
            setSelectedEmployee(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          },
        })}
      />

      {/* 新增/編輯員工 Modal */}
      <Modal
        title={selectedEmployee ? '編輯員工' : '新增員工'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="姓名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="is_day88_primary" label="8-8 主力" valuePropName="checked">
            <Select options={[{ value: true, label: '是' }, { value: false, label: '否' }]} />
          </Form.Item>
          <Form.Item name="status" label="狀態">
            <Select
              options={[
                { value: 1, label: '在職' },
                { value: 0, label: '停用' },
                { value: 2, label: '長期請假' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 限制設定 Modal */}
      <Modal
        title={`⚙️ ${selectedEmployee?.name} 的班別限制`}
        open={restrictionModalOpen}
        onCancel={() => setRestrictionModalOpen(false)}
        footer={null}
        width={600}
      >
        {/* 現有限制 */}
        <Table
          dataSource={restrictions}
          rowKey="ID"
          size="small"
          pagination={false}
          columns={[
            {
              title: '班別',
              dataIndex: 'shift_type',
              render: (v: ShiftType) => (
                <Tag color={SHIFT_CONFIG[v]?.color}>{SHIFT_CONFIG[v]?.label || v}</Tag>
              ),
            },
            {
              title: '限制類型',
              render: (_: unknown, r: ShiftRestriction) =>
                r.max_days === null ? (
                  <Tag color="red">完全禁止</Tag>
                ) : (
                  <Tag color="blue">最多 {r.max_days} 天</Tag>
                ),
            },
            { title: '備註', dataIndex: 'note' },
            {
              title: '',
              render: (_: unknown, r: ShiftRestriction) => (
                <Popconfirm title="刪除此限制？" onConfirm={() => handleDeleteRestriction(r.ID)}>
                  <Button size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              ),
            },
          ]}
        />

        {/* 新增限制表單 */}
        <div style={{ marginTop: 16, padding: 16, background: '#fafafa', borderRadius: 8 }}>
          <h4>新增限制</h4>
          <Form form={restrictionForm} layout="inline">
            <Form.Item name="shift_type" rules={[{ required: true }]}>
              <Select
                placeholder="班別"
                style={{ width: 120 }}
                options={[
                  { value: 'day', label: '白班' },
                  { value: 'evening', label: '小夜' },
                  { value: 'night', label: '大夜' },
                  { value: 'day88', label: '8-8' },
                ]}
              />
            </Form.Item>
            <Form.Item name="restriction_type" rules={[{ required: true }]}>
              <Select
                placeholder="類型"
                style={{ width: 140 }}
                options={[
                  { value: 'ban', label: '完全禁止' },
                  { value: 'limit', label: '天數上限' },
                ]}
              />
            </Form.Item>
            <Form.Item name="max_days">
              <InputNumber placeholder="天數" min={1} max={28} style={{ width: 80 }} />
            </Form.Item>
            <Form.Item name="note">
              <Input placeholder="備註" style={{ width: 120 }} />
            </Form.Item>
            <Button type="primary" onClick={handleAddRestriction}>
              新增
            </Button>
          </Form>
        </div>
      </Modal>
    </div>
  );
};

export default EmployeeManagement;
