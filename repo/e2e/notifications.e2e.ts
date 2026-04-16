import { test, expect } from '@playwright/test';

test.describe('Notifications E2E', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'buyer1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');
  });

  test('notifications page loads', async ({ page }) => {
    await page.click('text=Notifications');
    await expect(page.getByText('Notifications')).toBeVisible();
  });

  test('notification preferences page loads', async ({ page }) => {
    await page.goto('/notifications/prefs');
    await expect(page.getByText(/preferences|subscription/i)).toBeVisible();
  });
});

test.describe('Role-Based Access E2E', () => {
  test('buyer cannot access analytics', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'buyer1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/analytics');
    // Should redirect away since buyer lacks admin/analyst role
    await expect(page).not.toHaveURL('/analytics');
  });

  test('analyst can access analytics', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'analyst1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/analytics');
    await expect(page.getByText(/analytics|funnel|retention/i)).toBeVisible();
  });
});
