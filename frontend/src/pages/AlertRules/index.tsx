import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Modal, Form, Input, Select, InputNumber, Drawer, Checkbox, Upload, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ExportOutlined, ImportOutlined, InboxOutlined } from '@ant-design/icons';
import { alertRuleApi, alertChannelApi, bindingApi, businessGroupApi, batchApi, dataSourceApi, templateApi, AlertRule, AlertChannel, type AlertChannelBinding, type BusinessGroup, type DataSource, type ExclusionWindow } from '../../services/api';
import dayjs from 'dayjs';

const { Text } = Typography;
const { Dragger } = Upload;

function ExpressionInputWithTest({
  value,
  onChange,
  onTest,
}: {
  value?: string;
  onChange?: (v: string) => void;
  onTest: () => void | Promise<void>;
}) {
  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
      <Input.TextArea
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        rows={3}
        placeholder="例如: rate(http_requests_total[5m]) > 0.1"
        style={{ flex: 1, minWidth: 0 }}
      />
      <Button
        type="primary"
        ghost
        size="middle"
        style={{ flexShrink: 0, marginTop: 4 }}
        onClick={onTest}
      >
        测试
      </Button>
    </div>
  );
}

const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

export default function AlertRules() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({ group_id: '', severity: '', status: '' });
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [isBindDrawerOpen, setIsBindDrawerOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const [currentRuleId, setCurrentRuleId] = useState<string>('');
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: rulesData, isLoading } = useQuery({
    queryKey: ['alertRules', page, pageSize, filters],
    queryFn: async () => {
      const res = await alertRuleApi.list({ page, page_size: pageSize, ...filters });
      const body = res.data as unknown as { data?: { data: AlertRule[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const { data: groupsData, isLoading: groupsLoading } = useQuery({
    queryKey: ['businessGroups'],
    queryFn: async (): Promise<{ data: BusinessGroup[]; total: number }> => {
      const res = await businessGroupApi.list({ page: 1, page_size: 100 });
      const body = res.data as unknown as { data?: { data?: BusinessGroup[]; total?: number } };
      const inner = body?.data;
      const list = Array.isArray(inner?.data) ? inner.data : (Array.isArray(inner) ? inner : []);
      return { data: list, total: inner && typeof inner.total === 'number' ? inner.total : 0 };
    },
  });

  const { data: channelsData, isLoading: channelsLoading } = useQuery({
    queryKey: ['channels'],
    queryFn: async () => {
      const res = await alertChannelApi.list({ page: 1, page_size: 100, status: 'enabled' });
      const body = res.data as unknown as { data?: { data: AlertChannel[]; total: number } };
      return body?.data ?? { data: [], total: 0 };
    },
  });

  const { data: dataSourcesData, isLoading: dataSourcesLoading } = useQuery({
    queryKey: ['dataSources'],
    queryFn: async (): Promise<{ data: DataSource[]; total: number }> => {
      const res = await dataSourceApi.list({ page: 1, page_size: 200 });
      const body = res.data as unknown as { data?: { data?: DataSource[]; total?: number } };
      const inner = body?.data;
      const list = Array.isArray(inner?.data) ? inner.data : (Array.isArray(inner) ? inner : []);
      return { data: list, total: inner && typeof inner.total === 'number' ? inner.total : 0 };
    },
  });

  const { data: templatesData } = useQuery({
    queryKey: ['templates', 'all'],
    queryFn: async () => {
      const res = await templateApi.list({ page: 1, page_size: 200 });
      const body = res.data as unknown as { data?: { data?: { id: string; name: string }[]; total?: number } };
      const inner = body?.data;
      const list = Array.isArray(inner?.data) ? inner.data : [];
      return { data: list, total: inner && typeof (inner as { total?: number }).total === 'number' ? (inner as { total: number }).total : 0 };
    },
  });

  const { data: boundChannels, refetch: refetchBindings } = useQuery({
    queryKey: ['bindings', currentRuleId],
    queryFn: async () => {
      if (!currentRuleId) return [];
      const res = await bindingApi.getByRule(currentRuleId);
      const body = res.data as unknown as { data?: AlertChannelBinding[] };
      return Array.isArray(body?.data) ? body.data : [];
    },
    enabled: !!currentRuleId,
  });

  // When editing, sync bound channels into form (API returns channel objects with id)
  useEffect(() => {
    if (!isDrawerOpen || !editingRule || currentRuleId !== editingRule.id) return;
    if (boundChannels && Array.isArray(boundChannels)) {
      form.setFieldValue('channel_ids', boundChannels.map((c) => c.id));
    }
  }, [isDrawerOpen, editingRule?.id, currentRuleId, boundChannels, form]);

  const createMutation = useMutation({
    mutationFn: (data: Partial<AlertRule>) => alertRuleApi.create(data),
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<AlertRule> }) => alertRuleApi.update(id, data),
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => alertRuleApi.delete(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['alertRules'] });
    },
    onError: () => message.error('删除失败'),
  });

  const bindMutation = useMutation({
    mutationFn: ({ ruleId, channelIds }: { ruleId: string; channelIds: string[] }) =>
      bindingApi.bind(ruleId, channelIds),
    onSuccess: () => {
      setIsBindDrawerOpen(false);
      refetchBindings();
    },
    onError: () => message.error('渠道绑定失败'),
  });

  const columns = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      width: 160,
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
      title: '表达式',
      dataIndex: 'expression',
      key: 'expression',
      width: 240,
      ellipsis: true,
    },
    {
      title: '持续时间',
      dataIndex: 'for_duration',
      key: 'for_duration',
      width: 100,
      render: (v: number) => `${v}s`,
    },
    {
      title: '告警渠道',
      key: 'bound_channels',
      width: 160,
      ellipsis: true,
      render: (_: unknown, record: AlertRule) => {
        const channels = record.bound_channels;
        if (!Array.isArray(channels) || channels.length === 0) {
          return <Text type="secondary">—</Text>;
        }
        return (
          <span title={channels.map((c) => `${c.name} (${c.type})`).join('、')}>
            {channels.map((c) => c.name).join('、')}
          </span>
        );
      },
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
      width: 200,
      render: (_: unknown, record: AlertRule) => (
        <Space>
          <Button
            type="link"
            size="small"
            onClick={() => {
              const next = record.status === 1 ? 0 : 1;
              updateMutation.mutate(
                { id: record.id, data: { status: next } },
                {
                  onSuccess: () => {
                    queryClient.invalidateQueries({ queryKey: ['alertRules'] });
                    message.success(next === 1 ? '已启用' : '已禁用');
                  },
                  onError: () => message.error('切换失败'),
                }
              );
            }}
          >
            {record.status === 1 ? '禁用' : '启用'}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingRule(record);
              setCurrentRuleId(record.id);
              const dsList = Array.isArray(dataSourcesData?.data) ? dataSourcesData.data : [];
              const matchingDs = dsList.find(
                (ds) => ds.endpoint === record.data_source_url && ds.type === record.data_source_type
              );
              let exclusionList: ExclusionWindow[] = [];
              if (record.exclusion_windows != null) {
                if (Array.isArray(record.exclusion_windows)) {
                  exclusionList = record.exclusion_windows;
                } else if (typeof record.exclusion_windows === 'string') {
                  try {
                    exclusionList = JSON.parse(record.exclusion_windows) ?? [];
                  } catch {
                    exclusionList = [];
                  }
                }
              }
              form.setFieldsValue({
                ...record,
                labels: record.labels,
                annotations: record.annotations,
                data_source_id: matchingDs?.id ?? undefined,
                channel_ids: Array.isArray(record.bound_channels) && record.bound_channels.length
                  ? record.bound_channels.map((c) => c.id)
                  : [],
                effective_start_time: record.effective_start_time ?? '00:00',
                effective_end_time: record.effective_end_time ?? '23:59',
                exclusion_windows: exclusionList.length > 0 ? exclusionList : undefined,
                status: record.status ?? 1,
                template_id: record.template_id ?? undefined,
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
                content: `确定要删除规则 "${record.name}" 吗？`,
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

  const handleExport = async () => {
    try {
      const exportParams: { group_id?: string; severity?: string; status?: string } = {};
      if (filters.group_id) exportParams.group_id = filters.group_id;
      if (filters.severity) exportParams.severity = filters.severity;
      if (filters.status) exportParams.status = filters.status;
      const res = await batchApi.exportRules(exportParams);
      const blob = new Blob([res.data], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `alert_rules_${dayjs().format('YYYYMMDDHHmmss')}.json`;
      link.click();
      message.success('导出成功');
    } catch {
      message.error('导出失败');
    }
  };

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [isImportModalOpen, setIsImportModalOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<{ success: number; failed: number; errors: string[] } | null>(null);

  const importMutation = useMutation({
    mutationFn: async (file: File) => {
      const text = await file.text();
      const rules = JSON.parse(text);
      return batchApi.importRules(rules);
    },
    onSuccess: (res) => {
      setImportResult(res.data);
      message.success(`导入完成: 成功 ${res.data.success} 条, 失败 ${res.data.failed} 条`);
      queryClient.invalidateQueries({ queryKey: ['alertRules'] });
    },
    onError: () => {
      message.error('导入失败');
    },
  });

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">告警规则</h1>
        <Space>
          <Button icon={<ImportOutlined />} onClick={() => setIsImportModalOpen(true)}>
            导入
          </Button>
          <Button icon={<ExportOutlined />} onClick={handleExport}>
            导出
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingRule(null);
              setCurrentRuleId('');
              form.resetFields();
              setIsDrawerOpen(true);
            }}
          >
            新建规则
          </Button>
        </Space>
      </div>

      <div className="filter-form">
        <Form layout="inline" onFinish={setFilters}>
          <Form.Item name="group_id" label="业务组">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 200 }}
              loading={groupsLoading}
              options={(Array.isArray(groupsData?.data) ? groupsData.data : []).map((g) => ({
                value: g.id,
                label: g.name || g.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="severity" label="级别">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 120 }}
              options={[
                { value: 'critical', label: '严重' },
                { value: 'warning', label: '警告' },
                { value: 'info', label: '信息' },
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

      <div style={{ overflow: 'auto' }}>
        <Table
          columns={columns}
          dataSource={Array.isArray(rulesData?.data) ? rulesData.data : []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 1280 }}
          pagination={{
            current: page,
            pageSize,
            total: rulesData?.total || 0,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </div>

      <Drawer
        title={editingRule ? '编辑告警规则' : '新建告警规则'}
        open={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false);
          setEditingRule(null);
          setCurrentRuleId('');
          form.resetFields();
        }}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ effective_start_time: '00:00', effective_end_time: '23:59', evaluation_interval_seconds: 60, status: 1 }}
          onFinish={async (values) => {
          const { data_source_id, channel_ids = [], exclusion_windows, template_id, ...rest } = values;
          const data = {
            ...rest,
            template_id: template_id ? template_id : (editingRule ? null : undefined),
            status: rest.status !== undefined && rest.status !== null ? Number(rest.status) : 1,
            evaluation_interval_seconds: rest.evaluation_interval_seconds != null && rest.evaluation_interval_seconds >= 1 ? rest.evaluation_interval_seconds : 60,
            effective_start_time: rest.effective_start_time && rest.effective_start_time.trim() ? rest.effective_start_time.trim() : '00:00',
            effective_end_time: rest.effective_end_time && rest.effective_end_time.trim() ? rest.effective_end_time.trim() : '23:59',
            exclusion_windows: Array.isArray(exclusion_windows) ? exclusion_windows.filter((w: ExclusionWindow) => w && (w.start || w.end)) : [],
            labels: typeof rest.labels === 'object' ? JSON.stringify(rest.labels || {}) : rest.labels,
            annotations: typeof rest.annotations === 'object' ? JSON.stringify(rest.annotations || {}) : rest.annotations,
          };
          const channelIdList = Array.isArray(channel_ids) ? channel_ids : [];
          try {
            if (editingRule) {
              await updateMutation.mutateAsync({ id: editingRule.id, data });
              await bindMutation.mutateAsync({ ruleId: editingRule.id, channelIds: channelIdList });
              message.success('更新成功');
              queryClient.invalidateQueries({ queryKey: ['alertRules'] });
              queryClient.invalidateQueries({ queryKey: ['bindings', editingRule.id] });
              setIsDrawerOpen(false);
              setEditingRule(null);
              form.resetFields();
            } else {
              const res = await createMutation.mutateAsync(data);
              const newId = (res?.data as { data?: { id?: string } })?.data?.id ?? (res?.data as { id?: string })?.id;
              if (newId) await bindMutation.mutateAsync({ ruleId: newId, channelIds: channelIdList });
              message.success('创建成功');
              queryClient.invalidateQueries({ queryKey: ['alertRules'] });
              setIsDrawerOpen(false);
              form.resetFields();
            }
          } catch {
            // Error already shown by mutation
          }
        }}>
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input placeholder="请输入规则名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="规则描述" />
          </Form.Item>
          <Form.Item
            name="expression"
            label="表达式 (PromQL)"
            rules={[{ required: true }]}
          >
            <ExpressionInputWithTest
              onTest={async () => {
                const expression = form.getFieldValue('expression')?.trim();
                const dataSourceUrl = form.getFieldValue('data_source_url');
                const dataSourceType = form.getFieldValue('data_source_type') || 'prometheus';
                if (!expression) {
                  message.warning('请先输入表达式');
                  return;
                }
                if (!dataSourceUrl) {
                  message.warning('请先选择数据源');
                  return;
                }
                try {
                  const res = await alertRuleApi.testExpression({
                    expression,
                    data_source_type: dataSourceType,
                    data_source_url: dataSourceUrl,
                  });
                  const payload = (res.data as { data?: { count?: number; data?: unknown[] } })?.data;
                  const count = payload?.count ?? 0;
                  const data = payload?.data ?? [];
                  Modal.success({
                    title: '表达式测试成功',
                    width: 560,
                    content: (
                      <div>
                        <p>返回 <strong>{count}</strong> 条结果。</p>
                        {Array.isArray(data) && data.length > 0 && (
                          <pre style={{ marginTop: 8, padding: 12, background: '#f5f5f5', borderRadius: 4, fontSize: 12, maxHeight: 240, overflow: 'auto' }}>
                            {JSON.stringify(data.slice(0, 10), null, 2)}
                            {data.length > 10 ? `\n... 共 ${data.length} 条` : ''}
                          </pre>
                        )}
                      </div>
                    ),
                  });
                } catch (e: unknown) {
                  const err = e as { response?: { data?: { message?: string } } };
                  const msg = err?.response?.data?.message || (err as Error)?.message || '请求失败';
                  Modal.error({ title: '表达式测试失败', content: String(msg) });
                }
              }}
            />
          </Form.Item>
          <Form.Item name="evaluation_interval_seconds" label="执行频率(秒)" rules={[{ required: true }]} initialValue={60}>
            <InputNumber min={1} style={{ width: '100%' }} placeholder="60" />
          </Form.Item>
          <Form.Item name="for_duration" label="持续时间(秒)" rules={[{ required: true }]}>
            <InputNumber min={1} style={{ width: '100%' }} placeholder="60" />
          </Form.Item>
          <Form.Item name="severity" label="严重级别" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'critical', label: '严重 (Critical)' },
                { value: 'warning', label: '警告 (Warning)' },
                { value: 'info', label: '信息 (Info)' },
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
          <Form.Item name="group_id" label="业务组" rules={[{ required: true, message: '请选择业务组' }]}>
            <Select
              placeholder="请选择业务组"
              allowClear={false}
              loading={groupsLoading}
              showSearch
              optionFilterProp="label"
              notFoundContent={
                (Array.isArray(groupsData?.data) ? groupsData.data : []).length === 0 && !groupsLoading
                  ? '暂无业务组，请先在系统设置-业务组管理中创建'
                  : null
              }
              options={(Array.isArray(groupsData?.data) ? groupsData.data : []).map((g) => ({
                value: g.id,
                label: g.name || g.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="template_id" label="关联告警模板">
            <Select
              placeholder="可选，选择后告警通知将使用该模板渲染内容"
              allowClear
              showSearch
              optionFilterProp="label"
              options={(Array.isArray(templatesData?.data) ? templatesData.data : []).map((t) => ({
                value: t.id,
                label: t.name || t.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="data_source_id" label="数据源" rules={[{ required: true, message: '请选择数据源' }]}>
            <Select
              placeholder="请选择已配置的数据源"
              allowClear={false}
              loading={dataSourcesLoading}
              showSearch
              optionFilterProp="label"
              notFoundContent={
                (Array.isArray(dataSourcesData?.data) ? dataSourcesData.data : []).length === 0 && !dataSourcesLoading
                  ? '暂无数据源，请先在数据源管理中添加'
                  : null
              }
              options={(Array.isArray(dataSourcesData?.data) ? dataSourcesData.data : []).map((ds) => ({
                value: ds.id,
                label: `${ds.name} (${ds.type}) · ${ds.endpoint}`,
              }))}
              onChange={(id) => {
                const ds = (Array.isArray(dataSourcesData?.data) ? dataSourcesData.data : []).find((d) => d.id === id);
                if (ds) form.setFieldsValue({ data_source_type: ds.type, data_source_url: ds.endpoint });
              }}
            />
          </Form.Item>
          <Form.Item name="data_source_type" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="data_source_url" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="channel_ids" label="告警渠道">
            <Select
              mode="multiple"
              placeholder="从已配置的渠道中选择，告警将通知到所选渠道"
              allowClear
              showSearch
              optionFilterProp="label"
              loading={channelsLoading}
              options={(Array.isArray(channelsData?.data) ? channelsData.data : []).map((ch) => ({
                value: ch.id,
                label: `${ch.name} (${ch.type})`,
              }))}
            />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle" align="start">
            <Form.Item name="effective_start_time" label="生效开始时间" tooltip="每日规则生效开始时间，默认 00:00（24 小时生效）">
              <Input placeholder="00:00" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="effective_end_time" label="生效结束时间" tooltip="每日规则生效结束时间，默认 23:59">
              <Input placeholder="23:59" style={{ width: 100 }} />
            </Form.Item>
          </Space>
          <Form.Item label="排除时间" tooltip="在此时间段内不触发告警（不评估或跳过触发）">
            <Form.List name="exclusion_windows">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="start">
                      <Form.Item {...restField} name={[name, 'start']} rules={[{ pattern: /^\d{1,2}:\d{2}$/, message: 'HH:mm' }]}>
                        <Input placeholder="开始 02:00" style={{ width: 90 }} />
                      </Form.Item>
                      <Form.Item {...restField} name={[name, 'end']} rules={[{ pattern: /^\d{1,2}:\d{2}$/, message: 'HH:mm' }]}>
                        <Input placeholder="结束 06:00" style={{ width: 90 }} />
                      </Form.Item>
                      <Form.Item {...restField} name={[name, 'days']} label="星期">
                        <Checkbox.Group
                          options={[
                            { value: 0, label: '日' },
                            { value: 1, label: '一' },
                            { value: 2, label: '二' },
                            { value: 3, label: '三' },
                            { value: 4, label: '四' },
                            { value: 5, label: '五' },
                            { value: 6, label: '六' },
                          ]}
                        />
                      </Form.Item>
                      <Button type="text" danger onClick={() => remove(name)}>删除</Button>
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add({ start: '', end: '', days: [] })} block style={{ marginBottom: 8 }}>
                    添加排除时段
                  </Button>
                </>
              )}
            </Form.List>
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

      <Drawer
        title="绑定告警渠道"
        open={isBindDrawerOpen}
        onClose={() => {
          setIsBindDrawerOpen(false);
          setSelectedChannels([]);
        }}
        width={500}
      >
        <div style={{ marginBottom: 16 }}>
          <p>选择需要绑定的告警渠道：</p>
        </div>
        <Checkbox.Group
          value={selectedChannels}
          onChange={(values) => setSelectedChannels(values as string[])}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {(Array.isArray(channelsData?.data) ? channelsData.data : []).map((channel: AlertChannel) => (
              <Checkbox key={channel.id} value={channel.id}>
                <Tag>{channel.type}</Tag> {channel.name}
              </Checkbox>
            ))}
          </div>
        </Checkbox.Group>
        <div style={{ marginTop: 24 }}>
          <Button
            type="primary"
            onClick={() => bindMutation.mutate({ ruleId: currentRuleId, channelIds: selectedChannels })}
            loading={bindMutation.isPending}
          >
            保存绑定
          </Button>
        </div>
      </Drawer>

      <Modal
        title="批量导入规则"
        open={isImportModalOpen}
        onCancel={() => {
          setIsImportModalOpen(false);
          setImportFile(null);
          setImportResult(null);
        }}
        footer={null}
        width={600}
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary">上传 JSON 格式的规则文件，每条规则包含以下字段：name, expression, severity, for_duration, group_id 等</Text>
        </div>
        <Dragger
          beforeUpload={(file) => {
            if (file.type !== 'application/json') {
              message.error('只能上传 JSON 文件');
              return false;
            }
            setImportFile(file);
            return false;
          }}
          showUploadList={false}
        >
          <p className="ant-upload-drag-icon">
            <InboxOutlined />
          </p>
          <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
          <p className="ant-upload-hint">{importFile ? importFile.name : '支持 JSON 格式'}</p>
        </Dragger>
        {importFile && (
          <div style={{ marginTop: 16 }}>
            <Button
              type="primary"
              loading={importMutation.isPending}
              onClick={() => importMutation.mutate(importFile)}
            >
              开始导入
            </Button>
          </div>
        )}
        {importResult && (
          <div style={{ marginTop: 16, padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
            <Text strong>导入结果：</Text>
            <div style={{ marginTop: 8 }}>
              <Text type="success">成功: {importResult.success} 条</Text>
            </div>
            <div>
              <Text type="danger">失败: {importResult.failed} 条</Text>
            </div>
            {importResult.errors.length > 0 && (
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">错误详情：</Text>
                <ul style={{ marginTop: 4, paddingLeft: 16 }}>
                  {importResult.errors.slice(0, 5).map((err, idx) => (
                    <li key={idx}><Text type="danger" style={{ fontSize: 12 }}>{err}</Text></li>
                  ))}
                  {importResult.errors.length > 5 && (
                    <li><Text type="secondary">... 共 {importResult.errors.length} 条错误</Text></li>
                  )}
                </ul>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
