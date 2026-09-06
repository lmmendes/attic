import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import CollectionsPage from '../../app/pages/collections.vue'

const { api, mutate, toast } = vi.hoisted(() => ({ api: vi.fn(), mutate: vi.fn(), toast: vi.fn() }))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useToast', () => () => ({ add: toast }))

const modal = { props: ['open'], template: '<div v-if="open"><slot name="body" /></div>' }
const mountPage = () => mountSuspended(CollectionsPage, { global: { stubs: { UModal: modal } } })

describe('Collections management', () => {
  const refresh = vi.fn()
  beforeEach(() => {
    vi.clearAllMocks()
    api.mockReturnValue({ data: ref([{ id: 'games', name: 'PS5 games', description: 'Our favorites', icon: 'i-lucide-gamepad-2', asset_count: 3 }]), status: ref('success'), error: ref(null), refresh })
    mutate.mockResolvedValue({})
  })

  it('links to filtered inventory and saves edited metadata', async () => {
    const wrapper = await mountPage()
    expect(wrapper.find('a[href="/assets?collection_id=games"]').text()).toContain('View assets')
    expect(wrapper.get('article').text()).toContain('3 assets')
    await wrapper.get('button[aria-label="Edit PS5 games"]').trigger('click')
    await wrapper.get('form input').setValue('PlayStation 5')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(mutate).toHaveBeenCalledWith('/api/collections/games', {
      method: 'PUT', body: JSON.stringify({ name: 'PlayStation 5', description: 'Our favorites', icon: 'i-lucide-gamepad-2' })
    })
    expect(refresh).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('preserves input and displays duplicate-name errors', async () => {
    mutate.mockRejectedValue({ data: { error: 'a collection with this name already exists' } })
    const wrapper = await mountPage()
    await wrapper.findAll('button').find(button => button.text() === 'New collection')!.trigger('click')
    await wrapper.get('form input').setValue('PS5 games')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('already exists')
    expect((wrapper.get('form input').element as HTMLInputElement).value).toBe('PS5 games')
    expect(mutate.mock.calls[0]?.[1].method).toBe('POST')
    wrapper.unmount()
  })

  it('requires confirmation before deleting a collection', async () => {
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Actions for PS5 games"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const deleteItem = document.querySelector<HTMLElement>('[role="menuitem"]')!
    expect(deleteItem.textContent).toContain('Delete')
    deleteItem.click()
    await flushPromises()
    expect(mutate).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(button => button.text() === 'Delete collection')!.trigger('click')
    await flushPromises()
    expect(mutate).toHaveBeenCalledWith('/api/collections/games', { method: 'DELETE' })
    expect(refresh).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('searches names and descriptions without changing summary totals', async () => {
    api.mockReturnValue({ data: ref([
      { id: 'games', name: 'PS5 games', description: 'Our favorites', icon: 'i-lucide-gamepad-2', asset_count: 3 },
      { id: 'books', name: 'Books', description: 'Weekend reading', icon: 'i-lucide-book-open', asset_count: 0 }
    ]), status: ref('success'), error: ref(null), refresh })
    const wrapper = await mountPage()
    expect(wrapper.get('dl').text()).toContain('Collections2')
    expect(wrapper.get('dl').text()).toContain('With assets1')
    expect(wrapper.get('dl').text()).toContain('Empty1')
    await wrapper.get('input[aria-label="Search collections"]').setValue(' WEEKEND ')
    expect(wrapper.findAll('article')).toHaveLength(1)
    expect(wrapper.get('article').text()).toContain('Books')
    expect(wrapper.get('[role="status"]').text()).toBe('Showing 1 of 2')
    expect(wrapper.get('dl').text()).toContain('Collections2')
    await wrapper.get('input[aria-label="Search collections"]').setValue('unknown')
    expect(wrapper.text()).toContain('No matching collections')
    await wrapper.findAll('button').find(button => button.text() === 'Clear search')!.trigger('click')
    expect(wrapper.findAll('article')).toHaveLength(2)
    wrapper.unmount()
  })
})
