import { expect, test } from '@playwright/test'
import type { Asset, Location } from '../app/types/api'

// Mock read responses so both pages render identical records without creating data.
test('dashboard and location cards share metadata and responsive styling', async ({ page }, testInfo) => {
  await page.goto('/login')
  await page.getByPlaceholder('Enter your email').fill(process.env.E2E_EMAIL || 'admin')
  await page.getByPlaceholder('Enter your password').fill(process.env.E2E_PASSWORD || 'admin')
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page).toHaveURL(url => url.pathname === '/')
  const timestamp = new Date().toISOString()
  const location: Location = { id: '11111111-1111-4111-8111-111111111111', organization_id: 'org', name: 'Card preview room', created_at: timestamp, updated_at: timestamp }
  const assets: Asset[] = [
    {
      id: '22222222-2222-4222-8222-222222222222', organization_id: 'org',
      name: 'A camera with a deliberately long name for responsive card layouts', quantity: 1,
      created_at: timestamp, updated_at: timestamp, location,
      category: { id: 'category', organization_id: 'org', name: 'Electronics', created_at: timestamp, updated_at: timestamp },
      collections: ['Photography equipment with a long collection name', 'Travel', 'Favorites'].map((name, index) => ({ id: `collection-${index}`, organization_id: 'org', name, icon: 'i-lucide-library', asset_count: 1, created_at: timestamp, updated_at: timestamp })),
      main_attachment_url: 'data:image/svg+xml,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200"><rect width="400" height="200" fill="#dbeafe"/><rect x="140" y="65" width="120" height="80" rx="12" fill="#334155"/><circle cx="200" cy="105" r="25" fill="#60a5fa"/></svg>')
    },
    { id: '33333333-3333-4333-8333-333333333333', organization_id: 'org', name: 'Asset without optional metadata', quantity: 1, created_at: timestamp, updated_at: timestamp }
  ]
  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url())
    if (route.request().method() !== 'GET') return route.continue()
    if (url.pathname === '/api/assets') return route.fulfill({ json: { assets, total: assets.length, limit: 20, offset: 0 } })
    if (url.pathname === '/api/locations') return route.fulfill({ json: [location] })
    if (url.pathname === `/api/locations/${location.id}/photos`) return route.fulfill({ json: [] })
    return route.continue()
  })
  for (const viewport of [{ width: 1440, height: 1000 }, { width: 393, height: 844 }]) {
    await page.setViewportSize(viewport)
    for (const theme of ['light', 'dark']) {
      await page.emulateMedia({ colorScheme: theme as 'light' | 'dark' })
      const dashboardCards: Array<{ text: string, classes: string }> = []
      for (const path of ['/', '/locations']) {
        await page.goto(path)
        // Set the theme explicitly to cover both modes regardless of saved preference.
        await page.evaluate((value) => {
          document.documentElement.classList.toggle('dark', value === 'dark')
          document.documentElement.classList.toggle('light', value === 'light')
        }, theme)
        if (path === '/locations') {
          await page.getByRole('button', { name: `View ${location.name}`, exact: true }).click()
        }
        for (const [index, asset] of assets.entries()) {
          const card = page.locator(`a[href="/assets/${asset.id}"]`)
          await expect(card).toContainText(asset.name)
          await expect(card).toContainText('Today')
          if (index === 0) {
            await expect(card).toContainText('Card preview room')
            await expect(card).toContainText('+2')
            await expect(card.locator('img')).toHaveAttribute('alt', asset.name)
            await expect.poll(() => card.locator('img').evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true)
          } else {
            await expect(card).toContainText('Uncategorized')
            await expect(card).toContainText('No location')
            await expect(card).toContainText('No collections')
            await expect(card.locator('img')).toHaveCount(0)
          }
          if (path === '/') dashboardCards[index] = { text: (await card.textContent())!, classes: (await card.getAttribute('class'))! }
          else {
            expect(await card.textContent()).toBe(dashboardCards[index]!.text)
            expect(await card.getAttribute('class')).toBe(dashboardCards[index]!.classes)
          }
          const box = await card.boundingBox()
          expect(box!.x).toBeGreaterThanOrEqual(0)
          expect(box!.x + box!.width).toBeLessThanOrEqual(viewport.width)
          expect(await card.evaluate(element => element.scrollWidth <= element.clientWidth + 1)).toBe(true)
        }
        await page.locator(`a[href="/assets/${assets[0]!.id}"]`).scrollIntoViewIfNeeded()
        await page.screenshot({ path: testInfo.outputPath(`${path === '/' ? 'dashboard' : 'locations'}-${theme}-${viewport.width}.png`) })
      }
    }
  }
})
