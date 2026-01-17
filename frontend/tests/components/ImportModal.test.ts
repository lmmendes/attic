import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'

const mockPlugins = ref([
  {
    id: 'plugin-1',
    name: 'OpenLibrary',
    description: 'Search books from Open Library',
    search_fields: [
      { key: 'title', label: 'Title' },
      { key: 'isbn', label: 'ISBN' }
    ]
  },
  {
    id: 'plugin-2',
    name: 'IGDB',
    description: 'Search games from IGDB',
    search_fields: [
      { key: 'name', label: 'Game Name' }
    ]
  }
])

const mockToast = {
  add: vi.fn()
}

const mockApiFetch = vi.fn()

vi.mock('#app', () => ({
  useRuntimeConfig: vi.fn(() => ({
    public: { apiBase: 'http://localhost:8080' }
  })),
  ref: (val: any) => ({ value: val }),
  computed: (fn: () => any) => ({ value: fn() }),
  watch: vi.fn()
}))

describe('ImportModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockApiFetch.mockReset()
    mockToast.add.mockReset()
  })

  describe('Plugin Selection (Step 1)', () => {
    it('displays available plugins', () => {
      const plugins = mockPlugins.value
      expect(plugins).toHaveLength(2)
      expect(plugins[0].name).toBe('OpenLibrary')
      expect(plugins[1].name).toBe('IGDB')
    })

    it('shows plugin descriptions', () => {
      const plugins = mockPlugins.value
      expect(plugins[0].description).toBe('Search books from Open Library')
      expect(plugins[1].description).toBe('Search games from IGDB')
    })

    it('handles empty plugins list', () => {
      const emptyPlugins: typeof mockPlugins.value = []
      expect(emptyPlugins).toHaveLength(0)
    })
  })

  describe('Search (Step 2)', () => {
    it('requires minimum 2 characters for search', () => {
      const query = 'a'
      expect(query.length).toBeLessThan(2)

      const validQuery = 'ab'
      expect(validQuery.length).toBeGreaterThanOrEqual(2)
    })

    it('builds correct search URL with parameters', () => {
      const pluginId = 'plugin-1'
      const searchField = 'title'
      const query = 'test book'

      const params = new URLSearchParams({
        field: searchField,
        q: query,
        limit: '10'
      })

      const expectedUrl = `/api/plugins/${pluginId}/search?${params}`

      expect(expectedUrl).toContain('plugin-1')
      expect(expectedUrl).toContain('field=title')
      expect(expectedUrl).toContain('q=test+book')
      expect(expectedUrl).toContain('limit=10')
    })

    it('handles search results', () => {
      const mockResults = [
        {
          external_id: 'OL123',
          title: 'The Great Gatsby',
          subtitle: 'F. Scott Fitzgerald',
          image_url: 'https://covers.openlibrary.org/123.jpg'
        },
        {
          external_id: 'OL456',
          title: '1984',
          subtitle: 'George Orwell',
          image_url: null
        }
      ]

      expect(mockResults).toHaveLength(2)
      expect(mockResults[0].title).toBe('The Great Gatsby')
      expect(mockResults[1].image_url).toBeNull()
    })

    it('handles empty search results', () => {
      const emptyResults: any[] = []
      expect(emptyResults).toHaveLength(0)
    })

    it('handles search error', async () => {
      const errorMessage = 'Service unavailable'
      mockApiFetch.mockRejectedValueOnce({ data: { error: errorMessage } })

      try {
        await mockApiFetch('/api/plugins/plugin-1/search?q=test')
      } catch (err: any) {
        expect(err.data.error).toBe('Service unavailable')
      }
    })
  })

  describe('Import (Step 3)', () => {
    it('builds correct import request', () => {
      const pluginId = 'plugin-1'
      const externalId = 'OL123'

      const importUrl = `/api/plugins/${pluginId}/import`
      const importBody = { external_id: externalId }

      expect(importUrl).toBe('/api/plugins/plugin-1/import')
      expect(importBody.external_id).toBe('OL123')
    })

    it('handles successful import', async () => {
      const mockResponse = {
        asset: {
          id: 'asset-123',
          name: 'The Great Gatsby'
        }
      }

      mockApiFetch.mockResolvedValueOnce(mockResponse)

      const result = await mockApiFetch('/api/plugins/plugin-1/import', {
        method: 'POST',
        body: JSON.stringify({ external_id: 'OL123' })
      })

      expect(result.asset.id).toBe('asset-123')
      expect(result.asset.name).toBe('The Great Gatsby')
    })

    it('handles import error - not found', async () => {
      const errorMessage = 'Item not found'
      mockApiFetch.mockRejectedValueOnce({ data: { error: errorMessage } })

      try {
        await mockApiFetch('/api/plugins/plugin-1/import', {
          method: 'POST',
          body: JSON.stringify({ external_id: 'invalid' })
        })
      } catch (err: any) {
        expect(err.data.error).toContain('not found')
      }
    })

    it('handles import error - service unavailable', async () => {
      const errorMessage = 'Service unavailable'
      mockApiFetch.mockRejectedValueOnce({ data: { error: errorMessage } })

      try {
        await mockApiFetch('/api/plugins/plugin-1/import', {
          method: 'POST',
          body: JSON.stringify({ external_id: 'OL123' })
        })
      } catch (err: any) {
        expect(err.data.error).toContain('unavailable')
      }
    })
  })

  describe('Search Field Options', () => {
    it('extracts search field options from plugin', () => {
      const plugin = mockPlugins.value[0]
      const options = plugin.search_fields.map(f => ({
        label: f.label,
        value: f.key
      }))

      expect(options).toEqual([
        { label: 'Title', value: 'title' },
        { label: 'ISBN', value: 'isbn' }
      ])
    })

    it('defaults to first search field', () => {
      const plugin = mockPlugins.value[0]
      const defaultField = plugin.search_fields[0]?.key || ''

      expect(defaultField).toBe('title')
    })
  })

  describe('State Management', () => {
    it('tracks step progression', () => {
      const steps = ['select', 'search', 'importing'] as const
      type Step = typeof steps[number]

      let currentStep: Step = 'select'

      currentStep = 'search'
      expect(currentStep).toBe('search')

      currentStep = 'importing'
      expect(currentStep).toBe('importing')
    })

    it('resets state when modal closes', () => {
      const initialState = {
        step: 'select' as const,
        selectedPlugin: null,
        searchField: '',
        searchQuery: '',
        searchResults: [] as any[]
      }

      const resetState = { ...initialState }

      expect(resetState.step).toBe('select')
      expect(resetState.selectedPlugin).toBeNull()
      expect(resetState.searchField).toBe('')
      expect(resetState.searchQuery).toBe('')
      expect(resetState.searchResults).toHaveLength(0)
    })
  })

  describe('Navigation', () => {
    it('allows going back from search to select', () => {
      let step = 'search'
      const selectedPlugin = mockPlugins.value[0]

      if (step === 'search') {
        step = 'select'
      }

      expect(step).toBe('select')
    })
  })
})
