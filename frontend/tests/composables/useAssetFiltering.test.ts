import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, reactive, ref } from 'vue'
import { useAssetFiltering } from '../../app/composables/useAssetFiltering'
import type { AssetFilters, SavedFilter } from '../../app/types/api'

const { fetch, api, replace, route } = vi.hoisted(() => ({ fetch: vi.fn(), api: vi.fn(), replace: vi.fn(), route: { query: {} as Record<string, string> } }))
mockNuxtImport('useApiFetch', () => () => fetch)
mockNuxtImport('useApi', () => api)
mockNuxtImport('useRoute', () => () => route)
mockNuxtImport('useRouter', () => () => ({ replace }))
const mocks = { fetch, api, replace, route }

const saved: SavedFilter = { id: 'saved-1', name: 'Retro', pinned: false, criteria: { version: 1, q: 'retro', category_id: 'old-category' }, created_at: '', updated_at: '' }
let state: ReturnType<typeof useAssetFiltering>
let filters: AssetFilters
async function setup(query: Record<string, string> = {}) {
  route.query = reactive(query)
  const wrapper = mount(defineComponent({
    setup() {
      filters = reactive({ limit: 24, offset: 0 })
      state = useAssetFiltering(filters)
      return () => h('div')
    }
  }))
  return wrapper
}

describe('personal asset filters', () => {
  beforeEach(() => {
    mocks.fetch.mockReset()
    mocks.replace.mockReset()
    mocks.api.mockReturnValue({ data: ref([saved]), error: ref(null), refresh: vi.fn() })
  })

  it('selects saved criteria, keeps modified drafts private, and saves without pagination', async () => {
    const wrapper = await setup()
    mocks.fetch.mockResolvedValue(saved)
    filters.offset = 72
    await state.selectSaved(saved.id)
    expect(filters.offset).toBe(0)
    expect(state.structured.value).toBe(true)
    expect(state.criteria.value).toEqual(saved.criteria)
    filters.q = 'changed'
    expect(state.modified.value).toBe(true)
    expect(saved.criteria.q).toBe('retro')
    expect(mocks.fetch).toHaveBeenCalledTimes(1)
    filters.offset = 48
    const criteria = state.criteria.value
    mocks.fetch.mockResolvedValue({ ...saved, criteria })
    await state.save('Updated', criteria, true)
    expect(mocks.fetch).toHaveBeenLastCalledWith('/api/saved-filters/saved-1', { method: 'PUT', body: JSON.stringify({ name: 'Updated', criteria }) })
    expect(JSON.parse(mocks.fetch.mock.calls.at(-1)![1].body).criteria).not.toHaveProperty('offset')
    expect(state.modified.value).toBe(false)
    wrapper.unmount()
  })

  it('keeps top-only advanced mode in the URL and restores browser navigation', async () => {
    const wrapper = await setup()
    state.apply({ version: 1, category_id: 'category' })
    await nextTick()
    const query = mocks.replace.mock.calls.at(-1)![0].query
    expect(JSON.parse(query.criteria)).toEqual({ version: 1, category_id: 'category' })
    state.clear()
    await nextTick()
    expect(state.structured.value).toBe(false)
    Object.assign(route.query, query)
    await nextTick()
    expect(state.structured.value).toBe(true)
    expect(filters.category_id).toBe('category')
    filters.offset = 96
    await nextTick()
    expect(filters.offset).toBe(96)
    filters.q = 'reset paging'
    expect(filters.offset).toBe(0)
    wrapper.unmount()
  })

  it('loads a saved-filter-only URL and retains stale definitions for repair', async () => {
    const stale = { ...saved, issues: [{ path: 'category_id', message: 'Category no longer exists' }] }
    mocks.fetch.mockResolvedValue(stale)
    const wrapper = await setup({ saved_filter_id: saved.id })
    await nextTick()
    expect(state.criteria.value).toEqual(saved.criteria)
    expect(state.issues.value).toEqual(stale.issues)
    expect(state.message.value).toContain('repair')
    mocks.fetch.mockResolvedValue({ ...stale, name: 'Renamed' })
    await state.rename('Renamed')
    expect(mocks.fetch).toHaveBeenLastCalledWith('/api/saved-filters/saved-1', { method: 'PUT', body: JSON.stringify({ name: 'Renamed' }) })
    await state.remove()
    expect(mocks.fetch).toHaveBeenLastCalledWith('/api/saved-filters/saved-1', { method: 'DELETE' })
    expect(state.selectedId.value).toBeUndefined()
    expect(state.criteria.value).toEqual(saved.criteria)
    expect(state.structured.value).toBe(true)
    wrapper.unmount()
  })

  it('pins and unpins the selected saved search', async () => {
    const wrapper = await setup()
    mocks.fetch.mockResolvedValueOnce(saved)
    await state.selectSaved(saved.id)
    mocks.fetch.mockResolvedValueOnce({ ...saved, pinned: true })
    expect(await state.togglePin()).toBe(true)
    expect(mocks.fetch).toHaveBeenLastCalledWith('/api/saved-filters/saved-1', {
      method: 'PUT', body: JSON.stringify({ name: 'Retro', pinned: true })
    })
    expect(state.selected.value?.pinned).toBe(true)
    mocks.fetch.mockResolvedValueOnce({ ...saved, pinned: false })
    expect(await state.togglePin()).toBe(true)
    expect(state.selected.value?.pinned).toBe(false)
    wrapper.unmount()
  })

  it('does not replace explicit modified criteria with the saved definition on reload', async () => {
    mocks.fetch.mockResolvedValue(saved)
    const criteria = { version: 1, q: 'modified' }
    const wrapper = await setup({ saved_filter_id: saved.id, criteria: JSON.stringify(criteria) })
    await nextTick()
    expect(state.criteria.value).toEqual(criteria)
    expect(state.modified.value).toBe(true)
    wrapper.unmount()
  })

  it('ignores late saved responses after Clear', async () => {
    const wrapper = await setup()
    let resolve!: (value: SavedFilter) => void
    mocks.fetch.mockReturnValue(new Promise<SavedFilter>((done) => {
      resolve = done
    }))
    const loading = state.selectSaved(saved.id)
    state.clear()
    resolve(saved)
    await loading
    expect(state.criteria.value).toEqual({ version: 1 })
    expect(state.selectedId.value).toBeUndefined()
    wrapper.unmount()
  })

  it('shows repair issues when reloading unchanged explicit saved criteria', async () => {
    const issues = [{ path: 'category_id', message: 'Category was deleted' }]
    mocks.fetch.mockResolvedValue({ ...saved, issues })
    const wrapper = await setup({ saved_filter_id: saved.id, criteria: JSON.stringify(saved.criteria) })
    await nextTick()
    expect(state.modified.value).toBe(false)
    expect(state.issues.value).toEqual(issues)
    expect(state.message.value).toContain('repair')
    wrapper.unmount()
  })

  it('surfaces server rule issues without applying a failed save', async () => {
    const wrapper = await setup()
    const issues = [{ path: 'expression.children[0]', message: 'Attribute changed type' }]
    mocks.fetch.mockRejectedValue({ data: { error: 'Invalid filter', issues } })
    expect(await state.save('Broken', { version: 1, q: 'draft' }, false)).toBe(false)
    expect(state.criteria.value).toEqual({ version: 1 })
    expect(state.issues.value).toEqual(issues)
    expect(state.message.value).toBe('Invalid filter')
    wrapper.unmount()
  })
})
