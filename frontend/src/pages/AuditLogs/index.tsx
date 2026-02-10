import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Table, Tag, DatePicker, Button, Space, Card, Select } from 'antd';
import { DownloadOutlined, FilterOutlined } from '@ant-design/icons';
import { auditLogApi, type AuditLog } from '../../services/api';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

const actionColors: Record<string, string> = {
  create: 'green',
  update: 'blue',
  delete: 'red',
  login: 'cyan',
  logout: 'default',
  bind: 'purple',
  unbind: 'orange',
  export: 'geekblue',
};

const actionNames: Record<string, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  login: '登录',
  logout: '登出',
  bind: '绑定',
  unbind: '解绑',
  export: '导出',
};

const resourceNames: Record<string, string> = {
  user: '用户',
  user_group: '用户组',
  alert_rule: '告警规则',
  alert_channel: '告警渠道',
  alert_template: '告警模板',
  alert_history: '告警历史',
  binding: '渠道绑定',
};

export default function AuditLogs() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filters, setFilters] = useState({
    action: '',
    resource: '',
    start_time: '',
    end_time: '',
  });

  const { data: logsData, isLoading, refetch } = useQuery({
    queryKey: ['auditLogs', page, pageSize, filters],
    queryFn: async () => {
      const res = await auditLogApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: AuditLog[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const handleFilter = (values: any) => {
    const filters: any = {};
    if (values.action) filters.action = values.action;
    if (values.resource) filters.resource = values.resource;
    if (values.dateRange) {
      filters.start_time = values.dateRange[0].format('YYYY-MM-DD');
      filters.end_time = values.dateRange[1].format('YYYY-MM-DD');
    }
    setFilters(filters);
  };

  const handleExport = async () => {
    try {
      const res = await auditLogApi.export(filters);
      const blob = new Blob([res.data], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `audit_logs_${dayjs().format('YYYYMMDDHHmmss')}.json`;
      link.click();
    } catch {
      console.error('导出失败');
    }
  };

  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作用户',
      dataIndex: 'user_id',
      key: 'user_id',
      width: 180,
      render: (id: string) => <span style={{ fontFamily: 'monospace' }}>{id.slice(0, 8)}...</span>,
    },
    {
      title: '操作类型',
      dataIndex: 'action',
      key: 'action',
      width: 100,
      render: (action: string) => (
        <Tag color={actionColors[action] || 'default'}>
          {actionNames[action] || action}
        </Tag>
      ),
    },
    {
      title: '资源类型',
      dataIndex: 'resource',
      key: 'resource',
      width: 120,
      render: (resource: string) => (
        <Tag>{resourceNames[resource] || resource}</Tag>
      ),
    },
    {
      title: '资源ID',
      dataIndex: 'resource_id',
      key: 'resource_id',
      width: 200,
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {id?.slice(0, 16) || '-'}
        </span>
      ),
    },
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      width: 140,
    },
    {
      title: '详情',
      dataIndex: 'detail',
      key: 'detail',
      ellipsis: true,
      render: (detail: string) => {
        try {
          const parsed = JSON.parse(detail);
          return (
            <span style={{ fontSize: 12 }}>
              {Object.entries(parsed).slice(0, 3).map(([k, v]) => (
                `${k}: ${v}`
              )).join(', ')}
            </span>
          );
        } catch {
          return detail || '-';
        }
      },
    },
  ];

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">审计日志</h1>
        <Space>
          <Button icon={<DownloadOutlined />} onClick={handleExport}>
            导出
          </Button>
        </Space>
      </div>

      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="操作类型"
            allowClear
            style={{ width: 120 }}
            options={Object.entries(actionNames).map(([value, label]) => ({
              value,
              label,
            }))}
            onChange={(value) => handleFilter({ ...filters, action: value || '' })}
          />
          <Select
            placeholder="资源类型"
            allowClear
            style={{ width: 140 }}
            options={Object.entries(resourceNames).map(([value, label]) => ({
              value,
              label,
            }))}
            onChange={(value) => handleFilter({ ...filters, resource: value || '' })}
          />
          <RangePicker
            onChange={(_dates, dateStrings) => {
              handleFilter({
                ...filters,
                start_time: dateStrings[0] || '',
                end_time: dateStrings[1] || '',
              });
            }}
          />
          <Button
            icon={<FilterOutlined />}
            onClick={() => {
              setFilters({ action: '', resource: '', start_time: '', end_time: '' });
              refetch();
            }}
          >
            重置
          </Button>
        </Space>
      </Card>

      <Table
        columns={columns}
        dataSource={Array.isArray(logsData?.data) ? logsData.data : []}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize,
          total: logsData?.total || 0,
          showSizeChanger: true,
          showQuickJumper: true,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />
    </div>
  );
}
