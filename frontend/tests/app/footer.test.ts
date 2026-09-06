import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('authenticated app footer', () => {
  const source = readFileSync(resolve(process.cwd(), 'app/app.vue'), 'utf8')

  it('loads and displays the running backend version', () => {
    expect(source).toContain('useApi<AppInfo>(\'/api/\'')
    expect(source).toContain('Attic version {{ softwareVersion }}')
  })

  it('stays at the bottom when page content is short', () => {
    expect(source).toContain('mx-auto flex min-h-full max-w-[1440px] flex-col')
    expect(source).toContain('<div class="flex-1">')
  })
})
