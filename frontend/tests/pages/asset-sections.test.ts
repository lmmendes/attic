import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import EditAsset from '../../app/pages/assets/[id]/edit.vue'
import NewAsset from '../../app/pages/assets/new.vue'
import AssetCategoryField from '../../app/components/AssetCategoryField.vue'

const { api, mutate, toast, clearAsset, featureFlags, featureRef } = vi.hoisted(() => {
  const featureFlags = {
    locations: true, collections: true, categories: true, attributes: true,
    conditions: true, warranties: true, plugins: true
  }
  return {
    api: vi.fn(), mutate: vi.fn(), toast: vi.fn(), clearAsset: vi.fn(), featureFlags,
    featureRef: { __v_isRef: true, value: featureFlags }
  }
})
mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useToast', () => () => ({ add: toast }))
mockNuxtImport('useRoute', () => () => ({ params: { id: 'asset' }, query: {} }))
mockNuxtImport('useFeatures', () => () => ({ features: featureRef }))

describe('Asset form sections', () => {
  const asset = ref<Record<string, unknown> | null>(null)
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(featureFlags, {
      locations: true, collections: true, categories: true, attributes: true,
      conditions: true, warranties: true, plugins: true
    })
    asset.value = { id: 'asset', name: 'Desk', quantity: 1 }
    api.mockImplementation((url: unknown) => ({
      data: typeof url === 'function' ? asset : ref([]),
      status: ref('success'),
      error: ref(null),
      clear: clearAsset
    }))
    mutate.mockResolvedValue({ id: 'new-asset' })
  })

  it('always shows both sections when creating an asset', async () => {
    const wrapper = await mountSuspended(NewAsset)
    for (const title of ['Details and notes', 'Purchase information']) {
      const heading = wrapper.findAll('h2').find(node => node.text() === title)!
      expect(heading.element.closest('section')).not.toBeNull()
      expect(heading.element.closest('details')).toBeNull()
    }
    wrapper.unmount()
  })

  describe.each([
    ['create', NewAsset, 'Failed to create asset'],
    ['edit', EditAsset, 'Failed to update asset']
  ] as const)('%s save errors', (mode, component, fallback) => {
    it.each([
      [
        { data: { error: 'required category attributes are missing: Serial number' }, message: '[PUT] /api/assets/asset: 400 Bad Request' },
        'required category attributes are missing: Serial number'
      ],
      [new Error('Network unavailable'), 'Network unavailable'],
      [{}, fallback]
    ])('shows the useful error message for %j', async (error, expected) => {
      mutate.mockImplementation(async (url: string) => {
        if (url.startsWith('/api/categories/')) {
          return {
            id: 'electronics',
            name: 'Electronics',
            attributes: [{
              attribute_id: 'serial', required: true,
              attribute: { key: 'serial', name: 'Serial number', data_type: 'string' }
            }]
          }
        }
        throw error
      })
      if (mode === 'edit') asset.value = { ...asset.value, category_id: 'electronics' }
      const wrapper = await mountSuspended(component)
      if (mode === 'create') {
        await wrapper.get('#asset-name').setValue('Desk')
        wrapper.getComponent(AssetCategoryField).vm.$emit('update:modelValue', 'electronics')
      }
      await flushPromises()
      await wrapper.get('form').trigger('submit')
      await flushPromises()

      expect(mutate).toHaveBeenCalledWith(
        mode === 'create' ? '/api/assets' : '/api/assets/asset',
        expect.objectContaining({ method: mode === 'create' ? 'POST' : 'PUT' })
      )
      expect(toast).toHaveBeenCalledExactlyOnceWith({ title: expected, color: 'error' })
      wrapper.unmount()
    })
  })

  it.each([
    [{}, false, false],
    [{ description: '  ', notes: '\n', purchase_note: ' ' }, false, false],
    [{ description: 'Oak desk' }, true, false],
    [{ notes: 'Upstairs' }, true, false],
    [{ condition_id: 'used' }, true, false],
    [{ quantity: 2 }, true, false],
    [{ purchase_at: '2026-09-06' }, false, true],
    [{ purchase_price: 0 }, false, true],
    [{ purchase_note: 'Gift' }, false, true]
  ])('sets edit expansion from saved fields %j', async (fields, details, purchase) => {
    const wrapper = await mountSuspended(EditAsset)
    asset.value = { ...asset.value, ...fields }
    await nextTick()
    const sections = wrapper.findAll('details')
    expect((sections[0]!.element as HTMLDetailsElement).open).toBe(details)
    expect((sections[1]!.element as HTMLDetailsElement).open).toBe(purchase)
    wrapper.unmount()
  })

  it('does not collapse an expanded section when its text is cleared', async () => {
    asset.value = { ...asset.value, notes: 'A note' }
    const wrapper = await mountSuspended(EditAsset)
    const section = wrapper.findAll('details')[0]!
    await section.findAll('textarea')[1]!.setValue('')
    expect((section.element as HTMLDetailsElement).open).toBe(true)
    wrapper.unmount()
  })

  it('creates an asset without a category', async () => {
    const wrapper = await mountSuspended(NewAsset)
    await wrapper.get('#asset-name').setValue('Unsorted item')
    await wrapper.get('#new-asset-form').trigger('submit')
    await flushPromises()
    const payload = JSON.parse(mutate.mock.calls[0]![1].body)
    expect(payload.name).toBe('Unsorted item')
    expect(payload).not.toHaveProperty('category_id')
    wrapper.unmount()
  })

  it.each(['locations', 'collections', 'categories', 'attributes', 'conditions'] as const)(
    'omits disabled %s data from asset creation',
    async (feature) => {
      featureFlags[feature] = false
      const wrapper = await mountSuspended(NewAsset)
      await wrapper.get('#asset-name').setValue('Feature-safe asset')
      await wrapper.get('#new-asset-form').trigger('submit')
      await flushPromises()

      const payload = JSON.parse(mutate.mock.calls.find(([url]) => url === '/api/assets')![1].body)
      const property = {
        locations: 'location_id', collections: 'collection_ids', categories: 'category_id',
        attributes: 'attributes', conditions: 'condition_id'
      }[feature]
      expect(payload).not.toHaveProperty(property)
      wrapper.unmount()
    }
  )

  it('does not request or render disabled asset feature controls', async () => {
    Object.assign(featureFlags, {
      locations: false, collections: false, categories: false,
      attributes: false, conditions: false
    })
    const wrapper = await mountSuspended(NewAsset)

    expect(api).toHaveBeenCalledWith('/api/categories', { immediate: false })
    expect(api).toHaveBeenCalledWith('/api/locations', { immediate: false })
    expect(api).toHaveBeenCalledWith('/api/conditions', { immediate: false })
    expect(wrapper.text()).not.toContain('Manage categories')
    expect(wrapper.text()).not.toContain('Collections (optional)')
    expect(wrapper.text()).not.toContain('Select condition')
    wrapper.unmount()
  })

  it('omits preserved feature data when editing with features disabled', async () => {
    Object.assign(featureFlags, {
      locations: false, collections: false, categories: false,
      attributes: false, conditions: false, plugins: false
    })
    asset.value = {
      ...asset.value,
      category_id: 'books',
      location_id: 'office',
      collection_ids: ['favorites'],
      condition_id: 'good',
      attributes: {
        'serial': 'user-value',
        'plugin.google_books.books.isbn': '123'
      }
    }
    const wrapper = await mountSuspended(EditAsset)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    const updateCall = mutate.mock.calls.find(([url]) => url === '/api/assets/asset')!
    const payload = JSON.parse(updateCall[1].body)
    for (const property of [
      'category_id', 'location_id', 'collection_ids', 'condition_id', 'attributes'
    ]) {
      expect(payload).not.toHaveProperty(property)
    }
    wrapper.unmount()
  })

  it('removes a category and its attributes while editing', async () => {
    asset.value = { ...asset.value, category_id: 'games', attributes: { platform: 'PS5' } }
    mutate.mockImplementation(async (url: string) => url === '/api/categories/games'
      ? { id: 'games', name: 'Games', attributes: [] }
      : undefined)
    const wrapper = await mountSuspended(EditAsset)
    await nextTick()
    wrapper.getComponent(AssetCategoryField).vm.$emit('update:modelValue', undefined)
    await nextTick()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    const updateCall = mutate.mock.calls.find(([url]) => url === '/api/assets/asset')!
    const payload = JSON.parse(updateCall[1].body)
    expect(payload).not.toHaveProperty('category_id')
    expect(payload).not.toHaveProperty('attributes')
    expect(clearAsset).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('loads category fields when categorizing an uncategorized asset while editing', async () => {
    mutate.mockImplementation(async (url: string) => url === '/api/categories/games?inherited=true'
      ? {
          id: 'games',
          name: 'Games',
          attributes: [{ attribute_id: 'platform', attribute: { key: 'platform', name: 'Platform', data_type: 'string' } }]
        }
      : undefined)
    const wrapper = await mountSuspended(EditAsset)

    wrapper.getComponent(AssetCategoryField).vm.$emit('update:modelValue', 'games')
    await flushPromises()

    expect(mutate).toHaveBeenCalledWith('/api/categories/games?inherited=true')
    wrapper.unmount()
  })
})
