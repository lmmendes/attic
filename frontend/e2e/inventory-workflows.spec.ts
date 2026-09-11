import { expect, test } from '@playwright/test'

const runID = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
const categoryName = `E2E Cameras ${runID}`
const firstAttributeName = `Serial ${runID}`
const secondAttributeName = `Edition ${runID}`

async function signIn(page: import('@playwright/test').Page) {
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill('admin')
  await page.getByPlaceholder('Enter your password').fill('admin')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(/\/$/)
}

test.describe.configure({ mode: 'serial' })

test('creates attributes from category create and edit flows', async ({ page }) => {
  await signIn(page)

  await page.goto('/categories/new')
  await page.getByPlaceholder('e.g. Rare Books').fill(categoryName)
  await page.getByRole('button', { name: 'New Attribute' }).click()
  await expect(page).toHaveURL(/\/attributes\/new/)
  expect(new URL(page.url()).searchParams.get('returnTo')).toBe('/categories/new')

  await page.getByPlaceholder('e.g. Purchase Date').fill(firstAttributeName)
  await page.getByRole('button', { name: 'Save Attribute' }).click()
  await expect(page).toHaveURL(/\/categories\/new\?resume=attribute/)
  await expect(page.getByText(firstAttributeName, { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Save Category' }).click()
  await expect(page).toHaveURL(/\/categories$/)
  await expect(page.getByRole('heading', { name: categoryName })).toBeVisible()

  await page.getByRole('button', { name: `Edit ${categoryName}` }).click()
  await expect(page).toHaveURL(/\/categories\/[^/]+\/edit$/)
  await page.getByRole('button', { name: 'New Attribute' }).click()
  await expect(page).toHaveURL(/\/attributes\/new/)
  expect(new URL(page.url()).searchParams.get('returnTo')).toMatch(/^\/categories\/[^/]+\/edit$/)

  await page.getByPlaceholder('e.g. Purchase Date').fill(secondAttributeName)
  await page.getByRole('button', { name: 'Save Attribute' }).click()
  await expect(page).toHaveURL(/\/categories\/[^/]+\/edit\?resume=attribute/)
  await expect(page.getByText(secondAttributeName, { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Save Changes' }).click()
  await expect(page).toHaveURL(/\/categories$/)
  await expect(page.getByText(categoryName, { exact: true })).toBeVisible()
})

test('applies feature settings to navigation and asset forms', async ({ page }) => {
  await signIn(page)

  await page.goto('/settings')
  const locations = page.getByRole('switch', { name: 'Enable Locations' })
  await expect(locations).toBeChecked()
  await locations.click()
  await expect(locations).not.toBeChecked()
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(page.getByText('Settings saved')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Locations' })).toHaveCount(0)

  await page.goto('/assets/new')
  await expect(page.getByLabel('Stored in')).toHaveCount(0)

  await page.goto('/settings')
  await locations.click()
  await expect(locations).toBeChecked()
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(page.getByText('Settings saved')).toBeVisible()
})
