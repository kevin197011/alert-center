import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Statistic, DatePicker, Spin, Table, Tag, Button, Space, Dropdown, message } from 'antd';
import {
  WarningOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  DownloadOutlined,
  DownOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import { statisticsApi } from '../../services/api';
import type { AlertStatistics, DashboardSummary } from '../../services/api';
import { exportToCSV } from '../../utils/export';
import dayjs from 'dayjs';
import './statistics.css';

const { RangePicker } = DatePicker;

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

export default function Statistics() {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [filters, setFilters] = useState<{ start_time?: string; end_time?: string }>({});

  const { data: stats, isLoading } = useQuery({
    queryKey: ['statistics', filters],
    queryFn: async (): Promise<AlertStatistics | null> => {
      const params: { start_time?: string; end_time?: string } = {};
      if (dateRange?.[0]) params.start_time = dateRange[0].format('YYYY-MM-DD');
      if (dateRange?.[1]) params.end_time = dateRange[1].format('YYYY-MM-DD');
      const res = await statisticsApi.getStatistics(params);
      const body = res.data as unknown as { data?: AlertStatistics };
      return body?.data ?? null;
    },
  });

  const { data: dashboard } = useQuery({
    queryKey: ['dashboardSummary'],
    queryFn: async (): Promise<DashboardSummary | null> => {
      const res = await statisticsApi.getDashboard();
      const body = res.data as unknown as { data?: DashboardSummary };
      return body?.data ?? null;
    },
  });

  const handleDateChange = (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null) => {
    setDateRange(dates && dates[0] && dates[1] ? [dates[0], dates[1]] : null);
    if (dates?.[0] && dates?.[1]) {
      setFilters({
        start_time: dates[0].format('YYYY-MM-DD'),
        end_time: dates[1].format('YYYY-MM-DD'),
      });
    } else {
      setFilters({});
    }
  };

  const dailyColumns = [
    { title: '日期', dataIndex: 'date', key: 'date', width: 120 },
    { title: '总告警', dataIndex: 'total', key: 'total', width: 90 },
    {
      title: '进行中',
      dataIndex: 'firing',
      key: 'firing',
      width: 90,
      render: (v: number) => <Tag color="red">{v}</Tag>,
    },
    {
      title: '已恢复',
      dataIndex: 'resolved',
      key: 'resolved',
      width: 90,
      render: (v: number) => <Tag color="green">{v}</Tag>,
    },
    {
      title: '严重',
      dataIndex: 'critical',
      key: 'critical',
      width: 80,
      render: (v: number) => <Tag color="red">{v}</Tag>,
    },
    {
      title: '警告',
      dataIndex: 'warning',
      key: 'warning',
      width: 80,
      render: (v: number) => <Tag color="orange">{v}</Tag>,
    },
  ];

  const topRuleColumns = [
    { title: '规则名称', dataIndex: 'rule_name', key: 'rule_name', ellipsis: true },
    {
      title: '告警次数',
      dataIndex: 'alert_count',
      key: 'alert_count',
      width: 120,
      render: (v: number) => <Tag color="red">{v}</Tag>,
    },
  ];

  const handleExport = (type: string) => {
    if (type === 'daily') {
      exportToCSV(stats?.by_day || [], [
        { title: '日期', dataIndex: 'date' },
        { title: '总告警', dataIndex: 'total' },
        { title: '进行中', dataIndex: 'firing' },
        { title: '已恢复', dataIndex: 'resolved' },
        { title: '严重', dataIndex: 'critical' },
        { title: '警告', dataIndex: 'warning' },
      ], 'alert_statistics_daily');
      message.success('每日统计导出成功');
    } else if (type === 'rules') {
      exportToCSV(stats?.top_firing_rules || [], [
        { title: '规则ID', dataIndex: 'rule_id' },
        { title: '规则名称', dataIndex: 'rule_name' },
        { title: '告警次数', dataIndex: 'alert_count' },
      ], 'alert_top_rules');
      message.success('TOP规则导出成功');
    }
  };

  const exportItems = [
    { key: 'daily', label: '导出每日统计 (CSV)', onClick: () => handleExport('daily') },
    { key: 'rules', label: '导出TOP规则 (CSV)', onClick: () => handleExport('rules') },
  ];

  return (
    <div className="statistics-page">
      <header className="statistics-header">
        <div className="statistics-header-content">
          <h1>告警统计</h1>
          <p>按时间范围查看告警趋势、级别分布与活跃规则</p>
        </div>
        <Space className="statistics-toolbar" size="middle">
          <RangePicker value={dateRange ?? undefined} onChange={handleDateChange} />
          <Dropdown menu={{ items: exportItems }} placement="bottomRight">
            <Button icon={<DownloadOutlined />}>
              导出 <DownOutlined />
            </Button>
          </Dropdown>
        </Space>
      </header>

      {isLoading ? (
        <div className="statistics-loading">
          <Spin size="large" />
        </div>
      ) : (
        <>
          <section className="statistics-stats">
            <Row gutter={[20, 20]}>
              <Col xs={24} sm={12} lg={6}>
                <Card className="statistics-stat-card" hoverable>
                  <Statistic
                    title="总告警数"
                    value={stats?.total_alerts ?? 0}
                    prefix={<WarningOutlined className="stat-icon stat-icon--warning" />}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="statistics-stat-card" hoverable>
                  <Statistic
                    title="进行中告警"
                    value={stats?.firing_alerts ?? 0}
                    prefix={<CloseCircleOutlined className="stat-icon stat-icon--firing" />}
                    valueStyle={{ color: '#ff4d4f' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="statistics-stat-card" hoverable>
                  <Statistic
                    title="已恢复告警"
                    value={stats?.resolved_alerts ?? 0}
                    prefix={<CheckCircleOutlined className="stat-icon stat-icon--resolved" />}
                    valueStyle={{ color: '#52c41a' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="statistics-stat-card" hoverable>
                  <Statistic
                    title="严重告警"
                    value={stats?.critical_alerts ?? 0}
                    prefix={<InfoCircleOutlined className="stat-icon stat-icon--critical" />}
                    valueStyle={{ color: '#ff4d4f' }}
                  />
                </Card>
              </Col>
            </Row>
          </section>

          <section className="statistics-grid">
            <Row gutter={[20, 20]}>
              <Col xs={24} lg={16}>
                <Card className="statistics-card" title="每日告警趋势">
                  <Table
                    columns={dailyColumns}
                    dataSource={stats?.by_day ?? []}
                    rowKey="date"
                    pagination={false}
                    size="small"
                    scroll={{ x: 520 }}
                  />
                </Card>
              </Col>
              <Col xs={24} lg={8}>
                <Card className="statistics-card" title="告警级别分布">
                  <div className="statistics-severity-list">
                    {(stats?.by_severity ?? []).map((item: { severity: string; count: number }) => (
                      <div key={item.severity} className="statistics-severity-item">
                        <Tag color={severityColors[item.severity] || 'default'} className="label">
                          {item.severity}
                        </Tag>
                        <span className="value">{item.count}</span>
                      </div>
                    ))}
                    {(!stats?.by_severity || stats.by_severity.length === 0) && (
                      <div style={{ color: 'rgba(0,0,0,0.45)', fontSize: 13 }}>暂无数据</div>
                    )}
                  </div>
                </Card>
              </Col>
            </Row>
          </section>

          <section className="statistics-section">
            <Card className="statistics-card" title="TOP 10 活跃告警规则">
              <Table
                columns={topRuleColumns}
                dataSource={stats?.top_firing_rules ?? []}
                rowKey="rule_id"
                pagination={false}
                size="small"
              />
            </Card>
          </section>

          <section className="statistics-overview">
            <Row gutter={[20, 20]}>
              <Col xs={24} sm={8}>
                <Card size="small" className="statistics-stat-card" hoverable>
                  <Statistic title="告警规则总数" value={dashboard?.total_rules ?? 0} prefix={<BarChartOutlined style={{ color: '#1890ff', marginRight: 8 }} />} />
                </Card>
              </Col>
              <Col xs={24} sm={8}>
                <Card size="small" className="statistics-stat-card" hoverable>
                  <Statistic title="启用规则" value={dashboard?.enabled_rules ?? 0} />
                </Card>
              </Col>
              <Col xs={24} sm={8}>
                <Card size="small" className="statistics-stat-card" hoverable>
                  <Statistic title="今日告警" value={dashboard?.today_alerts ?? 0} />
                </Card>
              </Col>
            </Row>
          </section>
        </>
      )}
    </div>
  );
}
