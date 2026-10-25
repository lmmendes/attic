import { expect, test, type Page } from '@playwright/test'

async function mockApp(page: Page, settings: (route: import('@playwright/test').Route) => Promise<void>) {
  await page.route('**/auth/session', route => route.fulfill({ json: {
    authenticated: true, auth_disabled: true, user: { name: 'Reviewer', email: 'review@example.test', role: 'admin' }
  } }))
  await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/organization/settings') return settings(route)
    if (path === '/api/organization/features') return route.fulfill({ json: {
      locations: true, collections: true, tags: true, categories: true, attributes: true, conditions: true, warranties: true, plugins: true
    } })
    if (path === '/api/assets/stats') return route.fulfill({ json: { total_assets: 0, total_quantity: 0, total_value: 0.5 } })
    if (path === '/api/assets') return route.fulfill({ json: { assets: [], total: 0 } })
    await route.fulfill({ json: path === '/api/' ? { status: 'ok', version: 'test' } : [] })
  })
}

test('hides price-dependent content after a currency load failure and retries with the saved currency', async ({ page }) => {
  let fail = true
  await mockApp(page, route => route.fulfill(fail ? { status: 503, json: { error: 'offline' } } : { json: { currency: 'EUR' } }))
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Currency settings could not be loaded', exact: true })).toBeVisible()
  await expect(page.getByText('$1', { exact: true })).toHaveCount(0)
  await expect(page.getByText('€0.5', { exact: true })).toHaveCount(0)
  fail = false
  await page.getByRole('button', { name: 'Try again', exact: true }).click()
  await expect(page.getByText('€0.5', { exact: true })).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)
})

test('shares the pending settings load between the app and Settings page', async ({ page }) => {
  let reads = 0
  let release!: (settings: { currency: string }) => void
  const pending = new Promise<{ currency: string }>((resolve) => {
    release = resolve
  })
  await mockApp(page, async (route) => {
    reads++
    await route.fulfill({ json: await pending })
  })
  await page.goto('/settings')
  const picker = page.getByRole('button', { name: 'Currency', exact: true })
  await expect(picker).toBeVisible()
  await expect(picker).toBeDisabled()
  release({ currency: 'EUR' })
  await expect(picker).toBeEnabled()
  await expect(picker).toHaveText('EUR')
  expect(reads).toBe(1)
})
