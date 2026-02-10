import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import AntdTable from 'antd/lib/table';
import AntdButton from 'antd/lib/button';
import AntdSpace from 'antd/lib/space';
import AntdTag from 'antd/lib/tag';
import AntdCard from 'antd/lib/card';
import AntdRow from 'antd/lib/row';
import AntdCol from 'antd/lib/col';
import AntdStatistic from 'antd/lib/statistic';
import AntdModal from 'antd/lib/modal';
import AntdForm from 'antd/lib/form';
import AntdInput from 'antd/lib/input';
import AntdSelect from 'antd/lib/select';
import AntdTypography from 'antd/lib/typography';
import AntdMessage from 'antd/lib/message';
import AntdDrawer from 'antd/lib/drawer';
import AntdDescriptions from 'antd/lib/descriptions';
import AntdTooltip from 'antd/lib/tooltip';
import AntdPopconfirm from 'antd/lib/popconfirm';
import { PlusOutlined, EditOutlined, ReloadOutlined, FileTextOutlined, CheckOutlined, CloseOutlined, UserOutlined } from '@ant-design/icons';
import { ticketApi, Ticket } from '../../services/api';
import dayjs from 'dayjs';

const { Text } = AntdTypography;
const { TextArea } = AntdInput;
const { Option } = AntdSelect;

const priorityColors: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'default',
};

const statusColors: Record<string, string> = {
  open: 'red',
  in_progress: 'blue',
  resolved: 'green',
  closed: 'default',
};

const statusLabels: Record<string, string> = {
  open: '待处理',
  in_progress: '处理中',
  resolved: '已解决',
  closed: '已关闭',
};

export default function TicketManagement() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const [editingTicket, setEditingTicket] = useState<Ticket | null>(null);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [form] = AntdForm.useForm();
  const queryClient = useQueryClient();

  const { data: ticketsResponse, isLoading, refetch } = useQuery({
    queryKey: ['tickets', page, pageSize],
    queryFn: async () => {
      const res = await ticketApi.list({ page, page_size: pageSize });
      const body = res.data as unknown as { data?: { data: Ticket[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<Ticket>) =>
      ticketApi.create(data as any),
    onSuccess: () => {
      AntdMessage.success('工单创建成功');
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      setIsModalOpen(false);
      form.resetFields();
    },
    onError: (error: Error) => AntdMessage.error(`创建失败: ${error.message}`),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Ticket> }) =>
      ticketApi.update(id, data),
    onSuccess: () => {
      AntdMessage.success('工单更新成功');
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      setIsModalOpen(false);
      setEditingTicket(null);
    },
    onError: (error: Error) => AntdMessage.error(`更新失败: ${error.message}`),
  });

  const resolveMutation = useMutation({
    mutationFn: (id: string) =>
      ticketApi.resolve(id),
    onSuccess: () => {
      AntdMessage.success('工单已标记为已解决');
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: Error) => AntdMessage.error(`操作失败: ${error.message}`),
  });

  const closeMutation = useMutation({
    mutationFn: (id: string) =>
      ticketApi.close(id),
    onSuccess: () => {
      AntdMessage.success('工单已关闭');
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: Error) => AntdMessage.error(`操作失败: ${error.message}`),
  });

  const handleCreate = () => {
    setEditingTicket(null);
    form.resetFields();
    setIsModalOpen(true);
  };

  const handleEdit = (record: Ticket) => {
    setEditingTicket(record);
    form.setFieldsValue({
      title: record.title,
      description: record.description,
      priority: record.priority,
      status: record.status,
      assignee_name: record.assignee_name,
    });
    setIsModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingTicket) {
        await updateMutation.mutateAsync({ id: editingTicket.id, data: values });
      } else {
        await createMutation.mutateAsync({
          ...values,
          status: 'open',
        });
      }
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  const columns = [
    {
      title: '工单ID',
      dataIndex: 'id',
      key: 'id',
      width: 140,
      render: (id: string) => <Text code>{id.slice(0, 12)}</Text>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (title: string, record: Ticket) => (
        <AntdButton type="link" onClick={() => {
          setSelectedTicket(record);
          setIsDetailOpen(true);
        }}>
          {title}
        </AntdButton>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => (
        <AntdTag color={priorityColors[priority] || 'default'}>
          {priority?.toUpperCase()}
        </AntdTag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <AntdTag color={statusColors[status] || 'default'}>
          {statusLabels[status] || status}
        </AntdTag>
      ),
    },
    {
      title: '负责人',
      dataIndex: 'assignee_name',
      key: 'assignee_name',
      width: 120,
      render: (name: string) => name || <Text type="secondary">未分配</Text>,
    },
    {
      title: '创建人',
      dataIndex: 'creator_name',
      key: 'creator_name',
      width: 120,
      render: (name: string) => (
        <AntdSpace>
          <UserOutlined />
          <Text>{name}</Text>
        </AntdSpace>
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
      width: 200,
      render: (_: unknown, record: Ticket) => (
        <AntdSpace>
          <AntdTooltip title="查看详情">
            <AntdButton
              type="text"
              icon={<FileTextOutlined />}
              onClick={() => {
                setSelectedTicket(record);
                setIsDetailOpen(true);
              }}
            />
          </AntdTooltip>
          <AntdTooltip title="编辑">
            <AntdButton
              type="text"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </AntdTooltip>
          {record.status === 'open' && (
            <AntdTooltip title="标记为解决">
              <AntdButton
                type="text"
                icon={<CheckOutlined />}
                onClick={() => resolveMutation.mutate(record.id)}
              />
            </AntdTooltip>
          )}
          {record.status === 'resolved' && (
            <AntdTooltip title="关闭工单">
              <AntdPopconfirm
                title="确定关闭此工单？"
                onConfirm={() => closeMutation.mutate(record.id)}
              >
                <AntdButton type="text" danger icon={<CloseOutlined />} />
              </AntdPopconfirm>
            </AntdTooltip>
          )}
        </AntdSpace>
      ),
    },
  ];

  const tickets = Array.isArray(ticketsResponse?.data) ? ticketsResponse.data : [];
  const total = ticketsResponse?.total || 0;

  return (
    <div style={{ padding: 24 }}>
      <AntdRow gutter={16} style={{ marginBottom: 24 }}>
        <AntdCol span={6}>
          <AntdCard>
            <AntdStatistic
              title="待处理工单"
              value={tickets.filter((t: Ticket) => t.status === 'open').length}
              valueStyle={{ color: '#cf1322' }}
              prefix={<FileTextOutlined />}
            />
          </AntdCard>
        </AntdCol>
        <AntdCol span={6}>
          <AntdCard>
            <AntdStatistic
              title="处理中工单"
              value={tickets.filter((t: Ticket) => t.status === 'in_progress').length}
              valueStyle={{ color: '#1890ff' }}
              prefix={<FileTextOutlined />}
            />
          </AntdCard>
        </AntdCol>
        <AntdCol span={6}>
          <AntdCard>
            <AntdStatistic
              title="已解决工单"
              value={tickets.filter((t: Ticket) => t.status === 'resolved').length}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckOutlined />}
            />
          </AntdCard>
        </AntdCol>
        <AntdCol span={6}>
          <AntdCard>
            <AntdStatistic
              title="总工单数"
              value={total}
              prefix={<FileTextOutlined />}
            />
          </AntdCard>
        </AntdCol>
      </AntdRow>

      <AntdCard
        title={
          <AntdSpace>
            <FileTextOutlined />
            <span>工单管理</span>
          </AntdSpace>
        }
        extra={
          <AntdSpace>
            <AntdButton icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </AntdButton>
            <AntdButton type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建工单
            </AntdButton>
          </AntdSpace>
        }
      >
        <AntdTable
          columns={columns}
          dataSource={tickets}
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
      </AntdCard>

      <AntdModal
        title={editingTicket ? '编辑工单' : '创建工单'}
        open={isModalOpen}
        onOk={handleSubmit}
        onCancel={() => {
          setIsModalOpen(false);
          setEditingTicket(null);
          form.resetFields();
        }}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <AntdForm form={form} layout="vertical">
          <AntdForm.Item
            name="title"
            label="标题"
            rules={[{ required: true, message: '请输入标题' }]}
          >
            <AntdInput placeholder="请输入工单标题" />
          </AntdForm.Item>

          <AntdForm.Item
            name="description"
            label="描述"
            rules={[{ required: true, message: '请输入描述' }]}
          >
            <TextArea rows={4} placeholder="请输入工单描述" />
          </AntdForm.Item>

          <AntdForm.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <AntdSelect placeholder="选择优先级">
              <Option value="critical">紧急</Option>
              <Option value="high">高</Option>
              <Option value="medium">中</Option>
              <Option value="low">低</Option>
            </AntdSelect>
          </AntdForm.Item>

          {editingTicket && (
            <AntdForm.Item name="status" label="状态">
              <AntdSelect placeholder="选择状态">
                <Option value="open">待处理</Option>
                <Option value="in_progress">处理中</Option>
                <Option value="resolved">已解决</Option>
                <Option value="closed">已关闭</Option>
              </AntdSelect>
            </AntdForm.Item>
          )}

          <AntdForm.Item name="assignee_name" label="负责人">
            <AntdInput placeholder="输入负责人用户名" />
          </AntdForm.Item>
        </AntdForm>
      </AntdModal>

      <AntdDrawer
        title="工单详情"
        width={640}
        open={isDetailOpen}
        onClose={() => {
          setIsDetailOpen(false);
          setSelectedTicket(null);
        }}
        extra={
          <AntdSpace>
            {selectedTicket?.status === 'open' && (
              <AntdButton type="primary" onClick={() => {
                handleEdit(selectedTicket);
                setIsDetailOpen(false);
              }}>
                编辑
              </AntdButton>
            )}
          </AntdSpace>
        }
      >
        {selectedTicket && (
          <AntdDescriptions column={1} bordered size="small">
            <AntdDescriptions.Item label="工单ID">
              <Text code>{selectedTicket.id}</Text>
            </AntdDescriptions.Item>
            <AntdDescriptions.Item label="标题">{selectedTicket.title}</AntdDescriptions.Item>
            <AntdDescriptions.Item label="描述">{selectedTicket.description}</AntdDescriptions.Item>
            <AntdDescriptions.Item label="优先级">
              <AntdTag color={priorityColors[selectedTicket.priority]}>
                {selectedTicket.priority?.toUpperCase()}
              </AntdTag>
            </AntdDescriptions.Item>
            <AntdDescriptions.Item label="状态">
              <AntdTag color={statusColors[selectedTicket.status]}>
                {statusLabels[selectedTicket.status]}
              </AntdTag>
            </AntdDescriptions.Item>
            <AntdDescriptions.Item label="负责人">
              {selectedTicket.assignee_name || '未分配'}
            </AntdDescriptions.Item>
            <AntdDescriptions.Item label="创建人">{selectedTicket.creator_name}</AntdDescriptions.Item>
            <AntdDescriptions.Item label="创建时间">
              {dayjs(selectedTicket.created_at).format('YYYY-MM-DD HH:mm:ss')}
            </AntdDescriptions.Item>
            <AntdDescriptions.Item label="更新时间">
              {dayjs(selectedTicket.updated_at).format('YYYY-MM-DD HH:mm:ss')}
            </AntdDescriptions.Item>
            {selectedTicket.resolved_at && (
              <AntdDescriptions.Item label="解决时间">
                {dayjs(selectedTicket.resolved_at).format('YYYY-MM-DD HH:mm:ss')}
              </AntdDescriptions.Item>
            )}
          </AntdDescriptions>
        )}
      </AntdDrawer>
    </div>
  );
}
