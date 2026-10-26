import { expect, test } from '@playwright/test'

const sources = [
  { id: 'rebrickable_sets', name: 'Rebrickable LEGO Sets', category: 'LEGO Sets', number: '75192-2', title: 'Millennium Falcon' },
  { id: 'rebrickable_minifigs', name: 'Rebrickable Minifigures', category: 'LEGO Minifigures', number: 'fig-000001', title: 'Toy Store Employee' }
]

for (const viewport of [{ width: 390, height: 844 }, { width: 1440, height: 1000 }]) {
  for (const colorScheme of ['light', 'dark'] as const) {
    test(`Rebrickable imports at ${viewport.width}px in ${colorScheme} mode`, async ({ page }, testInfo) => {
      await page.setViewportSize(viewport)
      await page.emulateMedia({ colorScheme })
      await page.addInitScript((theme) => {
        localStorage.setItem('nuxt-color-mode', theme)
      }, colorScheme)
      let enabled = false
      const searches: string[] = []
      const imports: string[] = []
      await page.route('**/auth/session', route => route.fulfill({ json: { authenticated: true, auth_disabled: true } }))
      await page.route('**/api/**', async (route) => {
        const url = new URL(route.request().url())
        if (url.pathname === '/api/organization/features') {
          await route.fulfill({ json: { locations: true, collections: true, tags: true, categories: true, attributes: true, conditions: true, warranties: true, plugins: true } })
        } else if (url.pathname === '/api/plugins') {
          await route.fulfill({ json: { plugins: sources.map(source => ({
            id: source.id, name: source.name, category_name: source.category, category_id: enabled ? 'test-category' : undefined,
            description: `Import ${source.category} from the Rebrickable catalog`,
            enabled, disabled_reason: enabled ? '' : 'Missing API key: ATTIC_REBRICKABLE_API_KEY',
            search_fields: [{ key: 'name', label: 'Name' }, { key: 'catalog_number', label: 'Catalog Number' }],
            attributes: [
              { key: `plugin.${source.id}.catalog_number`, name: 'Catalog Number', data_type: 'string', required: false },
              { key: `plugin.${source.id}.url`, name: 'Rebrickable URL', data_type: 'url', required: false }
            ]
          })) } })
        } else if (url.pathname.endsWith('/search')) {
          searches.push(url.toString())
          const source = sources.find(source => url.pathname.includes(source.id))!
          await route.fulfill({ json: { results: [{ external_id: source.number, title: source.title, subtitle: `${source.number} · 42 pieces`, image_url: null }] } })
        } else if (url.pathname.endsWith('/import')) {
          const source = sources.find(source => url.pathname.includes(source.id))!
          expect(route.request().postDataJSON()).toEqual({ external_id: source.number })
          imports.push(source.id)
          await route.fulfill({ status: 201, json: { asset: { id: 'test-asset', name: source.title, quantity: 1 } } })
        } else {
          await route.fulfill({ json: url.pathname === '/api/' ? { status: 'ok', version: 'test' } : [] })
        }
      })
      await page.goto('/plugins')
      const setupButton = page.getByRole('button', { name: 'Setup instructions for Rebrickable', exact: true })
      await expect(setupButton).toHaveCount(1)
      await setupButton.click()
      const setupPanel = page.getByRole('dialog', { name: 'Connect Rebrickable', exact: true })
      await expect(setupPanel.locator('code')).toHaveText('ATTIC_REBRICKABLE_API_KEY')
      await expect(setupPanel.getByText('One key enables both LEGO Sets and LEGO Minifigures.')).toBeVisible()
      await page.keyboard.press('Escape')
      await expect(setupPanel).toHaveCount(0)
      await expect(setupButton).toBeFocused()
      await expect(page.getByRole('button', { name: /^Import LEGO/ })).toHaveCount(0)
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
      enabled = true
      await page.reload()
      for (const source of sources) {
        if (source.id === 'rebrickable_sets') {
          await page.getByRole('button', { name: `Details for ${source.category}`, exact: true }).click()
          const panel = page.getByRole('dialog', { name: source.category, exact: true })
          await expect(panel.getByRole('heading', { name: 'Imported fields', exact: true })).toBeVisible()
          await expect(panel.locator('dl > div').filter({ hasText: 'Rebrickable URL' }).locator('dd')).toHaveText('url')
          await panel.getByRole('button', { name: `Import ${source.category}`, exact: true }).click()
          await expect(panel).toHaveCount(0)
        } else {
          await page.getByRole('button', { name: `Import ${source.category}`, exact: true }).click()
        }
        const dialog = page.getByRole('dialog')
        await expect(dialog.getByRole('heading', { name: `Search ${source.name}`, exact: true })).toBeVisible()
        await dialog.getByPlaceholder('Search...').fill(source.title)
        await dialog.getByRole('button', { name: 'Search', exact: true }).click()
        await expect(dialog.getByRole('heading', { name: source.title, exact: true })).toBeVisible()
        await dialog.getByRole('button', { name: 'Search field', exact: true }).click()
        await page.getByRole('option', { name: 'Catalog Number', exact: true }).click()
        await expect(page.getByRole('listbox')).toHaveCount(0)
        await dialog.getByPlaceholder('Search...').fill(source.number)
        await dialog.getByRole('button', { name: 'Search', exact: true }).click()
        await expect.poll(() => searches.length).toBe((sources.indexOf(source) + 1) * 2)
        const query = new URL(searches.at(-1)!).searchParams
        expect(query.get('field')).toBe('catalog_number')
        expect(query.get('q')).toBe(source.number)
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
        await expect(dialog.getByRole('heading', { name: source.title, exact: true })).toBeVisible()
        await page.screenshot({ path: testInfo.outputPath(`${source.id}.png`) })
        await dialog.getByRole('button', { name: 'Import', exact: true }).click()
        await expect(dialog).toHaveCount(0)
        await expect(page.getByText(`"${source.title}" has been added to your inventory`, { exact: true })).toBeVisible()
      }
      expect(imports).toEqual(sources.map(source => source.id))
    })
  }
}
