import { test, expect } from '@playwright/test';
import dotenv from 'dotenv';

dotenv.config();

test.describe('Devtron Authentication', () => {
  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/login');
    
    await page.fill('input[name="username"]', process.env.ADMIN_USERNAME || 'admin');
    await page.fill('input[name="password"]', process.env.ADMIN_PASSWORD || 'password');
    
    await page.click('button[type="submit"]');
    
    // Wait for dashboard or home page
    await expect(page).toHaveURL(/dashboard|home/);
    await expect(page.locator('nav')).toBeVisible();
  });

  test('should prevent login with invalid credentials', async ({ page }) => {
    await page.goto('/login');
    
    await page.fill('input[name="username"]', 'invalid_user');
    await page.fill('input[name="password"]', 'wrong_password');
    
    await page.click('button[type="submit"]');
    
    // Check for error message
    const errorMessage = page.locator('.error-message');
    await expect(errorMessage).toBeVisible();
    await expect(errorMessage).toContainText('Invalid credentials');
  });

  test('should logout successfully', async ({ page }) => {
    // Perform login first
    await page.goto('/login');
    await page.fill('input[name="username"]', process.env.ADMIN_USERNAME || 'admin');
    await page.fill('input[name="password"]', process.env.ADMIN_PASSWORD || 'password');
    await page.click('button[type="submit"]');
    
    // Navigate to logout
    await page.click('button[data-testid="logout-button"]');
    
    // Verify redirected to login page
    await expect(page).toHaveURL('/login');
  });

  test('should handle password reset', async ({ page }) => {
    await page.goto('/login');
    await page.click('a[href="/forgot-password"]');
    
    await page.fill('input[name="email"]', process.env.ADMIN_USERNAME || 'admin@devtron.ai');
    await page.click('button[type="submit"]');
    
    // Check for password reset confirmation
    const confirmationMessage = page.locator('.reset-confirmation');
    await expect(confirmationMessage).toBeVisible();
    await expect(confirmationMessage).toContainText('Password reset instructions sent');
  });
});