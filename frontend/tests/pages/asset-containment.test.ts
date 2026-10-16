import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { clearNuxtData } from '#app'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { ref } from 'vue'
import { createFetch } from 'ofetch'

import type { Asset } from '../../app/types/api'
import AssetPicker from '../../app/components/AssetPicker.vue'
import AssetContents from '../../app/components/AssetContents.vue'
import AssetContentsBrowser from '../../app/components/AssetContentsBrowser.vue'
import NewAsset from '../../app/pages/assets/new.vue'
import EditAsset from '../../app/pages/assets/[id]/edit.vue'

// Restore Fetch globals before MSW evaluates its Request/Response subclasses.
const fetchGlobals = await vi.hoisted(async () => {
  const { fetch, Headers, Request, Response } = await import('undici')
  const original = { fetch: globalThis.fetch, Headers: globalThis.Headers, Request: globalThis.Request, Response: globalThis.Response }
  Object.assign(globalThis, { fetch, Headers, Request, Response })
  return original
})

const { route, toast, flags } = vi.hoisted(() => ({
  route: { params: { id: 'pc' }, query: {} as Record<string, string> },
  toast: vi.fn(),
  flags: { locations: true, categories: false, collections: false, tags: false, conditions: false, warranties: false, plugins: false, attributes: false }
}))
// Nuxt 4.6 imports its fetch client from #build/fetch rather than the global.
mockNuxtImport('$fetch', () => (...args: Parameters<typeof globalThis.$fetch>) => globalThis.$fetch(...args))
mockNuxtImport('useRuntimeConfig', () => () => ({ public: { apiBase: 'http://attic.test' }, app: { baseURL: '/', buildAssetsDir: '/_nuxt/', cdnURL: '' } }))
mockNuxtImport('useFeatures', () => () => ({ features: ref(flags) }))
mockNuxtImport('useRoute', () => () => route)
mockNuxtImport('useToast', () => () => ({ add: toast }))

const server = setupServer()
const saves: { method: string, body: Record<string, unknown> }[] = []
const listQueries: URLSearchParams[] = []
let assets: Record<string, Asset>
const wrappers: { unmount: () => void }[] = []

function fixture(id: string, name: string, fields: Partial<Asset> = {}): Asset {
  return { id, name, organization_id: 'org', quantity: 1, created_at: '', updated_at: '', ...fields }
}

async function mounted(component: Parameters<typeof mountSuspended>[0], options: Parameters<typeof mountSuspended>[1] = {}) {
  const wrapper = await mountSuspended(component, options)
  wrappers.push(wrapper)
  await flushPromises()
  return wrapper
}

async function settle(assertion: () => void) {
  await vi.waitFor(assertion)
}

const originalFetch = globalThis.$fetch
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'bypass' })
  globalThis.$fetch = createFetch({ fetch: globalThis.fetch })
})
afterAll(() => {
  server.close()
  Object.assign(globalThis, fetchGlobals)
  globalThis.$fetch = originalFetch
})
afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
})
beforeEach(() => {
  vi.clearAllMocks()
  clearNuxtData()
  saves.length = 0
  listQueries.length = 0
  route.params.id = 'pc'
  route.query = {}
  flags.locations = true
  assets = {
    pc: fixture('pc', 'Gaming PC', { location_id: 'garage', location: { id: 'garage', name: 'Garage', organization_id: 'org', created_at: '', updated_at: '' } }),
    gpu: fixture('gpu', 'Old GPU', { parent_id: 'pc', parent: { id: 'pc', name: 'Gaming PC' }, purchase_price: 50 }),
    replacement: fixture('replacement', 'New GPU')
  }
  server.resetHandlers(
    http.get('http://attic.test/api/locations', () => HttpResponse.json([])),
    http.get('http://attic.test/api/assets', ({ request }) => {
      const params = new URL(request.url).searchParams
      listQueries.push(params)
      return HttpResponse.json({ assets: params.has('parent_id') ? [assets.gpu] : [assets.pc, assets.replacement], total: params.has('parent_id') ? 1 : 2, limit: 20, offset: 0 })
    }),
    http.get('http://attic.test/api/assets/:id', ({ params }) => {
      const asset = assets[String(params.id)]
      return asset ? HttpResponse.json(asset) : HttpResponse.json({ error: 'asset not found' }, { status: 404 })
    }),
    http.post('http://attic.test/api/assets', async ({ request }) => {
      saves.push({ method: 'POST', body: await request.json() as Record<string, unknown> })
      return HttpResponse.json({ id: 'created' }, { status: 201 })
    }),
    http.put('http://attic.test/api/assets/:id', async ({ request }) => {
      saves.push({ method: 'PUT', body: await request.json() as Record<string, unknown> })
      return HttpResponse.json(assets.pc)
    }),
    http.patch('http://attic.test/api/assets/:id/parent', async ({ request }) => {
      saves.push({ method: 'PATCH', body: await request.json() as Record<string, unknown> })
      return HttpResponse.json(assets.pc)
    })
  )
})

describe('Asset containment', () => {
  it('loads branches lazily and browses direct contents without eager grandchildren', async () => {
    assets.motherboard = fixture('motherboard', 'Motherboard', { parent_id: 'pc', child_count: 1 })
    assets.ram = fixture('ram', '32 GB RAM', { parent_id: 'motherboard', child_count: 0 })
    server.use(http.get('http://attic.test/api/assets', ({ request }) => {
      const params = new URL(request.url).searchParams
      listQueries.push(params)
      const children = params.get('parent_id') === 'pc' ? [assets.motherboard] : [assets.ram]
      return HttpResponse.json({ assets: children, total: children.length })
    }))
    const wrapper = await mounted(AssetContentsBrowser, { props: { asset: assets.pc! } })
    await settle(() => expect(wrapper.find('button[aria-label="Expand Motherboard"]').exists()).toBe(true))
    expect(listQueries.map(query => query.get('parent_id'))).toEqual(['pc'])
    await wrapper.get('button[aria-label="Browse Motherboard"]').trigger('click')
    await settle(() => expect(wrapper.get('ul[aria-label="Asset results"]').text()).toContain('32 GB RAM'))
    expect(listQueries.map(query => query.get('parent_id'))).toEqual(['pc', 'motherboard'])
    expect(wrapper.get('nav[aria-label="Contents path"]').text()).toContain('Gaming PC')
    await wrapper.get('button[aria-label="Collapse Motherboard"]').trigger('click')
    await wrapper.get('button[aria-label="Expand Motherboard"]').trigger('click')
    expect(listQueries).toHaveLength(2)
    await wrapper.get('button[aria-label="Browse 32 GB RAM"]').trigger('click')
    await settle(() => expect(wrapper.text()).toContain('No contents yet'))
    expect(listQueries).toHaveLength(2)
  })

  it('shows paginated descendants and their parent paths without loading their branches', async () => {
    server.use(http.get('http://attic.test/api/assets', ({ request }) => {
      const params = new URL(request.url).searchParams
      listQueries.push(params)
      const children = params.has('descendant_of')
        ? [fixture('ram', '32 GB RAM', { parent_id: 'motherboard', containment_path: [{ id: 'pc', name: 'Gaming PC' }, { id: 'motherboard', name: 'Motherboard' }] })]
        : [assets.gpu]
      return HttpResponse.json({ assets: children, total: params.has('descendant_of') ? 25 : 1 })
    }))
    const wrapper = await mounted(AssetContentsBrowser, { props: { asset: assets.pc! } })
    await settle(() => expect(wrapper.text()).toContain('Old GPU'))
    await wrapper.get('input[type="checkbox"]').setValue(true)
    await settle(() => expect(wrapper.text()).toContain('Inside Gaming PC › Motherboard'))
    expect(listQueries.map(query => query.get('parent_id') || query.get('descendant_of'))).toEqual(['pc', 'pc'])
    await wrapper.findAll('button').find(button => button.text() === 'Next contents')!.trigger('click')
    await settle(() => expect(listQueries.at(-1)?.get('offset')).toBe('24'))
    await wrapper.get('input[type="search"]').setValue('RAM')
    await settle(() => expect(listQueries.at(-1)?.get('q')).toBe('RAM'))
    expect(listQueries.at(-1)?.get('offset')).toBe('0')
  })

  it('retains loaded tree siblings when switching containers and paginates results independently', async () => {
    const children = Array.from({ length: 25 }, (_, index) => fixture(`part-${index}`, `Part ${index}`, { parent_id: 'pc', child_count: 0 }))
    server.use(http.get('http://attic.test/api/assets', ({ request }) => {
      const params = new URL(request.url).searchParams
      listQueries.push(params)
      const start = Number(params.get('offset') || 0)
      return HttpResponse.json({ assets: children.slice(start, start + 24), total: children.length })
    }))
    const wrapper = await mounted(AssetContentsBrowser, { props: { asset: assets.pc! } })
    await settle(() => expect(wrapper.find('button[aria-label="Browse Part 0"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Load more contents of Gaming PC"]').trigger('click')
    await settle(() => expect(wrapper.find('button[aria-label="Browse Part 24"]').exists()).toBe(true))
    expect(wrapper.get('ul[aria-label="Asset results"]').findAll('li')).toHaveLength(24)
    await wrapper.get('button[aria-label="Browse Part 0"]').trigger('click')
    await wrapper.get('button[aria-label="Browse Gaming PC"]').trigger('click')
    await settle(() => expect(wrapper.get('ul[aria-label="Asset results"]').findAll('li')).toHaveLength(24))
    expect(wrapper.find('button[aria-label="Browse Part 24"]').exists()).toBe(true)
    expect(listQueries).toHaveLength(2)
  })

  it('retries failed branches without requesting unopened grandchildren', async () => {
    assets.motherboard = fixture('motherboard', 'Motherboard', { parent_id: 'pc', child_count: 1 })
    let attempts = 0
    server.use(http.get('http://attic.test/api/assets', ({ request }) => {
      const params = new URL(request.url).searchParams
      listQueries.push(params)
      if (params.get('parent_id') === 'motherboard') {
        attempts++
        if (attempts === 1) return HttpResponse.json({ error: 'offline' }, { status: 400 })
        return HttpResponse.json({ assets: [fixture('ram', 'RAM', { parent_id: 'motherboard', child_count: 0 })], total: 1 })
      }
      return HttpResponse.json({ assets: [assets.motherboard], total: 1 })
    }))
    const wrapper = await mounted(AssetContentsBrowser, { props: { asset: assets.pc! } })
    await settle(() => expect(wrapper.find('button[aria-label="Expand Motherboard"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Expand Motherboard"]').trigger('click')
    await settle(() => expect(wrapper.text()).toContain('Could not load Motherboard'))
    await wrapper.findAll('button').find(button => button.text() === 'Retry')!.trigger('click')
    await settle(() => expect(wrapper.find('button[aria-label="Browse RAM"]').exists()).toBe(true))
    expect(attempts).toBe(2)
    expect(wrapper.get('ul[aria-label="Asset results"]').text()).toContain('Motherboard')
    expect(listQueries.some(query => query.get('parent_id') === 'ram')).toBe(false)
  })

  it('keeps the latest selection when an earlier container response arrives late', async () => {
    assets.motherboard = fixture('motherboard', 'Motherboard', { parent_id: 'pc', child_count: 1 })
    let release: (() => void) | undefined
    let returned = false
    server.use(http.get('http://attic.test/api/assets', async ({ request }) => {
      const params = new URL(request.url).searchParams
      if (params.get('parent_id') === 'motherboard') {
        await new Promise<void>((resolve) => {
          release = resolve
        })
        returned = true
        return HttpResponse.json({ assets: [fixture('ram', 'Late RAM', { parent_id: 'motherboard' })], total: 1 })
      }
      return HttpResponse.json({ assets: [assets.motherboard], total: 1 })
    }))
    const wrapper = await mounted(AssetContentsBrowser, { props: { asset: assets.pc! } })
    await settle(() => expect(wrapper.find('button[aria-label="Browse Motherboard"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Browse Motherboard"]').trigger('click')
    await settle(() => expect(release).toBeDefined())
    await wrapper.get('button[aria-label="Browse Gaming PC"]').trigger('click')
    release!()
    await settle(() => expect(returned).toBe(true))
    await flushPromises()
    expect(wrapper.get('ul[aria-label="Asset results"]').text()).toContain('Motherboard')
    expect(wrapper.get('ul[aria-label="Asset results"]').text()).not.toContain('Late RAM')
    expect(wrapper.get('button[aria-label="Browse Gaming PC"]').attributes('aria-pressed')).toBe('true')
  })

  it('links into the selected nested container and refreshes its contents', async () => {
    assets.motherboard = fixture('motherboard', 'Motherboard', { parent_id: 'pc', child_count: 0 })
    let linked = false
    server.use(
      http.get('http://attic.test/api/assets', ({ request }) => {
        const params = new URL(request.url).searchParams
        listQueries.push(params)
        const children = params.get('parent_id') === 'pc' ? [assets.motherboard] : linked ? [assets.replacement] : []
        return HttpResponse.json({ assets: children, total: children.length })
      }),
      http.patch('http://attic.test/api/assets/:id/parent', async ({ request }) => {
        saves.push({ method: 'PATCH', body: await request.json() as Record<string, unknown> })
        linked = true
        assets.motherboard!.child_count = 1
        assets.replacement!.parent_id = 'motherboard'
        return HttpResponse.json(assets.replacement)
      })
    )
    const wrapper = await mounted(AssetContentsBrowser, { props: { asset: assets.pc! } })
    await settle(() => expect(wrapper.find('button[aria-label="Browse Motherboard"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Browse Motherboard"]').trigger('click')
    await wrapper.findAll('button').find(button => button.text() === 'Link existing asset')!.trigger('click')
    const picker = wrapper.getComponent(AssetPicker)
    picker.vm.$emit('update:modelValue', 'replacement')
    await settle(() => expect(wrapper.findAll('button').find(button => button.text() === 'Link asset')!.attributes('disabled')).toBeUndefined())
    await wrapper.findAll('button').find(button => button.text() === 'Link asset')!.trigger('click')
    await settle(() => expect(wrapper.get('ul[aria-label="Asset results"]').text()).toContain('New GPU'))
    expect(saves).toEqual([{ method: 'PATCH', body: { parent_id: 'motherboard' } }])
    expect(wrapper.emitted('changed')).toHaveLength(1)
    expect(wrapper.get('button[aria-label="Browse Motherboard"]').attributes('aria-pressed')).toBe('true')
  })

  it('resolves a prefilled parent and submits it without an independent location', async () => {
    route.query = { parent_id: 'pc' }
    const wrapper = await mounted(NewAsset)
    await settle(() => expect(wrapper.text()).toContain('Location: Garage'))
    expect(wrapper.find('#location').exists()).toBe(false)
    await wrapper.get('#asset-name').setValue('New component')
    await wrapper.get('form').trigger('submit')
    await settle(() => expect(saves).toHaveLength(1))
    expect(saves[0]?.body).toMatchObject({ parent_id: 'pc', name: 'New component' })
    expect(saves[0]?.body).not.toHaveProperty('location_id')
  })

  it('blocks creation with an unavailable prefilled parent and lets the user clear it', async () => {
    route.query = { parent_id: 'missing' }
    const wrapper = await mounted(NewAsset)
    await settle(() => expect(wrapper.text()).toContain('Selected asset is unavailable'))
    await wrapper.get('#asset-name').setValue('Component')
    await wrapper.get('form').trigger('submit')
    expect(saves).toHaveLength(0)
    await wrapper.findAll('button').find(button => button.text() === 'Clear inside')!.trigger('click')
    await wrapper.get('form').trigger('submit')
    await settle(() => expect(saves).toHaveLength(1))
    expect(saves[0]?.body.parent_id).toBeNull()
  })

  it('retains parent selection but hides its location when locations are disabled', async () => {
    route.query = { parent_id: 'pc' }
    flags.locations = false
    const wrapper = await mounted(NewAsset)
    await settle(() => expect(wrapper.text()).toContain('Gaming PC'))
    expect(wrapper.text()).not.toContain('Location: Garage')
  })

  it('stages removal until the main edit form is saved', async () => {
    const wrapper = await mounted(EditAsset)
    await settle(() => expect(wrapper.find('button[aria-label="Detach Old GPU"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Detach Old GPU"]').trigger('click')
    expect(wrapper.text()).toContain('Will detach on save')
    expect(saves).toHaveLength(0)
    await wrapper.get('form').trigger('submit')
    await settle(() => expect(saves).toHaveLength(1))
    expect(saves[0]).toMatchObject({ method: 'PUT', body: { parent_id: null, remove_child_ids: ['gpu'], add_child_ids: [] } })
  })

  it('discards staged removals when cancelling the edit form', async () => {
    const wrapper = await mounted(EditAsset)
    await settle(() => expect(wrapper.find('button[aria-label="Detach Old GPU"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Detach Old GPU"]').trigger('click')
    const cancel = wrapper.findAll('a').find(link => link.text() === 'Cancel')
    expect(cancel).toBeDefined()
    expect(cancel?.attributes('href')).toBe('/assets/pc')
    wrapper.unmount()
    expect(saves).toHaveLength(0)
  })

  it('lets the user undo a staged removal without writing to the API', async () => {
    const wrapper = await mounted(AssetContents, { props: { assetId: 'pc', editing: true } })
    await settle(() => expect(wrapper.find('button[aria-label="Detach Old GPU"]').exists()).toBe(true))
    await wrapper.get('button[aria-label="Detach Old GPU"]').trigger('click')
    await wrapper.get('button[aria-label="Undo removal of Old GPU"]').trigger('click')
    expect(wrapper.text()).not.toContain('Will detach on save')
    expect(saves).toHaveLength(0)
  })

  it('offers retry when contents loading fails', async () => {
    server.use(http.get('http://attic.test/api/assets', () => HttpResponse.json({ error: 'unavailable' }, { status: 400 })))
    const wrapper = await mounted(AssetContents, { props: { assetId: 'pc' } })
    await settle(() => expect(wrapper.text()).toContain('Contents could not be loaded'))
    server.use(http.get('http://attic.test/api/assets', () => HttpResponse.json({ assets: [assets.gpu], total: 1 })))
    await wrapper.findAll('button').find(button => button.text() === 'Try again')!.trigger('click')
    await settle(() => expect(wrapper.text()).toContain('Old GPU'))
  })

  it('queries valid parent candidates using server-side exclusions', async () => {
    const wrapper = await mounted(AssetPicker, { props: { label: 'Inside', excludeSubtreeOf: 'pc' } })
    await wrapper.get('[aria-label="Inside"]').trigger('click')
    await settle(() => expect(listQueries.some(query => query.get('exclude_subtree_of') === 'pc')).toBe(true))
  })
})
