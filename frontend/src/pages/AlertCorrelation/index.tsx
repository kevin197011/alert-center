import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, Table, Tag, Timeline, Typography, Space, Button, Select, Row, Col, Statistic, Descriptions, Alert, Tooltip, Badge, Collapse, List, CollapseProps } from 'antd';
import { ReloadOutlined, LinkOutlined, WarningOutlined, ClockCircleOutlined, ExperimentOutlined, NodeIndexOutlined, ThunderboltOutlined, AppstoreOutlined, UnorderedListOutlined } from '@ant-design/icons';
import { alertHistoryApi, correlationApi } from '../../services/api';
import type { AlertHistory } from '../../services/api';
import dayjs from 'dayjs';

const { Text, Title } = Typography;
const { Option } = Select;

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

interface CorrelatedAlert {
  id: string;
  rule_id: string;
  fingerprint: string;
  severity: string;
  status: string;
  started_at: string;
  ended_at: string | null;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  created_at: string;
}

interface CorrelationResult {
  related_alerts?: CorrelatedAlert[];
  root_cause?: { id?: string };
  correlation_score?: number;
  common_labels?: Record<string, string>;
}

export default function AlertCorrelation() {
  const [selectedAlertId, setSelectedAlertId] = useState<string | null>(null);
  const [timeWindow, setTimeWindow] = useState<number>(30);
  const [viewMode, setViewMode] = useState<'list' | 'group'>('list');

  const { data: firingAlerts } = useQuery({
    queryKey: ['firing-alerts'],
    queryFn: async () => {
      const res = await alertHistoryApi.list({ status: 'firing', page: 1, page_size: 100 });
      const body = res.data as unknown as { data?: { data: AlertHistory[] } };
      return body?.data ?? { data: [] as AlertHistory[] };
    },
  });

  const { data: correlationData, isLoading, refetch } = useQuery({
    queryKey: ['correlation', selectedAlertId, timeWindow],
    queryFn: async (): Promise<CorrelationResult | null> => {
      if (!selectedAlertId) return null;
      const res = await correlationApi.getAnalyze(selectedAlertId, { window_minutes: timeWindow });
      const body = res.data as { data?: CorrelationResult };
      return (body?.data ?? body ?? null) as CorrelationResult | null;
    },
    enabled: !!selectedAlertId,
  });

  const { data: patternData } = useQuery({
    queryKey: ['patterns', timeWindow],
    queryFn: async () => {
      const res = await correlationApi.getPatterns({ hours: timeWindow * 2, min_occurrences: 3 });
      const body = res.data as unknown as { data?: unknown };
      return { data: Array.isArray(body?.data) ? body.data : [] };
    },
    enabled: !selectedAlertId,
  });

  const { data: flappingData } = useQuery({
    queryKey: ['flapping'],
    queryFn: async () => {
      const res = await correlationApi.getFlapping();
      const body = res.data as unknown as { data?: unknown };
      return { data: Array.isArray(body?.data) ? body.data : [] };
    },
  });

  const columns = [
    {
      title: '告警时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
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
      title: '规则',
      dataIndex: 'rule_id',
      key: 'rule_id',
      render: (id: string) => <Text code>{id.slice(0, 8)}...</Text>,
    },
    {
      title: '标签',
      dataIndex: 'labels',
      key: 'labels',
      render: (labels: Record<string, string>) => (
        <Space wrap>
          {Object.entries(labels || {}).slice(0, 3).map(([k, v]) => (
            <Tag key={k} color="blue">{k}={v}</Tag>
          ))}
          {Object.keys(labels || {}).length > 3 && (
            <Tooltip title={JSON.stringify(labels, null, 2)}>
              <Tag>+{Object.keys(labels).length - 3}</Tag>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      render: (_: unknown, record: CorrelatedAlert) => (
        <Button
          type="link"
          icon={<NodeIndexOutlined />}
          onClick={() => setSelectedAlertId(record.id)}
        >
          关联分析
        </Button>
      ),
    },
  ];

  const relatedColumns = [
    {
      title: '关联度',
      key: 'score',
      width: 100,
      render: (_: unknown, record: CorrelatedAlert & { correlation_score?: number }) => {
        const score = record.correlation_score || 0;
        let color = '#52c41a';
        if (score < 0.5) color = '#faad14';
        if (score < 0.3) color = '#f5222d';
        return (
          <Badge count={`${Math.round(score * 100)}%`} style={{ backgroundColor: color }} />
        );
      },
    },
    {
      title: '告警时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 160,
      render: (time: string) => dayjs(time).format('MM-DD HH:mm:ss'),
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
      title: '共同标签',
      dataIndex: 'common_labels',
      key: 'common_labels',
      render: (labels: Record<string, string>) => (
        <Space wrap>
          {Object.entries(labels || {}).map(([k, v]) => (
            <Tag key={k} color="purple">{k}={v}</Tag>
          ))}
        </Space>
      ),
    },
  ];

  const firing: CorrelatedAlert[] = Array.isArray(firingAlerts?.data)
    ? (firingAlerts.data as AlertHistory[]).map((a) => ({
        id: a.id,
        rule_id: a.rule_id,
        fingerprint: a.fingerprint ?? '',
        severity: a.severity ?? '',
        status: a.status ?? '',
        started_at: a.started_at,
        ended_at: a.ended_at ?? null,
        labels: typeof a.labels === 'string' ? (JSON.parse(a.labels || '{}') as Record<string, string>) : (a.labels ?? {}),
        annotations: typeof a.annotations === 'string' ? (JSON.parse(a.annotations || '{}') as Record<string, string>) : (a.annotations ?? {}),
        created_at: a.created_at ?? '',
      }))
    : [];

  const groupedAlerts = firing.reduce((groups: Record<string, CorrelatedAlert[]>, alert) => {
    const key = alert.labels?.severity || 'unknown';
    if (!groups[key]) {
      groups[key] = [];
    }
    groups[key].push(alert);
    return groups;
  }, {});

  const groupItems: CollapseProps['items'] = Object.entries(groupedAlerts).map(([key, alerts]) => ({
    key,
    label: (
      <Space>
        <Tag color={severityColors[key] || 'default'}>{key?.toUpperCase()}</Tag>
        <Text>({alerts.length} 条)</Text>
      </Space>
    ),
    children: (
      <List
        dataSource={alerts}
        renderItem={(item: CorrelatedAlert) => (
          <List.Item
            actions={[
              <Button
                key="analyze"
                type="link"
                size="small"
                icon={<NodeIndexOutlined />}
                onClick={() => setSelectedAlertId(item.id)}
              >
                分析
              </Button>,
            ]}
          >
            <List.Item.Meta
              title={
                <Space>
                  <Text code>{item.id.slice(0, 12)}</Text>
                  <Text type="secondary">{dayjs(item.started_at).format('HH:mm:ss')}</Text>
                </Space>
              }
              description={
                <Space wrap>
                  {Object.entries(item.labels || {}).slice(0, 4).map(([k, v]) => (
                    <Tag key={k} color="blue">{k}={v}</Tag>
                  ))}
                </Space>
              }
            />
          </List.Item>
        )}
      />
    ),
  }));

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="当前告警数"
              value={firing.length}
              prefix={<ThunderboltOutlined />}
              valueStyle={{ color: firing.length > 10 ? '#cf1322' : '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="关联告警组"
              value={patternData?.data?.length || 0}
              prefix={<LinkOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="频繁告警"
              value={flappingData?.data?.length || 0}
              prefix={<ExperimentOutlined />}
              valueStyle={{ color: (flappingData?.data?.length || 0) > 0 ? '#cf1322' : undefined }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="分析窗口"
              value={timeWindow}
              suffix="分钟"
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <Space>
            <NodeIndexOutlined />
            <span>告警关联分析</span>
          </Space>
        }
        extra={
          <Space>
            <Button
              type={viewMode === 'list' ? 'primary' : 'default'}
              icon={<UnorderedListOutlined />}
              onClick={() => setViewMode('list')}
            >
              列表
            </Button>
            <Button
              type={viewMode === 'group' ? 'primary' : 'default'}
              icon={<AppstoreOutlined />}
              onClick={() => setViewMode('group')}
            >
              分组
            </Button>
            <Select value={timeWindow} onChange={setTimeWindow} style={{ width: 120 }}>
              <Option value={15}>15分钟</Option>
              <Option value={30}>30分钟</Option>
              <Option value={60}>1小时</Option>
              <Option value={120}>2小时</Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={() => { refetch(); }}>
              刷新
            </Button>
          </Space>
        }
      >
        {selectedAlertId ? (
          <div>
            <Alert
              message="关联分析模式"
              description={
                <Space>
                  <Text>正在分析告警: </Text>
                  <Text code>{selectedAlertId.slice(0, 8)}</Text>
                  <Button type="link" onClick={() => setSelectedAlertId(null)}>
                    返回列表
                  </Button>
                </Space>
              }
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Row gutter={16}>
              <Col span={16}>
                <Title level={5}>
                  <LinkOutlined /> 关联告警
                </Title>
                <Table
                  columns={relatedColumns}
                  dataSource={correlationData?.related_alerts || []}
                  rowKey="id"
                  pagination={false}
                  size="small"
                />
              </Col>
              <Col span={8}>
                <Title level={5}>
                  <WarningOutlined /> 根因分析
                </Title>
                <Card size="small">
                  <Descriptions column={1} size="small">
                    <Descriptions.Item label="根因告警">
                      <Tag color="red">
                        {correlationData?.root_cause?.id?.slice(0, 8)}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="关联度">
                      <Text strong style={{ color: '#1890ff' }}>
                        {Math.round((correlationData?.correlation_score || 0) * 100)}%
                      </Text>
                    </Descriptions.Item>
                    <Descriptions.Item label="共同标签">
                      <Space wrap>
                        {Object.entries(correlationData?.common_labels || {}).map(([k, v]) => (
                          <Tag key={k}>{k}={String(v)}</Tag>
                        ))}
                      </Space>
                    </Descriptions.Item>
                  </Descriptions>
                </Card>

                <Title level={5} style={{ marginTop: 16 }}>
                  <ClockCircleOutlined /> 时间线
                </Title>
                <Timeline
                  items={correlationData?.related_alerts?.map((a: CorrelatedAlert) => ({
                    color: severityColors[a.severity] || 'blue',
                    children: `${dayjs(a.started_at).format('HH:mm:ss')} - ${a.severity}`,
                  })) || []}
                />
              </Col>
            </Row>
          </div>
        ) : viewMode === 'group' ? (
          <Collapse items={groupItems} defaultActiveKey={Object.keys(groupedAlerts)} />
        ) : (
          <Table
            columns={columns}
            dataSource={firing}
            rowKey="id"
            loading={isLoading}
            pagination={{ pageSize: 10 }}
          />
        )}
      </Card>

      {!selectedAlertId && (
        <Row gutter={16} style={{ marginTop: 24 }}>
          <Col span={12}>
            <Card title={<><ExperimentOutlined /> 频繁告警模式</>}>
              {(flappingData?.data?.length ?? 0) > 0 ? (
                <ul>
                  {(flappingData?.data || []).slice(0, 5).map((item: string, idx: number) => (
                    <li key={idx} style={{ marginBottom: 8 }}>
                      <Tag color="red"><ExperimentOutlined /> {item}</Tag>
                    </li>
                  ))}
                </ul>
              ) : (
                <Text type="secondary">未检测到频繁告警</Text>
              )}
            </Card>
          </Col>
          <Col span={12}>
            <Card title={<><LinkOutlined /> 告警模式</>}>
              {(patternData?.data?.length ?? 0) > 0 ? (
                <ul>
                  {(patternData?.data || []).slice(0, 5).map((pattern: { common_labels: Record<string, string>; occurrence_count: number }, idx: number) => (
                    <li key={idx} style={{ marginBottom: 8 }}>
                      <Space>
                        <Tag color="blue">{pattern.occurrence_count}次</Tag>
                        <Text>
                          {Object.entries(pattern.common_labels || {}).map(([k, v]) => `${k}=${v}`).join(', ')}
                        </Text>
                      </Space>
                    </li>
                  ))}
                </ul>
              ) : (
                <Text type="secondary">未检测到告警模式</Text>
              )}
            </Card>
          </Col>
        </Row>
      )}
    </div>
  );
}
