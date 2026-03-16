import { useState } from 'react';
import { ConfigProvider, Layout, Menu, theme } from 'antd';
import { TeamOutlined, CalendarOutlined, SettingOutlined, ScheduleOutlined } from '@ant-design/icons';
import zhTW from 'antd/locale/zh_TW';
import EmployeeManagement from './pages/EmployeeManagement';
import StaffingSettings from './pages/StaffingSettings';
import TemplateEditor from './pages/TemplateEditor';
import MonthlySchedule from './pages/MonthlySchedule';

const { Sider, Content, Header } = Layout;

function App() {
  const [current, setCurrent] = useState('template');

  const menuItems = [
    { key: 'monthly', icon: <ScheduleOutlined />, label: '月度班表' },
    { key: 'template', icon: <CalendarOutlined />, label: '排班模板' },
    { key: 'employees', icon: <TeamOutlined />, label: '員工管理' },
    { key: 'staffing', icon: <SettingOutlined />, label: '人力需求' },
  ];

  const renderPage = () => {
    switch (current) {
      case 'monthly':
        return <MonthlySchedule />;
      case 'employees':
        return <EmployeeManagement />;
      case 'staffing':
        return <StaffingSettings />;
      case 'template':
      default:
        return <TemplateEditor />;
    }
  };

  return (
    <ConfigProvider
      locale={zhTW}
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 8,
        },
      }}
    >
      <Layout style={{ minHeight: '100vh' }}>
        <Sider
          theme="light"
          width={200}
          style={{
            borderRight: '1px solid #f0f0f0',
          }}
        >
          <div
            style={{
              height: 64,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderBottom: '1px solid #f0f0f0',
              fontWeight: 'bold',
              fontSize: 16,
            }}
          >
            🗓️ 排班系統
          </div>
          <Menu
            mode="inline"
            selectedKeys={[current]}
            onClick={(e) => setCurrent(e.key)}
            items={menuItems}
            style={{ borderRight: 0 }}
          />
        </Sider>
        <Layout>
          <Header
            style={{
              background: '#fff',
              padding: '0 24px',
              borderBottom: '1px solid #f0f0f0',
              display: 'flex',
              alignItems: 'center',
              fontSize: 18,
              fontWeight: 600,
            }}
          >
            {menuItems.find((m) => m.key === current)?.label}
          </Header>
          <Content style={{ padding: 24, background: '#f5f5f5', overflow: 'auto' }}>
            <div style={{ background: '#fff', padding: 24, borderRadius: 8, minHeight: 'calc(100vh - 130px)' }}>
              {renderPage()}
            </div>
          </Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
}

export default App;
