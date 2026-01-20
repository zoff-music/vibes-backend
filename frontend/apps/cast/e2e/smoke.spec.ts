import { expect, test } from '@playwright/test';

test('has title', async ({ page }) => {
  await page.goto('/');

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle(/Vibez/);
});

test('shows ready status', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('Ready for Casting')).toBeVisible();
});
