import { expect, test } from '@playwright/test'

const adminEmail = process.env.E2E_EMAIL || 'admin'
const adminPassword = process.env.E2E_PASSWORD || 'admin'
const apiURL = (path: string) => `${process.env.E2E_API_BASE || process.env.E2E_BASE_URL || 'http://127.0.0.1:3000'}${path}`

async function signIn(page: import('@playwright/test').Page, email = adminEmail, password = adminPassword) {
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill(email)
  await page.getByPlaceholder('Enter your password').fill(password)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page).toHaveURL(url => url.pathname === '/')
}

test('settings preserve currency controls alongside MCP integration settings', async ({ page }) => {
  await signIn(page)
  const original = await (await page.request.get(apiURL('/api/organization/mcp'))).json() as { enabled: boolean }
  try {
    await page.goto('/settings')
    await expect(page.getByRole('heading', { name: 'General settings', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Currency', exact: true })).toBeEnabled()
    await page.getByRole('tab', { name: 'Integrations', exact: true }).click()
    const toggle = page.getByRole('switch', { name: 'Enable MCP server', exact: true })
    await expect(toggle).toBeEnabled()
    await toggle.click()
    await expect.poll(async () => (await (await page.request.get(apiURL('/api/organization/mcp'))).json()).enabled).toBe(!original.enabled)
    await page.getByRole('tab', { name: 'Inventory', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Save settings', exact: true })).toBeEnabled()
  } finally {
    expect((await page.request.put(apiURL('/api/organization/mcp'), { data: original })).status()).toBe(200)
  }
})

test('private active keys and aggregate usage in the people list', async ({ page, browser }) => {
  const marker = Date.now().toString()
  const memberEmail = `keys-${marker}@example.test`
  const memberPassword = 'browser-test-password'
  await signIn(page)
  const created = await page.request.post(apiURL('/api/users'), { data: { email: memberEmail, password: memberPassword, name: 'API key tester', role: 'user' } })
  expect(created.status()).toBe(201)
  const member = await created.json() as { id: string }
  const memberContext = await browser.newContext()
  const memberPage = await memberContext.newPage()
  try {
    await signIn(memberPage, memberEmail, memberPassword)
    await memberPage.goto('/api-keys')
    await expect(memberPage.getByRole('heading', { name: 'No active API keys' })).toBeVisible()
    await expect(memberPage.getByRole('textbox', { name: 'Name', exact: true })).toHaveCount(0)
    await memberPage.getByRole('button', { name: 'Create key', exact: true }).click()
    const creation = memberPage.getByRole('dialog', { name: 'Create API key', exact: true })
    await creation.getByRole('textbox', { name: 'Name', exact: true }).fill(`Reader ${marker}`)
    await creation.getByRole('button', { name: 'Customize permissions' }).click()
    await expect(creation.getByRole('combobox', { name: 'Attributes access', exact: true }).locator('option')).toHaveText(['No access', 'Read only'])
    await creation.getByRole('button', { name: 'Create key', exact: true }).click()
    const secret = await memberPage.getByTestId('new-api-key').innerText()
    expect(secret).toMatch(/^attic_[A-Za-z0-9_-]{43}$/)
    const metadata = await memberPage.request.get(apiURL('/api/me/api-keys'))
    const keys = await metadata.json() as Array<{ id: string }>
    expect(keys).toHaveLength(1)
    expect(JSON.stringify(keys)).not.toContain(secret)
    await memberPage.getByRole('button', { name: 'I saved my key' }).click()
    await expect(memberPage.getByTestId('new-api-key')).toHaveCount(0)
    await expect(memberPage.getByRole('cell', { name: 'Never used', exact: true })).toBeVisible()
    const read = await page.request.get(apiURL('/api/assets'), { headers: { Authorization: `Bearer ${secret}` } })
    expect(read.status()).toBe(200)
    const denied = await page.request.post(apiURL('/api/assets'), { headers: { Authorization: `Bearer ${secret}` }, data: { name: 'Denied' } })
    expect(denied.status()).toBe(403)
    await memberPage.getByRole('button', { name: 'Refresh API keys' }).click()
    await expect(memberPage.getByRole('cell', { name: 'Just now', exact: true })).toBeVisible()
    await memberPage.getByRole('button', { name: `Edit Reader ${marker}`, exact: true }).click()
    const edit = memberPage.getByRole('dialog', { name: 'Edit API key', exact: true })
    await edit.getByRole('textbox', { name: 'Name', exact: true }).fill(`Renamed ${marker}`)
    await edit.getByRole('button', { name: 'Save changes' }).click()
    await expect(memberPage.getByText(`Renamed ${marker}`, { exact: true })).toBeVisible()

    await page.goto('/api-keys')
    await expect(page.getByRole('checkbox', { name: 'Show workspace keys' })).toHaveCount(0)
    await expect(page.getByText(`Renamed ${marker}`, { exact: true })).toHaveCount(0)
    expect((await page.request.get(apiURL('/api/organization/api-keys'))).status()).toBe(404)
    expect((await page.request.delete(apiURL(`/api/me/api-keys/${keys[0]!.id}`))).status()).toBe(404)
    await page.goto('/users')
    const memberRow = page.getByRole('row').filter({ hasText: memberEmail })
    await expect(memberRow.getByTestId('user-api-key-usage')).toContainText('1 active key')
    await expect(memberRow.getByTestId('user-api-key-usage')).toContainText('Last used: Just now')

    await memberPage.getByRole('button', { name: `Delete Renamed ${marker}`, exact: true }).click()
    await memberPage.getByRole('dialog', { name: 'Delete API key?', exact: true }).getByRole('button', { name: 'Delete key', exact: true }).click()
    await expect(memberPage.getByRole('heading', { name: 'No active API keys' })).toBeVisible()
    expect((await page.request.get(apiURL('/api/assets'), { headers: { Authorization: `Bearer ${secret}` } })).status()).toBe(401)
    await page.reload()
    await expect(memberRow.getByTestId('user-api-key-usage')).toContainText('0 active keys')

    await memberPage.goto('/profile')
    await memberPage.getByRole('button', { name: 'Change password', exact: true }).click()
    const passwordDialog = memberPage.getByRole('dialog', { name: 'Change password', exact: true })
    await passwordDialog.getByLabel('Current password', { exact: true }).fill('incorrect-password')
    await passwordDialog.getByLabel('New password', { exact: true }).fill('updated-browser-password')
    await passwordDialog.getByLabel('Confirm new password', { exact: true }).fill('updated-browser-password')
    await passwordDialog.getByRole('button', { name: 'Update password', exact: true }).click()
    await expect(passwordDialog.getByText(/current password is incorrect/i)).toBeVisible()
    await passwordDialog.getByLabel('Current password', { exact: true }).fill(memberPassword)
    await passwordDialog.getByRole('button', { name: 'Update password', exact: true }).click()
    await expect(passwordDialog.getByText('Password changed successfully', { exact: true })).toBeVisible()
    await passwordDialog.getByRole('button', { name: 'Done', exact: true }).click()
    await memberPage.getByRole('button', { name: 'Change password', exact: true }).click()
    await expect(passwordDialog.getByLabel('Current password', { exact: true })).toHaveValue('')
    await passwordDialog.getByRole('button', { name: 'Cancel', exact: true }).click()
    expect((await memberPage.request.post(apiURL('/auth/logout'))).ok()).toBe(true)
    await signIn(memberPage, memberEmail, 'updated-browser-password')
  } finally {
    await page.request.delete(apiURL(`/api/users/${member.id}`))
    await memberContext.close()
  }
})
