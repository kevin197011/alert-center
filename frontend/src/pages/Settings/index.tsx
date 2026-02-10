import { useState, useEffect } from 'react';
import { Card, Form, Input, Button, Switch, message, Tabs, Table, Tag, Space, Modal, InputNumber, Select, Spin } from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '../../store/auth';
import { businessGroupApi } from '../../services/api';
import type { BusinessGroup } from '../../services/api';

export default function Settings() {
  const { user } = useAuthStore();
  const [activeTab, setActiveTab] = useState('profile');
  const [darkMode, setDarkMode] = useState(() => {
    const saved = localStorage.getItem('darkMode');
    return saved ? JSON.parse(saved) : false;
  });

  useEffect(() => {
    localStorage.setItem('darkMode', JSON.stringify(darkMode));
  }, [darkMode]);

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">系统设置</h1>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <Tabs.TabPane tab="个人设置" key="profile">
          <Card title="个人信息">
            <Form layout="vertical" initialValues={user || undefined}>
              <Form.Item name="username" label="用户名">
                <Input disabled />
              </Form.Item>
              <Form.Item name="email" label="邮箱">
                <Input />
              </Form.Item>
              <Form.Item name="phone" label="手机号">
                <Input />
              </Form.Item>
              <Form.Item>
                <Button type="primary">保存</Button>
              </Form.Item>
            </Form>
          </Card>
        </Tabs.TabPane>

        <Tabs.TabPane tab="显示设置" key="display">
          <Card title="显示配置">
            <Form layout="vertical">
              <Form.Item label="深色模式">
                <Switch
                  checked={darkMode}
                  onChange={setDarkMode}
                  checkedChildren="开启"
                  unCheckedChildren="关闭"
                />
              </Form.Item>
              <Form.Item label="紧凑模式">
                <Switch />
              </Form.Item>
              <Form.Item>
                <Button type="primary">保存</Button>
              </Form.Item>
            </Form>
          </Card>
        </Tabs.TabPane>

        <Tabs.TabPane tab="业务组管理" key="groups">
          <BusinessGroupSettings />
        </Tabs.TabPane>

        <Tabs.TabPane tab="系统配置" key="system">
          <Card title="全局配置">
            <Form layout="vertical">
              <Form.Item label="会话超时时间(分钟)">
                <InputNumber min={5} max={1440} defaultValue={60} />
              </Form.Item>
              <Form.Item label="最大登录失败次数">
                <InputNumber min={1} max={10} defaultValue={5} />
              </Form.Item>
              <Form.Item label="启用两因素认证">
                <Switch />
              </Form.Item>
              <Form.Item label="默认通知渠道">
                <Select
                  placeholder="选择默认渠道"
                  style={{ width: 200 }}
                  options={[]}
                />
              </Form.Item>
              <Form.Item>
                <Button type="primary">保存配置</Button>
              </Form.Item>
            </Form>
          </Card>
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
}

function BusinessGroupSettings() {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<BusinessGroup | null>(null);
  const [form] = Form.useForm();

  const { data: groupsData, isLoading } = useQuery({
    queryKey: ['businessGroups'],
    queryFn: async (): Promise<{ data: BusinessGroup[]; total: number }> => {
      const res = await businessGroupApi.list({ page: 1, page_size: 100 });
      const body = res.data as unknown as { data?: { data?: BusinessGroup[]; total?: number } };
      const inner = body?.data;
      const list = Array.isArray(inner?.data) ? inner.data : (Array.isArray(inner) ? inner : []);
      return { data: list, total: inner && typeof inner.total === 'number' ? inner.total : 0 };
    },
  });

  const groups = Array.isArray(groupsData?.data) ? groupsData.data : [];

  const handleSubmit = () => {
    message.success(editingGroup ? '更新成功' : '创建成功');
    setIsModalOpen(false);
    setEditingGroup(null);
    form.resetFields();
  };

  const columns = [
    { title: '组名', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: number) => (
        <Tag color={status === 1 ? 'green' : 'red'}>
          {status === 1 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: unknown, record: BusinessGroup) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => {
            setEditingGroup(record);
            form.setFieldsValue(record);
            setIsModalOpen(true);
          }}>编辑</Button>
          <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
          setEditingGroup(null);
          form.resetFields();
          setIsModalOpen(true);
        }}>
          新建业务组
        </Button>
      </div>

      <Spin spinning={isLoading}>
        <Table dataSource={groups} rowKey="id" columns={columns} />
      </Spin>

      <Modal
        title={editingGroup ? '编辑业务组' : '新建业务组'}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onOk={form.submit}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="组名" rules={[{ required: true }]}>
            <Input placeholder="请输入组名" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: 1, label: '启用' },
                { value: 0, label: '禁用' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
