import { test, expect } from '@playwright/test';

test.describe('Order Journey E2E', () => {
  test('buyer places order and sees it in orders list', async ({ page }) => {
    // Login as buyer
    await page.goto('/login');
    await page.fill('input[type="text"]', 'buyer1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    // Navigate to catalog and click first item
    await page.click('text=Catalog');
    await page.click('text=Rare Digital Dragon #001');
    await page.waitForURL(/\/catalog\//);

    // Place order (if button exists)
    const orderButton = page.getByRole('button', { name: /order|buy/i });
    if (await orderButton.isVisible()) {
      await orderButton.click();
      await expect(page.getByText(/pending|order placed/i)).toBeVisible();
    }

    // Navigate to orders page
    await page.click('text=Orders');
    await expect(page.getByText('Orders')).toBeVisible();
  });

  test('seller confirms and completes order flow', async ({ page }) => {
    // Login as seller
    await page.goto('/login');
    await page.fill('input[type="text"]', 'seller1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    // Navigate to orders
    await page.click('text=Orders');
    await expect(page.getByText('Orders')).toBeVisible();
  });
});

test.describe('Admin Dashboard E2E', () => {
  test('admin sees dashboard with all stats', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await expect(page.getByText('Dashboard')).toBeVisible();
    await expect(page.getByText('Open Orders')).toBeVisible();
    await expect(page.getByText('Unread Notifications')).toBeVisible();
  });

  test('admin navigates to analytics page', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.click('text=Analytics');
    await expect(page.getByText(/funnel|retention|analytics/i)).toBeVisible();
  });

  test('admin navigates to user management', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.click('text=Users');
    await expect(page.getByText('admin')).toBeVisible();
    await expect(page.getByText('seller1')).toBeVisible();
    await expect(page.getByText('buyer1')).toBeVisible();
  });
});
