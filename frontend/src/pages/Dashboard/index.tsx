import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Statistic, Spin, Table, Tag, Typography } from 'antd';
import {
  WarningOutlined,
  CheckCircleOutlined,
  ApiOutlined,
  BellOutlined,
  CalendarOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { alertHistoryApi, statisticsApi } from '../../services/api';
import type { AlertHistory, DashboardSummary } from '../../services/api';
import './dashboard.css';

const { Text } = Typography;

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const statusColors: Record<string, string> = {
  firing: 'red',
  resolved: 'green',
};

export default function Dashboard() {
  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ['dashboardSummary'],
    queryFn: async (): Promise<DashboardSummary | null> => {
      const res = await statisticsApi.getDashboard();
      const body = res.data as unknown as { data?: DashboardSummary };
      return body?.data ?? null;
    },
  });

  const { data: recentData, isLoading: recentLoading } = useQuery({
    queryKey: ['alertHistory', 'recent'],
    queryFn: async () => {
      const res = await alertHistoryApi.list({ page: 1, page_size: 5 });
      const body = res.data as unknown as {
        data?: { data: AlertHistory[]; total: number; page: number; size: number };
      };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 5 };
      return {
        data: Array.isArray(payload.data) ? payload.data : [],
        total: payload.total ?? 0,
      };
    },
  });

  const loading = summaryLoading;
  const recentList = recentData?.data ?? [];

  const recentColumns = [
    {
      title: '规则 ID',
      dataIndex: 'rule_id',
      key: 'rule_id',
      ellipsis: true,
      render: (id: string) => (
        <Text copyable={{ text: id }} style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {id.slice(0, 8)}…
        </Text>
      ),
    },
    {
      title: '严重级别',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (s: string) => <Tag color={severityColors[s] || 'default'}>{s}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s === 'firing' ? '进行中' : '已恢复'}</Tag>,
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 180,
      render: (t: string) => (t ? new Date(t).toLocaleString('zh-CN') : '—'),
    },
  ];

  return (
    <div className="dashboard-page">
      <header className="dashboard-header">
        <div>
          <h1 className="dashboard-title">仪表盘</h1>
          <p className="dashboard-subtitle">概览告警规则、渠道与近期告警</p>
        </div>
      </header>

      {loading ? (
        <div className="dashboard-loading">
          <Spin size="large" />
        </div>
      ) : (
        <>
          <section className="dashboard-stats">
            <Row gutter={[20, 20]}>
              <Col xs={24} sm={12} lg={6}>
                <Card className="dashboard-stat-card" hoverable>
                  <Statistic
                    title="告警规则数"
                    value={summary?.total_rules ?? 0}
                    prefix={<BellOutlined className="stat-icon stat-icon--blue" />}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="dashboard-stat-card" hoverable>
                  <Statistic
                    title="告警渠道数"
                    value={summary?.total_channels ?? 0}
                    prefix={<ApiOutlined className="stat-icon stat-icon--cyan" />}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="dashboard-stat-card dashboard-stat-card--alert" hoverable>
                  <Statistic
                    title="进行中告警"
                    value={summary?.firing_alerts ?? 0}
                    prefix={<WarningOutlined className="stat-icon stat-icon--orange" />}
                    valueStyle={{ color: 'var(--stat-firing, #fa8c16)' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="dashboard-stat-card" hoverable>
                  <Statistic
                    title="今日告警"
                    value={summary?.today_alerts ?? 0}
                    prefix={<CalendarOutlined className="stat-icon stat-icon--green" />}
                  />
                </Card>
              </Col>
            </Row>
          </section>

          <section className="dashboard-recent">
            <Card
              className="dashboard-recent-card"
              title={
                <span>
                  <BellOutlined style={{ marginRight: 8 }} />
                  最近告警
                </span>
              }
              extra={
                <Link to="/history" className="dashboard-recent-link">
                  查看更多 <RightOutlined />
                </Link>
              }
            >
              {recentLoading ? (
                <div className="dashboard-recent-loading">
                  <Spin />
                </div>
              ) : recentList.length === 0 ? (
                <div className="dashboard-recent-empty">
                  <CheckCircleOutlined style={{ fontSize: 40, color: 'var(--color-success, #52c41a)' }} />
                  <p>暂无告警记录</p>
                </div>
              ) : (
                <Table
                  dataSource={recentList}
                  columns={recentColumns}
                  rowKey="id"
                  pagination={false}
                  size="small"
                  showHeader
                />
              )}
            </Card>
          </section>
        </>
      )}
    </div>
  );
}
