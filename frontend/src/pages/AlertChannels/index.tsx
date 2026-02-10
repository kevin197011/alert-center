import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Modal, Form, Input, Select, Drawer, Dropdown, Tooltip } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ExportOutlined, DownOutlined, SendOutlined } from '@ant-design/icons';
import { alertChannelApi, batchApi, AlertChannel } from '../../services/api';
import dayjs from 'dayjs';

const channelTypes = [
  { value: 'lark', label: '飞书', icon: '📱' },
  { value: 'telegram', label: 'Telegram', icon: '✈️' },
  { value: 'email', label: '邮件', icon: '📧' },
  { value: 'webhook', label: 'Webhook', icon: '🔗' },
];

export default function AlertChannels() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({ type: '', status: '' });
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<AlertChannel | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: channelsData, isLoading } = useQuery({
    queryKey: ['channels', page, pageSize, filters],
    queryFn: async () => {
      const res = await alertChannelApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: AlertChannel[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<AlertChannel>) => alertChannelApi.create(data),
    onSuccess: () => {
      message.success('创建成功');
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<AlertChannel> }) => alertChannelApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功');
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      setIsDrawerOpen(false);
      setEditingChannel(null);
      form.resetFields();
    },
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => alertChannelApi.delete(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['channels'] });
    },
    onError: () => message.error('删除失败'),
  });

  const testMutation = useMutation({
    mutationFn: (id: string) => alertChannelApi.test(id),
    onSuccess: () => message.success('测试消息已发送，请检查渠道是否收到'),
    onError: (err: { response?: { data?: { message?: string } } }) =>
      message.error(err?.response?.data?.message || '测试发送失败'),
  });

  const testConfigMutation = useMutation({
    mutationFn: (data: { type: string; config: Record<string, unknown> }) => alertChannelApi.testWithConfig(data),
    onSuccess: () => message.success('测试消息已发送，请检查渠道是否收到'),
    onError: (err: { response?: { data?: { message?: string } } }) =>
      message.error(err?.response?.data?.message || '测试发送失败'),
  });

  const handleExportChannels = async () => {
    try {
      const res = await batchApi.exportChannels(filters);
      const blob = new Blob([res.data], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `alert_channels_${dayjs().format('YYYYMMDDHHmmss')}.json`;
      link.click();
      message.success('导出成功');
    } catch {
      message.error('导出失败');
    }
  };

  const exportItems = [
    {
      key: 'channels',
      label: '导出渠道',
      onClick: handleExportChannels,
    },
  ];

  const renderConfigFields = (type: string) => {
    switch (type) {
      case 'lark':
        return (
          <>
            <Form.Item name={['config', 'webhook_url']} label="Webhook URL" rules={[{ required: true }]}>
              <Input.Password placeholder="飞书机器人 Webhook URL" />
            </Form.Item>
          </>
        );
      case 'telegram':
        return (
          <>
            <Form.Item name={['config', 'bot_token']} label="Bot Token" rules={[{ required: true }]}>
              <Input.Password placeholder="Telegram Bot Token" />
            </Form.Item>
            <Form.Item name={['config', 'chat_id']} label="Chat ID" rules={[{ required: true }]}>
              <Input placeholder="Telegram Chat ID" />
            </Form.Item>
          </>
        );
      case 'email':
        return (
          <>
            <Form.Item name={['config', 'smtp_host']} label="SMTP 主机" rules={[{ required: true }]}>
              <Input placeholder="smtp.example.com" />
            </Form.Item>
            <Form.Item name={['config', 'smtp_port']} label="SMTP 端口" rules={[{ required: true }]}>
              <Input type="number" placeholder="587" />
            </Form.Item>
            <Form.Item name={['config', 'from_address']} label="发件地址" rules={[{ required: true, type: 'email' }]}>
              <Input placeholder="alert@example.com" />
            </Form.Item>
          </>
        );
      case 'webhook':
        return (
          <Form.Item
            name={['config', 'url']}
            label="Webhook URL"
            rules={[{ required: true, message: '请输入 Webhook URL' }]}
            extra="支持通用 Webhook；飞书机器人地址也可填于此，将自动按飞书格式推送。"
          >
            <Input placeholder="https://your-webhook.com 或飞书机器人 Webhook 地址" />
          </Form.Item>
        );
      default:
        return null;
    }
  };

  const columns = [
    {
      title: '渠道名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => {
        const t = channelTypes.find((t) => t.value === type);
        return (
          <Tag icon={t?.icon}>
            {t?.label || type}
          </Tag>
        );
      },
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: { showTitle: false },
      render: (desc: string) => (
        desc ? (
          <Tooltip placement="topLeft" title={desc}>
            <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {desc}
            </span>
          </Tooltip>
        ) : (
          <span style={{ color: 'rgba(0,0,0,0.25)' }}>—</span>
        )
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => (
        <Tag color={status === 1 ? 'green' : 'red'}>
          {status === 1 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: AlertChannel) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<SendOutlined />}
            loading={testMutation.isPending && testMutation.variables === record.id}
            onClick={() => testMutation.mutate(record.id)}
          >
            测试
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingChannel(record);
              form.setFieldsValue({
                ...record,
                config: typeof record.config === 'string' ? JSON.parse(record.config || '{}') : record.config,
              });
              setIsDrawerOpen(true);
            }}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: '确认删除',
                content: `确定要删除渠道 "${record.name}" 吗？`,
                onOk: () => deleteMutation.mutate(record.id),
              });
            }}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">告警渠道</h1>
        <Space>
          <Dropdown menu={{ items: exportItems }} placement="bottomRight">
            <Button icon={<ExportOutlined />}>
              导出 <DownOutlined />
            </Button>
          </Dropdown>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingChannel(null);
              form.resetFields();
              setIsDrawerOpen(true);
            }}
          >
            新建渠道
          </Button>
        </Space>
      </div>

      <div className="filter-form">
        <Form layout="inline" onFinish={setFilters}>
          <Form.Item name="type" label="渠道类型">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 150 }}
              options={channelTypes}
            />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 100 }}
              options={[
                { value: 'enabled', label: '启用' },
                { value: 'disabled', label: '禁用' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">
              查询
            </Button>
          </Form.Item>
        </Form>
      </div>

      <Table
        columns={columns}
        dataSource={Array.isArray(channelsData?.data) ? channelsData.data : []}
        rowKey="id"
        loading={isLoading}
        scroll={{ x: 800 }}
        pagination={{
          current: page,
          pageSize,
          total: channelsData?.total || 0,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />

      <Drawer
        title={editingChannel ? '编辑告警渠道' : '新建告警渠道'}
        open={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false);
          setEditingChannel(null);
          form.resetFields();
        }}
        width={500}
      >
        <Form form={form} layout="vertical" onFinish={(values) => {
          const configObj = values.config && typeof values.config === 'object' ? values.config : {};
          const data = { ...values, config: configObj };
          if (editingChannel) {
            updateMutation.mutate({ id: editingChannel.id, data });
          } else {
            createMutation.mutate(data);
          }
        }}>
          <Form.Item name="name" label="渠道名称" rules={[{ required: true }]}>
            <Input placeholder="请输入渠道名称" />
          </Form.Item>
          <Form.Item name="type" label="渠道类型" rules={[{ required: true }]}>
            <Select
              placeholder="选择渠道类型"
              options={channelTypes}
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="渠道描述" />
          </Form.Item>
          <Form.Item noStyle dependencies={['type']}>
            {() => renderConfigFields(form.getFieldValue('type'))}
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending || updateMutation.isPending}>
                保存
              </Button>
              <Button
                onClick={async () => {
                  try {
                    const type = form.getFieldValue('type');
                    const config = form.getFieldValue('config');
                    if (!type) {
                      message.warning('请先选择渠道类型');
                      return;
                    }
                    const configObj = config && typeof config === 'object' ? config : {};
                    await testConfigMutation.mutateAsync({ type, config: configObj });
                  } catch {
                    // Error already shown by mutation
                  }
                }}
                loading={testConfigMutation.isPending}
                icon={<SendOutlined />}
              >
                测试
              </Button>
              <Button onClick={() => setIsDrawerOpen(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
}
