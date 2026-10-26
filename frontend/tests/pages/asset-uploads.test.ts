import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import AssetPage from '../../app/pages/assets/[id]/index.vue'

const { api, toast, refreshAsset, refreshAttachments } = vi.hoisted(() => ({
  api: vi.fn(), toast: vi.fn(), refreshAsset: vi.fn(), refreshAttachments: vi.fn()
}))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useToast', () => () => ({ add: toast }))
mockNuxtImport('useRoute', () => () => ({ params: { id: 'asset' }, query: {} }))
mockNuxtImport('useFeatures', () => () => ({ features: ref({ warranties: false }) }))

describe('Asset attachment uploads', () => {
  const upload = vi.fn()
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('fetch', upload)
    upload.mockReset().mockResolvedValue({ ok: true })
    api.mockImplementation((url: () => string) => {
      const path = url()
      return {
        data: ref(path.endsWith('/attachments')
          ? []
          : path.endsWith('/warranty')
            ? null
            : {
                id: 'asset', name: 'Desk', quantity: 1
              }),
        refresh: path.endsWith('/attachments') ? refreshAttachments : refreshAsset,
        status: ref('success'), error: ref(null)
      }
    })
  })
  afterEach(() => vi.unstubAllGlobals())

  const mountPage = () => mountSuspended(AssetPage, {
    global: { stubs: { AssetHistory: true, UModal: true } }
  })
  const file = (name: string) => new File([name], name, { type: 'text/plain' })
  const sentNames = () => upload.mock.calls.map(([, options]) => options.body.get('file').name)

  it('retains failed and unattempted files across new batches and retries only those files', async () => {
    const wrapper = await mountPage()
    const input = wrapper.get('input[type="file"]')
    expect(input.attributes()).toHaveProperty('multiple')
    upload.mockResolvedValueOnce({ ok: true }).mockResolvedValueOnce({ ok: false, status: 500 })
    Object.defineProperty(input.element, 'files', {
      configurable: true, value: [file('first.txt'), file('failed.txt'), file('remaining.txt')]
    })
    await input.trigger('change')
    await flushPromises()
    expect(sentNames()).toEqual(['first.txt', 'failed.txt'])
    expect(wrapper.text()).toContain('2 files not uploaded')
    expect(refreshAttachments).toHaveBeenCalledOnce()
    expect(refreshAsset).toHaveBeenCalledOnce()
    expect(toast).toHaveBeenCalledWith({ title: 'Uploaded 1 of 3 files', color: 'warning' })

    const section = input.element.closest('section')!
    section.dispatchEvent(Object.assign(new Event('drop', { bubbles: true, cancelable: true }), {
      dataTransfer: { files: [file('new.txt')] }
    }))
    await flushPromises()
    expect(wrapper.text()).toContain('2 files not uploaded')
    Array.from(section.querySelectorAll('button')).find(button => button.textContent?.trim() === 'Retry')!.click()
    await flushPromises()
    expect(sentNames()).toEqual(['first.txt', 'failed.txt', 'new.txt', 'failed.txt', 'remaining.txt'])
    expect(wrapper.text()).not.toContain('not uploaded')
    wrapper.unmount()
  })

  it('queues a second drop while an upload is in progress', async () => {
    const wrapper = await mountPage()
    let complete!: (response: { ok: boolean }) => void
    upload.mockImplementationOnce(() => new Promise((resolve) => {
      complete = resolve
    }))
    const section = wrapper.get('input[type="file"]').element.closest('section')!
    const drop = (name: string) => section.dispatchEvent(Object.assign(new Event('drop', { bubbles: true, cancelable: true }), {
      dataTransfer: { files: [file(name)] }
    }))
    drop('first.txt')
    drop('queued.txt')
    expect(sentNames()).toEqual(['first.txt'])
    complete({ ok: true })
    await flushPromises()
    expect(sentNames()).toEqual(['first.txt', 'queued.txt'])
    wrapper.unmount()
  })
})
