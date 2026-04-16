import { test, expect } from '@playwright/test';

test.describe('Catalog E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Log in as buyer
    await page.goto('/login');
    await page.fill('input[type="text"]', 'buyer1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');
  });

  test('catalog page displays seeded collectibles', async ({ page }) => {
    await page.click('text=Catalog');
    await expect(page.getByText('Rare Digital Dragon #001')).toBeVisible();
    await expect(page.getByText('Cyber Punk Portrait #042')).toBeVisible();
    await expect(page.getByText('Abstract Waves #007')).toBeVisible();
  });

  test('clicking a collectible navigates to detail page', async ({ page }) => {
    await page.click('text=Catalog');
    await page.click('text=Rare Digital Dragon #001');
    await page.waitForURL(/\/catalog\//);
    await expect(page.getByText('Rare Digital Dragon #001')).toBeVisible();
  });

  test('collectible detail shows price and description', async ({ page }) => {
    await page.click('text=Catalog');
    await page.click('text=Rare Digital Dragon #001');
    await expect(page.getByText('$999.00')).toBeVisible();
    await expect(page.getByText('fire breathing animation')).toBeVisible();
  });
});
