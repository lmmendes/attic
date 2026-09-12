import { expect, test } from '@playwright/test'

const runID = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
const loginEmail = process.env.E2E_EMAIL || 'admin'
const loginPassword = process.env.E2E_PASSWORD || 'admin'

async function signIn(page: import('@playwright/test').Page) {
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill(loginEmail)
  await page.getByPlaceholder('Enter your password').fill(loginPassword)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(url => url.pathname === '/')
  await expect(page.getByRole('link', { name: 'All Assets', exact: true })).toBeVisible()
}

async function selectOption(
  page: import('@playwright/test').Page,
  label: string,
  option: string
) {
  const trigger = page.getByRole('button', { name: label, exact: true })
  await trigger.click()
  await page.getByRole('option', { name: option, exact: true }).click()
  await page.keyboard.press('Escape')
  await expect(trigger).toContainText(option)
}

test.describe('core browser workflows', () => {
  test('creates, edits, and searches collections', async ({ page }) => {
    const collectionName = `E2E Games ${runID}`
    const updatedName = `${collectionName} Updated`

    await signIn(page)
    await page.goto('/collections')
    await page.getByRole('button', { name: 'New collection' }).click()
    await expect(page.getByRole('heading', { name: 'New collection' })).toBeVisible()
    await page.getByPlaceholder('e.g. PS5 games').fill(collectionName)
    await page.getByPlaceholder('What belongs in this collection?').fill('Browser test collection')
    await page.getByRole('button', { name: 'Create collection' }).click()

    await expect(page.getByText('Collection created', { exact: true })).toBeVisible()
    const collection = page.getByRole('listitem').filter({ hasText: collectionName })
    await expect(collection).toBeVisible()

    await collection.getByRole('button', { name: `Edit ${collectionName}` }).click()
    await expect(page.getByRole('heading', { name: 'Edit collection' })).toBeVisible()
    await page.getByPlaceholder('e.g. PS5 games').fill(updatedName)
    await page.getByPlaceholder('What belongs in this collection?').fill('Updated browser test collection')
    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect(page.getByText('Collection updated', { exact: true })).toBeVisible()
    await expect(page.getByText(updatedName, { exact: true })).toBeVisible()

    await page.getByPlaceholder('Search collections').fill('Updated browser test')
    await expect(page.getByRole('listitem')).toHaveCount(1)
    await expect(page.getByText(updatedName, { exact: true })).toBeVisible()
  })

  test('creates, edits, and searches attributes', async ({ page }) => {
    const attributeName = `E2E Serial ${runID}`
    const updatedName = `${attributeName} Updated`

    await signIn(page)
    await page.goto('/attributes/new')
    await page.getByPlaceholder('e.g. Purchase Date').fill(attributeName)
    await page.getByRole('button', { name: 'Save Attribute' }).click()

    await expect(page).toHaveURL(/\/attributes$/)
    const attribute = page.getByRole('article').filter({ hasText: attributeName })
    await expect(attribute).toBeVisible()

    await attribute.getByRole('link', { name: 'Edit' }).click()
    await expect(page).toHaveURL(/\/attributes\/[^/]+\/edit$/)
    await page.getByPlaceholder('e.g. Purchase Date').fill(updatedName)
    await page.getByRole('button', { name: 'Save Changes' }).click()

    await expect(page).toHaveURL(/\/attributes$/)
    await expect(page.getByText('Attribute updated successfully', { exact: true })).toBeVisible()
    await expect(page.getByText(updatedName, { exact: true })).toBeVisible()

    await page.getByPlaceholder('Search name or key').fill('Updated')
    await expect(page.getByRole('article')).toHaveCount(1)
    await expect(page.getByText(updatedName, { exact: true })).toBeVisible()
  })

  test('creates, edits, searches, and filters assets', async ({ page }) => {
    const assetName = `E2E Camera ${runID}`
    const updatedName = `${assetName} Updated`

    await signIn(page)
    await page.goto('/assets/new')
    await page.getByPlaceholder('e.g. Vintage Canon AE-1').fill(assetName)
    await page.getByPlaceholder('Product description, features, specifications...').fill('Created by browser integration tests')
    await page.getByRole('button', { name: 'Save Asset' }).first().click()

    await expect(page).toHaveURL(/\/assets\/[^/]+$/)
    await expect(page.getByRole('heading', { name: assetName })).toBeVisible()

    await page.getByRole('link', { name: 'Edit Asset' }).click()
    await expect(page).toHaveURL(/\/assets\/[^/]+\/edit$/)
    await page.getByPlaceholder('e.g. Vintage Canon AE-1').fill(updatedName)
    await page.getByPlaceholder('Product description, features, specifications...').fill('Updated by browser integration tests')
    await page.getByRole('button', { name: 'Save Changes' }).first().click()

    await expect(page).toHaveURL(/\/assets\/[^/]+$/)
    await expect(page.getByRole('heading', { name: updatedName })).toBeVisible()

    await page.goto('/assets')
    const asset = page.getByRole('row').filter({ hasText: updatedName })
    await expect(asset).toBeVisible()
    await page.getByPlaceholder('Search by name, tag, or serial number...').fill('Updated by browser')
    await expect(page.getByRole('row').filter({ hasText: updatedName })).toHaveCount(1)

    await page.getByPlaceholder('Search by name, tag, or serial number...').fill('does-not-exist')
    await expect(page.getByText('No assets match these filters')).toBeVisible()
  })

  test('creates, edits, and searches users as an administrator', async ({ page }) => {
    const email = `e2e-${runID}@example.com`
    const name = `E2E Member ${runID}`
    const updatedName = `${name} Updated`

    await signIn(page)
    await page.goto('/users')
    await page.getByRole('button', { name: 'Add person' }).click()
    await expect(page.getByRole('heading', { name: 'Add a person' })).toBeVisible()
    await page.getByPlaceholder('user@example.com').fill(email)
    await page.getByPlaceholder('e.g. John Doe').fill(name)
    await page.getByPlaceholder('At least 8 characters').fill('e2e-password')
    await page.getByRole('button', { name: 'Add person' }).click()

    await expect(page.getByText('User created successfully', { exact: true })).toBeVisible()
    let user = page.getByRole('row').filter({ hasText: email })
    await expect(user).toBeVisible()

    await user.getByTitle('Edit User').click()
    await expect(page.getByRole('heading', { name: `Edit ${name}` })).toBeVisible()
    await page.getByPlaceholder('e.g. John Doe').fill(updatedName)
    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect(page.getByText('User updated successfully', { exact: true })).toBeVisible()
    user = page.getByRole('row').filter({ hasText: email })
    await expect(user).toContainText(updatedName)

    await page.getByPlaceholder('Search name or email').fill('Updated')
    await expect(page.getByRole('row').filter({ hasText: email })).toHaveCount(1)
    await page.getByPlaceholder('Search name or email').fill('not-a-real-user')
    await expect(page.getByText('No results found')).toBeVisible()
  })

  test('assigns a collection while creating and editing an asset', async ({ page }) => {
    const collectionName = `E2E Assignment ${runID}`
    const assetName = `E2E Assigned Asset ${runID}`

    await signIn(page)
    await page.goto('/collections')
    await page.getByRole('button', { name: 'New collection' }).click()
    await page.getByPlaceholder('e.g. PS5 games').fill(collectionName)
    await page.getByRole('button', { name: 'Create collection' }).click()
    await expect(page.getByText(collectionName, { exact: true })).toBeVisible()

    await page.goto('/assets/new')
    await page.getByPlaceholder('e.g. Vintage Canon AE-1').fill(assetName)
    await selectOption(page, 'Collections', collectionName)
    await page.getByRole('button', { name: 'Save Asset' }).first().click()

    await expect(page).toHaveURL(/\/assets\/[^/]+$/)
    await expect(page.getByRole('heading', { name: assetName })).toBeVisible()
    await expect(page.getByText(collectionName, { exact: true })).toBeVisible()

    await page.getByRole('link', { name: 'Edit Asset' }).click()
    await expect(page).toHaveURL(/\/assets\/[^/]+\/edit$/)
    await expect(page.getByRole('button', { name: 'Collections', exact: true })).toContainText(collectionName)
    await page.getByRole('button', { name: 'Save Changes' }).first().click()
    await expect(page).toHaveURL(/\/assets\/[^/]+$/)
    await expect(page.getByText(collectionName, { exact: true })).toBeVisible()
  })
})
