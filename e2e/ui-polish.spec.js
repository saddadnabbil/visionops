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

test('route navigation keeps the current workspace stable while route data arrives', async ({ page }) => {
  await login(page, 'Admin');
  await expect(page.getByRole('heading', { name: 'Keep operations accountable.' })).toBeVisible();
  await page.route('**/api/v1/jobs', async route => {
    await new Promise(resolve => setTimeout(resolve, 500));
    await route.continue();
  });
  const menu = page.locator('#menu-toggle');
  if (await menu.isVisible()) await menu.click();
  await page.getByRole('link', { name: 'Delivery', exact: true }).click();
  await expect(page.locator('.spinner')).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Every alert, accounted for.' })).toBeVisible();
});
