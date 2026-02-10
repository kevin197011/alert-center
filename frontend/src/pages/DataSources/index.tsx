import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Modal, Form, Input, Select, Drawer, Badge, Tooltip } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { dataSourceApi, type DataSource } from '../../services/api';
import dayjs from 'dayjs';

const typeOptions = [
  { value: 'prometheus', label: 'Prometheus' },
  { value: 'victoria-metrics', label: 'VictoriaMetrics' },
];

const healthStatusColors: Record<string, string> = {
  healthy: 'success',
  unhealthy: 'error',
  unknown: 'default',
};

export default function DataSources() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({ type: '', status: '' });
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<any>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: sourcesData, isLoading, refetch } = useQuery({
    queryKey: ['dataSources', page, pageSize, filters],
    queryFn: async () => {
      const res = await dataSourceApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: DataSource[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: any) => dataSourceApi.create(data),
    onSuccess: () => {
      message.success('创建成功');
      queryClient.invalidateQueries({ queryKey: ['dataSources'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: (error: any) => message.error(error.response?.data?.message || '创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => dataSourceApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功');
      queryClient.invalidateQueries({ queryKey: ['dataSources'] });
      setIsDrawerOpen(false);
      setEditingSource(null);
      form.resetFields();
    },
    onError: (error: any) => message.error(error.response?.data?.message || '更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => dataSourceApi.delete(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['dataSources'] });
    },
    onError: (error: any) => message.error(error.response?.data?.message || '删除失败'),
  });

  const healthCheckMutation = useMutation({
    mutationFn: (id: string) => dataSourceApi.healthCheck(id),
    onSuccess: () => {
      message.success('健康检查完成');
      refetch();
    },
    onError: () => message.error('健康检查失败'),
  });

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 150,
      render: (type: string) => (
        <Tag color={type === 'prometheus' ? 'orange' : 'blue'}>
          {type?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '端点',
      dataIndex: 'endpoint',
      key: 'endpoint',
      width: 280,
      ellipsis: { showTitle: false },
      render: (endpoint: string) => (
        <Tooltip placement="topLeft" title={endpoint}>
          <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {endpoint}
          </span>
        </Tooltip>
      ),
    },
    {
      title: '健康状态',
      dataIndex: 'health_status',
      key: 'health_status',
      width: 120,
      render: (status: string) => {
        const color = healthStatusColors[status] || 'default';
        return <Badge status={color as 'success' | 'error' | 'default' | 'processing' | 'warning'} text={status || 'unknown'} />;
      },
    },
    {
      title: '最后检查',
      dataIndex: 'last_check_at',
      key: 'last_check_at',
      width: 180,
      render: (time: string | null) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-',
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
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: any) => (
        <Space>
          <Button
            type="link"
            icon={<ReloadOutlined />}
            onClick={() => healthCheckMutation.mutate(record.id)}
          >
            检查
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingSource(record);
              form.setFieldsValue(record);
              setIsDrawerOpen(true);
            }}
          >
            编辑
          </Button>
          <Button
            type="link"
            danger
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: '确认删除',
                content: `确定要删除数据源 "${record.name}" 吗？`,
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
        <h1 className="page-title">数据源管理</h1>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingSource(null);
            form.resetFields();
            setIsDrawerOpen(true);
          }}
        >
          新建数据源
        </Button>
      </div>

      <div className="filter-form">
        <Form layout="inline" onFinish={setFilters}>
          <Form.Item name="type" label="类型">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 150 }}
              options={typeOptions}
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
        dataSource={Array.isArray(sourcesData?.data) ? sourcesData.data : []}
        rowKey="id"
        loading={isLoading}
        scroll={{ x: 900 }}
        pagination={{
          current: page,
          pageSize,
          total: sourcesData?.total || 0,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />

      <Drawer
        title={editingSource ? '编辑数据源' : '新建数据源'}
        open={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false);
          setEditingSource(null);
          form.resetFields();
        }}
        width={500}
        styles={{ body: { overflow: 'auto' } }}
      >
        <Form form={form} layout="vertical" onFinish={(values) => {
          if (editingSource) {
            updateMutation.mutate({ id: editingSource.id, data: values });
          } else {
            createMutation.mutate(values);
          }
        }}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="请输入数据源名称" />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Select placeholder="选择数据源类型" options={typeOptions} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="数据源描述" />
          </Form.Item>
          <Form.Item name="endpoint" label="端点地址" rules={[{ required: true }]}>
            <Input placeholder="http://prometheus:9090" style={{ width: '100%', minWidth: 0, boxSizing: 'border-box' }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: 1, label: '启用' },
                { value: 0, label: '禁用' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending || updateMutation.isPending}>
                保存
              </Button>
              <Button onClick={() => setIsDrawerOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
}
