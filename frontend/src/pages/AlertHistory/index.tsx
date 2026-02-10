import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Tag, Space, DatePicker, Select, Button, Form, Input, message, Drawer, Tooltip } from 'antd';
import { DownloadOutlined, StopOutlined } from '@ant-design/icons';
import { alertHistoryApi } from '../../services/api';
import type { AlertHistory } from '../../services/api';
import { silenceApi } from '../../services/api';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const statusColors: Record<string, string> = {
  firing: 'red',
  resolved: 'green',
};

export default function AlertHistory() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({
    rule_id: '',
    status: '',
    start_time: '',
    end_time: '',
  });
  const [isSilenceDrawerOpen, setIsSilenceDrawerOpen] = useState(false);
  const [selectedAlert, setSelectedAlert] = useState<AlertHistory | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: historyData, isLoading } = useQuery({
    queryKey: ['alertHistory', page, pageSize, filters],
    queryFn: async () => {
      const res = await alertHistoryApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: AlertHistory[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: pageSize };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createSilenceMutation = useMutation({
    mutationFn: (data: { name: string; description?: string; matchers: Record<string, string>[]; start_time: string; end_time: string }) =>
      silenceApi.create(data),
    onSuccess: () => {
      message.success('静默规则创建成功');
      setIsSilenceDrawerOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['silences'] });
    },
    onError: () => message.error('创建失败'),
  });

  const columns = [
    {
      title: '告警编号',
      dataIndex: 'alert_no',
      key: 'alert_no',
      width: 220,
      ellipsis: true,
      render: (alertNo: string) => alertNo || '-',
    },
    {
      title: '规则ID',
      dataIndex: 'rule_id',
      key: 'rule_id',
      width: 200,
      render: (id: string) => <a href={`/rules`}>{id?.slice(0, 8)}...</a>,
    },
    {
      title: '指纹',
      dataIndex: 'fingerprint',
      key: 'fingerprint',
      width: 200,
      ellipsis: true,
    },
    {
      title: '严重级别',
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
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status === 'firing' ? '进行中' : '已恢复'}
        </Tag>
      ),
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '结束时间',
      dataIndex: 'ended_at',
      key: 'ended_at',
      width: 180,
      render: (time: string | null) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_: unknown, record: AlertHistory) => (
        <Space>
          <Tooltip title="快速静默此告警">
          <Button
            type="link"
            icon={<StopOutlined />}
            onClick={() => {
                setSelectedAlert(record);
                const labels = typeof record.labels === 'string' ? JSON.parse(record.labels) : record.labels;
                const matchers = Object.entries(labels).map(([key, value]) => ({ [key]: value }));
                form.setFieldsValue({
                  name: `静默-${record.alert_no || record.rule_id?.slice(0, 8)}`,
                  description: `静默此告警 (${dayjs(record.started_at).format('YYYY-MM-DD HH:mm')})`,
                  matchers,
                  start_time: dayjs(),
                  end_time: dayjs().add(2, 'hour'),
                });
                setIsSilenceDrawerOpen(true);
              }}
            >
              静默
            </Button>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">告警历史</h1>
        <Space>
          <Button icon={<DownloadOutlined />}>
            导出报表
          </Button>
        </Space>
      </div>

      <div className="filter-form">
        <Space wrap>
          <RangePicker onChange={(dates) => {
            if (dates && dates[0] && dates[1]) {
              setFilters({
                ...filters,
                start_time: dates[0].format('YYYY-MM-DD'),
                end_time: dates[1].format('YYYY-MM-DD'),
              });
            }
          }} />
          <Select
            placeholder="告警状态"
            allowClear
            style={{ width: 120 }}
            options={[
              { value: 'firing', label: '进行中' },
              { value: 'resolved', label: '已恢复' },
            ]}
            onChange={(value) => setFilters({ ...filters, status: value || '' })}
          />
          <Button type="primary" onClick={() => {
            // Export functionality
          }}>
            导出
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={Array.isArray(historyData?.data) ? historyData.data : []}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize,
          total: historyData?.total || 0,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />

      <Drawer
        title="快速创建静默规则"
        open={isSilenceDrawerOpen}
        onClose={() => {
          setIsSilenceDrawerOpen(false);
          form.resetFields();
        }}
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            const matchers = values.matchers?.map((m: Record<string, string>) => {
              const [key, value] = Object.entries(m)[0];
              return { [key]: value };
            }) || [];
            createSilenceMutation.mutate({
              name: values.name,
              description: values.description,
              matchers,
              start_time: values.start_time.toISOString(),
              end_time: values.end_time.toISOString(),
            });
          }}
        >
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input placeholder="请输入规则名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="规则描述" />
          </Form.Item>
          <Form.Item label="匹配标签（自动从告警中提取）">
            <Input.TextArea
              rows={4}
              value={selectedAlert ? (() => {
                const labels = typeof selectedAlert.labels === 'string' 
                  ? JSON.parse(selectedAlert.labels) 
                  : selectedAlert.labels;
                return JSON.stringify(labels, null, 2);
              })() : ''}
              disabled
              style={{ background: '#f5f5f5' }}
            />
          </Form.Item>
          <Form.Item name="start_time" label="开始时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="end_time" label="结束时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createSilenceMutation.isPending}>
                创建静默
              </Button>
              <Button onClick={() => setIsSilenceDrawerOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  );
}
