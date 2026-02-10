import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Table, Button, Space, Tag, message, Modal, Form, Input, Drawer, DatePicker, Tooltip, Typography, Badge, Collapse, Row, Col, Result, Upload, Dropdown } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, InfoCircleOutlined, CheckCircleOutlined, ExperimentOutlined, ImportOutlined, ExportOutlined, DownOutlined, InboxOutlined } from '@ant-design/icons';
import { silenceApi, batchApi, AlertSilence, SilenceMatcher } from '../../services/api';
import dayjs from 'dayjs';

const { Text } = Typography;
const { Panel } = Collapse;
const { Dragger } = Upload;

const getSilenceStatus = (silence: AlertSilence) => {
  if (silence.status !== 1) {
    return { status: 'disabled', text: '已禁用', color: 'default' };
  }
  const now = dayjs();
  const startTime = dayjs(silence.start_time);
  const endTime = dayjs(silence.end_time);
  
  if (now.isBefore(startTime)) {
    return { status: 'pending', text: '待生效', color: 'blue' };
  }
  if (now.isAfter(endTime)) {
    return { status: 'expired', text: '已过期', color: 'default' };
  }
  return { status: 'active', text: '生效中', color: 'green' };
};

export default function AlertSilences() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [editingSilence, setEditingSilence] = useState<AlertSilence | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: silencesData, isLoading } = useQuery({
    queryKey: ['silences', page, pageSize],
    queryFn: async () => {
      const res = await silenceApi.list({ page, page_size: pageSize });
      const body = res.data as unknown as { data?: { data: AlertSilence[]; total: number; page: number; size: number } };
      const payload = body?.data ?? { data: [], total: 0, page: 1, size: 0 };
      return { ...payload, data: Array.isArray(payload.data) ? payload.data : [] };
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; description?: string; matchers: SilenceMatcher[]; start_time: string; end_time: string }) =>
      silenceApi.create(data),
    onSuccess: () => {
      message.success('创建成功');
      queryClient.invalidateQueries({ queryKey: ['silences'] });
      setIsDrawerOpen(false);
      form.resetFields();
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<AlertSilence> }) => silenceApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功');
      queryClient.invalidateQueries({ queryKey: ['silences'] });
      setIsDrawerOpen(false);
      setEditingSilence(null);
      form.resetFields();
    },
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => silenceApi.delete(id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['silences'] });
    },
    onError: () => message.error('删除失败'),
  });

  const batchEnableMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      await Promise.all(ids.map(id => silenceApi.update(id, { status: 1 })));
    },
    onSuccess: () => {
      message.success('批量启用成功');
      queryClient.invalidateQueries({ queryKey: ['silences'] });
      setSelectedRowKeys([]);
    },
    onError: () => message.error('批量启用失败'),
  });

  const batchDisableMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      await Promise.all(ids.map(id => silenceApi.update(id, { status: 0 })));
    },
    onSuccess: () => {
      message.success('批量禁用成功');
      queryClient.invalidateQueries({ queryKey: ['silences'] });
      setSelectedRowKeys([]);
    },
    onError: () => message.error('批量禁用失败'),
  });

  const batchDeleteMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      await Promise.all(ids.map(id => silenceApi.delete(id)));
    },
    onSuccess: () => {
      message.success('批量删除成功');
      queryClient.invalidateQueries({ queryKey: ['silences'] });
      setSelectedRowKeys([]);
    },
    onError: () => message.error('批量删除失败'),
  });

  const [testLabels, setTestLabels] = useState<{ key: string; value: string }[]>([
    { key: 'severity', value: 'critical' },
    { key: 'instance', value: 'localhost:9090' },
  ]);
  const [testResult, setTestResult] = useState<boolean | null>(null);
  const [isTesting, setIsTesting] = useState(false);

  const checkSilence = async () => {
    const labels: Record<string, string> = {};
    testLabels.forEach((l) => {
      if (l.key && l.value) labels[l.key] = l.value;
    });
    setIsTesting(true);
    try {
      const res = await silenceApi.check(labels);
      setTestResult(res.data.silenced);
    } catch {
      message.error('检查失败');
      setTestResult(null);
    } finally {
      setIsTesting(false);
    }
  };

  const addTestLabel = () => {
    setTestLabels([...testLabels, { key: '', value: '' }]);
  };

  const removeTestLabel = (index: number) => {
    setTestLabels(testLabels.filter((_, i) => i !== index));
  };

  const updateTestLabel = (index: number, field: 'key' | 'value', val: string) => {
    const newLabels = [...testLabels];
    newLabels[index][field] = val;
    setTestLabels(newLabels);
  };

  const [isImportModalOpen, setIsImportModalOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<{ success: number; failed: number; errors: string[] } | null>(null);

  const importMutation = useMutation({
    mutationFn: async (file: File) => {
      const text = await file.text();
      const data = JSON.parse(text);
      return batchApi.importSilences(data.silences || data);
    },
    onSuccess: (res) => {
      setImportResult(res.data);
      message.success(`导入完成: 成功 ${res.data.success} 条, 失败 ${res.data.failed} 条`);
      queryClient.invalidateQueries({ queryKey: ['silences'] });
    },
    onError: () => {
      message.error('导入失败');
    },
  });

  const handleExport = async () => {
    try {
      const res = await batchApi.exportSilences();
      const blob = new Blob([res.data], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `alert_silences_${dayjs().format('YYYYMMDDHHmmss')}.json`;
      link.click();
      message.success('导出成功');
    } catch {
      message.error('导出失败');
    }
  };

  const exportItems = [
    {
      key: 'silences',
      label: '导出静默规则',
      onClick: handleExport,
    },
  ];

  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys),
  };

  const hasSelected = selectedRowKeys.length > 0;

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '匹配标签',
      dataIndex: 'matchers',
      key: 'matchers',
      width: 250,
      render: (matchers: string) => {
        try {
          const parsed = JSON.parse(matchers || '[]');
          return (
            <Space wrap size={[4, 4]}>
              {parsed.map((m: Record<string, string>, idx: number) => {
                const [key, value] = Object.entries(m)[0];
                const isRegex = value.startsWith('~');
                return (
                  <Tooltip key={idx} title={isRegex ? '正则表达式匹配' : '精确匹配'}>
                    <Tag color={isRegex ? 'purple' : 'blue'} style={{ margin: 0 }}>
                      {key}=~{isRegex ? value.slice(1) : value}
                    </Tag>
                  </Tooltip>
                );
              })}
            </Space>
          );
        } catch {
          return '-';
        }
      },
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      key: 'end_time',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, record: AlertSilence) => {
        const { text, color } = getSilenceStatus(record);
        return (
          <Badge status={color as 'success' | 'processing' | 'error' | 'default' | 'warning'} text={text} />
        );
      },
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
      width: 150,
      render: (_: unknown, record: AlertSilence) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingSilence(record);
              const matchers = JSON.parse(record.matchers || '[]');
              form.setFieldsValue({
                ...record,
                start_time: dayjs(record.start_time),
                end_time: dayjs(record.end_time),
                matchers: matchers.map((m: Record<string, string>) => {
                  const [key, value] = Object.entries(m)[0];
                  return { key, value: value.startsWith('~') ? value.slice(1) : value, isRegex: value.startsWith('~') };
                }),
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
                content: `确定要删除静默规则 "${record.name}" 吗？`,
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

  const [matcherForms, setMatcherForms] = useState<{ key: string; value: string; isRegex: boolean }[]>([
    { key: '', value: '', isRegex: false },
  ]);

  const addMatcher = () => {
    setMatcherForms([...matcherForms, { key: '', value: '', isRegex: false }]);
  };

  const removeMatcher = (index: number) => {
    const newMatchers = matcherForms.filter((_, i) => i !== index);
    setMatcherForms(newMatchers);
  };

  const updateMatcher = (index: number, field: 'key' | 'value' | 'isRegex', val: string | boolean) => {
    const newMatchers = [...matcherForms];
    if (field === 'isRegex') {
      newMatchers[index].isRegex = val as boolean;
    } else {
      newMatchers[index][field] = val as string;
    }
    setMatcherForms(newMatchers);
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">告警静默</h1>
        <Space>
          <Dropdown menu={{ items: exportItems }} placement="bottomRight">
            <Button icon={<ExportOutlined />}>
              导出 <DownOutlined />
            </Button>
          </Dropdown>
          <Button icon={<ImportOutlined />} onClick={() => {
            setIsImportModalOpen(true);
            setImportFile(null);
            setImportResult(null);
          }}>
            导入
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingSilence(null);
              form.resetFields();
              setMatcherForms([{ key: '', value: '', isRegex: false }]);
              setIsDrawerOpen(true);
            }}
          >
            新建静默规则
          </Button>
        </Space>
      </div>

      {hasSelected && (
        <div style={{ marginBottom: 16, padding: 16, background: '#e6f7ff', borderRadius: 8, border: '1px solid #91d5ff' }}>
          <Space>
            <Text strong>已选择 {selectedRowKeys.length} 条规则</Text>
            <Button
              size="small"
              onClick={() => batchEnableMutation.mutate(selectedRowKeys as string[])}
              loading={batchEnableMutation.isPending}
            >
              批量启用
            </Button>
            <Button
              size="small"
              onClick={() => batchDisableMutation.mutate(selectedRowKeys as string[])}
              loading={batchDisableMutation.isPending}
            >
              批量禁用
            </Button>
            <Button
              size="small"
              danger
              onClick={() => {
                Modal.confirm({
                  title: '确认批量删除',
                  content: `确定要删除选中的 ${selectedRowKeys.length} 条静默规则吗？`,
                  onOk: () => batchDeleteMutation.mutate(selectedRowKeys as string[]),
                });
              }}
              loading={batchDeleteMutation.isPending}
            >
              批量删除
            </Button>
            <Button size="small" onClick={() => setSelectedRowKeys([])}>
              取消选择
            </Button>
          </Space>
        </div>
      )}

      <Table
        rowSelection={rowSelection}
        columns={columns}
        dataSource={Array.isArray(silencesData?.data) ? silencesData.data : []}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize,
          total: silencesData?.total || 0,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />

      <Collapse defaultActiveKey={[]} style={{ marginTop: 16 }}>
        <Panel header={
          <span>
            <ExperimentOutlined style={{ marginRight: 8 }} />
            <Text strong>静默规则测试</Text>
            <Text type="secondary" style={{ marginLeft: 8 }}>输入标签组合，测试是否会被静默</Text>
          </span>
        } key="test">
          <Row gutter={16}>
            <Col span={16}>
              <div style={{ marginBottom: 16 }}>
                {testLabels.map((label, index) => (
                  <Row key={index} gutter={8} style={{ marginBottom: 8 }}>
                    <Col span={10}>
                      <Input
                        placeholder="标签键 (如: severity)"
                        value={label.key}
                        onChange={(e) => updateTestLabel(index, 'key', e.target.value)}
                      />
                    </Col>
                    <Col span={10}>
                      <Input
                        placeholder="标签值 (如: critical)"
                        value={label.value}
                        onChange={(e) => updateTestLabel(index, 'value', e.target.value)}
                      />
                    </Col>
                    <Col span={4}>
                      <Button
                        type="link"
                        danger
                        onClick={() => removeTestLabel(index)}
                        disabled={testLabels.length <= 1}
                      >
                        移除
                      </Button>
                    </Col>
                  </Row>
                ))}
                <Button type="dashed" onClick={addTestLabel} block>
                  添加标签
                </Button>
              </div>
              <Button
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={checkSilence}
                loading={isTesting}
              >
                测试静默
              </Button>
            </Col>
            <Col span={8}>
              <div style={{ padding: 16, background: '#f5f5f5', borderRadius: 8, minHeight: 120 }}>
                <Text strong>测试结果：</Text>
                {testResult !== null && (
                  <div style={{ marginTop: 12 }}>
                    {testResult ? (
                      <Result
                        status="success"
                        title="会被静默"
                        subTitle="当前标签组合匹配至少一条静默规则"
                        style={{ padding: 0 }}
                      />
                    ) : (
                      <Result
                        status="error"
                        title="不会被静默"
                        subTitle="当前标签组合不匹配任何静默规则"
                        style={{ padding: 0 }}
                      />
                    )}
                  </div>
                )}
                {testResult === null && (
                  <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
                    点击"测试静默"查看结果
                  </Text>
                )}
              </div>
            </Col>
          </Row>
        </Panel>
      </Collapse>

      <Drawer
        title={editingSilence ? '编辑静默规则' : '新建静默规则'}
        open={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false);
          setEditingSilence(null);
          form.resetFields();
        }}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            const matchers = matcherForms
              .filter((m) => m.key && m.value)
              .map((m) => {
                const pattern = m.isRegex ? '~' + m.value : m.value;
                return { [m.key]: pattern };
              });
            if (editingSilence) {
              const data = {
                name: values.name,
                description: values.description,
                matchers: JSON.stringify(matchers),
                start_time: values.start_time.toISOString(),
                end_time: values.end_time.toISOString(),
              };
              updateMutation.mutate({ id: editingSilence.id, data });
            } else {
              const data = {
                name: values.name,
                description: values.description,
                matchers,
                start_time: values.start_time.toISOString(),
                end_time: values.end_time.toISOString(),
              };
              createMutation.mutate(data);
            }
          }}
        >
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input placeholder="请输入规则名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="规则描述" />
          </Form.Item>

          <Form.Item label="匹配标签">
            <div style={{ marginBottom: 8 }}>
              <Text type="secondary">
                <Tooltip title="使用 ~ 前缀启用正则表达式匹配，例如: ~.*error.* 匹配包含 error 的值">
                  <span>
                    <InfoCircleOutlined style={{ marginRight: 4 }} />
                    支持正则表达式匹配
                  </span>
                </Tooltip>
              </Text>
            </div>
            {matcherForms.map((matcher, index) => (
              <Space key={index} style={{ display: 'flex', marginBottom: 8 }}>
                <Input
                  placeholder="标签键"
                  value={matcher.key}
                  onChange={(e) => updateMatcher(index, 'key', e.target.value)}
                  style={{ width: 150 }}
                />
                <span>=</span>
                <Input
                  placeholder={matcher.isRegex ? '正则表达式，如: .*error.*' : '标签值'}
                  value={matcher.value}
                  onChange={(e) => updateMatcher(index, 'value', e.target.value)}
                  addonBefore={
                    <Button
                      type={matcher.isRegex ? 'primary' : 'default'}
                      size="small"
                      onClick={() => updateMatcher(index, 'isRegex', !matcher.isRegex)}
                      style={{ margin: -5, height: 20, padding: '0 8px' }}
                    >
                      ~
                    </Button>
                  }
                  style={{ width: 200 }}
                />
                {matcherForms.length > 1 && (
                  <Button type="link" danger onClick={() => removeMatcher(index)}>
                    移除
                  </Button>
                )}
              </Space>
            ))}
            <Button type="dashed" onClick={addMatcher} block>
              添加标签匹配
            </Button>
          </Form.Item>

          <Form.Item name="start_time" label="开始时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="end_time" label="结束时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
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

      <Modal
        title="批量导入静默规则"
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
          <Text type="secondary">上传 JSON 格式的静默规则文件，包含 silences 数组，每条规则包含以下字段：name, matchers, start_time, end_time</Text>
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
