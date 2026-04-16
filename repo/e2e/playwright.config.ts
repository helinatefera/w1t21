import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: 'http://localhost',
    headless: true,
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: 'docker-compose up',
    url: 'http://localhost',
    reuseExistingServer: true,
    timeout: 120000,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
});
