import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Form, Input, InputNumber, Drawer, Select, Typography, Popconfirm, Statistic, Row, Col, Card, Progress, Tooltip } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, SafetyCertificateOutlined, CheckCircleOutlined, ClockCircleOutlined, WarningOutlined, ExperimentOutlined } from '@ant-design/icons';
import { slaApi, SLAConfig } from '../../services/api';
import dayjs from 'dayjs';

const { Text, Title } = Typography;
const { Option } = Select;

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

interface SeverityOrder {
  critical: number;
  warning: number;
  info: number;
}

const severityOrder: SeverityOrder = { critical: 0, warning: 1, info: 2 };

export default function SLAConfigs() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<SLAConfig | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const [reportDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'days'),
    dayjs(),
  ]);

  const { data: configsData, isLoading, refetch } = useQuery({
    queryKey: ['sla-configs'],
    queryFn: async () => {
      const res = await slaApi.listConfigs();
      const body = res.data as unknown as { data?: { data?: SLAConfig[]; total?: number } };
      const payload = body?.data ?? {};
      return { data: Array.isArray(payload.data) ? payload.data : [], total: payload.total ?? 0 };
    },
  });

  const { data: reportData } = useQuery({
    queryKey: ['sla-report', reportDateRange],
    queryFn: async () => {
      const res = await slaApi.getReport({
        start_time: reportDateRange[0].format('YYYY-MM-DD'),
        end_time: reportDateRange[1].format('YYYY-MM-DD'),
      });
      const body = res.data as unknown as { data?: { met_count?: number; breached_count?: number; compliance_rate?: number } };
      return body?.data ?? {};
    },
    enabled: !!reportDateRange,
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; severity: string; response_time_mins: number; resolution_time_mins: number; priority?: number }) =>
      slaApi.createConfig(data),
    onSuccess: () => {
      message.success('SLA配置创建成功');
      queryClient.invalidateQueries({ queryKey: ['sla-configs'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: (error: Error) => message.error(`创建失败: ${error.message}`),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => slaApi.deleteConfig(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['sla-configs'] });
    },
    onError: (error: Error) => message.error(`删除失败: ${error.message}`),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<SLAConfig> }) => slaApi.updateConfig(id, data),
    onSuccess: () => {
      message.success('更新成功');
      queryClient.invalidateQueries({ queryKey: ['sla-configs'] });
      setIsDrawerOpen(false);
      setEditingConfig(null);
      form.resetFields();
    },
    onError: (error: Error) => message.error(`更新失败: ${error.message}`),
  });

  const seedMutation = useMutation({
    mutationFn: () => slaApi.seedConfigs(),
    onSuccess: () => {
      message.success('默认SLA配置已创建');
      queryClient.invalidateQueries({ queryKey: ['sla-configs'] });
    },
    onError: (error: Error) => message.error(`创建失败: ${error.message}`),
  });

  const handleCreate = () => {
    setEditingConfig(null);
    form.resetFields();
    setIsDrawerOpen(true);
  };

  const handleEdit = (record: SLAConfig) => {
    setEditingConfig(record);
    form.setFieldsValue({
      name: record.name,
      severity: record.severity,
      response_time_mins: record.response_time_mins,
      resolution_time_mins: record.resolution_time_mins,
      priority: record.priority,
    });
    setIsDrawerOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingConfig) {
        await deleteMutation.mutateAsync(editingConfig.id);
      }
      await createMutation.mutateAsync({
        name: values.name,
        severity: values.severity,
        response_time_mins: values.response_time_mins,
        resolution_time_mins: values.resolution_time_mins,
        priority: values.priority ?? 0,
      });
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  const columns = [
    {
      title: 'SLA名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '告警级别',
      dataIndex: 'severity',
      key: 'severity',
      width: 120,
      render: (severity: string) => (
        <Tag color={severityColors[severity] || 'default'}>
          {severity?.toUpperCase()}
        </Tag>
      ),
      sorter: (a: SLAConfig, b: SLAConfig) => severityOrder[a.severity as keyof SeverityOrder] - severityOrder[b.severity as keyof SeverityOrder],
    },
    {
      title: '响应时限',
      dataIndex: 'response_time_mins',
      key: 'response_time_mins',
      width: 140,
      render: (mins: number) => (
        <Space>
          <ClockCircleOutlined />
          <Text>{mins} 分钟</Text>
        </Space>
      ),
    },
    {
      title: '解决时限',
      dataIndex: 'resolution_time_mins',
      key: 'resolution_time_mins',
      width: 140,
      render: (mins: number) => {
        const hours = Math.round(mins / 60);
        const days = Math.floor(hours / 24);
        const remainingHours = hours % 24;
        let text = '';
        if (days > 0) text += `${days}天`;
        if (remainingHours > 0) text += `${remainingHours}小时`;
        return (
          <Space>
            <SafetyCertificateOutlined />
            <Text>{text || `${hours}小时`}</Text>
          </Space>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      sorter: (a: SLAConfig, b: SLAConfig) => b.priority - a.priority,
      render: (priority: number) => (
        <Tag color={priority >= 100 ? 'red' : priority >= 50 ? 'orange' : 'blue'}>
          {priority}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: SLAConfig) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定删除此SLA配置？"
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Tooltip title="删除">
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const configs = Array.isArray(configsData?.data) ? configsData.data : [];
  const sortedConfigs = [...configs].sort((a, b) => severityOrder[a.severity as keyof SeverityOrder] - severityOrder[b.severity as keyof SeverityOrder] || b.priority - a.priority);

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="SLA配置总数"
              value={configs.length}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已满足"
              value={reportData?.met_count || 0}
              valueStyle={{ color: '#3f8600' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="违反SLA"
              value={reportData?.breached_count || 0}
              valueStyle={{ color: '#cf1322' }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <Text type="secondary">SLA达成率</Text>
              <Progress
                percent={Math.round((reportData?.compliance_rate || 0))}
                status={(reportData?.compliance_rate || 0) >= 95 ? 'success' : (reportData?.compliance_rate || 0) >= 80 ? 'normal' : 'exception'}
                strokeColor={(reportData?.compliance_rate || 0) >= 95 ? '#52c41a' : (reportData?.compliance_rate || 0) >= 80 ? '#1890ff' : '#f5222d'}
              />
            </div>
          </Card>
        </Col>
      </Row>

      <Card
        title="SLA配置管理"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
            <Button onClick={() => seedMutation.mutate()}>
              创建默认配置
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              新建SLA配置
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={sortedConfigs}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize,
            total: sortedConfigs.length,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
            showSizeChanger: true,
            showQuickJumper: true,
          }}
        />
      </Card>

      <Drawer
        title={editingConfig ? '编辑SLA配置' : '新建SLA配置'}
        width={480}
        open={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => setIsDrawerOpen(false)} style={{ marginRight: 8 }}>
              取消
            </Button>
            <Button type="primary" onClick={handleSubmit} loading={createMutation.isPending || updateMutation.isPending}>
              保存
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="SLA名称"
            rules={[{ required: true, message: '请输入SLA名称' }]}
          >
            <Input placeholder="例如: Critical SLA" />
          </Form.Item>

          <Form.Item
            name="severity"
            label="告警级别"
            rules={[{ required: true, message: '请选择告警级别' }]}
          >
            <Select placeholder="选择告警级别">
              <Option value="critical">
                <Tag color="red">CRITICAL</Tag> 严重
              </Option>
              <Option value="warning">
                <Tag color="orange">WARNING</Tag> 警告
              </Option>
              <Option value="info">
                <Tag color="blue">INFO</Tag> 信息
              </Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="response_time_mins"
            label="响应时限 (分钟)"
            rules={[{ required: true, message: '请输入响应时限' }]}
            tooltip="告警触发后需要响应的最长时间"
          >
            <InputNumber min={1} max={10080} style={{ width: '100%' }} placeholder="例如: 5" />
          </Form.Item>

          <Form.Item
            name="resolution_time_mins"
            label="解决时限 (分钟)"
            rules={[{ required: true, message: '请输入解决时限' }]}
            tooltip="告警触发后需要解决的最长时间"
          >
            <InputNumber min={1} max={10080} style={{ width: '100%' }} placeholder="例如: 60" />
          </Form.Item>

          <Form.Item
            name="priority"
            label="优先级"
            initialValue={0}
            tooltip="当同一级别有多个SLA配置时，优先级高的生效"
          >
            <InputNumber min={0} max={1000} style={{ width: '100%' }} />
          </Form.Item>

          <Card size="small" style={{ marginTop: 16, backgroundColor: '#fafafa' }}>
            <Title level={5}>
              <ExperimentOutlined /> 级别建议
            </Title>
            <ul style={{ margin: 0, paddingLeft: 20 }}>
              <li><Text type="danger">Critical</Text>: 响应 5-15分钟, 解决 1-4小时</li>
              <li><Text type="warning">Warning</Text>: 响应 15-60分钟, 解决 4-24小时</li>
              <li><Text type="secondary">Info</Text>: 响应 1-24小时, 解决 24小时-7天</li>
            </ul>
          </Card>
        </Form>
      </Drawer>
    </div>
  );
}
