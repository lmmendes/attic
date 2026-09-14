import { expect, test, type Locator, type Page } from '@playwright/test'

// Point both URLs at an isolated app/database; fixtures are deliberately retained
// for inspecting a failed run and disappear with the disposable database.
const apiBase = process.env.E2E_API_BASE || process.env.E2E_BASE_URL || 'http://127.0.0.1:3000'

async function seed(page: Page) {
  const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill(process.env.E2E_EMAIL || 'admin')
  await page.getByPlaceholder('Enter your password').fill(process.env.E2E_PASSWORD || 'admin')
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page).toHaveURL(url => url.pathname === '/')
  async function create(path: string, data: Record<string, unknown>) {
    const response = await page.request.post(`${apiBase}/api/${path}`, { data })
    expect(response.ok(), await response.text()).toBeTruthy()
    return await response.json() as { id: string }
  }
  const vendorName = `Vendor ${suffix}`
  const key = `vendor_${suffix.replaceAll('-', '_')}`
  const vendor = await create('attributes', { name: vendorName, key, data_type: 'string' })
  const category = await create('categories', { name: `Machines ${suffix}`, attributes: [{ attribute_id: vendor.id, required: false, sort_order: 0 }] })
  const collectionNames = [`Computers ${suffix}`, `Favorites ${suffix}`]
  const collections = []
  for (const name of collectionNames) collections.push(await create('collections', { name, icon: 'i-lucide-library' }))
  const names = [`C64 ${suffix}`, `Amiga ${suffix}`, `Spectrum ${suffix}`]
  for (const [index, name] of names.entries()) {
    await create('assets', {
      name, quantity: 1, category_id: category.id,
      attributes: { [key]: index === 2 ? `Sinclair ${suffix}` : `Commodore ${suffix}` },
      collection_ids: index === 0 ? collections.map(c => c.id) : [collections[index === 1 ? 0 : 1]!.id]
    })
  }
  await page.goto('/assets')
  return { names, vendorName, collectionNames, suffix }
}

async function choose(page: Page, trigger: Locator, label: string) {
  await trigger.click()
  await page.getByRole('option', { name: label, exact: true }).click()
  await expect(page.getByRole('listbox', { includeHidden: true })).toHaveCount(0)
}

async function results(page: Page, names: string[], included: number[]) {
  for (const [index, name] of names.entries()) {
    const asset = page.getByRole('link', { name, exact: true })
    if (included.includes(index)) await expect(asset.first()).toBeVisible()
    else await expect(asset).toHaveCount(0)
  }
}

test('attribute search and nested collection filters survive the saved-filter lifecycle', async ({ page }) => {
  test.setTimeout(120_000)
  const data = await seed(page)
  const quick = page.getByRole('textbox', { name: 'Search attribute values' })
  await quick.fill(`cOmMoDoRe ${data.suffix}`)
  await results(page, data.names, [0, 1])
  await page.reload()
  await expect(quick).toHaveValue(`cOmMoDoRe ${data.suffix}`)
  await results(page, data.names, [0, 1])
  await page.getByRole('button', { name: 'Clear', exact: true }).click()
  await results(page, data.names, [0, 1, 2])

  await page.getByRole('button', { name: 'Advanced filter', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: 'Advanced filter', exact: true })
  await dialog.getByRole('button', { name: 'Add rules', exact: true }).click()
  const rules = dialog.getByRole('group', { name: 'Filter rule', exact: true })
  await choose(page, rules.nth(0).getByLabel('Rule field'), `Attribute: ${data.vendorName}`)
  await rules.nth(0).getByLabel('Rule value', { exact: true }).fill(`Commodore ${data.suffix}`)
  await dialog.getByRole('button', { name: 'Add group', exact: true }).click()
  await choose(page, dialog.getByLabel('Group match').nth(1), 'Match any (OR)')
  await choose(page, rules.nth(1).getByLabel('Rule field'), 'Collections')
  await rules.nth(1).getByLabel('Rule values', { exact: true }).click()
  for (const name of data.collectionNames) await page.getByRole('option', { name, exact: true }).click()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('listbox', { includeHidden: true })).toHaveCount(0)
  // The second OR branch deliberately does not match any seeded asset.
  await dialog.getByRole('button', { name: 'Add rule', exact: true }).nth(1).click()
  await rules.nth(2).getByLabel('Rule value', { exact: true }).fill(`unmatched-${data.suffix}`)
  await expect(rules.nth(0).getByLabel('Rule value', { exact: true })).toHaveValue(`Commodore ${data.suffix}`)
  await expect(rules.nth(2).getByLabel('Rule value', { exact: true })).toHaveValue(`unmatched-${data.suffix}`)
  await dialog.getByRole('button', { name: 'Apply', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  await results(page, data.names, [0, 1])

  await page.getByRole('button', { name: 'Advanced filter', exact: true }).click()
  await choose(page, rules.nth(1).getByLabel('Rule comparison'), 'Includes all')
  const filterName = `Retro ${data.suffix}`
  await dialog.getByLabel('Filter name', { exact: true }).fill(filterName)
  await dialog.getByRole('button', { name: 'Save as new', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  await results(page, data.names, [0])
  await page.reload()
  await results(page, data.names, [0])
  await expect(page.getByLabel('Saved filters', { exact: true })).toContainText(filterName)

  await page.getByRole('button', { name: 'Advanced filter', exact: true }).click()
  await choose(page, rules.nth(1).getByLabel('Rule comparison'), 'Includes any')
  await dialog.getByRole('button', { name: 'Cancel', exact: true }).click()
  await results(page, data.names, [0])
  await page.getByRole('button', { name: 'Advanced filter', exact: true }).click()
  await expect(rules.nth(1).getByLabel('Rule comparison')).toContainText('Includes all')
  await choose(page, rules.nth(1).getByLabel('Rule comparison'), 'Includes any')
  await dialog.getByRole('button', { name: 'Apply', exact: true }).click()
  await results(page, data.names, [0, 1])
  await expect(page.getByText('(modified)', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Clear', exact: true }).click()
  await choose(page, page.getByLabel('Saved filters', { exact: true }), filterName)
  await results(page, data.names, [0])
  await page.getByRole('button', { name: 'Advanced filter', exact: true }).click()
  await choose(page, rules.nth(1).getByLabel('Rule comparison'), 'Includes any')
  await dialog.getByRole('button', { name: 'Update saved filter', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  await page.reload()
  await results(page, data.names, [0, 1])
  await expect(page.getByText('(modified)', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: 'Delete filter', exact: true }).click()
  await page.getByRole('dialog', { name: 'Delete saved filter' }).getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Delete filter', exact: true })).toHaveCount(0)
  await page.reload()
  await page.getByLabel('Saved filters', { exact: true }).click()
  await expect(page.getByRole('option', { name: filterName, exact: true })).toHaveCount(0)
})

test('mobile filter dialog fits the viewport and supports keyboard dismissal and apply', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const data = await seed(page)
  const advanced = page.getByRole('button', { name: 'Advanced filter', exact: true })
  await advanced.focus()
  await page.keyboard.press('Enter')
  const dialog = page.getByRole('dialog', { name: 'Advanced filter', exact: true })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: 'Add rules', exact: true }).click()
  await dialog.getByRole('button', { name: 'Add group', exact: true }).click()
  await expect.poll(() => dialog.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true)
  const bounds = await dialog.boundingBox()
  expect(bounds).not.toBeNull()
  expect(bounds!.x).toBeGreaterThanOrEqual(0)
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390)
  await dialog.getByLabel('Filter name', { exact: true }).focus()
  for (let i = 0; i < 12; i++) {
    await page.keyboard.press('Tab')
    await expect.poll(() => dialog.evaluate(el => el.contains(document.activeElement))).toBe(true)
  }
  await page.keyboard.press('Escape')
  await expect(dialog).not.toBeVisible()
  await expect(advanced).toBeFocused()
  await page.keyboard.press('Enter')
  await dialog.getByRole('button', { name: 'Add rules', exact: true }).click()
  await dialog.getByLabel('Rule value', { exact: true }).fill(data.names[0]!)
  await dialog.getByRole('button', { name: 'Apply', exact: true }).focus()
  await page.keyboard.press('Enter')
  await expect(dialog).not.toBeVisible()
  await results(page, data.names, [0])
})
