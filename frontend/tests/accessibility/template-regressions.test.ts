import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

function readAppFile(path: string): string {
  return readFileSync(resolve(process.cwd(), 'app', path), 'utf8')
}

describe('accessibility template regressions', () => {
  it('keeps the custom import modal title and description connected to the dialog', () => {
    const source = readAppFile('components/ImportModal.vue')

    expect(source).toContain('<DialogTitle')
    expect(source).toContain('<DialogDescription')
    expect(source).toContain('{{ modalTitle }}')
    expect(source).toContain('{{ modalDescription }}')
  })

  it('provides title and description props to every custom-content modal', () => {
    const files = [
      'app.vue',
      'pages/assets/[id]/index.vue',
      'pages/attributes/index.vue',
      'pages/categories/index.vue',
      'pages/conditions/index.vue',
      'pages/locations.vue',
      'pages/users.vue'
    ]

    for (const file of files) {
      const source = readAppFile(file)
      const modalTags = source.match(/<UModal[\s\S]*?>/g) || []
      expect(modalTags.length, file).toBeGreaterThan(0)
      for (const tag of modalTags) {
        expect(tag, file).toMatch(/\s:?title=/)
        expect(tag, file).toMatch(/\s:?description=/)
      }
    }
  })

  it('names icon and data-type choices and exposes their selected state', () => {
    for (const file of ['pages/locations.vue', 'pages/categories/new.vue', 'pages/categories/[id]/edit.vue']) {
      const source = readAppFile(file)
      expect(source, file).toContain(':aria-label=')
      expect(source, file).toContain(':aria-pressed=')
    }

    for (const file of ['pages/attributes/new.vue', 'pages/attributes/[id]/edit.vue']) {
      const source = readAppFile(file)
      expect(source, file).toContain('role="radiogroup"')
      expect(source, file).toContain('role="radio"')
      expect(source, file).toContain(':aria-checked=')
    }
  })

  it('keeps the account menu trigger named', () => {
    const source = readAppFile('app.vue')

    expect(source).toContain('aria-label="Open account menu"')
    expect(source).toContain('aria-label="Open navigation"')
    expect(source).toContain('aria-label="Close navigation"')
    expect(source).toContain('aria-label="Add asset"')
  })

  it('requests search focus when navigating from the dashboard', () => {
    expect(readAppFile('pages/index.vue')).toContain('to="/assets?focus=search"')

    const assetsSource = readAppFile('pages/assets/index.vue')
    expect(assetsSource).toContain('route.query.focus !== \'search\'')
    expect(assetsSource).toContain('searchContainer.value?.querySelector<HTMLInputElement>(\'input\')?.focus()')
  })

  it('uses an asset-name link and names each asset actions menu', () => {
    const source = readAppFile('pages/assets/index.vue')

    expect(source).toContain(':to="`/assets/${asset.id}`"')
    expect(source).toContain(':aria-label="`Actions for ${asset.name}`"')
  })

  it('distinguishes an asset load failure from an empty inventory and offers retry', () => {
    const source = readAppFile('pages/assets/index.vue')

    expect(source).toContain('v-else-if="error"')
    expect(source).toContain('Could not load assets')
    expect(source).toContain('@click="refresh()"')
  })

  it('names the mobile asset delete control', () => {
    expect(readAppFile('pages/assets/[id]/index.vue')).toContain(':aria-label="`Delete ${asset.name}`"')
  })

  it('lets category icon grids contribute their full mobile height', () => {
    expect(readAppFile('pages/categories/new.vue')).toContain('sm:max-h-[280px] sm:overflow-y-auto')
    expect(readAppFile('pages/categories/[id]/edit.vue')).toContain('sm:max-h-40 sm:overflow-y-auto')
  })

  it('allows the attribute filters to shrink without overlapping search', () => {
    const source = readAppFile('pages/attributes/index.vue')

    expect(source).toContain('flex min-w-0 flex-1 gap-2 overflow-x-auto')
    expect(source).toContain('class="w-full shrink-0 2xl:w-64"')
  })
})
