import { test, expect } from '@playwright/test';

test.describe('Authentication E2E', () => {
  test('login page loads with branding and form', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText('LedgerMint')).toBeVisible();
    await expect(page.getByText('Digital Collectibles Exchange')).toBeVisible();
    await expect(page.getByText('Username')).toBeVisible();
    await expect(page.getByText('Password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible();
  });

  test('admin can log in and see dashboard', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');
    await expect(page.getByText('Dashboard')).toBeVisible();
    await expect(page.getByText('Open Orders')).toBeVisible();
  });

  test('seller can log in and see catalog with Add button', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'seller1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');
    await page.click('text=Catalog');
    await expect(page.getByText('+ Add Collectible')).toBeVisible();
  });

  test('buyer can log in and browse catalog without Add button', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'buyer1');
    await page.fill('input[type="password"]', 'testpass123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/');
    await page.click('text=Catalog');
    await expect(page.getByText('Catalog')).toBeVisible();
    await expect(page.getByText('+ Add Collectible')).not.toBeVisible();
  });

  test('invalid credentials show error message', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');
    await expect(page.getByText(/invalid|failed/i)).toBeVisible();
  });

  test('unauthenticated user is redirected to login', async ({ page }) => {
    await page.goto('/');
    await page.waitForURL('/login');
    await expect(page.getByText('Sign In')).toBeVisible();
  });
});
