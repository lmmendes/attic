import { expect, test, type Page } from '@playwright/test'

async function mockAccount(page: Page, options: { role?: 'user' | 'admin', oidc?: boolean, authDisabled?: boolean } = {}) {
  const state = { enabled: false, failed: false, pending: undefined as Promise<void> | undefined }
  await page.route('**/auth/session', route => route.fulfill({ json: {
    authenticated: true, auth_disabled: options.authDisabled || false, oidc_enabled: options.oidc || false,
    user: { id: 'profile-user', email: 'alex@example.test', name: 'Alex Morgan', role: options.role || 'user' }
  } }))
  await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/organization/mcp') {
      if (state.pending) await state.pending
      return state.failed ? route.fulfill({ status: 503, json: { error: 'Unavailable' } }) : route.fulfill({ json: { enabled: state.enabled } })
    }
    if (path === '/api/organization/features') return route.fulfill({ json: { locations: true, collections: true, tags: true, categories: true, attributes: true, conditions: true, warranties: true, plugins: true } })
    if (path === '/api/organization/settings') return route.fulfill({ json: { currency: 'USD' } })
    if (path === '/api/organization/settings/currencies') return route.fulfill({ json: { currencies: ['USD', 'EUR'] } })
    if (path === '/api/me/api-keys/permissions') return route.fulfill({ json: [{ resource: 'assets', label: 'Assets', levels: ['read', 'write'] }] })
    return route.fulfill({ json: path === '/api/' ? { status: 'ok', version: 'test' } : [] })
  })
  return state
}

test('profile tabs support direct links, browser history, and the account menu', async ({ page }) => {
  await mockAccount(page)
  await page.goto('/profile')
  await expect(page.getByRole('heading', { name: 'Profile', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Account security' })).toBeVisible()
  await page.getByRole('button', { name: 'Open account menu' }).first().click()
  await expect(page.getByRole('menuitem', { name: 'Profile', exact: true })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: 'Change Password', exact: true })).toHaveCount(0)
  await page.keyboard.press('Escape')
  await page.getByRole('tab', { name: 'API keys', exact: true }).click()
  await expect(page).toHaveURL(/tab=api-keys/)
  await expect(page.getByRole('heading', { name: 'No active API keys' })).toBeVisible()
  await page.getByRole('tab', { name: 'MCP connection', exact: true }).click()
  await expect(page).toHaveURL(/tab=mcp/)
  await page.goBack()
  await expect(page.getByRole('tab', { name: 'API keys', exact: true })).toHaveAttribute('aria-selected', 'true')
  await page.goForward()
  await expect(page.getByRole('tab', { name: 'MCP connection', exact: true })).toHaveAttribute('aria-selected', 'true')
  await page.goto('/api-keys')
  await expect(page).toHaveURL(/\/profile\?tab=api-keys$/)
  await page.goto('/profile?tab=invalid')
  await expect(page.getByRole('heading', { name: 'Account security' })).toBeVisible()
  await page.getByRole('tab', { name: 'Account', exact: true }).focus()
  await page.keyboard.press('ArrowRight')
  await expect(page.getByRole('tab', { name: 'API keys', exact: true })).toBeFocused()
})

for (const mode of ['oidc', 'disabled'] as const) {
  test(`profile explains password management for ${mode} authentication`, async ({ page }) => {
    await mockAccount(page, { oidc: mode === 'oidc', authDisabled: mode === 'disabled' })
    await page.goto('/profile')
    await expect(page.getByRole('button', { name: 'Change password', exact: true })).toHaveCount(0)
    await expect(page.getByText(mode === 'oidc' ? /Your password is managed by your sign-in provider/ : /Password changes are unavailable/)).toBeVisible()
  })
}

test('members see MCP availability, collapsed instructions, and recoverable status failures', async ({ page, baseURL }) => {
  const state = await mockAccount(page)
  await page.goto('/profile?tab=mcp')
  await expect(page.getByText('Disabled by your administrator', { exact: true })).toBeVisible()
  await expect(page.getByRole('switch')).toHaveCount(0)
  await expect(page.getByRole('link', { name: 'Manage workspace MCP' })).toHaveCount(0)
  const guide = page.locator('details').filter({ has: page.getByText('View setup instructions', { exact: true }) })
  await expect(guide).not.toHaveAttribute('open')
  await page.getByText('View setup instructions', { exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Add Attic to your assistant' })).toBeVisible()
  state.enabled = true
  await page.getByRole('button', { name: 'Refresh MCP status' }).click()
  await expect(page.getByText('Enabled by your administrator', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Create an API key' })).toBeVisible()
  await page.context().grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.getByRole('button', { name: 'Copy server URL', exact: true }).click()
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(`${baseURL}/mcp`)
  await page.getByText('JSON configuration example', { exact: true }).click()
  await page.getByRole('button', { name: 'Copy client configuration' }).click()
  const config = JSON.parse(await page.evaluate(() => navigator.clipboard.readText()))
  expect(config.mcpServers.attic.headers.Authorization).toBe('Bearer <your-api-key>')
  state.failed = true
  await page.getByRole('button', { name: 'Refresh MCP status' }).click()
  await expect(page.getByText('Could not load MCP status', { exact: true })).toBeVisible()
  await expect(page.getByText('Disabled by your administrator', { exact: true })).toHaveCount(0)
  let release!: () => void
  state.pending = new Promise<void>((resolve) => {
    release = resolve
  })
  state.failed = false
  await page.getByRole('button', { name: 'Retry', exact: true }).click()
  await expect(page.getByText('Checking MCP availability…')).toBeVisible()
  release()
  await expect(page.getByText('Enabled by your administrator', { exact: true })).toBeVisible()
  await page.getByRole('link', { name: 'Create API key', exact: true }).click()
  await expect(page).toHaveURL(/tab=api-keys/)
})

test('admin settings keep the workspace switch and link to personal instructions', async ({ page }) => {
  await mockAccount(page, { role: 'admin' })
  await page.goto('/settings?tab=integrations')
  await expect(page.getByRole('switch', { name: 'Enable MCP server' })).toBeVisible()
  await expect(page.getByText('JSON configuration example', { exact: true })).toHaveCount(0)
  await page.getByRole('link', { name: 'View connection instructions in your profile' }).click()
  await expect(page).toHaveURL(/\/profile\?tab=mcp$/)
  await expect(page.getByRole('link', { name: 'Manage workspace MCP' })).toBeVisible()
})

for (const width of [390, 1440]) {
  for (const theme of ['light', 'dark'] as const) {
    test(`profile is readable at ${width}px in ${theme} mode`, async ({ page }, testInfo) => {
      await mockAccount(page)
      await page.setViewportSize({ width, height: 900 })
      await page.addInitScript(value => localStorage.setItem('nuxt-color-mode', value), theme)
      await page.emulateMedia({ colorScheme: theme })
      for (const tab of ['account', 'api-keys', 'mcp']) {
        await page.goto(`/profile?tab=${tab}`)
        await expect(page.getByRole('tab', { selected: true })).toBeVisible()
        if (tab === 'mcp') await page.getByText('View setup instructions', { exact: true }).click()
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
        const tabs = await page.getByRole('tab').all()
        for (const item of tabs) await expect(item).toBeInViewport()
        await page.screenshot({ path: testInfo.outputPath(`${tab}-${width}-${theme}.png`), fullPage: true })
      }
    })
  }
}
