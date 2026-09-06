import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import EditAsset from '../../app/pages/assets/[id]/edit.vue'
import NewAsset from '../../app/pages/assets/new.vue'

const { api, mutate, toast } = vi.hoisted(() => ({ api: vi.fn(), mutate: vi.fn(), toast: vi.fn() }))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useApiFetch', () => () => mutate)
mockNuxtImport('useToast', () => () => ({ add: toast }))
mockNuxtImport('useRoute', () => () => ({ params: { id: 'asset' }, query: {} }))

describe('Asset form sections', () => {
  const asset = ref<Record<string, unknown> | null>(null)
  beforeEach(() => {
    vi.clearAllMocks()
    asset.value = { id: 'asset', name: 'Desk', quantity: 1 }
    api.mockImplementation((url: unknown) => ({
      data: typeof url === 'function' ? asset : ref([]), status: ref('success'), error: ref(null)
    }))
    mutate.mockResolvedValue({ id: 'new-asset' })
  })

  it('always shows both sections when creating an asset', async () => {
    const wrapper = await mountSuspended(NewAsset)
    for (const title of ['Details and notes', 'Purchase information']) {
      const heading = wrapper.findAll('h2').find(node => node.text() === title)!
      expect(heading.element.closest('section')).not.toBeNull()
      expect(heading.element.closest('details')).toBeNull()
    }
    wrapper.unmount()
  })

  it.each([
    [{}, false, false],
    [{ description: '  ', notes: '\n', purchase_note: ' ' }, false, false],
    [{ description: 'Oak desk' }, true, false],
    [{ notes: 'Upstairs' }, true, false],
    [{ condition_id: 'used' }, true, false],
    [{ quantity: 2 }, true, false],
    [{ purchase_at: '2026-09-06' }, false, true],
    [{ purchase_price: 0 }, false, true],
    [{ purchase_note: 'Gift' }, false, true]
  ])('sets edit expansion from saved fields %j', async (fields, details, purchase) => {
    const wrapper = await mountSuspended(EditAsset)
    asset.value = { ...asset.value, ...fields }
    await nextTick()
    const sections = wrapper.findAll('details')
    expect((sections[0]!.element as HTMLDetailsElement).open).toBe(details)
    expect((sections[1]!.element as HTMLDetailsElement).open).toBe(purchase)
    wrapper.unmount()
  })

  it('does not collapse an expanded section when its text is cleared', async () => {
    asset.value = { ...asset.value, notes: 'A note' }
    const wrapper = await mountSuspended(EditAsset)
    const section = wrapper.findAll('details')[0]!
    await section.findAll('textarea')[1]!.setValue('')
    expect((section.element as HTMLDetailsElement).open).toBe(true)
    wrapper.unmount()
  })

  it('creates an asset without a category', async () => {
    const wrapper = await mountSuspended(NewAsset)
    await wrapper.get('#asset-name').setValue('Unsorted item')
    await wrapper.get('#new-asset-form').trigger('submit')
    await flushPromises()
    const payload = JSON.parse(mutate.mock.calls[0]![1].body)
    expect(payload.name).toBe('Unsorted item')
    expect(payload).not.toHaveProperty('category_id')
    wrapper.unmount()
  })

  it('removes a category and its attributes while editing', async () => {
    asset.value = { ...asset.value, category_id: 'games', attributes: { platform: 'PS5' } }
    mutate.mockImplementation(async (url: string) => url === '/api/categories/games'
      ? { id: 'games', name: 'Games', attributes: [] }
      : undefined)
    const wrapper = await mountSuspended(EditAsset)
    await nextTick()
    await wrapper.get('input[name="category"]').setValue()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    const updateCall = mutate.mock.calls.find(([url]) => url === '/api/assets/asset')!
    const payload = JSON.parse(updateCall[1].body)
    expect(payload).not.toHaveProperty('category_id')
    expect(payload).not.toHaveProperty('attributes')
    wrapper.unmount()
  })
})
