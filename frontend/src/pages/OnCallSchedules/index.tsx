import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Form, Input, Drawer, Select, Typography, Popconfirm, Tooltip, Card, Tabs, List, Avatar, Row, Col, Statistic, Divider } from 'antd';
import { PlusOutlined, DeleteOutlined, ReloadOutlined, TeamOutlined, UserOutlined, CalendarOutlined, PhoneOutlined, MailOutlined, ClockCircleOutlined, SwapOutlined, UserSwitchOutlined } from '@ant-design/icons';
import { oncallApi, OnCallSchedule, OnCallAssignment } from '../../services/api';
import dayjs from 'dayjs';

const { Text, Title } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

const rotationTypes: Record<string, string> = {
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
};

export default function OnCallSchedules() {
  const [activeTab, setActiveTab] = useState('schedules');
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [, setEditingSchedule] = useState<OnCallSchedule | null>(null);
  const [selectedScheduleId, setSelectedScheduleId] = useState<string | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  function unwrapList<T>(res: { data: unknown }): { data: T[] } {
    const inner = (res.data as { data?: unknown })?.data;
    const list = Array.isArray(inner) ? inner : Array.isArray((inner as { data?: unknown })?.data) ? (inner as { data: T[] }).data : [];
    return { data: list };
  }

  const { data: schedulesData, isLoading, refetch } = useQuery({
    queryKey: ['oncall-schedules'],
    queryFn: async () => {
      const res = await oncallApi.listSchedules();
      return unwrapList<OnCallSchedule>(res);
    },
  });

  const { data: currentOnCallData } = useQuery({
    queryKey: ['oncall-current'],
    queryFn: async () => {
      const res = await oncallApi.getCurrentOnCall();
      return unwrapList<OnCallAssignment>(res);
    },
  });

  const { data: assignmentsData, refetch: refetchAssignments } = useQuery({
    queryKey: ['oncall-assignments', selectedScheduleId],
    queryFn: async () => {
      if (!selectedScheduleId) return { data: [] };
      const res = await oncallApi.getScheduleAssignments(selectedScheduleId);
      return unwrapList<OnCallAssignment>(res);
    },
    enabled: !!selectedScheduleId,
  });

  const { data: _membersData, refetch: _refetchMembers } = useQuery({
    queryKey: ['oncall-members', selectedScheduleId],
    queryFn: async () => {
      if (!selectedScheduleId) return { data: [] };
      const res = await oncallApi.getMembers(selectedScheduleId);
      return unwrapList(res);
    },
    enabled: !!selectedScheduleId,
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; description?: string; timezone?: string; rotation_type?: string }) =>
      oncallApi.createSchedule(data),
    onSuccess: () => {
      message.success('值班表创建成功');
      queryClient.invalidateQueries({ queryKey: ['oncall-schedules'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: (error: Error) => message.error(`创建失败: ${error.message}`),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => oncallApi.deleteSchedule(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['oncall-schedules'] });
    },
    onError: (error: Error) => message.error(`删除失败: ${error.message}`),
  });

  // Member mutations - available for future use
  // const addMemberMutation = useMutation({
  //   mutationFn: ({ scheduleId, data }: { scheduleId: string; data: { user_id: string; username: string; priority?: number } }) =>
  //     oncallApi.addMember(scheduleId, data),
  //   onSuccess: () => {
  //     message.success('成员添加成功');
  //     queryClient.invalidateQueries({ queryKey: ['oncall-members'] });
  //   },
  //   onError: (error: Error) => message.error(`添加失败: ${error.message}`),
  // });

  // const deleteMemberMutation = useMutation({
  //   mutationFn: ({ scheduleId, memberId }: { scheduleId: string; memberId: string }) =>
  //     oncallApi.deleteMember(scheduleId, memberId),
  //   onSuccess: () => {
  //     message.success('成员删除成功');
  //     queryClient.invalidateQueries({ queryKey: ['oncall-members'] });
  //   },
  //   onError: (error: Error) => message.error(`删除失败: ${error.message}`),
  // });

  const seedMutation = useMutation({
    mutationFn: () => oncallApi.seedSchedules(),
    onSuccess: () => {
      message.success('默认值班表已创建');
      queryClient.invalidateQueries({ queryKey: ['oncall-schedules'] });
    },
    onError: (error: Error) => message.error(`创建失败: ${error.message}`),
  });

  const escalateMutation = useMutation({
    mutationFn: (scheduleId: string) =>
      oncallApi.escalate(scheduleId, { current_user_id: 'current-user-id' }),
    onSuccess: () => {
      message.success('已升级给下一位值班人员');
      queryClient.invalidateQueries({ queryKey: ['oncall-current'] });
    },
    onError: (error: Error) => message.error(`升级失败: ${error.message}`),
  });

  const handleCreate = () => {
    setEditingSchedule(null);
    form.resetFields();
    setIsDrawerOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      await createMutation.mutateAsync({
        name: values.name,
        description: values.description,
        timezone: values.timezone || 'Asia/Shanghai',
        rotation_type: values.rotation_type || 'weekly',
      });
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  const schedules = Array.isArray(schedulesData?.data) ? schedulesData.data : [];
  const currentOnCall = Array.isArray(currentOnCallData?.data) ? currentOnCallData.data : [];
  const assignments = Array.isArray(assignmentsData?.data) ? assignmentsData.data : [];

  const scheduleColumns = [
    {
      title: '值班表名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: OnCallSchedule) => (
        <Space direction="vertical" size={0}>
          <Text strong>{text}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{record.description}</Text>
        </Space>
      ),
    },
    {
      title: '时区',
      dataIndex: 'timezone',
      key: 'timezone',
      width: 140,
      render: (tz: string) => <Tag>{tz}</Tag>,
    },
    {
      title: '轮换类型',
      dataIndex: 'rotation_type',
      key: 'rotation_type',
      width: 100,
      render: (type: string) => (
        <Tag color="blue">{rotationTypes[type] || type}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'default'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_: unknown, record: OnCallSchedule) => (
        <Space>
          <Button
            type="text"
            icon={<TeamOutlined />}
            onClick={() => {
              setSelectedScheduleId(record.id);
              setActiveTab('members');
            }}
          >
            成员
          </Button>
          <Popconfirm
            title="确定删除此值班表？"
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const assignmentColumns = [
    {
      title: '值班人员',
      dataIndex: 'username',
      key: 'username',
      render: (username: string) => (
        <Space>
          <Avatar icon={<UserOutlined />} />
          <Text strong>{username}</Text>
        </Space>
      ),
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 160,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      key: 'end_time',
      width: 160,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '联系方式',
      key: 'contact',
      render: (_: unknown, record: OnCallAssignment) => (
        <Space>
          {record.email && <Tooltip title={record.email}><Button type="text" icon={<MailOutlined />} /></Tooltip>}
          {record.phone && <Tooltip title={record.phone}><Button type="text" icon={<PhoneOutlined />} /></Tooltip>}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="值班表数量"
              value={schedules.length}
              prefix={<CalendarOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="当前值班人员"
              value={currentOnCall.length}
              valueStyle={{ color: '#1890ff' }}
              prefix={<UserSwitchOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="排班记录"
              value={assignments.length}
              prefix={<SwapOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title="值班管理"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => { refetch(); refetchAssignments(); }}>
              刷新
            </Button>
            <Button onClick={() => seedMutation.mutate()}>
              创建默认值班表
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              新建值班表
            </Button>
          </Space>
        }
      >
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={<span><TeamOutlined /> 当前值班</span>}
            key="current"
          >
            <List
              grid={{ gutter: 16, column: 2 }}
              dataSource={currentOnCall}
              renderItem={(item: OnCallAssignment) => (
                <List.Item>
                  <Card size="small">
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Space>
                          <Avatar size="large" icon={<UserOutlined />} style={{ backgroundColor: '#1890ff' }} />
                          <div>
                            <Text strong style={{ fontSize: 16 }}>{item.username}</Text>
                            <br />
                            <Text type="secondary" style={{ fontSize: 12 }}>值班中</Text>
                          </div>
                        </Space>
                        <Button
                          icon={<SwapOutlined />}
                          onClick={() => escalateMutation.mutate(item.schedule_id)}
                        >
                          升级
                        </Button>
                      </div>
                      <Divider style={{ margin: '12px 0' }} />
                      <Space>
                        {item.email && <Text><MailOutlined /> {item.email}</Text>}
                        {item.phone && <Text><PhoneOutlined /> {item.phone}</Text>}
                      </Space>
                      <Text type="secondary">
                        <ClockCircleOutlined /> 值班至: {dayjs(item.end_time).format('MM-DD HH:mm')}
                      </Text>
                    </Space>
                  </Card>
                </List.Item>
              )}
            />
            {currentOnCall.length === 0 && (
              <div style={{ textAlign: 'center', padding: 40 }}>
                <Text type="secondary">当前暂无值班人员</Text>
              </div>
            )}
          </TabPane>

          <TabPane
            tab={<span><CalendarOutlined /> 值班表</span>}
            key="schedules"
          >
            <Table
              columns={scheduleColumns}
              dataSource={schedules}
              rowKey="id"
              loading={isLoading}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane
            tab={<span><SwapOutlined /> 排班记录</span>}
            key="members"
          >
            <Row gutter={16}>
              <Col span={6}>
                <Select
                  style={{ width: '100%', marginBottom: 16 }}
                  placeholder="选择值班表查看排班"
                  value={selectedScheduleId}
                  onChange={setSelectedScheduleId}
                >
                  {schedules.map((s: OnCallSchedule) => (
                    <Option key={s.id} value={s.id}>{s.name}</Option>
                  ))}
                </Select>
              </Col>
              <Col span={18}>
                <Table
                  columns={assignmentColumns}
                  dataSource={assignments}
                  rowKey="id"
                  pagination={{ pageSize: 10 }}
                />
              </Col>
            </Row>
            {!selectedScheduleId && (
              <div style={{ textAlign: 'center', padding: 40 }}>
                <Text type="secondary">请选择值班表查看排班记录</Text>
              </div>
            )}
          </TabPane>
        </Tabs>
      </Card>

      <Drawer
        title="新建值班表"
        width={480}
        open={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => setIsDrawerOpen(false)} style={{ marginRight: 8 }}>
              取消
            </Button>
            <Button type="primary" onClick={handleSubmit} loading={createMutation.isPending}>
              创建
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="值班表名称"
            rules={[{ required: true, message: '请输入值班表名称' }]}
          >
            <Input placeholder="例如: SRE值班表" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={3} placeholder="值班表描述" />
          </Form.Item>

          <Form.Item
            name="timezone"
            label="时区"
            initialValue="Asia/Shanghai"
          >
            <Select>
              <Option value="Asia/Shanghai">Asia/Shanghai (UTC+8)</Option>
              <Option value="UTC">UTC</Option>
              <Option value="America/New_York">America/New_York (UTC-5)</Option>
              <Option value="Europe/London">Europe/London (UTC+0)</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="rotation_type"
            label="轮换类型"
            initialValue="weekly"
          >
            <Select>
              <Option value="daily">每日轮换</Option>
              <Option value="weekly">每周轮换</Option>
              <Option value="monthly">每月轮换</Option>
            </Select>
          </Form.Item>

          <Card size="small" style={{ marginTop: 16, backgroundColor: '#fafafa' }}>
            <Title level={5}>
              <TeamOutlined /> 使用说明
            </Title>
            <ul style={{ margin: 0, paddingLeft: 20 }}>
              <li>创建值班表后，需要添加值班成员</li>
              <li>系统会根据轮换类型自动生成排班</li>
              <li>当前值班人员会收到告警通知</li>
            </ul>
          </Card>
        </Form>
      </Drawer>
    </div>
  );
}
