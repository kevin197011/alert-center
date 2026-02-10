import { defineConfig } from '@playwright/test';

const baseURL = process.env.UI_BASE_URL || 'http://localhost:3000';

export default defineConfig({
  timeout: 90_000,
  retries: 0,
  use: {
    baseURL,
    headless: true,
    viewport: { width: 1440, height: 900 },
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  testDir: './tests',
  reporter: [['list']],
});
