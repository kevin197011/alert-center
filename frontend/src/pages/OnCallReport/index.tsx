import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, Table, Row, Col, Statistic, DatePicker, Button, Tag, Typography, Space, Progress } from 'antd';
import { ReloadOutlined, UserOutlined, CalendarOutlined, ClockCircleOutlined, BellOutlined, CheckCircleOutlined, WarningOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text, Title } = Typography;
const { RangePicker } = DatePicker;

interface UserOnCallStats {
  user_id: string;
  username: string;
  shift_count: number;
  total_hours: number;
  alert_count: number;
  resolved_count: number;
  avg_response_time: number;
}

interface ScheduleOnCallStats {
  schedule_id: string;
  schedule_name: string;
  shift_count: number;
  coverage_percent: number;
}

interface AlertOnCallStats {
  total_alerts: number;
  assigned_alerts: number;
  resolved_alerts: number;
  avg_response_time: number;
  escalation_count: number;
}

interface OnCallReport {
  period_start: string;
  period_end: string;
  total_shifts: number;
  total_hours: number;
  by_user: UserOnCallStats[];
  by_schedule: ScheduleOnCallStats[];
  alert_stats: AlertOnCallStats;
}

export default function OnCallReport() {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'days'),
    dayjs(),
  ]);

  const { data: reportData, isLoading, refetch } = useQuery({
    queryKey: ['oncall-report', dateRange],
    queryFn: async () => {
      const res = await fetch(`/api/v1/oncall/report?start_time=${dateRange[0].format('YYYY-MM-DD')}&end_time=${dateRange[1].format('YYYY-MM-DD')}`);
      return res.json();
    },
    enabled: !!dateRange,
  });

  const userColumns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (name: string) => (
        <Space>
          <UserOutlined />
          <Text strong>{name}</Text>
        </Space>
      ),
    },
    {
      title: '值班班次',
      dataIndex: 'shift_count',
      key: 'shift_count',
      width: 100,
      render: (count: number) => <Tag>{count} 次</Tag>,
    },
    {
      title: '总时长',
      dataIndex: 'total_hours',
      key: 'total_hours',
      width: 120,
      render: (hours: number) => (
        <Space>
          <ClockCircleOutlined />
          <Text>{hours.toFixed(1)} 小时</Text>
        </Space>
      ),
    },
    {
      title: '处理告警',
      dataIndex: 'alert_count',
      key: 'alert_count',
      width: 100,
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: '已解决',
      dataIndex: 'resolved_count',
      key: 'resolved_count',
      width: 100,
      render: (count: number) => <Tag color="green">{count}</Tag>,
    },
    {
      title: '平均响应',
      dataIndex: 'avg_response_time',
      key: 'avg_response_time',
      width: 120,
      render: (secs: number) => {
        const mins = Math.round(secs / 60);
        return <Text type="warning">{mins} 分钟</Text>;
      },
    },
  ];

  const scheduleColumns = [
    {
      title: '排班名称',
      dataIndex: 'schedule_name',
      key: 'schedule_name',
      render: (name: string) => (
        <Space>
          <CalendarOutlined />
          <Text strong>{name}</Text>
        </Space>
      ),
    },
    {
      title: '班次数量',
      dataIndex: 'shift_count',
      key: 'shift_count',
      width: 120,
      render: (count: number) => <Tag>{count}</Tag>,
    },
    {
      title: '覆盖率',
      dataIndex: 'coverage_percent',
      key: 'coverage_percent',
      width: 200,
      render: (percent: number) => (
        <Progress percent={Math.round(percent)} size="small" status={percent >= 95 ? 'success' : percent >= 80 ? 'normal' : 'exception'} />
      ),
    },
  ];

  const report: OnCallReport | null = reportData?.data || null;

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总值班时长"
              value={report?.total_hours || 0}
              suffix="小时"
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="值班班次"
              value={report?.total_shifts || 0}
              prefix={<CalendarOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="处理告警"
              value={report?.alert_stats?.total_alerts || 0}
              prefix={<BellOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="升级次数"
              value={report?.alert_stats?.escalation_count || 0}
              valueStyle={{ color: '#fa8c16' }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <Space>
            <CalendarOutlined />
            <span>值班报告</span>
          </Space>
        }
        extra={
          <Space>
            <RangePicker
              value={dateRange}
              onChange={(dates) => {
                if (dates && dates[0] && dates[1]) {
                  setDateRange([dates[0], dates[1]]);
                }
              }}
            />
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
          </Space>
        }
      >
        {report && (
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}>
              <Card size="small">
                <Statistic
                  title="平均响应时间"
                  value={Math.round((report.alert_stats?.avg_response_time || 0) / 60)}
                  suffix="分钟"
                  valueStyle={{ color: '#1890ff' }}
                  prefix={<ClockCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic
                  title="已解决告警"
                  value={report.alert_stats?.resolved_alerts || 0}
                  valueStyle={{ color: '#52c41a' }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic
                  title="解决率"
                  value={report.alert_stats?.total_alerts ? Math.round((report.alert_stats?.resolved_alerts / report.alert_stats?.total_alerts) * 100) : 0}
                  suffix="%"
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic
                  title="报告周期"
                  value={`${report.period_start} ~ ${report.period_end}`}
                  valueStyle={{ fontSize: 14 }}
                />
              </Card>
            </Col>
          </Row>
        )}

        <Row gutter={16}>
          <Col span={14}>
            <Title level={5}>
              <UserOutlined /> 个人值班统计
            </Title>
            <Table
              columns={userColumns}
              dataSource={report?.by_user || []}
              rowKey="user_id"
              loading={isLoading}
              pagination={false}
              size="small"
            />
          </Col>
          <Col span={10}>
            <Title level={5}>
              <CalendarOutlined /> 排班覆盖统计
            </Title>
            <Table
              columns={scheduleColumns}
              dataSource={report?.by_schedule || []}
              rowKey="schedule_id"
              loading={isLoading}
              pagination={false}
              size="small"
            />
          </Col>
        </Row>
      </Card>
    </div>
  );
}
