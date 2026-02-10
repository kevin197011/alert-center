import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Table, Card, Row, Col, Statistic, DatePicker, Select, Button, Tag, Typography, Space, Tooltip, Badge } from 'antd';
import { ReloadOutlined, ArrowUpOutlined, CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, UserOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;

interface EscalationHistory {
  id: string;
  alert_id: string;
  rule_id: string;
  rule_name: string;
  severity: string;
  from_user_id: string;
  from_username: string;
  to_user_id: string;
  to_username: string;
  reason: string;
  status: string;
  created_at: string;
  accepted_at?: string;
  resolved_at?: string;
  response_time: number;
}

interface EscalationHistoryStats {
  total_escalations: number;
  pending_escalations: number;
  accepted_escalations: number;
  resolved_escalations: number;
  rejected_escalations: number;
  by_user: UserEscalationStats[];
}

interface UserEscalationStats {
  user_id: string;
  username: string;
  escalated_count: number;
  accepted_count: number;
  resolved_count: number;
}

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const statusColors: Record<string, string> = {
  pending: 'orange',
  accepted: 'blue',
  resolved: 'green',
  rejected: 'red',
};

const statusLabels: Record<string, string> = {
  pending: '待处理',
  accepted: '已接受',
  resolved: '已解决',
  rejected: '已拒绝',
};

export default function EscalationHistory() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [status, setStatus] = useState<string>('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);

  const { data: historyData, isLoading, refetch } = useQuery({
    queryKey: ['escalation-history', page, pageSize, status],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
      });
      if (status && status !== 'all') {
        params.append('status', status);
      }
      const res = await fetch(`/api/v1/escalations?${params}`);
      return res.json();
    },
  });

  const { data: statsData } = useQuery({
    queryKey: ['escalation-stats'],
    queryFn: async () => {
      const res = await fetch('/api/v1/escalations/stats');
      return res.json();
    },
  });

  const columns = [
    {
      title: '升级时间',
      dataIndex: 'created_at',
      key: 'created_at',
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
      title: '规则',
      dataIndex: 'rule_name',
      key: 'rule_name',
      render: (name: string) => name || '-',
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
      title: '发起人',
      dataIndex: 'from_username',
      key: 'from_username',
      width: 120,
      render: (name: string) => (
        <Space>
          <UserOutlined />
          <Text>{name}</Text>
        </Space>
      ),
    },
    {
      title: '升级给',
      dataIndex: 'to_username',
      key: 'to_username',
      width: 120,
      render: (name: string) => (
        <Space>
          <UserOutlined />
          <Text strong>{name}</Text>
        </Space>
      ),
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
      render: (reason: string) => (
        <Tooltip title={reason}>
          <Text type="secondary">{reason || '-'}</Text>
        </Tooltip>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Badge status={statusColors[status] as any} text={statusLabels[status] || status} />
      ),
    },
    {
      title: '响应时间',
      dataIndex: 'response_time',
      key: 'response_time',
      width: 120,
      render: (secs: number) => {
        const mins = Math.round(secs / 60);
        return (
          <Space>
            <ClockCircleOutlined />
            <Text type={mins > 30 ? 'danger' : mins > 15 ? 'warning' : undefined}>
              {mins} 分钟
            </Text>
          </Space>
        );
      },
    },
  ];

  const stats: EscalationHistoryStats | null = statsData?.data || null;
  const history = historyData?.data?.data || [];
  const total = historyData?.data?.total || 0;

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总升级数"
              value={stats?.total_escalations || 0}
              prefix={<ArrowUpOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="待处理"
              value={stats?.pending_escalations || 0}
              valueStyle={{ color: '#fa8c16' }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={stats?.resolved_escalations || 0}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已拒绝"
              value={stats?.rejected_escalations || 0}
              valueStyle={{ color: '#cf1322' }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <Space>
            <ArrowUpOutlined />
            <span>升级历史</span>
          </Space>
        }
        extra={
          <Space>
            <RangePicker
              value={dateRange}
              onChange={(dates) => {
                if (dates && dates[0] && dates[1]) {
                  setDateRange([dates[0], dates[1]]);
                } else {
                  setDateRange(null);
                }
              }}
            />
            <Select
              value={status}
              onChange={setStatus}
              style={{ width: 120 }}
              placeholder="状态"
            >
              <Option value="all">全部</Option>
              <Option value="pending">待处理</Option>
              <Option value="accepted">已接受</Option>
              <Option value="resolved">已解决</Option>
              <Option value="rejected">已拒绝</Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={history}
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

      {stats?.by_user && stats.by_user.length > 0 && (
        <Card title={<><UserOutlined /> 个人升级统计</>} style={{ marginTop: 24 }}>
          <Table
            dataSource={stats.by_user}
            rowKey="user_id"
            pagination={false}
            size="small"
            columns={[
              {
                title: '用户',
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
                title: '发起升级',
                dataIndex: 'escalated_count',
                key: 'escalated_count',
                width: 120,
                render: (count: number) => <Tag>{count}</Tag>,
              },
              {
                title: '接受升级',
                dataIndex: 'accepted_count',
                key: 'accepted_count',
                width: 120,
                render: (count: number) => <Tag color="blue">{count}</Tag>,
              },
              {
                title: '解决升级',
                dataIndex: 'resolved_count',
                key: 'resolved_count',
                width: 120,
                render: (count: number) => <Tag color="green">{count}</Tag>,
              },
              {
                title: '解决率',
                key: 'resolution_rate',
                width: 150,
                render: (_: unknown, record: UserEscalationStats) => {
                  const rate = record.accepted_count > 0
                    ? Math.round((record.resolved_count / record.accepted_count) * 100)
                    : 0;
                  return (
                    <Text type={rate >= 80 ? 'success' : rate >= 50 ? 'warning' : 'danger'}>
                      {rate}%
                    </Text>
                  );
                },
              },
            ]}
          />
        </Card>
      )}
    </div>
  );
}
