import { expect, test, type Page, type Route } from '@playwright/test'

async function mockApp(page: Page, settings: (route: Route) => Promise<void>, catalog?: (route: Route) => Promise<void>) {
  await page.route('**/auth/session', route => route.fulfill({ json: {
    authenticated: true, auth_disabled: true, user: { name: 'Reviewer', email: 'review@example.test', role: 'admin' }
  } }))
  await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/organization/settings') return settings(route)
    if (path === '/api/organization/settings/currencies') return catalog ? catalog(route) : route.fulfill({ json: { currencies: ['EUR', 'JPY', 'USD'] } })
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
  let catalogReads = 0
  await mockApp(page, route => route.fulfill(fail ? { status: 503, json: { error: 'offline' } } : { json: { currency: 'EUR' } }), async (route) => {
    catalogReads++
    await route.fulfill({ json: { currencies: ['EUR', 'USD'] } })
  })
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Currency settings could not be loaded', exact: true })).toBeVisible()
  await expect(page.getByText('$1', { exact: true })).toHaveCount(0)
  await expect(page.getByText('€0.5', { exact: true })).toHaveCount(0)
  fail = false
  await page.getByRole('button', { name: 'Try again', exact: true }).click()
  await expect(page.getByText('€0.5', { exact: true })).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)
  expect(catalogReads).toBe(0)
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

test('currency choices can fail and retry without blocking feature saves or changing the saved selection', async ({ page }) => {
  let fail = true
  await mockApp(page, route => route.fulfill({ json: { currency: 'EUR' } }), route => route.fulfill(fail
    ? { status: 503, json: { error: 'offline' } }
    : { json: { currencies: ['EUR', 'USD'] } }))
  await page.goto('/settings')
  await expect(page.getByRole('alert')).toContainText('Currency choices could not be loaded')
  const picker = page.getByRole('button', { name: 'Currency', exact: true })
  await expect(picker).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Save settings', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'Save features', exact: true }).click()
  await expect(page.getByText('Features saved', { exact: true })).toBeVisible()
  fail = false
  await page.getByRole('button', { name: 'Retry currencies', exact: true }).click()
  await expect(picker).toBeEnabled()
  await expect(picker).toHaveText('EUR')
  await page.getByRole('button', { name: 'Save settings', exact: true }).click()
  await expect(page.getByText('General settings saved', { exact: true })).toBeVisible()
})

test('uses server choices and keeps the currency editor disabled until they arrive', async ({ page }) => {
  let release!: () => void
  const pending = new Promise<void>((resolve) => {
    release = resolve
  })
  await mockApp(page, route => route.fulfill({ json: { currency: 'USD' } }), async (route) => {
    await pending
    await route.fulfill({ json: { currencies: ['JPY', 'USD'] } })
  })
  await page.goto('/settings')
  await expect(page.getByRole('status')).toContainText('Loading currency choices')
  const picker = page.getByRole('button', { name: 'Currency', exact: true })
  await expect(picker).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Save settings', exact: true })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Save features', exact: true })).toBeEnabled()
  release()
  await expect(picker).toBeEnabled()
  await picker.click()
  await expect(page.getByRole('option', { name: 'JPY', exact: true })).toBeVisible()
  await expect(page.getByRole('option', { name: 'EUR', exact: true })).toHaveCount(0)
})
