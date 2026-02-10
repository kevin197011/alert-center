import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import 'dayjs/locale/en';

dayjs.locale('zh-cn');

export type Locale = 'zh-CN' | 'en-US';

export interface LocaleConfig {
  locale: Locale;
  dayjsLocale: string;
  label: string;
}

export const locales: Record<Locale, LocaleConfig> = {
  'zh-CN': {
    locale: 'zh-CN',
    dayjsLocale: 'zh-cn',
    label: '简体中文',
  },
  'en-US': {
    locale: 'en-US',
    dayjsLocale: 'en',
    label: 'English',
  },
};

export const translations: Record<string, Record<string, string>> = {
  'zh-CN': {
    // Common
    'common.loading': '加载中...',
    'common.save': '保存',
    'common.cancel': '取消',
    'common.delete': '删除',
    'common.edit': '编辑',
    'common.create': '创建',
    'common.search': '搜索',
    'common.refresh': '刷新',
    'common.export': '导出',
    'common.import': '导入',
    'common.success': '成功',
    'common.error': '错误',
    'common.confirm': '确认',
    'common.submit': '提交',

    // Navigation
    'nav.dashboard': '仪表盘',
    'nav.rules': '告警规则',
    'nav.channels': '告警渠道',
    'nav.templates': '告警模板',
    'nav.history': '告警历史',
    'nav.silences': '告警静默',
    'nav.sla': 'SLA配置',
    'nav.slaBreaches': 'SLA违约',
    'nav.oncall': '值班管理',
    'nav.oncallReport': '值班报告',
    'nav.escalations': '升级历史',
    'nav.correlation': '关联分析',
    'nav.tickets': '工单管理',
    'nav.statistics': '告警统计',
    'nav.dataSources': '数据源',
    'nav.users': '用户管理',
    'nav.auditLogs': '审计日志',
    'nav.settings': '系统设置',

    // Dashboard
    'dashboard.firingAlerts': '进行中告警',
    'dashboard.todayAlerts': '今日告警',
    'dashboard.activeRules': '活跃规则',
    'dashboard.activeChannels': '活跃渠道',

    // Alert Rules
    'rules.newRule': '新建规则',
    'rules.expression': '表达式',
    'rules.labels': '标签',
    'rules.annotations': '注解',
    'rules.enabled': '启用',
    'rules.disabled': '禁用',

    // SLA
    'sla.responseTime': '响应时限',
    'sla.resolutionTime': '解决时限',
    'sla.complianceRate': '达成率',
    'sla.breached': '违约',
    'sla.met': '达成',

    // OnCall
    'oncall.currentOnCall': '当前值班',
    'oncall.schedule': '排班',
    'oncall.rotation': '轮转',

    // Status
    'status.firing': '进行中',
    'status.resolved': '已恢复',
    'status.pending': '待处理',
    'status.acknowledged': '已确认',

    // Severity
    'severity.critical': '严重',
    'severity.warning': '警告',
    'severity.info': '信息',

    // Time
    'time.minutes': '分钟',
    'time.hours': '小时',
    'time.days': '天',
  },
  'en-US': {
    // Common
    'common.loading': 'Loading...',
    'common.save': 'Save',
    'common.cancel': 'Cancel',
    'common.delete': 'Delete',
    'common.edit': 'Edit',
    'common.create': 'Create',
    'common.search': 'Search',
    'common.refresh': 'Refresh',
    'common.export': 'Export',
    'common.import': 'Import',
    'common.success': 'Success',
    'common.error': 'Error',
    'common.confirm': 'Confirm',
    'common.submit': 'Submit',

    // Navigation
    'nav.dashboard': 'Dashboard',
    'nav.rules': 'Alert Rules',
    'nav.channels': 'Alert Channels',
    'nav.templates': 'Alert Templates',
    'nav.history': 'Alert History',
    'nav.silences': 'Alert Silences',
    'nav.sla': 'SLA Config',
    'nav.slaBreaches': 'SLA Breaches',
    'nav.oncall': 'On-Call',
    'nav.oncallReport': 'On-Call Report',
    'nav.escalations': 'Escalations',
    'nav.correlation': 'Correlation',
    'nav.tickets': 'Tickets',
    'nav.statistics': 'Statistics',
    'nav.dataSources': 'Data Sources',
    'nav.users': 'User Management',
    'nav.auditLogs': 'Audit Logs',
    'nav.settings': 'Settings',

    // Dashboard
    'dashboard.firingAlerts': 'Firing Alerts',
    'dashboard.todayAlerts': 'Today Alerts',
    'dashboard.activeRules': 'Active Rules',
    'dashboard.activeChannels': 'Active Channels',

    // Alert Rules
    'rules.newRule': 'New Rule',
    'rules.expression': 'Expression',
    'rules.labels': 'Labels',
    'rules.annotations': 'Annotations',
    'rules.enabled': 'Enabled',
    'rules.disabled': 'Disabled',

    // SLA
    'sla.responseTime': 'Response Time',
    'sla.resolutionTime': 'Resolution Time',
    'sla.complianceRate': 'Compliance Rate',
    'sla.breached': 'Breached',
    'sla.met': 'Met',

    // OnCall
    'oncall.currentOnCall': 'Current On-Call',
    'oncall.schedule': 'Schedule',
    'oncall.rotation': 'Rotation',

    // Status
    'status.firing': 'Firing',
    'status.resolved': 'Resolved',
    'status.pending': 'Pending',
    'status.acknowledged': 'Acknowledged',

    // Severity
    'severity.critical': 'Critical',
    'severity.warning': 'Warning',
    'severity.info': 'Info',

    // Time
    'time.minutes': 'minutes',
    'time.hours': 'hours',
    'time.days': 'days',
  },
};

export function t(key: string, locale: Locale = 'zh-CN'): string {
  return translations[locale]?.[key] || translations['zh-CN'][key] || key;
}

export function setDayjsLocale(locale: Locale) {
  dayjs.locale(locales[locale].dayjsLocale);
}

export function getCurrentLocale(): Locale {
  const saved = localStorage.getItem('locale');
  if (saved && saved in locales) {
    return saved as Locale;
  }
  const browserLang = navigator.language;
  if (browserLang.startsWith('zh')) {
    return 'zh-CN';
  }
  return 'en-US';
}
