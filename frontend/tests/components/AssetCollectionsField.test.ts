import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { ref } from 'vue'
import AssetCollectionsField from '../../app/components/AssetCollectionsField.vue'

const { api } = vi.hoisted(() => ({ api: vi.fn() }))
mockNuxtImport('useApi', () => api)

describe('AssetCollectionsField', () => {
  const refresh = vi.fn()
  beforeEach(() => {
    vi.clearAllMocks()
    api.mockReturnValue({
      data: ref([{ id: 'games', name: 'PS5 games', icon: 'i-lucide-gamepad-2' }, { id: 'favorites', name: 'Favorites', icon: 'i-lucide-library' }]),
      status: ref('success'), error: ref(null), refresh
    })
  })

  it('shows assigned collections and lets the user clear every assignment', async () => {
    const wrapper = await mountSuspended(AssetCollectionsField, { props: { modelValue: ['games', 'favorites'] } })
    expect(wrapper.text()).toContain('PS5 games')
    expect(wrapper.text()).toContain('Favorites')
    const clear = wrapper.findAll('button').find(button => button.text() === 'Clear selections')!
    await clear.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([[]])
    wrapper.unmount()
  })

  it('preserves selections when loading fails and offers retry', async () => {
    api.mockReturnValue({ data: ref(null), status: ref('error'), error: ref(new Error('offline')), refresh })
    const wrapper = await mountSuspended(AssetCollectionsField, { props: { modelValue: ['games'] } })
    expect(wrapper.get('[role="alert"]').text()).toContain('could not be loaded')
    await wrapper.findAll('button').find(button => button.text() === 'Try again')!.trigger('click')
    expect(refresh).toHaveBeenCalledOnce()
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    wrapper.unmount()
  })
})
