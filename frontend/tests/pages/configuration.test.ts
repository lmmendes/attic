import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('feature configuration UI', () => {
  const appSource = readFileSync(resolve(process.cwd(), 'app/app.vue'), 'utf8')
  const pageSource = readFileSync(resolve(process.cwd(), 'app/pages/configuration.vue'), 'utf8')
  const newAssetSource = readFileSync(resolve(process.cwd(), 'app/pages/assets/new.vue'), 'utf8')
  const editAssetSource = readFileSync(resolve(process.cwd(), 'app/pages/assets/[id]/edit.vue'), 'utf8')
  const inventorySource = readFileSync(resolve(process.cwd(), 'app/pages/assets/index.vue'), 'utf8')

  it('exposes configuration only in administrator navigation', () => {
    expect(appSource).toContain('if (isAdmin.value)')
    expect(appSource).toContain('to: \'/configuration\'')
    expect(pageSource).toContain('if (!isAdmin.value)')
  })

  it('provides all five feature switches', () => {
    for (const feature of ['collections', 'plugins', 'conditions', 'locations']) {
      expect(pageSource).toContain(`key: '${feature}'`)
    }
    expect(pageSource).toContain('<USwitch')
  })

  it('omits disabled features from asset forms and inventory filters', () => {
    for (const source of [newAssetSource, editAssetSource, inventorySource]) {
      expect(source).toContain('configuration.collections_enabled')
      expect(source).toContain('configuration.conditions_enabled')
      expect(source).toContain('configuration.locations_enabled')
    }
    expect(inventorySource).toContain('configuration.plugins_enabled')
  })
})
