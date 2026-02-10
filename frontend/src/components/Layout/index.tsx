import {
  Layout as AntLayout, Menu, Avatar, Dropdown, Space, Button, Badge, Typography
} from 'antd';
import {
  DashboardOutlined,
  BellOutlined,
  StopOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  ApiOutlined,
  FileTextOutlined,
  BellFilled,
  TeamOutlined,
  AuditOutlined,
  BarChartOutlined,
  DatabaseOutlined,
  MoonOutlined,
  SunOutlined,
  SafetyOutlined,
  CalendarOutlined,
  NodeIndexOutlined,
  WarningOutlined,
  FolderOpenOutlined,
  ArrowUpOutlined,
  GlobalOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../../store/auth';
import { useEffect, useState } from 'react';
import { useWebSocket } from '../../hooks/useWebSocket';
import { Locale, setDayjsLocale, getCurrentLocale } from '../../utils/i18n';

const { Text } = Typography;

const menuItems = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: '仪表盘',
  },
  {
    key: '/rules',
    icon: <BellOutlined />,
    label: '告警规则',
  },
  {
    key: '/channels',
    icon: <ApiOutlined />,
    label: '告警渠道',
  },
  {
    key: '/templates',
    icon: <FileTextOutlined />,
    label: '告警模板',
  },
  {
    key: '/history',
    icon: <SettingOutlined />,
    label: '告警历史',
  },
  {
    key: '/silences',
    icon: <StopOutlined />,
    label: '告警静默',
  },
  {
    key: '/sla',
    icon: <SafetyOutlined />,
    label: 'SLA配置',
  },
  {
    key: '/oncall',
    icon: <CalendarOutlined />,
    label: '值班管理',
  },
  {
    key: '/correlation',
    icon: <NodeIndexOutlined />,
    label: '关联分析',
  },
  {
    key: '/sla-breaches',
    icon: <WarningOutlined />,
    label: 'SLA违约',
  },
  {
    key: '/oncall/report',
    icon: <BarChartOutlined />,
    label: '值班报告',
  },
  {
    key: '/escalations',
    icon: <ArrowUpOutlined />,
    label: '升级历史',
  },
  {
    key: '/tickets',
    icon: <FolderOpenOutlined />,
    label: '工单管理',
  },
  {
    key: '/statistics',
    icon: <BarChartOutlined />,
    label: '告警统计',
  },
  {
    key: '/data-sources',
    icon: <DatabaseOutlined />,
    label: '数据源',
  },
  {
    key: '/users',
    icon: <TeamOutlined />,
    label: '用户管理',
  },
  {
    key: '/audit-logs',
    icon: <AuditOutlined />,
    label: '审计日志',
  },
  {
    key: '/settings',
    icon: <SettingOutlined />,
    label: '系统设置',
  },
];

interface LayoutProps {
  children: React.ReactNode;
  darkMode?: boolean;
  onToggleDark?: () => void;
}

export default function Layout({ children, darkMode, onToggleDark }: LayoutProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { alerts, slaBreaches, tickets, alertCount, slaBreachCount, ticketCount, clearAlerts, clearSLABreaches, clearTickets } = useWebSocket();
  const [locale, setLocale] = useState<Locale>(getCurrentLocale());
  const [siderCollapsed, setSiderCollapsed] = useState(false);

  useEffect(() => {
    localStorage.setItem('locale', locale);
    setDayjsLocale(locale);
  }, [locale]);

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key);
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const changeLocale = (newLocale: Locale) => {
    setLocale(newLocale);
  };

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: locale === 'zh-CN' ? '个人中心' : 'Profile',
    },
    {
      key: 'language',
      icon: <GlobalOutlined />,
      label: locale === 'zh-CN' ? '语言' : 'Language',
      children: [
        {
          key: 'zh-CN',
          label: '简体中文',
          onClick: () => changeLocale('zh-CN'),
        },
        {
          key: 'en-US',
          label: 'English',
          onClick: () => changeLocale('en-US'),
        },
      ],
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'theme',
      icon: darkMode ? <SunOutlined /> : <MoonOutlined />,
      label: (
        <span onClick={(e) => { e.stopPropagation(); onToggleDark?.(); }}>
          {darkMode ? (locale === 'zh-CN' ? '切换到亮色模式' : 'Switch to Light Mode') : (locale === 'zh-CN' ? '切换到深色模式' : 'Switch to Dark Mode')}
        </span>
      ),
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout,
    },
  ];

  const notificationItems = alerts.slice(0, 3).map((alert) => ({
    key: `alert-${alert.alert_id}`,
    label: (
      <div style={{ padding: '8px 0' }}>
        <Text strong style={{ color: alert.severity === 'critical' ? '#ff4d4f' : '#faad14' }}>
          告警: {alert.rule_name}
        </Text>
        <br />
        <Text type="secondary" style={{ fontSize: 12 }}>
          {new Date(alert.timestamp).toLocaleString('zh-CN')}
        </Text>
      </div>
    ),
  }));

  const breachItems = slaBreaches.slice(0, 2).map((breach) => ({
    key: `breach-${breach.breach_id}`,
    label: (
      <div style={{ padding: '8px 0' }}>
        <Text strong style={{ color: '#ff4d4f' }}>
          SLA违约: {breach.breach_type}
        </Text>
        <br />
        <Text type="secondary" style={{ fontSize: 12 }}>
          {new Date(breach.timestamp).toLocaleString('zh-CN')}
        </Text>
      </div>
    ),
  }));

  const ticketItems = tickets.slice(0, 2).map((ticket) => ({
    key: `ticket-${ticket.ticket_id}`,
    label: (
      <div style={{ padding: '8px 0' }}>
        <Text strong>
          工单: {ticket.title}
        </Text>
        <br />
        <Text type="secondary" style={{ fontSize: 12 }}>
          {ticket.action} - {new Date(ticket.timestamp).toLocaleString('zh-CN')}
        </Text>
      </div>
    ),
  }));

  const allNotifications = [
    ...notificationItems,
    ...breachItems,
    ...ticketItems,
  ];

  const headerStyle: React.CSSProperties = {
    background: darkMode ? '#141414' : '#fff',
    padding: '0 24px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
  };

  const contentStyle: React.CSSProperties = {
    margin: '24px 16px',
    padding: 24,
    background: darkMode ? '#1f1f1f' : '#fff',
    borderRadius: 8,
  };

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <AntLayout.Sider
        collapsible
        theme={darkMode ? 'dark' : 'light'}
        onCollapse={(collapsed) => setSiderCollapsed(collapsed)}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '0 16px',
            background: darkMode ? '#001529' : '#f0f5ff',
          }}
        >
          <img
            src={siderCollapsed ? '/alert-center-icon.png' : '/alert-center-logo.png'}
            alt="Alert Center"
            style={{
              maxHeight: 32,
              width: siderCollapsed ? 32 : 'auto',
              objectFit: 'contain',
              ...(darkMode ? { filter: 'brightness(0) invert(1)' } : {}),
            }}
          />
        </div>
        <Menu
          theme={darkMode ? 'dark' : 'light'}
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={handleMenuClick}
        />
      </AntLayout.Sider>
      <AntLayout>
        <AntLayout.Header style={headerStyle}>
          <div />
          <Space size="middle">
            <Dropdown
              menu={{
                items: allNotifications.length > 0 ? [
                  ...allNotifications,
                  { type: 'divider' as const },
                  {
                    key: 'clear',
                    label: (
                      <Space>
                        <Button type="link" size="small" onClick={clearAlerts}>清空告警</Button>
                        <Button type="link" size="small" onClick={clearSLABreaches}>清空违约</Button>
                        <Button type="link" size="small" onClick={clearTickets}>清空工单</Button>
                      </Space>
                    ),
                  },
                ] : [
                  {
                    key: 'empty',
                    label: (
                      <div style={{ padding: '12px 16px', minWidth: 200, textAlign: 'center' }}>
                        <Text type="secondary">暂无实时通知</Text>
                        <br />
                        <Text type="secondary" style={{ fontSize: 12 }}>新告警与工单将在此显示</Text>
                      </div>
                    ),
                  },
                ],
              }}
              placement="bottomRight"
              trigger={['click']}
            >
              <Badge count={alertCount + slaBreachCount + ticketCount} size="small">
                <Button type="text" icon={<BellFilled style={{ fontSize: 18 }} />} />
              </Badge>
            </Dropdown>
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }}>
                <Avatar icon={<UserOutlined />} />
                <span style={{ color: darkMode ? '#fff' : '#000' }}>{user?.username || '用户'}</span>
              </Space>
            </Dropdown>
            </Space>
          </AntLayout.Header>
          <AntLayout.Content style={contentStyle}>
            {children}
          </AntLayout.Content>
      </AntLayout>
    </AntLayout>
  );
}
