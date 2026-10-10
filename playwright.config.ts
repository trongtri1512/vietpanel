import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:2083',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'admin',
      use: { ...devices['Desktop Chrome'], baseURL: 'http://localhost:2087' },
    },
  ],
  webServer: [
    {
      command: 'cd panel && DATA_DIR=/tmp/vietpanel-test DATA_DIR_SQLITE=true go run ./cmd/panel/',
      port: 2083,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'cd admin && DATA_DIR=/tmp/vietadmin-test go run ./cmd/admin/',
      port: 2087,
      reuseExistingServer: !process.env.CI,
    },
  ],
});