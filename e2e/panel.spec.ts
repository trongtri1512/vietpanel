import { test, expect } from '@playwright/test';

test.describe('User Panel', () => {
  test('login page renders', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('h1')).toContainText('VietPanel');
  });

  test('login with demo user', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="username"]', 'demo');
    await page.fill('input[name="password"]', 'demo123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL('/dashboard');
    await expect(page.locator('h2')).toContainText('Welcome back');
  });

  test('dashboard renders after login', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="username"]', 'demo');
    await page.fill('input[name="password"]', 'demo123');
    await page.click('button[type="submit"]');
    await expect(page.locator('h2')).toContainText('Welcome back');
  });

  test('websites page requires auth', async ({ page }) => {
    await page.goto('/websites');
    await expect(page).toHaveURL(/\/login/);
  });

  test('filemanager page', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="username"]', 'demo');
    await page.fill('input[name="password"]', 'demo123');
    await page.click('button[type="submit"]');
    await page.goto('/files');
    await expect(page.locator('h2')).toContainText('Files');
  });
});