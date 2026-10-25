import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import PluginsPage from '../../app/pages/plugins.vue'
import type { Plugin } from '../../app/types/api'

const { api, useFeatureMock } = vi.hoisted(() => ({ api: vi.fn(), useFeatureMock: vi.fn() }))
mockNuxtImport('useApi', () => api)
mockNuxtImport('useFeatures', () => useFeatureMock)

const panel = {
  props: ['open', 'title', 'description'],
  emits: ['after:leave'],
  watch: { open(value: boolean) { if (!value) this.$emit('after:leave') } },
  template: '<aside v-if="open" aria-label="Source panel"><h2>{{ title }}</h2><p>{{ description }}</p><slot name="body" /><slot name="footer" /></aside>'
}
const importModal = {
  props: ['open', 'initialPluginId'],
  template: '<div v-if="open" data-testid="import" :data-plugin="initialPluginId" />'
}
const mountPage = () => mountSuspended(PluginsPage, { global: { stubs: { USlideover: panel, ImportModal: importModal } } })
function source(id: string, category: string, enabled = true): Plugin {
  return {
    id, name: id, category_name: category, category_description: `${category} imports`,
    description: `Import ${category} from this catalog`, enabled,
    disabled_reason: enabled ? undefined : 'Missing server credential',
    category_id: enabled ? `category-${id}` : undefined,
    search_fields: [{ key: 'name', label: 'Name' }],
    attributes: [{ key: 'catalog_number', name: 'Catalog Number', data_type: 'string', required: false }]
  }
}
const refresh = vi.fn()
let data = ref<{ plugins: Plugin[] }>({ plugins: [] })
let status = ref('success')
let error = ref<Error | null>(null)
let features = ref({ categories: true, attributes: true })
beforeEach(() => {
  vi.clearAllMocks()
  data = ref({ plugins: [
    source('rebrickable_sets', 'LEGO Sets'), source('tmdb_series', 'TV Series', false),
    source('igdb_games', 'Video Games'), source('google_books', 'Books'),
    source('tmdb_movies', 'Movies', false), source('rebrickable_minifigs', 'LEGO Minifigures')
  ] })
  status = ref('success')
  error = ref<Error | null>(null)
  features = ref({ categories: true, attributes: true })
  api.mockReturnValue({ data, status, error, refresh })
  useFeatureMock.mockReturnValue({ features })
})

describe('Import sources page', () => {
  it('leads with full category names and sorts configured sources before shared setup', async () => {
    const wrapper = await mountPage()
    const configured = wrapper.get('section[aria-labelledby="configured-heading"]')
    expect(configured.findAll('h3').map(heading => heading.text())).toEqual(['Books', 'LEGO Minifigures', 'LEGO Sets', 'Video Games'])
    expect(configured.text()).toContain('Rebrickable')
    const setup = wrapper.get('section[aria-labelledby="setup-heading"]')
    expect(setup.findAll('h3').map(heading => heading.text())).toEqual(['Movies & TV Series'])
    expect(setup.findAll('button')).toHaveLength(1)
    expect(setup.text()).toContain('Needs setup 2')
    expect(wrapper.text()).not.toContain('Unavailable')
    expect(wrapper.text()).not.toContain('Created')
    expect(wrapper.get('details').attributes('open')).toBeUndefined()
    wrapper.unmount()
  })
  it('opens metadata details, then preselects that source when importing from the panel', async () => {
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Details for LEGO Sets"]').trigger('click')
    const details = wrapper.get('aside')
    expect(details.text()).toContain('Search by')
    expect(details.text()).toContain('Imported fields')
    expect(details.text()).toContain('Catalog Number')
    expect(details.text()).toContain('Category')
    await details.get('button[aria-label="Import LEGO Sets"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('aside').exists()).toBe(false)
    expect(wrapper.get('[data-testid="import"]').attributes('data-plugin')).toBe('rebrickable_sets')
    wrapper.unmount()
  })
  it('launches import directly from a row', async () => {
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Import Books"]').trigger('click')
    expect(wrapper.get('[data-testid="import"]').attributes('data-plugin')).toBe('google_books')
    wrapper.unmount()
  })
  it('shows provider setup once with server-only credentials and both affected categories', async () => {
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Setup instructions for TMDB"]').trigger('click')
    const setup = wrapper.get('aside')
    expect(setup.text()).toContain('Connect TMDB')
    expect(setup.text()).toContain('Movies')
    expect(setup.text()).toContain('TV Series')
    expect(setup.findAll('code').map(code => code.text())).toEqual(['ATTIC_TMDB_API_KEY'])
    expect(setup.get('a').attributes('href')).toBe('https://www.themoviedb.org/settings/api')
    expect(setup.findAll('input')).toHaveLength(0)
    expect(setup.find('button[aria-label^="Import "]').exists()).toBe(false)
    wrapper.unmount()
  })
  it('preserves shared Rebrickable and IGDB setup instructions', async () => {
    data.value.plugins = [source('rebrickable_sets', 'LEGO Sets', false), source('rebrickable_minifigs', 'LEGO Minifigures', false), source('igdb_games', 'Video Games', false)]
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Setup instructions for Rebrickable"]').trigger('click')
    expect(wrapper.get('aside').text()).toContain('One key enables both LEGO Sets and LEGO Minifigures.')
    expect(wrapper.findAll('code').map(code => code.text())).toEqual(['ATTIC_REBRICKABLE_API_KEY'])
    await wrapper.get('button[aria-label="Setup instructions for IGDB"]').trigger('click')
    expect(wrapper.findAll('code').map(code => code.text())).toEqual(['ATTIC_IGDB_CLIENT_ID', 'ATTIC_IGDB_CLIENT_SECRET'])
    expect(wrapper.get('aside').text()).toContain('Client Secret private')
    wrapper.unmount()
  })
  it('hides category and attribute details when those features are disabled', async () => {
    features.value = { categories: false, attributes: false }
    data.value.plugins[0]!.category_id = undefined
    const wrapper = await mountPage()
    await wrapper.get('button[aria-label="Details for LEGO Sets"]').trigger('click')
    expect(wrapper.find('section[aria-labelledby="import-category-heading"]').exists()).toBe(false)
    expect(wrapper.find('section[aria-labelledby="imported-fields-heading"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Initialization failed')
    expect(wrapper.get('aside').text()).toContain('Search by')
    wrapper.unmount()
  })
  it('supports unknown providers and uses their disabled reason as setup guidance', async () => {
    data.value.plugins = [source('custom', 'Custom catalog', false)]
    data.value.plugins[0]!.disabled_reason = 'Ask the administrator to install the catalog credentials.'
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('Connect a catalog')
    await wrapper.get('button[aria-label="Setup instructions for custom"]').trigger('click')
    expect(wrapper.get('aside').text()).toContain('Ask the administrator to install the catalog credentials.')
    wrapper.unmount()
  })
  it('uses plugin IDs to break category-name sorting ties', async () => {
    data.value.plugins = [source('z', 'Books'), source('a', 'Books')]
    const wrapper = await mountPage()
    await wrapper.findAll('button[aria-label="Import Books"]')[0]!.trigger('click')
    expect(wrapper.get('[data-testid="import"]').attributes('data-plugin')).toBe('a')
    wrapper.unmount()
  })
  it('renders loading, retryable errors, and an empty catalog separately', async () => {
    status.value = 'pending'
    let wrapper = await mountPage()
    expect(wrapper.get('[role="status"]').text()).toContain('Loading import sources')
    wrapper.unmount()
    status.value = 'error'
    error.value = new Error('Unavailable')
    wrapper = await mountPage()
    expect(wrapper.get('[role="alert"]').text()).toContain('Couldn’t load import sources')
    await wrapper.findAll('button').find(button => button.text() === 'Try again')!.trigger('click')
    expect(refresh).toHaveBeenCalledOnce()
    expect(wrapper.text()).not.toContain('No import sources available')
    wrapper.unmount()
    status.value = 'success'
    error.value = null
    data.value.plugins = []
    wrapper = await mountPage()
    expect(wrapper.text()).toContain('No import sources available')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
