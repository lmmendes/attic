import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { ref } from 'vue'
import AssetHistory from '../../app/components/AssetHistory.vue'

const { api, mutate, refresh, toast } = vi.hoisted(() => ({
  api: vi.fn(), mutate: vi.fn(), refresh: vi.fn(), toast: vi.fn()
}))

mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useToast', () => () => ({ add: toast }))

const modal = {
  props: ['open'],
  emits: ['update:open'],
  template: '<div v-if="open" data-modal><slot name="content" /></div>'
}

const event = {
  id: 'event-1', asset_id: 'asset-1', title: 'Repaired', description: 'Changed belt',
  icon: 'i-lucide-wrench', event_date: '2026-09-07',
  created_at: '2026-09-08T09:00:00Z', updated_at: '2026-09-08T09:00:00Z'
}

async function mountHistory() {
  return mountSuspended(AssetHistory, {
    props: {
      assetId: 'asset-1',
      createdAt: '2026-01-01T10:00:00Z',
      updatedAt: '2026-09-08T10:00:00Z'
    },
    global: { stubs: { UModal: modal } }
  })
}

describe('AssetHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.mockReturnValue({ data: ref([event]), status: ref('success'), error: ref(null), refresh })
    mutate.mockResolvedValue({})
  })

  it('renders custom and generated entries in reverse chronological order', async () => {
    const wrapper = await mountHistory()
    const titles = wrapper.findAll('li p.font-bold').map(node => node.text())
    expect(titles).toEqual(['Last Updated', 'Repaired', 'Asset Created'])
    expect(wrapper.text()).toContain('Changed belt')
    expect(wrapper.find('button[aria-label="Actions for Repaired"]').exists()).toBe(true)
    expect(wrapper.get('button[aria-label="No actions available for Asset Created"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('creates an event and refreshes the timeline', async () => {
    const wrapper = await mountHistory()
    await wrapper.findAll('button').find(button => button.text() === 'Add event')!.trigger('click')
    const form = wrapper.get('form')
    await form.find('input[type="text"]').setValue('Inspected')
    await form.find('textarea').setValue('Everything works')
    await form.find('input[type="date"]').setValue('2030-01-02')
    await form.trigger('submit')
    await flushPromises()

    expect(mutate).toHaveBeenCalledWith('/api/assets/asset-1/events', {
      method: 'POST',
      body: JSON.stringify({
        title: 'Inspected', description: 'Everything works',
        icon: 'i-lucide-calendar', event_date: '2030-01-02'
      })
    })
    expect(refresh).toHaveBeenCalledOnce()
    expect(toast).toHaveBeenCalledWith({ title: 'Event added', color: 'success' })
    wrapper.unmount()
  })

  it('updates an event with pre-filled values', async () => {
    const wrapper = await mountHistory()
    await wrapper.get('button[aria-label="Actions for Repaired"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const editItem = document.querySelectorAll<HTMLElement>('[role="menuitem"]')
    Array.from(editItem).find(item => item.textContent?.includes('Edit'))!.click()
    await flushPromises()
    const form = wrapper.get('form')
    expect((form.find('input[type="text"]').element as HTMLInputElement).value).toBe('Repaired')
    await form.find('input[type="text"]').setValue('Serviced')
    await form.trigger('submit')
    await flushPromises()

    expect(mutate).toHaveBeenCalledWith('/api/assets/asset-1/events/event-1', expect.objectContaining({ method: 'PUT' }))
    expect(refresh).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('requires confirmation before deleting an event', async () => {
    const wrapper = await mountHistory()
    await wrapper.get('button[aria-label="Actions for Repaired"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const deleteItem = Array.from(document.querySelectorAll<HTMLElement>('[role="menuitem"]'))
      .find(item => item.textContent?.includes('Delete'))!
    deleteItem.click()
    await flushPromises()
    expect(mutate).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(button => button.text() === 'Delete event')!.trigger('click')
    await flushPromises()

    expect(mutate).toHaveBeenCalledWith('/api/assets/asset-1/events/event-1', { method: 'DELETE' })
    expect(refresh).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('preserves values and exposes API errors when saving fails', async () => {
    mutate.mockRejectedValue({ data: { error: 'event_date must be valid' } })
    const wrapper = await mountHistory()
    await wrapper.findAll('button').find(button => button.text() === 'Add event')!.trigger('click')
    const input = wrapper.get('form input[type="text"]')
    await input.setValue('Inspection')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('event_date must be valid')
    expect((input.element as HTMLInputElement).value).toBe('Inspection')
    wrapper.unmount()
  })
})
