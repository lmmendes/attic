import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import { useOrganizationSettings } from '../../app/composables/useOrganizationSettings'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

mockNuxtImport('useApiFetch', () => () => apiFetch)

async function mountSettingsConsumer() {
  let result!: ReturnType<typeof useOrganizationSettings>
  const wrapper = await mountSuspended(defineComponent({
    setup() {
      result = useOrganizationSettings()
      return () => h('div')
    }
  }))
  return { result, wrapper }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: Error) => void
  const promise = new Promise<T>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

beforeEach(() => apiFetch.mockReset())

describe('organization settings', () => {
  it('loads the currency, keeps it when a reload fails, and resets it on logout', async () => {
    const { result, wrapper } = await mountSettingsConsumer()
    result.reset()

    apiFetch.mockResolvedValueOnce({ currency: 'GBP' })
    await result.load()
    expect(result.settings.value.currency).toBe('GBP')
    expect(result.loaded.value).toBe(true)
    expect(result.error.value).toBeNull()

    apiFetch.mockRejectedValueOnce(new Error('offline'))
    await result.load()
    expect(result.settings.value.currency).toBe('GBP')
    expect(result.loaded.value).toBe(false)
    expect(result.error.value?.message).toBe('offline')

    result.reset()
    expect(result.settings.value.currency).toBe('USD')
    expect(result.loaded.value).toBe(false)
    expect(result.error.value).toBeNull()
    wrapper.unmount()
  })
})

describe('settings request invalidation', () => {
  it('shares an in-flight load across consumers', async () => {
    const first = await mountSettingsConsumer()
    const second = await mountSettingsConsumer()
    first.result.reset()
    const response = deferred<{ currency: string }>()
    apiFetch.mockReturnValueOnce(response.promise)
    const initialLoad = first.result.load()
    const settingsPageLoad = second.result.load()
    expect(apiFetch).toHaveBeenCalledOnce()
    response.resolve({ currency: 'EUR' })
    await Promise.all([initialLoad, settingsPageLoad])
    for (const consumer of [first, second]) {
      expect(consumer.result.settings.value.currency).toBe('EUR')
      expect(consumer.result.loaded.value).toBe(true)
      consumer.wrapper.unmount()
    }
  })

  it.each(['success', 'failure'])('ignores a late load %s after another consumer resets settings', async (outcome) => {
    const first = await mountSettingsConsumer()
    const second = await mountSettingsConsumer()
    first.result.reset()
    const response = deferred<{ currency: string }>()
    apiFetch.mockReturnValueOnce(response.promise)
    const pending = first.result.load()
    second.result.reset()
    if (outcome === 'success') response.resolve({ currency: 'GBP' })
    else response.reject(new Error('expired session'))
    await pending
    for (const consumer of [first, second]) {
      expect(consumer.result.settings.value.currency).toBe('USD')
      expect(consumer.result.loaded.value).toBe(false)
      expect(consumer.result.error.value).toBeNull()
      consumer.wrapper.unmount()
    }
  })

  it('ignores an update response received after reset', async () => {
    const { result, wrapper } = await mountSettingsConsumer()
    result.reset()
    const response = deferred<{ currency: string }>()
    apiFetch.mockReturnValueOnce(response.promise)
    const pending = result.update({ currency: 'GBP' })
    result.reset()
    response.resolve({ currency: 'GBP' })
    await pending
    expect(result.settings.value.currency).toBe('USD')
    expect(result.loaded.value).toBe(false)
    expect(result.error.value).toBeNull()
    wrapper.unmount()
  })

  it('keeps a new login value when the old session response arrives last', async () => {
    const { result, wrapper } = await mountSettingsConsumer()
    result.reset()
    const oldResponse = deferred<{ currency: string }>()
    apiFetch.mockReturnValueOnce(oldResponse.promise)
    const oldLoad = result.load()
    result.reset()
    apiFetch.mockResolvedValueOnce({ currency: 'EUR' })
    await result.load()
    oldResponse.resolve({ currency: 'GBP' })
    await oldLoad
    expect(result.settings.value.currency).toBe('EUR')
    expect(result.loaded.value).toBe(true)
    wrapper.unmount()
  })

  it('does not overwrite a saved update with an older load', async () => {
    const { result, wrapper } = await mountSettingsConsumer()
    result.reset()
    const oldResponse = deferred<{ currency: string }>()
    apiFetch.mockReturnValueOnce(oldResponse.promise)
    const oldLoad = result.load()
    apiFetch.mockResolvedValueOnce({ currency: 'EUR' })
    await result.update({ currency: 'EUR' })
    oldResponse.resolve({ currency: 'GBP' })
    await oldLoad
    expect(result.settings.value.currency).toBe('EUR')
    expect(result.loaded.value).toBe(true)
    wrapper.unmount()
  })
})
