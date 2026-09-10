import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import NewCategory from '../../app/pages/categories/new.vue'
import NewAttribute from '../../app/pages/attributes/new.vue'

const { api, mutate, push, replace, resolve, toast, route, draft, clearCategories } = vi.hoisted(() => ({
  api: vi.fn(),
  mutate: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((to: string | { path: string }) => ({ href: typeof to === 'string' ? to : to.path })),
  toast: vi.fn(),
  clearCategories: vi.fn(),
  route: { query: {} as Record<string, string> },
  draft: { value: null as Record<string, unknown> | null }
}))

mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useRouter', () => () => ({ push, replace, resolve }))
mockNuxtImport('useRoute', () => () => route)
mockNuxtImport('useToast', () => () => ({ add: toast }))
mockNuxtImport('useState', () => () => draft)
mockNuxtImport('useConfiguration', () => () => ({
  configuration: ref({
    collections_enabled: true,
    plugins_enabled: true,
    conditions_enabled: true,
    locations_enabled: true
  })
}))

describe('creating an attribute from a category draft', () => {
  const attributes = ref<Record<string, unknown>[]>([])
  const refreshAttributes = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {}
    draft.value = null
    attributes.value = []
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
    expect(push).toHaveBeenCalledWith({ path: '/attributes/new', query: { returnTo: 'category' } })
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
