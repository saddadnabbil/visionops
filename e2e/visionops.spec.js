const { test, expect } = require('@playwright/test');

async function login(page, role) {
  await page.goto('/login');
  await page.getByRole('button', { name: new RegExp(`^${role}$`, 'i') }).click();
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.locator('#app')).toBeVisible();
}

async function navigate(page, name) {
  const menu = page.locator('#menu-toggle');
  if (await menu.isVisible()) await menu.click();
  await page.getByRole('link', { name, exact: true }).click();
}

test.describe('role-specific safety operations', () => {
  test('public root is a landing page with a clear path into the workspace', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'See what needs attention.' })).toBeVisible();
    await expect(page.locator('.landing-nav')).toHaveCSS('display', 'flex');
    await expect(page.locator('.landing-hero')).toHaveCSS('padding-top', /\d+px/);
    await page.getByRole('link', { name: 'Enter workspace' }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole('heading', { name: 'Enter your workspace' })).toBeVisible();
  });

  test('each demo role enters its intended landing and sees only its routes', async ({ page }) => {
    await login(page, 'Admin');
    await expect(page.getByRole('heading', { name: 'Keep operations accountable.' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Organization' })).toBeVisible();
    await page.getByRole('button', { name: 'Sign out' }).click();

    await login(page, 'Operator');
    await expect(page.getByRole('heading', { name: 'Incidents, ready for action.' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Organization' })).toHaveCount(0);
    await page.getByRole('button', { name: 'Sign out' }).click();

    await login(page, 'Supervisor');
    await expect(page.getByRole('heading', { name: 'See the safety pattern.' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Analytics' })).toBeVisible();
    await page.getByRole('button', { name: 'Sign out' }).click();

    await login(page, 'Viewer');
    await expect(page.getByRole('heading', { name: 'See what needs attention.' })).toBeVisible();
    await expect(page.getByRole('button', { name: /simulate detection/i })).toHaveCount(0);
  });

  test('operator can create, acknowledge, and resolve an incident with a note', async ({ page }) => {
    await login(page, 'Operator');
    await page.getByRole('button', { name: /simulate detection/i }).click();
    await expect(page.getByText('Detection accepted and queued for delivery.')).toBeVisible();
    const incident = page.locator('[data-incident-id]').first();
    await expect(incident).toBeVisible();
    await incident.click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('dialog').getByRole('button', { name: 'Acknowledge', exact: true }).click();
    await expect(page.getByText('Incident acknowledged.')).toBeVisible();
    await page.locator('[data-incident-id]').first().click();
    await page.getByLabel('Operator note').fill('PPE was supplied and the operator confirmed compliance.');
    await page.getByRole('button', { name: 'Resolve incident' }).click();
    await expect(page.getByText('Incident resolved.')).toBeVisible();
  });

  test('viewer can inspect but cannot mutate an incident', async ({ page }) => {
    await login(page, 'Viewer');
    await navigate(page, 'Incidents');
    const incident = page.locator('[data-incident-id]').first();
    await expect(incident).toBeVisible();
    await incident.click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('dialog').getByRole('button', { name: 'Acknowledge', exact: true })).toHaveCount(0);
    await expect(page.getByRole('dialog').getByRole('button', { name: 'Resolve incident', exact: true })).toHaveCount(0);
  });

  test('camera route discloses the recorded scenario and serves the wide worksite preview', async ({ page }) => {
    await login(page, 'Viewer');
    await navigate(page, 'Cameras');
    await expect(page.getByText('RECORDED SCENARIO / SIMULATED DETECTOR — NOT LIVE CCTV')).toBeVisible();
    const video = page.getByLabel('Recorded worksite scenario preview');
    await expect(video).toBeVisible();
    await expect(video.locator('source')).toHaveAttribute('src', '/assets/demo/construction-site-wide-recorded-scenario.mp4');
  });

  test('admin can reach organization configuration', async ({ page }) => {
    await login(page, 'Admin');
    await navigate(page, 'Organization');
    await expect(page.getByRole('heading', { name: 'Configure the workspace.' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'API keys' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Webhook subscriptions' })).toBeVisible();
  });

  test('admin can configure a user, one-time ingest key, webhook, and camera', async ({ page }) => {
    const suffix = `${Date.now()}-${Math.floor(Math.random() * 10_000)}`;
    await login(page, 'Admin');
    await navigate(page, 'Organization');

    await page.getByRole('button', { name: 'Add user' }).click();
    await page.locator('#user-form').getByLabel('Email').fill(`operator-${suffix}@acme.test`);
    await page.locator('#user-form').getByLabel('Temporary password').fill('safe-demo-password');
    await page.locator('#user-role').click();
    await page.locator('#user-form [data-select-option="operator"]').click();
    await page.getByRole('button', { name: 'Create user' }).click();
    await expect(page.getByText('Configuration saved.')).toBeVisible();

    await page.getByRole('button', { name: 'Create key' }).click();
    await page.locator('#key-form').getByLabel('Key name').fill(`browser-e2e-${suffix}`);
    await page.getByRole('button', { name: 'Create one-time key' }).click();
    await expect(page.getByText('Copy this key now')).toBeVisible();
    await expect(page.locator('.secret-callout code')).toContainText('vo_');

    await page.getByRole('button', { name: 'Add webhook' }).click();
    await page.locator('#webhook-form').getByLabel('HTTPS URL').fill(`https://example.test/visionops-${suffix}`);
    await page.locator('#webhook-form').getByLabel('Signing secret').fill('browser-e2e-signing-secret');
    await page.getByRole('button', { name: 'Create subscription' }).click();
    await expect(page.getByText('Configuration saved.')).toBeVisible();

    await navigate(page, 'Cameras');
    await page.getByRole('button', { name: 'Add camera' }).click();
    await page.locator('#camera-form').getByLabel('Name').fill(`Browser Camera ${suffix}`);
    await page.locator('#camera-form').getByLabel('Location').fill('Browser E2E location');
    await page.getByRole('button', { name: 'Create camera' }).click();
    await expect(page.getByText(`Browser Camera ${suffix}`)).toBeVisible();
  });
});
