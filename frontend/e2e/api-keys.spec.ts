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
    await expect(memberRow.getByTestId('user-api-key-usage')).toContainText('Last used just now')

    await memberPage.getByRole('button', { name: `Delete Renamed ${marker}`, exact: true }).click()
    await memberPage.getByRole('dialog', { name: 'Delete API key?', exact: true }).getByRole('button', { name: 'Delete key', exact: true }).click()
    await expect(memberPage.getByRole('heading', { name: 'No active API keys' })).toBeVisible()
    expect((await page.request.get(apiURL('/api/assets'), { headers: { Authorization: `Bearer ${secret}` } })).status()).toBe(401)
    await page.reload()
    await expect(memberRow.getByTestId('user-api-key-usage')).toContainText('0 active keys')
  } finally {
    await page.request.delete(apiURL(`/api/users/${member.id}`))
    await memberContext.close()
  }
})
