import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import { ref } from 'vue'
import LocationsPage from '../../app/pages/locations.vue'

const { api, statsRefresh, statsClear } = vi.hoisted(() => ({ api: vi.fn(), statsRefresh: vi.fn(), statsClear: vi.fn() }))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => vi.fn())
mockNuxtImport('useToast', () => () => ({ add: vi.fn() }))
mockNuxtImport('useOrganizationSettings', () => () => ({ settings: ref({ currency: 'USD' }) }))

const stats = ref<{ total_value: number }>()
const statsStatus = ref('idle')
const assets = ref<{ assets: Array<{ id: string, name: string, purchase_price: number }>, total: number }>({ assets: [], total: 0 })

beforeEach(() => {
  vi.clearAllMocks()
  stats.value = undefined
  statsStatus.value = 'idle'
  assets.value = { assets: [], total: 0 }
  statsRefresh.mockImplementation(() => {
    statsStatus.value = 'pending'
    return Promise.resolve()
  })
  api.mockImplementation((url: string | (() => string), options?: { key?: string }) => {
    if (url === '/api/locations') return {
      data: ref([{ id: 'a', name: 'Location A' }, { id: 'b', name: 'Location B' }]), status: ref('success'), refresh: vi.fn()
    }
    if (options?.key === 'location-asset-stats') return {
      data: stats, status: statsStatus, refresh: statsRefresh, clear: statsClear
    }
    return { data: assets, status: ref('success'), refresh: vi.fn() }
  })
})

describe('selected location totals', () => {
  it('uses price precision on asset cards and concise precision on the total', async () => {
    assets.value = { assets: [{ id: 'asset', name: 'Sample asset', purchase_price: 0.5 }], total: 1 }
    const wrapper = await mountSuspended(LocationsPage, { global: { stubs: { UModal: true } } })
    await wrapper.get('[aria-label="View Location A"]').trigger('click')
    stats.value = { total_value: 0.5 }
    statsStatus.value = 'success'
    await wrapper.vm.$nextTick()
    expect(wrapper.get('a[href="/assets/asset"]').text()).toContain('$0.50')
    expect(wrapper.get('[role="status"]').text()).toBe('$0.5')
    wrapper.unmount()
  })

  it('clears the previous request and hides its total while the next location loads or fails', async () => {
    const wrapper = await mountSuspended(LocationsPage, { global: { stubs: { UModal: true } } })
    await wrapper.get('[aria-label="View Location A"]').trigger('click')
    stats.value = { total_value: 1250 }
    statsStatus.value = 'success'
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="status"]').text()).toBe('$1,250')
    // Deliberately retain old mock data: rendering must never label it as B's total.
    await wrapper.get('[aria-label="View Location B"]').trigger('click')
    expect(statsClear).toHaveBeenCalledTimes(2)
    expect(statsRefresh).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[role="status"]').text()).toBe('Loading…')
    statsStatus.value = 'error'
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="status"]').text()).toBe('Unavailable')
    expect(wrapper.text()).not.toContain('$1,250')
    stats.value = { total_value: 42 }
    statsStatus.value = 'success'
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="status"]').text()).toBe('$42')
    wrapper.unmount()
  })

  it('renders a successful zero total as zero rather than an unavailable result', async () => {
    const wrapper = await mountSuspended(LocationsPage, { global: { stubs: { UModal: true } } })
    await wrapper.get('[aria-label="View Location B"]').trigger('click')
    stats.value = { total_value: 0 }
    statsStatus.value = 'success'
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="status"]').text()).toBe('$0')
    wrapper.unmount()
  })
})
