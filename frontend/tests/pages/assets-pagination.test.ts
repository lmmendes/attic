import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { ref } from 'vue'
import AssetsPage from '../../app/pages/assets/index.vue'

const { api, routeQuery } = vi.hoisted(() => ({ api: vi.fn(), routeQuery: {} as Record<string, string> }))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useRoute', () => () => ({ query: routeQuery }))

const select = {
  props: ['modelValue', 'items'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option value="">All</option><option v-for="item in items" :key="item.value" :value="item.value">{{ item.label }}</option></select>'
}

describe('Inventory pagination', () => {
  beforeEach(() => {
    api.mockReset()
    delete routeQuery.category_id
    api.mockImplementation((url: string | (() => string)) => ({
      data: ref(typeof url === 'function'
        ? { assets: [], total: 240 }
        : [{ id: 'selected', name: 'Selected', label: 'Selected' }]),
      status: ref('success'), error: ref(null), refresh: vi.fn()
    }))
  })

  const mountPage = () => mountSuspended(AssetsPage, {
    route: '/assets',
    global: { stubs: { USelectMenu: select, ImportModal: true } }
  })

  it('applies the category from a library card link', async () => {
    routeQuery.category_id = 'selected'
    const wrapper = await mountPage()
    const getUrl = api.mock.calls.find(([url]) => typeof url === 'function')![0] as () => string
    expect(getUrl()).toContain('category_id=selected')
    expect((wrapper.findAll('select')[1]!.element as HTMLSelectElement).value).toBe('selected')
    wrapper.unmount()
  })

  it('keeps the current and last page reachable beyond page five', async () => {
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Page 10"]').trigger('click')
    expect(wrapper.get('[aria-current="page"]').text()).toBe('10')
    await wrapper.get('button[aria-label="Page 8"]').trigger('click')
    expect(wrapper.get('[aria-current="page"]').text()).toBe('8')
    expect(wrapper.findAll('button[aria-label^="Page "]')).toHaveLength(5)
    expect(wrapper.find('button[aria-label="Page 1"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Page 10"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it.each([1, 2, 3])('resets offset when filter %i changes on a later page', async (index) => {
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Page 10"]').trigger('click')
    const getUrl = api.mock.calls.find(([url]) => typeof url === 'function')![0] as () => string
    expect(getUrl()).toContain('offset=216')
    await wrapper.findAll('select')[index]!.setValue('selected')
    expect(getUrl()).toContain('offset=0')
    expect(getUrl()).toContain(`${['', 'category_id', 'location_id', 'condition_id'][index]}=selected`)
    expect(wrapper.get('[aria-current="page"]').text()).toBe('1')
    wrapper.unmount()
  })
})
