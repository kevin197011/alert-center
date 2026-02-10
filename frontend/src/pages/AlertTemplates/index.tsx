import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Modal, Form, Input, Select, Drawer, Card } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import { templateApi, AlertTemplate } from '../../services/api';
import dayjs from 'dayjs';

const templateTypes = [
  { value: 'markdown', label: 'Markdown' },
  { value: 'text', label: '纯文本' },
  { value: 'html', label: 'HTML' },
];

const defaultTemplateContent = `**告警通知**

**规则名称**: {{ruleName}}
**严重级别**: {{severity}}
**状态**: {{status}}
**触发时间**: {{startTime}}
**持续时间**: {{duration}}

**告警详情**:
{{labels}}

**描述**:
{{annotations}}

**恢复建议**:
请检查相关服务状态，确认问题后及时处理。`;

export default function AlertTemplates() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({ type: '', status: '' });
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [isPreviewOpen, setIsPreviewOpen] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
  const [editingTemplate, setEditingTemplate] = useState<AlertTemplate | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: templatesData, isLoading } = useQuery({
    queryKey: ['templates', page, pageSize, filters],
    queryFn: async () => {
      const res = await templateApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: AlertTemplate[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<AlertTemplate>) => templateApi.create(data),
    onSuccess: () => {
      message.success('创建成功');
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<AlertTemplate> }) => templateApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功');
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      setIsDrawerOpen(false);
      setEditingTemplate(null);
      form.resetFields();
    },
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => templateApi.delete(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['templates'] });
    },
    onError: () => message.error('删除失败'),
  });

  const handlePreview = (record: AlertTemplate) => {
    const sampleData = {
      ruleName: record.name,
      severity: 'warning',
      status: 'firing',
      startTime: new Date().toLocaleString('zh-CN'),
      duration: '5分钟',
      labels: JSON.stringify({ instance: 'server-01', job: 'nginx' }, null, 2),
      annotations: record.description || '告警触发',
    };

    let content = record.content;
    for (const [key, value] of Object.entries(sampleData)) {
      content = content.replace(new RegExp(`{{${key}}}`, 'g'), String(value));
    }

    setPreviewContent(content);
    setIsPreviewOpen(true);
  };

  const columns = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={type === 'markdown' ? 'blue' : type === 'html' ? 'green' : 'orange'}>
          {type?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 240,
      ellipsis: true,
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
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 180,
      render: (_: unknown, record: AlertTemplate) => (
        <Space>
          <Button type="link" icon={<EyeOutlined />} onClick={() => handlePreview(record)}>
            预览
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingTemplate(record);
              form.setFieldsValue({
                ...record,
                variables: typeof record.variables === 'string' ? record.variables : (record.variables ? JSON.stringify(record.variables, null, 2) : '{}'),
              });
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
                content: `确定要删除模板 "${record.name}" 吗？`,
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
        <h1 className="page-title">告警模板</h1>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingTemplate(null);
            form.resetFields();
            form.setFieldsValue({ content: defaultTemplateContent, type: 'markdown' });
            setIsDrawerOpen(true);
          }}
        >
          新建模板
        </Button>
      </div>

      <div className="filter-form">
        <Form layout="inline" onFinish={setFilters}>
          <Form.Item name="type" label="模板类型">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 150 }}
              options={templateTypes}
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

      <div style={{ overflow: 'auto' }}>
        <Table
          columns={columns}
          dataSource={Array.isArray(templatesData?.data) ? templatesData.data : []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 940 }}
          pagination={{
            current: page,
            pageSize,
            total: templatesData?.total || 0,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </div>

      <Drawer
        title={editingTemplate ? '编辑告警模板' : '新建告警模板'}
        open={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false);
          setEditingTemplate(null);
          form.resetFields();
        }}
        width={700}
      >
        <Form form={form} layout="vertical" onFinish={(values) => {
          let variablesObj: Record<string, string> = {};
          const raw = values.variables;
          if (typeof raw === 'string' && raw.trim()) {
            try {
              variablesObj = JSON.parse(raw);
            } catch {
              message.warning('变量定义不是合法 JSON，已忽略');
            }
          } else if (typeof raw === 'object' && raw !== null) {
            variablesObj = raw as Record<string, string>;
          }
          const data = {
            ...values,
            variables: variablesObj,
          };
          if (editingTemplate) {
            updateMutation.mutate({ id: editingTemplate.id, data });
          } else {
            createMutation.mutate(data);
          }
        }}>
          <Form.Item name="name" label="模板名称" rules={[{ required: true }]}>
            <Input placeholder="请输入模板名称" />
          </Form.Item>
          <Form.Item name="type" label="模板类型" rules={[{ required: true }]}>
            <Select
              placeholder="选择模板类型"
              options={templateTypes}
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="模板描述" />
          </Form.Item>
          <Form.Item name="content" label="模板内容" rules={[{ required: true }]}>
            <Input.TextArea rows={8} placeholder="支持变量替换，如: {{ruleName}}, {{severity}}" />
          </Form.Item>
          <Form.Item name="variables" label="变量定义">
            <Input.TextArea
              rows={4}
              placeholder='JSON格式，如: {"ruleName": "规则名称", "severity": "严重级别"}'
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending || updateMutation.isPending}>
                保存
              </Button>
              <Button onClick={() => setIsDrawerOpen(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>

      <Modal
        title="模板预览"
        open={isPreviewOpen}
        onCancel={() => setIsPreviewOpen(false)}
        footer={null}
        width={600}
      >
        <Card>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{previewContent}</pre>
        </Card>
      </Modal>
    </div>
  );
}
