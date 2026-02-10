import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, Card, Row, Col, Statistic, DatePicker, Typography, message, Progress, Drawer, Descriptions, Badge } from 'antd';
import { ReloadOutlined, WarningOutlined, BellOutlined, ExclamationCircleOutlined, SendOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { slaBreachApi, SLABreach, SLABreachStats } from '../../services/api';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const breachTypeColors: Record<string, string> = {
  response: 'orange',
  resolution: 'red',
};

const breachTypeLabels: Record<string, string> = {
  response: '响应超时',
  resolution: '解决超时',
};

export default function SLABreaches() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'days'),
    dayjs(),
  ]);
  const [selectedBreach, setSelectedBreach] = useState<SLABreach | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const queryClient = useQueryClient();

  const { data: breachesData, isLoading, refetch } = useQuery({
    queryKey: ['sla-breaches', page, pageSize, dateRange],
    queryFn: async () => {
      const res = await slaBreachApi.getBreaches({ page, page_size: pageSize });
      const body = res.data as unknown as { data?: { data?: SLABreach[]; total?: number }; total?: number };
      const inner = body?.data ?? body;
      const list = Array.isArray(inner?.data) ? inner.data : Array.isArray(inner) ? inner : [];
      const totalVal = typeof inner?.total === 'number' ? inner.total : typeof body?.total === 'number' ? body.total : list.length;
      return { data: list, total: totalVal };
    },
  });

  const { data: statsData } = useQuery({
    queryKey: ['sla-breach-stats', dateRange],
    queryFn: async (): Promise<Partial<SLABreachStats>> => {
      const res = await slaBreachApi.getStats({
        start_time: dateRange[0].format('YYYY-MM-DD'),
        end_time: dateRange[1].format('YYYY-MM-DD'),
      });
      const body = res.data as unknown as { data?: SLABreachStats };
      return (body?.data ?? body ?? {}) as Partial<SLABreachStats>;
    },
    enabled: !!dateRange,
  });

  const triggerCheckMutation = useMutation({
    mutationFn: () => slaBreachApi.triggerCheck(),
    onSuccess: (res) => {
      const payload = (res.data as { data?: { breaches_found?: number } })?.data ?? res.data as { breaches_found?: number };
      const count = payload?.breaches_found ?? 0;
      message.success(`检查完成，发现 ${count} 个SLA违约`);
      queryClient.invalidateQueries({ queryKey: ['sla-breaches'] });
      queryClient.invalidateQueries({ queryKey: ['sla-breach-stats'] });
    },
    onError: (error: Error) => message.error(`检查失败: ${error.message}`),
  });

  const triggerNotifyMutation = useMutation({
    mutationFn: () => slaBreachApi.triggerNotifications(),
    onSuccess: (res) => {
      const payload = (res.data as { data?: { notifications?: number } })?.data ?? res.data as { notifications?: number };
      const count = payload?.notifications ?? 0;
      message.success(`已发送 ${count} 个通知`);
    },
    onError: (error: Error) => message.error(`通知发送失败: ${error.message}`),
  });

  const columns = [
    {
      title: '违约时间',
      dataIndex: 'breach_time',
      key: 'breach_time',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '告警ID',
      dataIndex: 'alert_id',
      key: 'alert_id',
      width: 140,
      render: (id: string) => <Text code>{id.slice(0, 12)}</Text>,
    },
    {
      title: '规则ID',
      dataIndex: 'rule_id',
      key: 'rule_id',
      width: 140,
      render: (id: string) => <Text code>{id.slice(0, 12)}</Text>,
    },
    {
      title: '级别',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severityColors[severity] || 'default'}>
          {severity?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '违约类型',
      dataIndex: 'breach_type',
      key: 'breach_type',
      width: 120,
      render: (type: string) => (
        <Tag color={breachTypeColors[type] || 'default'}>
          {breachTypeLabels[type] || type}
        </Tag>
      ),
    },
    {
      title: '响应时间',
      dataIndex: 'response_time',
      key: 'response_time',
      width: 120,
      render: (secs: number) => {
        const mins = Math.round(secs / 60);
        return <Text type="warning">{mins} 分钟</Text>;
      },
    },
    {
      title: '责任人',
      dataIndex: 'assigned_name',
      key: 'assigned_name',
      width: 120,
      render: (name: string) => name || <Text type="secondary">未分配</Text>,
    },
    {
      title: '已通知',
      dataIndex: 'notified',
      key: 'notified',
      width: 80,
      render: (notified: boolean) => (
        <Badge status={notified ? 'success' : 'default'} text={notified ? '是' : '否'} />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 80,
      render: (_: unknown, record: SLABreach) => (
        <Button type="link" size="small" onClick={() => {
          setSelectedBreach(record);
          setIsDrawerOpen(true);
        }}>
          详情
        </Button>
      ),
    },
  ];

  const breaches = Array.isArray(breachesData?.data) ? breachesData.data : [];
  const total = typeof breachesData?.total === 'number' ? breachesData.total : 0;
  const stats: Partial<SLABreachStats> | undefined = statsData && typeof statsData === 'object' ? statsData : undefined;

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总违约数"
              value={stats?.total_breaches || 0}
              prefix={<WarningOutlined />}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="响应超时"
              value={stats?.total_response_breaches || 0}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="解决超时"
              value={stats?.total_resolution_breaches || 0}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="受影响告警"
              value={stats?.alerts_breached || 0}
              prefix={<BellOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <Space>
            <WarningOutlined />
            <span>SLA违约记录</span>
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
            <Button icon={<SendOutlined />} onClick={() => triggerNotifyMutation.mutate()} loading={triggerNotifyMutation.isPending}>
              发送通知
            </Button>
            <Button type="primary" danger icon={<ExclamationCircleOutlined />} onClick={() => triggerCheckMutation.mutate()} loading={triggerCheckMutation.isPending}>
              立即检查
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={breaches}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
        />
      </Card>

      {stats && (stats.response_breaches || stats.resolution_breaches) && (
        <Row gutter={16} style={{ marginTop: 24 }}>
          <Col span={12}>
            <Card title={<><ClockCircleOutlined /> 按级别统计 - 响应超时</>}>
              {Object.entries(stats.response_breaches || {}).map(([severity, count]: [string, number]) => (
                <div key={severity} style={{ marginBottom: 12 }}>
                  <Space>
                    <Tag color={severityColors[severity]}>{severity.toUpperCase()}</Tag>
                    <Text>{count} 次</Text>
                  </Space>
                  <Progress percent={Math.min((count / (stats.total_response_breaches || 1)) * 100, 100)} size="small" status={count > 0 ? 'exception' : 'success'} />
                </div>
              ))}
            </Card>
          </Col>
          <Col span={12}>
            <Card title={<><ExclamationCircleOutlined /> 按级别统计 - 解决超时</>}>
              {Object.entries(stats.resolution_breaches || {}).map(([severity, count]: [string, number]) => (
                <div key={severity} style={{ marginBottom: 12 }}>
                  <Space>
                    <Tag color={severityColors[severity]}>{severity.toUpperCase()}</Tag>
                    <Text>{count} 次</Text>
                  </Space>
                  <Progress percent={Math.min((count / (stats.total_resolution_breaches || 1)) * 100, 100)} size="small" status={count > 0 ? 'exception' : 'success'} />
                </div>
              ))}
            </Card>
          </Col>
        </Row>
      )}

      <Drawer
        title="SLA违约详情"
        width={600}
        open={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
      >
        {selectedBreach && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="违约ID">{selectedBreach.id}</Descriptions.Item>
            <Descriptions.Item label="告警ID">
              <Text code>{selectedBreach.alert_id}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="规则ID">
              <Text code>{selectedBreach.rule_id}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="告警级别">
              <Tag color={severityColors[selectedBreach.severity]}>
                {selectedBreach.severity?.toUpperCase()}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="违约类型">
              <Tag color={breachTypeColors[selectedBreach.breach_type]}>
                {breachTypeLabels[selectedBreach.breach_type]}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="违约时间">
              {dayjs(selectedBreach.breach_time).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
            <Descriptions.Item label="响应时间">
              {Math.round(selectedBreach.response_time / 60)} 分钟
            </Descriptions.Item>
            <Descriptions.Item label="责任人">
              {selectedBreach.assigned_name || '未分配'}
            </Descriptions.Item>
            <Descriptions.Item label="已通知">
              <Badge status={selectedBreach.notified ? 'success' : 'default'} text={selectedBreach.notified ? '是' : '否'} />
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {dayjs(selectedBreach.created_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
}
