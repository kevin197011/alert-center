import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Modal, Form, Input, Select, Drawer, Avatar } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, UserOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { userApi, User } from '../../services/api';
import dayjs from 'dayjs';

const roleColors: Record<string, string> = {
  admin: 'red',
  manager: 'blue',
  user: 'green',
};

export default function UserManagement() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({ role: '', status: '' });
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [isPasswordOpen, setIsPasswordOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [currentUserId, setCurrentUserId] = useState<string>('');
  const [form] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: usersData, isLoading } = useQuery({
    queryKey: ['users', page, pageSize, filters],
    queryFn: async () => {
      const res = await userApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: User[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<User>) => userApi.create(data),
    onSuccess: () => {
      message.success('创建成功');
      queryClient.invalidateQueries({ queryKey: ['users'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: (error: any) => message.error(error.response?.data?.message || '创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<User> }) => userApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功');
      queryClient.invalidateQueries({ queryKey: ['users'] });
      setIsDrawerOpen(false);
      setEditingUser(null);
      form.resetFields();
    },
    onError: (error: any) => message.error(error.response?.data?.message || '更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => userApi.delete(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
    onError: (error: any) => message.error(error.response?.data?.message || '删除失败'),
  });

  const passwordMutation = useMutation({
    mutationFn: ({ id, oldPassword, newPassword }: { id: string; oldPassword: string; newPassword: string }) =>
      userApi.changePassword(id, oldPassword, newPassword),
    onSuccess: () => {
      message.success('密码修改成功');
      setIsPasswordOpen(false);
      passwordForm.resetFields();
    },
    onError: (error: any) => message.error(error.response?.data?.message || '密码修改失败'),
  });

  const columns = [
    {
      title: '用户',
      key: 'user',
      width: 200,
      render: (_: unknown, record: User) => (
        <Space>
          <Avatar icon={<UserOutlined />} />
          <div>
            <div style={{ fontWeight: 500 }}>{record.username}</div>
            <div style={{ fontSize: 12, color: '#999' }}>{record.email}</div>
          </div>
        </Space>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: (role: string) => (
        <Tag color={roleColors[role] || 'default'}>
          {role?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '手机号',
      dataIndex: 'phone',
      key: 'phone',
      width: 130,
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
      title: '最后登录',
      dataIndex: 'last_login_at',
      key: 'last_login_at',
      width: 180,
      render: (time: string | null) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '从未登录',
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
      width: 200,
      render: (_: unknown, record: User) => (
        <Space>
          <Button
            type="link"
            icon={<SafetyCertificateOutlined />}
            onClick={() => {
              setCurrentUserId(record.id);
              setIsPasswordOpen(true);
            }}
          >
            修改密码
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingUser(record);
              form.setFieldsValue({
                ...record,
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
                content: `确定要删除用户 "${record.username}" 吗？`,
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
        <h1 className="page-title">用户管理</h1>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingUser(null);
            form.resetFields();
            setIsDrawerOpen(true);
          }}
        >
          新建用户
        </Button>
      </div>

      <div className="filter-form">
        <Form layout="inline" onFinish={setFilters}>
          <Form.Item name="role" label="角色">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 120 }}
              options={[
                { value: 'admin', label: '管理员' },
                { value: 'manager', label: '管理员' },
                { value: 'user', label: '普通用户' },
              ]}
            />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 100 }}
              options={[
                { value: '1', label: '启用' },
                { value: '0', label: '禁用' },
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
        dataSource={Array.isArray(usersData?.data) ? usersData.data : []}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize,
          total: usersData?.total || 0,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />

      <Drawer
        title={editingUser ? '编辑用户' : '新建用户'}
        open={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false);
          setEditingUser(null);
          form.resetFields();
        }}
        width={500}
      >
        <Form form={form} layout="vertical" onFinish={(values) => {
          if (editingUser) {
            updateMutation.mutate({ id: editingUser.id, data: values });
          } else {
            createMutation.mutate(values);
          }
        }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input placeholder="请输入用户名" disabled={!!editingUser} />
          </Form.Item>
          {!editingUser && (
            <Form.Item name="password" label="密码" rules={[{ required: true, min: 6 }]}>
              <Input.Password placeholder="请输入密码" />
            </Form.Item>
          )}
          <Form.Item name="email" label="邮箱">
            <Input placeholder="请输入邮箱" />
          </Form.Item>
          <Form.Item name="phone" label="手机号">
            <Input placeholder="请输入手机号" />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'admin', label: '管理员' },
                { value: 'manager', label: '业务管理员' },
                { value: 'user', label: '普通用户' },
              ]}
            />
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
              <Button onClick={() => setIsDrawerOpen(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>

      <Modal
        title="修改密码"
        open={isPasswordOpen}
        onCancel={() => {
          setIsPasswordOpen(false);
          passwordForm.resetFields();
        }}
        onOk={() => {
          passwordForm.validateFields().then((values) => {
            passwordMutation.mutate({
              id: currentUserId,
              oldPassword: values.old_password,
              newPassword: values.new_password,
            });
          });
        }}
        confirmLoading={passwordMutation.isPending}
      >
        <Form form={passwordForm} layout="vertical">
          <Form.Item name="old_password" label="旧密码" rules={[{ required: true }]}>
            <Input.Password placeholder="请输入旧密码" />
          </Form.Item>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, min: 6 }]}>
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item name="confirm_password" label="确认密码" dependencies={['new_password']} rules={[
            { required: true },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('new_password') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('两次输入的密码不一致'));
              },
            }),
          ]}>
            <Input.Password placeholder="请确认新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
