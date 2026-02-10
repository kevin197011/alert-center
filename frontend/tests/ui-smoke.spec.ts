import { test, expect, type Page } from '@playwright/test';

const adminUser = process.env.UI_ADMIN_USER || 'admin';
const adminPass = process.env.UI_ADMIN_PASS || 'admin123';

async function login(page: Page) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Alert Center' })).toBeVisible();
  await page.getByPlaceholder('用户名').fill(adminUser);
  await page.getByPlaceholder('密码').fill(adminPass);
  await page.getByRole('button', { name: '登 录' }).click();
  await page.waitForURL('**/');
  await expect(page.getByRole('heading', { name: '仪表盘' })).toBeVisible();
}

test.describe('UI Smoke', () => {
  test('login + key pages load', async ({ page }) => {
    await login(page);

    const pages = [
      { path: '/', heading: '仪表盘' },
      { path: '/rules', heading: '告警规则' },
      { path: '/channels', heading: '告警渠道' },
      { path: '/templates', heading: '告警模板' },
      { path: '/history', heading: '告警历史' },
      { path: '/statistics', heading: '告警统计' },
      { path: '/sla', heading: 'SLA配置管理' },
      { path: '/sla-breaches', heading: 'SLA违约详情' },
      { path: '/oncall', heading: '值班管理' },
      { path: '/oncall/report', heading: '值班报告' },
      { path: '/correlation', heading: '告警关联分析' },
      { path: '/escalations', heading: '升级历史' },
      { path: '/tickets', heading: '工单管理' },
      { path: '/silences', heading: '告警静默' },
      { path: '/data-sources', heading: '数据源管理' },
      { path: '/users', heading: '用户管理' },
      { path: '/audit-logs', heading: '审计日志' },
      { path: '/settings', heading: '系统设置' },
    ];

    for (const entry of pages) {
      await page.goto(entry.path);
      await expect(page.getByText(entry.heading, { exact: false }).first()).toBeVisible();
    }
  });

  test('dashboard tables render', async ({ page }) => {
    await login(page);
    await page.goto('/');
    await expect(page.getByRole('heading', { name: '仪表盘' })).toBeVisible();
    await expect(page.getByText('最近告警')).toBeVisible();
    await expect(page.getByRole('table').first()).toBeVisible();
  });

  test('statistics tables render', async ({ page }) => {
    await login(page);
    await page.goto('/statistics');
    await expect(page.getByRole('heading', { name: '告警统计' })).toBeVisible();
    await expect(page.getByText('每日告警趋势')).toBeVisible();
    await expect(page.getByRole('table').first()).toBeVisible();
  });
});
