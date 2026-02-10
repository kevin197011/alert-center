import axios from 'axios';
import { useAuthStore } from '../store/auth';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout();
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  size: number;
}

/** Exclusion window: 0=Sunday .. 6=Saturday; empty days = every day */
export interface ExclusionWindow {
  start: string;
  end: string;
  days?: number[];
}

export interface AlertRule {
  id: string;
  name: string;
  description: string;
  expression: string;
  /** 执行频率(秒)，规则评估间隔，默认 60 */
  evaluation_interval_seconds?: number;
  for_duration: number;
  severity: 'critical' | 'warning' | 'info';
  labels: Record<string, string>;
  annotations: Record<string, string>;
  group_id: string;
  /** 关联告警模板 ID，可选 */
  template_id?: string | null;
  data_source_type: string;
  data_source_url: string;
  status: number;
  /** 生效开始时间 HH:mm，默认 00:00 */
  effective_start_time?: string;
  /** 生效结束时间 HH:mm，默认 23:59 */
  effective_end_time?: string;
  /** 排除时间列表 */
  exclusion_windows?: ExclusionWindow[];
  /** 绑定的告警渠道（列表接口返回） */
  bound_channels?: { id: string; name: string; type: string }[];
  created_at: string;
  updated_at: string;
}

export interface AlertChannel {
  id: string;
  name: string;
  type: 'lark' | 'telegram' | 'email' | 'webhook';
  description: string;
  config: Record<string, unknown>;
  group_id: string;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface AlertHistory {
  id: string;
  /** Unique date-time related id, e.g. AL20250205143022-a1b2c3d4 */
  alert_no: string;
  rule_id: string;
  fingerprint: string;
  severity: string;
  status: 'firing' | 'resolved';
  started_at: string;
  ended_at: string | null;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  created_at: string;
}

export interface BusinessGroup {
  id: string;
  name: string;
  description: string;
  parent_id: string | null;
  manager_id: string | null;
  status: number;
  created_at: string;
  updated_at: string;
}

export const alertRuleApi = {
  list: (params: { page?: number; page_size?: number; group_id?: string; severity?: string; status?: string }) =>
    api.get<PaginatedResponse<AlertRule>>('/alert-rules', { params }),

  getById: (id: string) =>
    api.get<AlertRule>(`/alert-rules/${id}`),

  create: (data: Partial<AlertRule>) =>
    api.post<AlertRule>('/alert-rules', data),

  update: (id: string, data: Partial<AlertRule>) =>
    api.put<AlertRule>(`/alert-rules/${id}`, data),

  delete: (id: string) =>
    api.delete(`/alert-rules/${id}`),

  testExpression: (data: { expression: string; data_source_type?: string; data_source_url: string }) =>
    api.post<{ data?: { count: number; data: Array<{ metric?: Record<string, string>; value?: { value: number } }> } }>('/alert-rules/test-expression', data),

  export: (params: { start_time?: string; end_time?: string }) =>
    api.get('/alert-rules/export', { params, responseType: 'blob' }),
};

export const alertChannelApi = {
  list: (params: { page?: number; page_size?: number; type?: string; status?: string }) =>
    api.get<PaginatedResponse<AlertChannel>>('/channels', { params }),

  getById: (id: string) =>
    api.get<AlertChannel>(`/channels/${id}`),

  create: (data: Partial<AlertChannel>) =>
    api.post<AlertChannel>('/channels', data),

  update: (id: string, data: Partial<AlertChannel>) =>
    api.put<AlertChannel>(`/channels/${id}`, data),

  delete: (id: string) =>
    api.delete(`/channels/${id}`),

  test: (id: string) =>
    api.post(`/channels/${id}/test`),

  /** Test channel with current form config (e.g. before save in create/edit drawer). */
  testWithConfig: (data: { type: string; config: Record<string, unknown> }) =>
    api.post('/channels/test-config', data),
};

export const alertHistoryApi = {
  list: (params: { page?: number; page_size?: number; rule_id?: string; status?: string; start_time?: string; end_time?: string }) =>
    api.get<PaginatedResponse<AlertHistory>>('/alert-history', { params }),
};

export const businessGroupApi = {
  list: (params?: { page?: number; page_size?: number; status?: number }) =>
    api.get<PaginatedResponse<BusinessGroup>>('/business-groups', { params }),
};

/** Backend success response wrapper (code, message, data) */
export interface ApiResponse<T> {
  code?: number;
  message?: string;
  data?: T;
}

export const authApi = {
  login: (username: string, password: string) =>
    api.post<ApiResponse<{ user: User; token: string }>>('/auth/login', { username, password }),

  getProfile: () =>
    api.get<User>('/profile'),
};

export interface AlertTemplate {
  id: string;
  name: string;
  description: string;
  content: string;
  variables: string;
  type: string;
  group_id: string | null;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface AlertChannelBinding {
  id: string;
  rule_id: string;
  channel_id: string;
  channel_name?: string;
  channel_type?: string;
  status: number;
  created_at: string;
}

export interface User {
  id: string;
  username: string;
  email: string;
  role: string;
}

export const templateApi = {
  list: (params: { page?: number; page_size?: number; type?: string; status?: string }) =>
    api.get<PaginatedResponse<AlertTemplate>>('/templates', { params }),

  getById: (id: string) =>
    api.get<AlertTemplate>(`/templates/${id}`),

  create: (data: Partial<AlertTemplate>) =>
    api.post<AlertTemplate>('/templates', data),

  update: (id: string, data: Partial<AlertTemplate>) =>
    api.put<AlertTemplate>(`/templates/${id}`, data),

  delete: (id: string) =>
    api.delete(`/templates/${id}`),
};

export const bindingApi = {
  getByRule: (ruleId: string) =>
    api.get<AlertChannelBinding[]>(`/alert-rules/${ruleId}/bindings`),

  bind: (ruleId: string, channelIds: string[]) =>
    api.post(`/alert-rules/${ruleId}/bindings`, { channel_ids: channelIds }),

  unbind: (ruleId: string, channelId: string) =>
    api.delete(`/alert-rules/${ruleId}/bindings/${channelId}`),
};

export interface User {
  id: string;
  username: string;
  email: string;
  phone?: string;
  role: string;
  status: number;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

export interface AuditLog {
  id: string;
  user_id: string;
  action: string;
  resource: string;
  resource_id: string;
  detail: string;
  ip: string;
  created_at: string;
}

export const userApi = {
  list: (params: { page?: number; page_size?: number; role?: string; status?: string }) =>
    api.get<PaginatedResponse<User>>('/users', { params }),

  getById: (id: string) =>
    api.get<User>(`/users/${id}`),

  create: (data: Partial<User>) =>
    api.post<User>('/users', data),

  update: (id: string, data: Partial<User>) =>
    api.put<User>(`/users/${id}`, data),

  delete: (id: string) =>
    api.delete(`/users/${id}`),

  changePassword: (id: string, oldPassword: string, newPassword: string) =>
    api.post(`/users/${id}/password`, { old_password: oldPassword, new_password: newPassword }),
};

export const auditLogApi = {
  list: (params: { page?: number; page_size?: number; user_id?: string; action?: string; resource?: string; start_time?: string; end_time?: string }) =>
    api.get<PaginatedResponse<AuditLog>>('/audit-logs', { params }),

  export: (params: { user_id?: string; action?: string; resource?: string; start_time?: string; end_time?: string }) =>
    api.get('/audit-logs/export', { params, responseType: 'blob' }),
};

export interface DataSource {
  id: string;
  name: string;
  type: string;
  description: string;
  endpoint: string;
  config: Record<string, unknown>;
  status: number;
  health_status: string;
  last_check_at: string;
  created_at: string;
  updated_at: string;
}

export interface AlertStatistics {
  total_alerts: number;
  firing_alerts: number;
  resolved_alerts: number;
  critical_alerts: number;
  warning_alerts: number;
  info_alerts: number;
  by_severity: { severity: string; count: number }[];
  by_status: { status: string; count: number }[];
  by_day: { date: string; total: number; firing: number; resolved: number; critical: number; warning: number }[];
  top_firing_rules: { rule_id: string; rule_name: string; alert_count: number }[];
}

export interface DashboardSummary {
  total_rules: number;
  enabled_rules: number;
  total_channels: number;
  enabled_channels: number;
  today_alerts: number;
  firing_alerts: number;
}

export const dataSourceApi = {
  list: (params: { page?: number; page_size?: number; type?: string; status?: string }) =>
    api.get<PaginatedResponse<DataSource>>('/data-sources', { params }),

  getById: (id: string) =>
    api.get<DataSource>(`/data-sources/${id}`),

  create: (data: Partial<DataSource>) =>
    api.post<DataSource>('/data-sources', data),

  update: (id: string, data: Partial<DataSource>) =>
    api.put<DataSource>(`/data-sources/${id}`, data),

  delete: (id: string) =>
    api.delete(`/data-sources/${id}`),

  healthCheck: (id: string) =>
    api.post(`/data-sources/${id}/health-check`),
};

export const statisticsApi = {
  getStatistics: (params?: { start_time?: string; end_time?: string; group_id?: string }) =>
    api.get<AlertStatistics>('/statistics', { params }),

  getDashboard: () =>
    api.get<DashboardSummary>('/dashboard'),
};

export interface AlertSilence {
  id: string;
  name: string;
  description: string;
  matchers: string;
  start_time: string;
  end_time: string;
  created_by: string;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface SilenceMatcher {
  [key: string]: string;
}

export const silenceApi = {
  list: (params: { page?: number; page_size?: number; status?: number }) =>
    api.get<PaginatedResponse<AlertSilence>>('/silences', { params }),

  getById: (id: string) =>
    api.get<AlertSilence>(`/silences/${id}`),

  create: (data: { name: string; description?: string; matchers: SilenceMatcher[]; start_time: string; end_time: string }) =>
    api.post<AlertSilence>('/silences', data),

  update: (id: string, data: Partial<AlertSilence>) =>
    api.put<AlertSilence>(`/silences/${id}`, data),

  delete: (id: string) =>
    api.delete(`/silences/${id}`),

  check: (labels: Record<string, string>) =>
    api.post<{ silenced: boolean }>('/silences/check', { labels }),
};

export const batchApi = {
  importRules: (rules: Partial<AlertRule>[]) =>
    api.post<{ success: number; failed: number; errors: string[] }>('/batch/import/rules', { rules }),

  exportRules: (params?: { group_id?: string; severity?: string; status?: string }) =>
    api.get('/batch/export/rules', { params, responseType: 'blob' }),

  exportChannels: (params?: { type?: string }) =>
    api.get('/batch/export/channels', { params, responseType: 'blob' }),

  importSilences: (silences: { name: string; description?: string; matchers: SilenceMatcher[]; start_time: string; end_time: string }[]) =>
    api.post<{ success: number; failed: number; errors: string[] }>('/batch/import/silences', { silences }),

  exportSilences: () =>
    api.get('/batch/export/silences', { responseType: 'blob' }),
};

export interface SLAConfig {
  id: string;
  name: string;
  severity: string;
  response_time_mins: number;
  resolution_time_mins: number;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface AlertSLA {
  alert_id: string;
  rule_id: string;
  severity: string;
  sla_config_id: string;
  response_deadline: string;
  resolution_deadline: string;
  first_acked_at: string | null;
  resolved_at: string | null;
  status: string;
  response_breached: boolean;
  resolution_breached: boolean;
  response_time_secs: number;
  resolution_time_secs: number;
}

export interface SLAReport {
  period_start: string;
  period_end: string;
  total_alerts: number;
  met_count: number;
  breached_count: number;
  in_progress_count: number;
  pending_count: number;
  compliance_rate: number;
  avg_response_time_secs: number;
  avg_resolution_time_secs: number;
}

export const slaApi = {
  listConfigs: () =>
    api.get<{ data: SLAConfig[]; total: number }>('/sla/configs'),

  createConfig: (data: { name: string; severity: string; response_time_mins: number; resolution_time_mins: number; priority?: number }) =>
    api.post<SLAConfig>('/sla/configs', data),

  updateConfig: (id: string, data: Partial<SLAConfig>) =>
    api.put<SLAConfig>(`/sla/configs/${id}`, data),

  deleteConfig: (id: string) =>
    api.delete(`/sla/configs/${id}`),

  seedConfigs: () =>
    api.get('/sla/configs/seed'),

  getAlertSLA: (alertId: string) =>
    api.get<AlertSLA>(`/sla/alerts/${alertId}`),

  getReport: (params?: { start_time?: string; end_time?: string }) =>
    api.get<SLAReport>('/sla/report', { params }),
};

export interface OnCallSchedule {
  id: string;
  name: string;
  description: string;
  timezone: string;
  rotation_type: string;
  rotation_start: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface OnCallMember {
  id: string;
  schedule_id: string;
  user_id: string;
  username: string;
  email: string;
  phone: string;
  priority: number;
  start_time: string;
  end_time: string;
  is_active: boolean;
}

export interface OnCallAssignment {
  id: string;
  schedule_id: string;
  user_id: string;
  username: string;
  start_time: string;
  end_time: string;
  email: string;
  phone: string;
}

export const oncallApi = {
  listSchedules: () =>
    api.get<{ data: OnCallSchedule[] }>('/oncall/schedules'),

  createSchedule: (data: { name: string; description?: string; timezone?: string; rotation_type?: string; rotation_start?: string }) =>
    api.post<OnCallSchedule>('/oncall/schedules', data),

  getSchedule: (id: string) =>
    api.get<OnCallSchedule>(`/oncall/schedules/${id}`),

  deleteSchedule: (id: string) =>
    api.delete(`/oncall/schedules/${id}`),

  addMember: (scheduleId: string, data: { user_id: string; username: string; email?: string; phone?: string; priority?: number; start_time?: string; end_time?: string }) =>
    api.post<OnCallMember>(`/oncall/schedules/${scheduleId}/members`, data),

  getMembers: (scheduleId: string) =>
    api.get<{ data: OnCallMember[] }>(`/oncall/schedules/${scheduleId}/members`),

  deleteMember: (scheduleId: string, memberId: string) =>
    api.delete(`/oncall/schedules/${scheduleId}/members/${memberId}`),

  getScheduleAssignments: (scheduleId: string, params?: { start_time?: string; end_time?: string }) =>
    api.get<{ data: OnCallAssignment[] }>(`/oncall/schedules/${scheduleId}/assignments`, { params }),

  generateRotations: (scheduleId: string, data: { end_time: string }) =>
    api.post(`/oncall/schedules/${scheduleId}/generate-rotations`, data),

  escalate: (scheduleId: string, data: { current_user_id: string }) =>
    api.post<OnCallAssignment>(`/oncall/schedules/${scheduleId}/escalate`, data),

  getCurrentOnCall: () =>
    api.get<{ data: OnCallAssignment[] }>('/oncall/current'),

  whoIsOnCall: (params?: { at_time?: string }) =>
    api.get<{ data: OnCallAssignment[] }>('/oncall/who', { params }),

  getReport: (params?: { start_time?: string; end_time?: string }) =>
    api.get('/oncall/report', { params }),

  seedSchedules: () =>
    api.get('/oncall/seed'),
};

export interface ScheduleCoverage {
  start_time: string;
  end_time: string;
  duration: string;
}

export interface ScheduleValidation {
  schedule_id: string;
  start_time: string;
  end_time: string;
  gap_count: number;
  total_gap_duration: string;
  coverage_percent: number;
  is_valid: boolean;
}

export interface GeneratedShift {
  id: string;
  schedule_id: string;
  user_id: string;
  username: string;
  email: string;
  phone: string;
  start_time: string;
  end_time: string;
  created_at: string;
}

export const schedulingApi = {
  generate: (scheduleId: string, data: { start_time: string; end_time: string; shift_duration?: number; timezone?: string }) =>
    api.post<{ shifts: GeneratedShift[]; total: number }>(`/oncall/schedules/${scheduleId}/generate`, data),

  getCoverage: (scheduleId: string, params?: { start_time?: string; end_time?: string }) =>
    api.get<{ gaps: ScheduleCoverage[]; total_gaps: number }>(`/oncall/schedules/${scheduleId}/coverage`, { params }),

  suggestRotation: (scheduleId: string) =>
    api.get<{ suggestions: string[] }>(`/oncall/schedules/${scheduleId}/suggest`),

  validate: (scheduleId: string, params?: { start_time?: string; end_time?: string }) =>
    api.get<ScheduleValidation>(`/oncall/schedules/${scheduleId}/validate`, { params }),
};

export interface AlertEscalation {
  id: string;
  alert_id: string;
  from_user_id: string;
  from_username: string;
  to_user_id: string;
  to_username: string;
  reason: string;
  status: string;
  created_at: string;
  resolved_at?: string;
}

export const escalationApi = {
  create: (data: { alert_id: string; to_user_id: string; to_username: string; reason: string }) =>
    api.post<AlertEscalation>('/escalations', data),

  getByAlert: (alertId: string) =>
    api.get<{ data: AlertEscalation[] }>(`/escalations/alert/${alertId}`),

  getPending: () =>
    api.get<{ data: AlertEscalation[] }>('/escalations/pending'),

  accept: (id: string) =>
    api.post(`/escalations/${id}/accept`),

  reject: (id: string) =>
    api.post(`/escalations/${id}/reject`),

  resolve: (id: string) =>
    api.post(`/escalations/${id}/resolve`),
};

export interface SLABreach {
  id: string;
  alert_id: string;
  rule_id: string;
  severity: string;
  breach_type: string;
  breach_time: string;
  response_time: number;
  assigned_to?: string;
  assigned_name?: string;
  notified: boolean;
  created_at: string;
}

export interface SLABreachStats {
  period_start: string;
  period_end: string;
  total_breaches: number;
  total_response_breaches: number;
  total_resolution_breaches: number;
  alerts_breached: number;
  response_breaches: Record<string, number>;
  resolution_breaches: Record<string, number>;
}

export const slaBreachApi = {
  getBreaches: (params?: { page?: number; page_size?: number; status?: string }) =>
    api.get<{ data: SLABreach[]; total: number }>('/sla/breaches', { params }),

  getStats: (params?: { start_time?: string; end_time?: string }) =>
    api.get<SLABreachStats>('/sla/breaches/stats', { params }),

  triggerCheck: () =>
    api.post<{ breaches_found: number }>('/sla/breaches/check'),

  triggerNotifications: () =>
    api.post<{ notifications: number }>('/sla/breaches/notify'),
};

export const correlationApi = {
  getAnalyze: (alertId: string, params: { window_minutes: number }) =>
    api.get<unknown>(`/correlation/analyze/${alertId}`, { params }),

  getPatterns: (params: { hours: number; min_occurrences: number }) =>
    api.get<{ data?: unknown[] }>('/correlation/patterns', { params }),

  getFlapping: () =>
    api.get<{ data?: string[] }>('/correlation/flapping'),
};

export interface Ticket {
  id: string;
  title: string;
  description: string;
  alert_id?: string;
  rule_id?: string;
  priority: string;
  status: string;
  assignee_id?: string;
  assignee_name?: string;
  creator_id: string;
  creator_name: string;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
  closed_at?: string;
}

export interface TicketStats {
  open: number;
  in_progress: number;
  resolved: number;
  closed: number;
  total: number;
}

export const ticketApi = {
  list: (params?: { page?: number; page_size?: number; status?: string }) =>
    api.get<{ data: Ticket[]; total: number; page: number; size: number }>('/tickets', { params }),

  getById: (id: string) =>
    api.get<Ticket>(`/tickets/${id}`),

  create: (data: { title: string; description: string; alert_id?: string; rule_id?: string; priority: string; assignee_name?: string }) =>
    api.post<Ticket>('/tickets', data),

  update: (id: string, data: Partial<Ticket>) =>
    api.put<Ticket>(`/tickets/${id}`, data),

  resolve: (id: string) =>
    api.post(`/tickets/${id}/resolve`),

  close: (id: string) =>
    api.post(`/tickets/${id}/close`),

  delete: (id: string) =>
    api.delete(`/tickets/${id}`),

  getStats: () =>
    api.get<{ data: TicketStats }>('/tickets/stats'),
};
