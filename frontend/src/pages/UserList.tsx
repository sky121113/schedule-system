import { useEffect, useState } from 'react';
import { Table, Button, Space, message } from 'antd';
import { userApi } from '../services/api';
import { User } from '../types';
import { useUserStore } from '../store/userStore';

const UserList = () => {
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(false);
    const { currentUser, setCurrentUser } = useUserStore();

    const fetchUsers = async () => {
        setLoading(true);
        try {
            const data: any = await userApi.getUsers();
            setUsers(data);
        } catch (error) {
            message.error('載入使用者失敗');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchUsers();
    }, []);

    const columns = [
        {
            title: 'ID',
            dataIndex: 'ID',
            key: 'ID',
        },
        {
            title: '姓名',
            dataIndex: 'name',
            key: 'name',
        },
        {
            title: 'Email',
            dataIndex: 'email',
            key: 'email',
        },
        {
            title: '角色',
            dataIndex: 'role',
            key: 'role',
        },
        {
            title: '狀態',
            dataIndex: 'status',
            key: 'status',
            render: (status: number) => (status === 1 ? '啟用' : '停用'),
        },
        {
            title: '操作',
            key: 'action',
            render: (_: any, record: User) => (
                <Space size="middle">
                    <Button
                        type={currentUser?.ID === record.ID ? 'primary' : 'default'}
                        onClick={() => {
                            setCurrentUser(record);
                            message.success(`已選擇使用者：${record.name}`);
                        }}
                    >
                        {currentUser?.ID === record.ID ? '當前使用者' : '選擇'}
                    </Button>
                </Space>
            ),
        },
    ];

    return (
        <div>
            <h2>使用者管理</h2>
            {currentUser && (
                <div style={{ marginBottom: 16, padding: 12, background: '#e6f7ff', borderRadius: 4 }}>
                    當前使用者：<strong>{currentUser.name}</strong> ({currentUser.email})
                </div>
            )}
            <Table
                columns={columns}
                dataSource={users}
                rowKey="ID"
                loading={loading}
            />
        </div>
    );
};

export default UserList;
