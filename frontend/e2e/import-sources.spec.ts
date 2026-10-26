import { expect, test, type Locator } from '@playwright/test'
import type { Plugin } from '../app/types/api'

function source(id: string, category: string, enabled = true): Plugin {
  return {
    id, name: id, category_name: category, category_description: `${category} catalog`,
    description: `Import ${category} from this catalog`, enabled,
    category_id: enabled ? `category-${id}` : undefined,
    disabled_reason: enabled ? undefined : 'Missing API key: ATTIC_TMDB_API_KEY',
    search_fields: [{ key: 'name', label: 'Name' }],
    attributes: [{ key: 'catalog_number', name: 'Catalog Number', data_type: 'string', required: false }]
  }
}

const sources = [
  source('rebrickable_sets', 'LEGO Sets'), source('tmdb_series', 'TV Series', false),
  source('igdb_games', 'Video Games'), source('google_books', 'Books'),
  source('tmdb_movies', 'Movies', false), source('bgg_boardgames', 'Board Games'),
  source('rebrickable_minifigs', 'LEGO Minifigures')
]

async function textContrast(locator: Locator): Promise<number> {
  return locator.evaluate((element) => {
    const channels = (color: string) => (color.match(/[\d.]+/g) || []).map(Number)
    const foreground = channels(getComputedStyle(element).color)
    let background = [255, 255, 255]
    let parent: Element | null = element
    while (parent) {
      const color = channels(getComputedStyle(parent).backgroundColor)
      if (color.length === 3 || (color[3] || 0) >= 1) {
        background = color
        break
      }
      parent = parent.parentElement
    }
    const luminance = (color: number[]) => color.slice(0, 3).reduce((sum, channel, index) => {
      const value = channel / 255
      const linear = value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
      return sum + linear * [0.2126, 0.7152, 0.0722][index]!
    }, 0)
    const values = [luminance(foreground), luminance(background)].sort((a, b) => b - a)
    return (values[0]! + 0.05) / (values[1]! + 0.05)
  })
}

for (const width of [390, 1440]) {
  for (const colorScheme of ['light', 'dark'] as const) {
    test(`source list at ${width}px in ${colorScheme} mode`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 })
      await page.emulateMedia({ colorScheme })
      await page.addInitScript(theme => localStorage.setItem('nuxt-color-mode', theme), colorScheme)
      await page.route('**/auth/session', route => route.fulfill({ json: { authenticated: true, auth_disabled: true } }))
      await page.route('**/api/**', async (route) => {
        const path = new URL(route.request().url()).pathname
        if (path === '/api/plugins') {
          await route.fulfill({ json: { plugins: sources } })
        } else if (path === '/api/organization/features') {
          await route.fulfill({ json: { locations: true, collections: true, tags: true, categories: true, attributes: true, conditions: true, warranties: true, plugins: true } })
        } else {
          await route.fulfill({ json: path === '/api/' ? { status: 'ok', version: 'test' } : [] })
        }
      })
      await page.goto('/plugins')
      await expect(page.getByRole('heading', { name: 'Import sources', exact: true })).toBeVisible()
      await expect(page.getByRole('link', { name: 'Add asset', exact: true })).toHaveCount(0)
      const configured = page.getByRole('region', { name: 'Configured', exact: true })
      await expect(configured.getByRole('heading', { level: 3 })).toHaveText(['Board Games', 'Books', 'LEGO Minifigures', 'LEGO Sets', 'Video Games'])
      await expect(configured.getByRole('listitem')).toHaveCount(5)
      const needsSetup = page.getByRole('region', { name: 'Needs setup', exact: true })
      await expect(needsSetup.getByRole('listitem')).toHaveCount(1)
      await expect(needsSetup.getByRole('heading', { level: 3 })).toHaveText('Movies & TV Series')
      expect(await page.locator('h3').evaluateAll(headings => headings.every(heading => heading.scrollWidth <= heading.clientWidth))).toBeTruthy()
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
      const thirdImport = configured.getByRole('button', { name: 'Import LEGO Minifigures', exact: true })
      const bounds = await thirdImport.boundingBox()
      expect(bounds).not.toBeNull()
      expect(bounds!.y + bounds!.height).toBeLessThan(page.viewportSize()!.height)
      expect(bounds!.height).toBeGreaterThanOrEqual(44)
      await page.screenshot({ path: testInfo.outputPath('source-list.png') })

      const detailsButton = page.getByRole('button', { name: 'Details for LEGO Minifigures', exact: true })
      await detailsButton.focus()
      await page.keyboard.press('Enter')
      const details = page.getByRole('dialog', { name: 'LEGO Minifigures', exact: true })
      await expect(details.getByRole('heading', { name: 'Imported fields', exact: true })).toBeVisible()
      await expect(details.getByText('Catalog Number', { exact: true })).toBeVisible()
      expect(await textContrast(details.getByText('Catalog Number', { exact: true }))).toBeGreaterThanOrEqual(4.5)
      expect(await details.evaluate(element => element.contains(document.activeElement))).toBeTruthy()
      await details.evaluate(element => Promise.all(element.getAnimations().map(animation => animation.finished)))
      await page.screenshot({ path: testInfo.outputPath('source-details.png') })
      await page.keyboard.press('Escape')
      await expect(details).toHaveCount(0)
      await expect(detailsButton).toBeFocused()

      const setupButton = page.getByRole('button', { name: 'Setup instructions for TMDB', exact: true })
      await setupButton.click()
      const setup = page.getByRole('dialog', { name: 'Connect TMDB', exact: true })
      await expect(setup.locator('code')).toHaveText('ATTIC_TMDB_API_KEY')
      expect(await textContrast(setup.getByText('Movies', { exact: true }))).toBeGreaterThanOrEqual(4.5)
      await expect(setup.getByText('One key enables both Movies and TV Series.')).toBeVisible()
      await expect(setup.getByRole('textbox')).toHaveCount(0)
      expect(await setup.evaluate(element => element.scrollWidth <= element.clientWidth)).toBeTruthy()
      if (width === 390) expect((await setup.boundingBox())!.width).toBeCloseTo(390, 0)
      await setup.evaluate(element => Promise.all(element.getAnimations().map(animation => animation.finished)))
      await page.screenshot({ path: testInfo.outputPath('source-setup.png') })
      await page.keyboard.press('Escape')
      await expect(setup).toHaveCount(0)
      await expect(setupButton).toBeFocused()
      await page.getByText('How importing works', { exact: true }).click()
      await expect(page.getByText('Choose a source and search using its supported identifiers.')).toBeVisible()
    })
  }
}
