import { useState } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, theme } from 'antd';
import { CalendarOutlined, UserOutlined, SettingOutlined } from '@ant-design/icons';
import ScheduleCalendar from './pages/ScheduleCalendar';
import UserList from './pages/UserList';
import './App.css';

const { Header, Sider, Content } = Layout;

function AppContent() {
    const [collapsed, setCollapsed] = useState(false);
    const navigate = useNavigate();
    const location = useLocation();
    const {
        token: { colorBgContainer, borderRadiusLG },
    } = theme.useToken();

    // 根據當前路徑設定選中的選單項
    const getSelectedKey = () => {
        if (location.pathname === '/schedule') return '1';
        if (location.pathname === '/users') return '2';
        return '1';
    };

    // 處理選單點擊
    const handleMenuClick = (e: { key: string }) => {
        switch (e.key) {
            case '1':
                navigate('/schedule');
                break;
            case '2':
                navigate('/users');
                break;
            case '3':
                // 系統設定頁面尚未實作
                console.log('系統設定功能開發中...');
                break;
        }
    };

    return (
        <Layout style={{ minHeight: '100vh' }}>
            <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed}>
                <div style={{
                    height: '64px',
                    margin: '16px',
                    background: 'rgba(255, 255, 255, 0.2)',
                    borderRadius: '8px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: 'white',
                    fontSize: '20px',
                    fontWeight: 'bold'
                }}>
                    {collapsed ? '排' : '排班系統'}
                </div>
                <Menu
                    theme="dark"
                    selectedKeys={[getSelectedKey()]}
                    mode="inline"
                    onClick={handleMenuClick}
                    items={[
                        {
                            key: '1',
                            icon: <CalendarOutlined />,
                            label: '班表行事曆',
                        },
                        {
                            key: '2',
                            icon: <UserOutlined />,
                            label: '使用者管理',
                        },
                        {
                            key: '3',
                            icon: <SettingOutlined />,
                            label: '系統設定',
                        },
                    ]}
                />
            </Sider>
            <Layout>
                <Header style={{ padding: '0 24px', background: colorBgContainer }}>
                    <h2 style={{ margin: 0 }}>排班管理系統</h2>
                </Header>
                <Content style={{ margin: '24px 16px', padding: 24, background: colorBgContainer, borderRadius: borderRadiusLG }}>
                    <Routes>
                        <Route path="/" element={<Navigate to="/schedule" replace />} />
                        <Route path="/schedule" element={<ScheduleCalendar />} />
                        <Route path="/users" element={<UserList />} />
                    </Routes>
                </Content>
            </Layout>
        </Layout>
    );
}

function App() {
    return (
        <BrowserRouter>
            <AppContent />
        </BrowserRouter>
    );
}

export default App;
