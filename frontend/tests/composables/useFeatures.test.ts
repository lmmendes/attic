import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import type { OrganizationFeatures } from '../../app/types/api'
import { useFeatures } from '../../app/composables/useFeatures'

const { data, error, status, refresh } = vi.hoisted(() => ({
  data: { value: null as OrganizationFeatures | null },
  error: { value: null as Error | null },
  status: { value: 'idle' as 'idle' | 'pending' | 'success' | 'error' },
  refresh: vi.fn()
}))

mockNuxtImport('useApi', () => () => ({ data, error, status, refresh }))

const enabledFeatures: OrganizationFeatures = {
  locations: true,
  collections: true,
  categories: true,
  attributes: true,
  conditions: true,
  warranties: true,
  plugins: true
}

async function mountFeatureConsumer() {
  let result!: ReturnType<typeof useFeatures>
  const wrapper = await mountSuspended(defineComponent({
    setup() {
      result = useFeatures()
      return () => h('div')
    }
  }))
  return { result, wrapper }
}

describe('useFeatures loading', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    data.value = null
    error.value = null
    status.value = 'idle'
    refresh.mockResolvedValue(undefined)
  })

  it('opens the gate only after a successful valid response', async () => {
    const { result, wrapper } = await mountFeatureConsumer()
    data.value = { ...enabledFeatures }
    status.value = 'success'

    expect(await result.load()).toBe(true)
    expect(result.loaded.value).toBe(true)
    expect(result.features.value).toEqual(enabledFeatures)
    wrapper.unmount()
  })

  it('closes the gate and feature state when refresh fails', async () => {
    const { result, wrapper } = await mountFeatureConsumer()
    result.features.value = { ...enabledFeatures }
    result.loaded.value = true
    status.value = 'error'
    error.value = new Error('request failed')

    expect(await result.load()).toBe(false)
    expect(result.loaded.value).toBe(false)
    expect(result.error.value).toBeTruthy()
    expect(Object.values(result.features.value).every(value => value === false)).toBe(true)
    wrapper.unmount()
  })

  it('keeps the gate closed when a successful request has missing data', async () => {
    const { result, wrapper } = await mountFeatureConsumer()
    status.value = 'success'

    expect(await result.load()).toBe(false)
    expect(result.loaded.value).toBe(false)
    expect(result.error.value).toBeTruthy()
    expect(Object.values(result.features.value).every(value => value === false)).toBe(true)
    wrapper.unmount()
  })
})
