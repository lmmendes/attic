import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import CategoryEditor from '../../app/components/CategoryEditor.vue'
import NewCategory from '../../app/pages/categories/new.vue'
import NewAttribute from '../../app/pages/attributes/new.vue'

const { api, mutate, push, replace, resolve, toast, route, draft, clearCategories, featureFlags, featureRef } = vi.hoisted(() => {
  const featureFlags = {
    locations: true, collections: true, categories: true, attributes: true,
    conditions: true, warranties: true, plugins: true
  }
  return {
    api: vi.fn(),
    mutate: vi.fn(),
    push: vi.fn(),
    replace: vi.fn(),
    resolve: vi.fn((to: string | { path: string }) => ({ href: typeof to === 'string' ? to : to.path })),
    toast: vi.fn(),
    clearCategories: vi.fn(),
    route: { query: {} as Record<string, string> },
    draft: { value: null as Record<string, unknown> | null },
    featureFlags,
    featureRef: { __v_isRef: true, value: featureFlags }
  }
})

mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useRouter', () => () => ({ push, replace, resolve }))
mockNuxtImport('useRoute', () => () => route)
mockNuxtImport('useToast', () => () => ({ add: toast }))
mockNuxtImport('useState', () => () => draft)
mockNuxtImport('useFeatures', () => () => ({ features: featureRef }))

describe('creating an attribute from a category draft', () => {
  const attributes = ref<Record<string, unknown>[]>([])
  const refreshAttributes = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {}
    draft.value = null
    attributes.value = []
    Object.assign(featureFlags, {
      locations: true, collections: true, categories: true, attributes: true,
      conditions: true, warranties: true, plugins: true
    })
    api.mockImplementation((url: string) => url === '/api/attributes'
      ? { data: attributes, refresh: refreshAttributes }
      : { data: ref([]), clear: clearCategories })
    mutate.mockResolvedValue({ id: 'new-attribute' })
  })

  it('keeps the category draft when opening the attribute form', async () => {
    const wrapper = await mountSuspended(NewCategory)
    await wrapper.get('input[placeholder="e.g. Rare Books"]').setValue('Vintage cameras')
    await wrapper.get('textarea').setValue('Analog photography gear')
    const newAttribute = wrapper.findAll('button').find(button => button.text().includes('New Attribute'))!
    await newAttribute.trigger('click')

    expect(draft.value).toMatchObject({
      form: { name: 'Vintage cameras', description: 'Analog photography gear' },
      selectedAttributes: []
    })
    expect(push).toHaveBeenCalledWith({ path: '/attributes/new', query: { returnTo: '/categories/new' } })
    wrapper.unmount()
  })

  it('clears the cached category list after creating a category', async () => {
    const wrapper = await mountSuspended(NewCategory)
    await wrapper.get('input[placeholder="e.g. Rare Books"]').setValue('Vintage cameras')
    const save = wrapper.findAll('button').find(button => button.text().includes('Save Category'))!
    await save.trigger('click')
    await flushPromises()

    expect(clearCategories).toHaveBeenCalledOnce()
    expect(push).toHaveBeenCalledWith('/categories')
    expect(clearCategories.mock.invocationCallOrder[0]).toBeLessThan(push.mock.invocationCallOrder[0]!)
    wrapper.unmount()
  })

  it('uses the create-category editor when editing and submits an update', async () => {
    const category = {
      id: 'category-1',
      organization_id: 'organization-1',
      name: 'Vintage cameras',
      description: 'Analog photography gear',
      icon: 'i-lucide-camera',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      attributes: []
    }
    const wrapper = await mountSuspended(CategoryEditor, {
      props: { category }
    })

    expect(wrapper.text()).toContain('Attribute Schema')
    expect(wrapper.text()).toContain('Active Attributes')
    expect(wrapper.text()).not.toContain('Asset fields')
    expect((wrapper.get('input[placeholder="e.g. Rare Books"]').element as HTMLInputElement).value).toBe('Vintage cameras')

    await wrapper.get('input[placeholder="e.g. Rare Books"]').setValue('Film cameras')
    const save = wrapper.findAll('button').find(button => button.text().includes('Save Changes'))!
    await save.trigger('click')
    await flushPromises()

    expect(mutate).toHaveBeenCalledWith('/api/categories/category-1', expect.objectContaining({
      method: 'PUT',
      body: expect.stringContaining('Film cameras')
    }))
    expect(clearCategories).toHaveBeenCalledOnce()
    expect(push).toHaveBeenCalledWith('/categories')
    wrapper.unmount()
  })

  it('opens the attribute form from category editing and preserves the draft', async () => {
    const category = {
      id: 'category-1',
      organization_id: 'organization-1',
      name: 'Vintage cameras',
      icon: 'i-lucide-camera',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      attributes: []
    }
    const wrapper = await mountSuspended(CategoryEditor, { props: { category } })
    await wrapper.get('input[placeholder="e.g. Rare Books"]').setValue('Film cameras')
    const newAttribute = wrapper.findAll('button').find(button => button.text().includes('New Attribute'))!
    await newAttribute.trigger('click')

    expect(draft.value).toMatchObject({
      returnPath: '/categories/category-1/edit',
      form: { name: 'Film cameras' }
    })
    expect(push).toHaveBeenCalledWith({
      path: '/attributes/new',
      query: { returnTo: '/categories/category-1/edit' }
    })
    wrapper.unmount()
  })

  it('uses the categories switch for attributes when legacy state disagrees', async () => {
    featureFlags.attributes = false
    const category = {
      id: 'category-1',
      organization_id: 'organization-1',
      name: 'Vintage cameras',
      icon: 'i-lucide-camera',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      attributes: [{ attribute_id: 'serial', required: true, sort_order: 0 }]
    }
    const wrapper = await mountSuspended(CategoryEditor, { props: { category } })

    expect(api).toHaveBeenCalledWith('/api/attributes', { immediate: true })
    expect(wrapper.text()).toContain('Attribute Schema')
    expect(wrapper.text()).toContain('New Attribute')
    wrapper.unmount()
  })

  it('hides attribute controls and omits assignments with categories disabled', async () => {
    featureFlags.categories = false
    const category = {
      id: 'category-1',
      organization_id: 'organization-1',
      name: 'Vintage cameras',
      icon: 'i-lucide-camera',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      attributes: [{ attribute_id: 'serial', required: true, sort_order: 0 }]
    }
    const wrapper = await mountSuspended(CategoryEditor, { props: { category } })

    expect(api).toHaveBeenCalledWith('/api/attributes', { immediate: false })
    expect(wrapper.text()).not.toContain('Attribute Schema')
    expect(wrapper.text()).not.toContain('New Attribute')
    const save = wrapper.findAll('button').find(button => button.text().includes('Save Changes'))!
    await save.trigger('click')
    await flushPromises()

    const payload = JSON.parse(mutate.mock.calls.find(([url]) => url === '/api/categories/category-1')![1].body)
    expect(payload).not.toHaveProperty('attributes')
    wrapper.unmount()
  })

  it('returns to the category draft and identifies the created attribute', async () => {
    route.query = { returnTo: 'category' }
    const wrapper = await mountSuspended(NewAttribute)
    await wrapper.get('input[placeholder="e.g. Purchase Date"]').setValue('Serial number')
    const save = wrapper.findAll('button').find(button => button.text().includes('Save Attribute'))!
    await save.trigger('click')
    await flushPromises()

    expect(push).toHaveBeenCalledWith({
      path: '/categories/new',
      query: { resume: 'attribute', attribute_id: 'new-attribute' }
    })
    wrapper.unmount()
  })

  it('returns to the exact category edit page after creating an attribute', async () => {
    route.query = { returnTo: '/categories/category-1/edit' }
    const wrapper = await mountSuspended(NewAttribute)
    await wrapper.get('input[placeholder="e.g. Purchase Date"]').setValue('Serial number')
    const save = wrapper.findAll('button').find(button => button.text().includes('Save Attribute'))!
    await save.trigger('click')
    await flushPromises()

    expect(push).toHaveBeenCalledWith({
      path: '/categories/category-1/edit',
      query: { resume: 'attribute', attribute_id: 'new-attribute' }
    })
    wrapper.unmount()
  })

  it('restores the edited category and selects its newly created attribute', async () => {
    route.query = { resume: 'attribute', attribute_id: 'new-attribute' }
    draft.value = {
      returnPath: '/categories/category-1/edit',
      form: {
        name: 'Unsaved film cameras',
        description: 'Unsaved description',
        icon: 'i-lucide-camera'
      },
      selectedAttributes: [{ attribute_id: 'existing', required: true, sort_order: 0 }]
    }
    attributes.value = [
      { id: 'existing', name: 'Maker', key: 'maker', data_type: 'string' },
      { id: 'new-attribute', name: 'Serial number', key: 'serial_number', data_type: 'string' }
    ]
    const category = {
      id: 'category-1',
      organization_id: 'organization-1',
      name: 'Saved cameras',
      icon: 'i-lucide-camera',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      attributes: []
    }

    const wrapper = await mountSuspended(CategoryEditor, { props: { category } })
    await flushPromises()

    expect((wrapper.get('input[placeholder="e.g. Rare Books"]').element as HTMLInputElement).value)
      .toBe('Unsaved film cameras')
    expect(wrapper.text()).toContain('Maker')
    expect(wrapper.text()).toContain('Serial number')
    expect(refreshAttributes).toHaveBeenCalledOnce()
    expect(draft.value).toBeNull()
    wrapper.unmount()
  })

  it('restores entered values and selects the newly created attribute', async () => {
    route.query = { resume: 'attribute', attribute_id: 'new-attribute' }
    draft.value = {
      form: {
        name: 'Vintage cameras',
        description: 'Analog photography gear',
        icon: 'i-lucide-camera'
      },
      selectedAttributes: [{ attribute_id: 'existing', required: true, sort_order: 0 }]
    }
    attributes.value = [
      { id: 'existing', name: 'Maker', key: 'maker', data_type: 'string' },
      { id: 'new-attribute', name: 'Serial number', key: 'serial_number', data_type: 'string' }
    ]

    const wrapper = await mountSuspended(NewCategory)
    await flushPromises()

    expect((wrapper.get('input[placeholder="e.g. Rare Books"]').element as HTMLInputElement).value).toBe('Vintage cameras')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('Analog photography gear')
    expect(wrapper.text()).toContain('Maker')
    expect(wrapper.text()).toContain('Serial number')
    expect(refreshAttributes).toHaveBeenCalledOnce()
    expect(draft.value).toBeNull()
    wrapper.unmount()
  })

  it('returns to the draft when attribute creation is cancelled', async () => {
    route.query = { returnTo: 'category' }
    const wrapper = await mountSuspended(NewAttribute)
    const cancel = wrapper.findAll('button').find(button => button.text() === 'Cancel')!
    await cancel.trigger('click')

    expect(push).toHaveBeenCalledWith({
      path: '/categories/new',
      query: { resume: 'attribute' }
    })
    wrapper.unmount()
  })
})
