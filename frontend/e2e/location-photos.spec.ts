import { expect, test } from '@playwright/test'

const apiURL = (path: string) => `${process.env.E2E_API_BASE || process.env.E2E_BASE_URL || 'http://127.0.0.1:3000'}${path}`
const photo = { name: 'box.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j3ioAAAAASUVORK5CYII=', 'base64') }
interface Photo { id: string, file_name: string, is_cover: boolean, url: string }

test('Photos tab supports batches, retry, covers, viewer and cleanup across locations', async ({ page }, testInfo) => {
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill(process.env.E2E_EMAIL || 'admin')
  await page.getByPlaceholder('Enter your password').fill(process.env.E2E_PASSWORD || 'admin')
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page).toHaveURL(url => url.pathname === '/')
  const name = `Photo box ${Date.now()}`
  const ids: string[] = []
  const list = async (id: string): Promise<Photo[]> => (await page.request.get(apiURL(`/api/locations/${id}/photos`))).json()
  try {
    await page.goto('/locations')
    await page.getByRole('button', { name: 'Add Location', exact: true }).first().click()
    const create = page.getByRole('dialog', { name: 'New Location', exact: true })
    await create.getByPlaceholder('Location name').fill(name)
    await expect(create.locator('input[type=file]')).toHaveCount(0)
    await create.getByRole('button', { name: 'Create', exact: true }).click()
    await expect(create).not.toBeVisible()
    const locations = await (await page.request.get(apiURL('/api/locations'))).json() as Array<{ id: string, name: string }>
    const id = locations.find(item => item.name === name)!.id
    ids.push(id)
    const other = await (await page.request.post(apiURL('/api/locations'), { data: { name: `${name} other` } })).json() as { id: string }
    ids.push(other.id)
    await page.reload()
    await page.getByRole('button', { name: `View ${name}`, exact: true }).click()
    await expect(page.getByRole('tab', { name: 'Assets', exact: true })).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('button', { name: 'Add photos', exact: true })).not.toBeVisible()
    await page.getByRole('tab', { name: 'Assets', exact: true }).focus()
    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('tab', { name: 'Photos', exact: true })).toHaveAttribute('aria-selected', 'true')
    const input = page.getByLabel('Add location photos', { exact: true })
    let uploadCount = 0
    let releaseUpload!: () => void
    const gate = new Promise<void>((resolve) => {
      releaseUpload = resolve
    })
    await page.route(`**/api/locations/${id}/photos`, async (route) => {
      if (route.request().method() !== 'POST') return route.continue()
      uploadCount++
      if (uploadCount === 1) await gate
      if (uploadCount === 2) return route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'Please retry this photo' }) })
      return route.continue()
    })
    await expect(input).toBeEnabled()
    await input.setInputFiles([photo, { ...photo, name: 'shelf.png' }, { ...photo, name: 'room.png' }])
    await expect(page.getByText('Uploading 1 of 3 photos…')).toBeVisible()
    await page.getByRole('tab', { name: 'Assets', exact: true }).click()
    await page.getByRole('button', { name: `View ${name} other`, exact: true }).click()
    await page.getByRole('tab', { name: 'Photos', exact: true }).click()
    releaseUpload()
    await expect.poll(async () => (await list(id)).length).toBe(2)
    await expect(page.getByTestId('location-photo')).toHaveCount(0)
    expect(await list(other.id)).toHaveLength(0)
    await page.getByRole('button', { name: `View ${name}`, exact: true }).click()
    await expect(page.getByTestId('location-photo')).toHaveCount(2)
    await expect(page.getByRole('alert')).toContainText('shelf.png')
    await page.getByRole('button', { name: 'Retry failed photos', exact: true }).click()
    await expect(page.getByTestId('location-photo')).toHaveCount(3)
    expect(uploadCount).toBe(4)
    expect((await list(id)).map(item => item.file_name).sort()).toEqual(['box.png', 'room.png', 'shelf.png'])
    for (const item of await list(id)) {
      expect(new URL(item.url).pathname).toMatch(/\/locations\/[0-9a-f-]{36}\/location-photo\.png$/)
    }
    await expect(page.getByRole('alert')).toHaveCount(0)
    await expect.poll(() => page.getByTestId('location-photo').first().evaluate((element: HTMLImageElement) => element.complete && element.naturalWidth > 0)).toBe(true)
    expect((await list(id)).every(item => !item.is_cover)).toBe(true)
    const treeItem = page.getByRole('button', { name: `View ${name}`, exact: true })
    await expect(treeItem.locator('img')).toHaveCount(0)
    await page.getByRole('button', { name: 'Photo actions for shelf.png', exact: true }).click()
    await page.getByRole('menuitem', { name: 'Set as cover', exact: true }).click()
    await expect.poll(async () => (await list(id))[0]?.file_name).toBe('shelf.png')
    await expect(page.locator('article').filter({ hasText: 'shelf.png' })).toContainText('Cover')
    await expect(treeItem.locator('img')).toHaveCount(1)
    await page.getByRole('button', { name: 'Photo actions for shelf.png', exact: true }).click()
    await page.getByRole('menuitem', { name: 'Remove cover', exact: true }).click()
    await expect.poll(async () => (await list(id)).every(item => !item.is_cover)).toBe(true)
    await expect(page.getByTestId('location-photo')).toHaveCount(3)
    await expect(treeItem.locator('img')).toHaveCount(0)
    await page.getByRole('button', { name: 'Photo actions for shelf.png', exact: true }).click()
    await page.getByRole('menuitem', { name: 'Set as cover', exact: true }).click()
    await expect.poll(async () => (await list(id))[0]?.file_name).toBe('shelf.png')
    await page.getByRole('button', { name: 'View photo 1: shelf.png', exact: true }).click()
    await expect(page.getByTestId('location-photo-viewer')).toHaveAttribute('alt', 'shelf.png')
    await page.keyboard.press('ArrowRight')
    await expect(page.getByTestId('location-photo-viewer')).toHaveAttribute('alt', 'box.png')
    await page.keyboard.press('Escape')
    await expect(page.getByTestId('location-photo-viewer')).not.toBeVisible()
    for (const theme of ['light', 'dark']) {
      await page.evaluate((value) => {
        document.documentElement.classList.toggle('dark', value === 'dark')
      }, theme)
      await expect(page.getByRole('button', { name: 'Add photos', exact: true })).toBeVisible()
      await page.screenshot({ path: testInfo.outputPath(`gallery-${theme}.png`) })
    }
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(page.getByRole('tab', { name: 'Photos', exact: true })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await page.getByRole('region', { name: 'Location photos', exact: true }).scrollIntoViewIfNeeded()
    await expect(page.getByRole('button', { name: 'Add photos', exact: true })).toBeInViewport()
    await page.screenshot({ path: testInfo.outputPath('gallery-mobile.png') })
    await page.setViewportSize({ width: 1440, height: 1000 })
    const coverURL = (await list(id))[0]!.url
    await page.getByRole('button', { name: 'Photo actions for shelf.png', exact: true }).click()
    await page.getByRole('menuitem', { name: 'Delete photo', exact: true }).click()
    const confirmation = page.getByRole('dialog', { name: 'Delete photo?', exact: true })
    await confirmation.getByRole('button', { name: 'Delete photo', exact: true }).click()
    await expect(confirmation).not.toBeVisible()
    await expect(page.getByTestId('location-photo')).toHaveCount(2)
    expect((await list(id)).every(item => !item.is_cover)).toBe(true)
    await expect(treeItem.locator('img')).toHaveCount(0)
    expect((await page.request.get(coverURL)).status()).toBe(404)
    await page.reload()
    await page.getByRole('button', { name: `View ${name}`, exact: true }).click()
    await page.getByRole('tab', { name: 'Photos', exact: true }).click()
    await expect(page.getByTestId('location-photo')).toHaveCount(2)
    const remaining = await list(id)
    await page.getByRole('button', { name: 'Delete location', exact: true }).click()
    const deleteLocation = page.getByRole('dialog', { name: 'Delete location?', exact: true })
    await expect(deleteLocation).toContainText('All photos in this location will be permanently deleted.')
    await expect(deleteLocation).toContainText('Assets will stay in your inventory')
    await deleteLocation.getByRole('button', { name: 'Cancel', exact: true }).click()
    expect(await list(id)).toHaveLength(2)
    await page.getByRole('button', { name: 'Delete location', exact: true }).click()
    await deleteLocation.getByRole('button', { name: 'Delete location', exact: true }).click()
    await expect(deleteLocation).not.toBeVisible()
    await expect(page.getByRole('button', { name: `View ${name}`, exact: true })).toHaveCount(0)
    ids.splice(ids.indexOf(id), 1)
    for (const item of remaining) expect((await page.request.get(item.url)).status()).toBe(404)
  } finally {
    for (const id of ids) await page.request.delete(apiURL(`/api/locations/${id}`))
  }
})
