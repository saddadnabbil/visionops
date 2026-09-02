const { test, expect } = require('@playwright/test');

async function login(page, role) {
  await page.goto('/login');
  await page.getByRole('button', { name: new RegExp(`^${role}$`, 'i') }).click();
  await page.getByRole('button', { name: 'Sign in' }).click();
}

test('incident filter is a custom accessible menu rather than a native select', async ({ page }) => {
  await login(page, 'Operator');
  await expect(page.locator('#incident-filter')).toHaveAttribute('aria-haspopup', 'listbox');
  await expect(page.locator('select')).toHaveCount(0);
  await page.locator('#incident-filter').click();
  await expect(page.getByRole('option', { name: 'Resolved' })).toBeVisible();
  await page.getByRole('option', { name: 'Resolved' }).click();
  await expect(page.locator('#incident-filter')).toContainText('Resolved');
});
