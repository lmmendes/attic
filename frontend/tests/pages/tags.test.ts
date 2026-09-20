import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import { ref } from 'vue'
import TagsPage from '../../app/pages/tags.vue'

const { api, mutate, toast } = vi.hoisted(() => ({ api: vi.fn(), mutate: vi.fn(), toast: vi.fn() }))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useToast', () => () => ({ add: toast }))

const modal = {
  props: ['open', 'title'],
  template: '<div v-if="open" role="dialog" :aria-label="title"><slot name="body" /><slot name="footer" /></div>'
}

describe('Tags management', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.mockReturnValue({
      data: ref([{
        id: 'retro', organization_id: 'org', name: 'Retro', asset_count: 1,
        created_at: '', updated_at: ''
      }]),
      status: ref('success'),
      error: ref(null),
      refresh: vi.fn()
    })
  })

  it('shows tag deletion failures to the user', async () => {
    mutate.mockRejectedValueOnce({ data: { error: 'Tag deletion failed' } })
    const wrapper = await mountSuspended(TagsPage, { global: { stubs: { UModal: modal } } })

    await wrapper.get('button[aria-label="Delete Retro"]').trigger('click')
    await wrapper.get('[role="dialog"] button:last-child').trigger('click')
    await flushPromises()

    expect(toast).toHaveBeenCalledWith({ title: 'Tag deletion failed', color: 'error' })
    wrapper.unmount()
  })
})
