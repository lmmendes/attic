import { expect, test } from '@playwright/test'
import type { APIRequestContext, Page } from '@playwright/test'

const runID = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill(process.env.E2E_EMAIL || 'admin')
  await page.getByPlaceholder('Enter your password').fill(process.env.E2E_PASSWORD || 'admin')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(url => url.pathname === '/')
}

async function create(request: APIRequestContext, resource: string, data: Record<string, unknown>) {
  const response = await request.post(`/api/${resource}`, { data })
  expect(response.status(), await response.text()).toBe(201)
  return await response.json() as { id: string }
}

async function getAsset(request: APIRequestContext, id: string) {
  const response = await request.get(`/api/assets/${id}`)
  expect(response.ok()).toBe(true)
  return await response.json() as { id: string, parent_id: string | null, location_id?: string, containment_summary: { total_value: number } }
}

test('upgrades a PC atomically and detaches the replacement from its detail page', async ({ page }) => {
  await signIn(page)
  const ids: string[] = []
  let locationId: string | undefined
  try {
    const garage = await create(page.request, 'locations', { name: `PC garage ${runID}` })
    locationId = garage.id
    const pc = await create(page.request, 'assets', { name: `Gaming PC ${runID}`, location_id: garage.id, purchase_price: 100 })
    ids.push(pc.id)
    const oldGPU = await create(page.request, 'assets', { name: `Old GPU ${runID}`, parent_id: pc.id, purchase_price: 20 })
    ids.push(oldGPU.id)
    const newName = `Replacement GPU ${runID}`
    const newGPU = await create(page.request, 'assets', { name: newName, purchase_price: 30, quantity: 2 })
    ids.push(newGPU.id)

    await page.goto(`/assets/${pc.id}/edit`)
    await page.getByRole('button', { name: `Remove Old GPU ${runID}`, exact: true }).click()
    await page.getByRole('button', { name: 'Link existing asset' }).click()
    await page.getByRole('button', { name: 'Content asset', exact: true }).click()
    await page.getByRole('option', { name: newName, exact: true }).click()
    await page.getByRole('button', { name: 'Stage addition' }).click()
    expect((await getAsset(page.request, oldGPU.id)).parent_id).toBe(pc.id)
    expect((await getAsset(page.request, newGPU.id)).parent_id).toBeNull()
    await page.getByRole('button', { name: 'Save Changes', exact: true }).first().click()
    await expect(page).toHaveURL(url => url.pathname === `/assets/${pc.id}`)
    await expect(page.getByRole('region', { name: 'Contents' }).getByRole('link', { name: newName })).toBeVisible()
    expect((await getAsset(page.request, oldGPU.id)).parent_id).toBeNull()
    expect(await getAsset(page.request, newGPU.id)).toMatchObject({ parent_id: pc.id, location_id: garage.id })
    expect((await getAsset(page.request, pc.id)).containment_summary.total_value).toBe(160)

    await page.goto(`/assets/${newGPU.id}`)
    await expect(page.getByRole('region', { name: 'Inside' }).getByRole('link', { name: `Gaming PC ${runID}` })).toBeVisible()
    await page.getByRole('button', { name: 'Detach', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Inside' })).toHaveCount(0)
    expect(await getAsset(page.request, newGPU.id)).toMatchObject({ parent_id: null, location_id: garage.id })
  } finally {
    for (const id of ids.reverse()) await page.request.delete(`/api/assets/${id}`)
    if (locationId) await page.request.delete(`/api/locations/${locationId}`)
  }
})

test('creates nested contents from a prefilled parent and moves the entire box', async ({ page }) => {
  await signIn(page)
  const ids: string[] = []
  const locations: string[] = []
  try {
    const garage = await create(page.request, 'locations', { name: `Box garage ${runID}` })
    const loftName = `Box loft ${runID}`
    const loft = await create(page.request, 'locations', { name: loftName })
    locations.push(garage.id, loft.id)
    const crate = await create(page.request, 'assets', { name: `Crate ${runID}`, location_id: garage.id })
    ids.push(crate.id)
    const box = await create(page.request, 'assets', { name: `RAM box ${runID}`, parent_id: crate.id })
    ids.push(box.id)
    await page.goto(`/assets/${box.id}`)
    await page.getByRole('link', { name: 'Add asset', exact: true }).click()
    await expect(page).toHaveURL(url => url.searchParams.get('parent_id') === box.id)
    await expect(page.getByText(`Location: Box garage ${runID}`, { exact: false })).toBeVisible()
    await page.getByLabel('Asset Name', { exact: false }).fill(`RAM ${runID}`)
    await page.getByRole('button', { name: 'Save Asset', exact: true }).first().click()
    await expect(page).toHaveURL(/\/assets\/[0-9a-f-]+$/)
    const ramID = new URL(page.url()).pathname.split('/').at(-1)!
    ids.push(ramID)
    expect(await getAsset(page.request, ramID)).toMatchObject({ parent_id: box.id, location_id: garage.id })

    await page.goto(`/assets/${crate.id}`)
    const nestedContents = page.getByRole('list', { name: `Contents of RAM box ${runID}`, exact: true })
    await expect(nestedContents.getByRole('link', { name: `RAM ${runID}`, exact: true })).toBeVisible()
    await page.getByRole('button', { name: `Collapse RAM box ${runID}`, exact: true }).click()
    await expect(nestedContents).toHaveCount(0)
    await page.getByRole('button', { name: `Expand RAM box ${runID}`, exact: true }).click()
    await expect(nestedContents.getByRole('link', { name: `RAM ${runID}`, exact: true })).toBeVisible()
    await page.goto(`/assets/${crate.id}/edit`)
    await page.locator('#location').click()
    await page.getByRole('option', { name: loftName, exact: true }).click()
    await page.getByRole('button', { name: 'Save Changes', exact: true }).first().click()
    await expect(page).toHaveURL(url => url.pathname === `/assets/${crate.id}`)
    for (const id of [crate.id, box.id, ramID]) expect((await getAsset(page.request, id)).location_id).toBe(loft.id)
    await page.goto(`/assets/${box.id}`)
    await page.getByRole('button', { name: 'Detach', exact: true }).click()
    await expect(page.getByRole('region', { name: 'Inside' })).toHaveCount(0)
    expect(await getAsset(page.request, ramID)).toMatchObject({ parent_id: box.id, location_id: loft.id })
  } finally {
    for (const id of ids.reverse()) await page.request.delete(`/api/assets/${id}`)
    for (const id of locations) await page.request.delete(`/api/locations/${id}`)
  }
})
