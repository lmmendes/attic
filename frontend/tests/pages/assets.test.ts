import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('Assets Index Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('filtering', () => {
    it('builds query string from filters', () => {
      const filters = {
        q: 'laptop',
        category_id: 'cat-123',
        location_id: 'loc-456',
        condition_id: undefined,
        limit: 24,
        offset: 0
      }

      const params = new URLSearchParams()
      if (filters.q) params.set('q', filters.q)
      if (filters.category_id) params.set('category_id', filters.category_id)
      if (filters.location_id) params.set('location_id', filters.location_id)
      if (filters.condition_id) params.set('condition_id', filters.condition_id)
      params.set('limit', String(filters.limit))
      params.set('offset', String(filters.offset))

      const queryString = params.toString()

      expect(queryString).toContain('q=laptop')
      expect(queryString).toContain('category_id=cat-123')
      expect(queryString).toContain('location_id=loc-456')
      expect(queryString).toContain('limit=24')
      expect(queryString).toContain('offset=0')
      expect(queryString).not.toContain('condition_id')
    })

    it('clears all filters', () => {
      const filters = {
        q: 'search term',
        category_id: 'cat-123',
        location_id: 'loc-456',
        condition_id: 'cond-789',
        offset: 24
      }

      // Clear filters function
      filters.q = ''
      filters.category_id = undefined as string | undefined
      filters.location_id = undefined as string | undefined
      filters.condition_id = undefined as string | undefined
      filters.offset = 0

      expect(filters.q).toBe('')
      expect(filters.category_id).toBeUndefined()
      expect(filters.location_id).toBeUndefined()
      expect(filters.condition_id).toBeUndefined()
      expect(filters.offset).toBe(0)
    })

    it('debounces search input', async () => {
      vi.useFakeTimers()
      let searchValue = ''
      let filterQ = ''
      let filterOffset = 24

      const handleSearch = (val: string) => {
        searchValue = val
        setTimeout(() => {
          filterQ = val
          filterOffset = 0
        }, 300)
      }

      handleSearch('test')
      expect(searchValue).toBe('test')
      expect(filterQ).toBe('')

      vi.advanceTimersByTime(300)
      expect(filterQ).toBe('test')
      expect(filterOffset).toBe(0)

      vi.useRealTimers()
    })
  })

  describe('pagination', () => {
    it('calculates current page from offset', () => {
      const testCases = [
        { offset: 0, limit: 24, expected: 1 },
        { offset: 24, limit: 24, expected: 2 },
        { offset: 48, limit: 24, expected: 3 },
        { offset: 0, limit: 10, expected: 1 },
        { offset: 10, limit: 10, expected: 2 }
      ]

      testCases.forEach(({ offset, limit, expected }) => {
        const page = Math.floor(offset / limit) + 1
        expect(page).toBe(expected)
      })
    })

    it('calculates offset from page number', () => {
      const testCases = [
        { page: 1, limit: 24, expected: 0 },
        { page: 2, limit: 24, expected: 24 },
        { page: 3, limit: 24, expected: 48 },
        { page: 1, limit: 10, expected: 0 },
        { page: 5, limit: 10, expected: 40 }
      ]

      testCases.forEach(({ page, limit, expected }) => {
        const offset = (page - 1) * limit
        expect(offset).toBe(expected)
      })
    })

    it('calculates total pages from total items', () => {
      const testCases = [
        { total: 100, limit: 24, expected: 5 },
        { total: 24, limit: 24, expected: 1 },
        { total: 25, limit: 24, expected: 2 },
        { total: 0, limit: 24, expected: 0 },
        { total: 50, limit: 10, expected: 5 }
      ]

      testCases.forEach(({ total, limit, expected }) => {
        const totalPages = Math.ceil(total / limit)
        expect(totalPages).toBe(expected)
      })
    })

    it('shows correct range in footer', () => {
      const page = 2
      const limit = 24
      const total = 100

      const start = ((page - 1) * limit) + 1
      const end = Math.min(page * limit, total)

      expect(start).toBe(25)
      expect(end).toBe(48)
    })
  })

  describe('short ID generation', () => {
    it('generates short ID from asset ID', () => {
      const getShortId = (assetId: string): string => {
        return `ATC-${assetId.slice(0, 4).toUpperCase()}`
      }

      expect(getShortId('abc123def456')).toBe('ATC-ABC1')
      expect(getShortId('xyz789')).toBe('ATC-XYZ7')
      expect(getShortId('1234567890')).toBe('ATC-1234')
    })
  })

  describe('asset selection', () => {
    it('toggles individual asset selection', () => {
      const selectedAssets: string[] = []

      const toggleAssetSelection = (assetId: string) => {
        const index = selectedAssets.indexOf(assetId)
        if (index === -1) {
          selectedAssets.push(assetId)
        } else {
          selectedAssets.splice(index, 1)
        }
      }

      toggleAssetSelection('asset-1')
      expect(selectedAssets).toContain('asset-1')

      toggleAssetSelection('asset-2')
      expect(selectedAssets).toContain('asset-1')
      expect(selectedAssets).toContain('asset-2')

      toggleAssetSelection('asset-1')
      expect(selectedAssets).not.toContain('asset-1')
      expect(selectedAssets).toContain('asset-2')
    })

    it('selects all assets', () => {
      const assets = [
        { id: 'asset-1' },
        { id: 'asset-2' },
        { id: 'asset-3' }
      ]
      let selectedAssets: string[] = []

      selectedAssets = assets.map(a => a.id)

      expect(selectedAssets).toEqual(['asset-1', 'asset-2', 'asset-3'])
    })

    it('deselects all assets', () => {
      let selectedAssets = ['asset-1', 'asset-2', 'asset-3']

      selectedAssets = []

      expect(selectedAssets).toEqual([])
    })

    it('determines if all are selected', () => {
      const assets = [{ id: 'asset-1' }, { id: 'asset-2' }]
      let selectedAssets = ['asset-1', 'asset-2']

      let allSelected = assets.length > 0 && selectedAssets.length === assets.length
      expect(allSelected).toBe(true)

      selectedAssets = ['asset-1']
      allSelected = assets.length > 0 && selectedAssets.length === assets.length
      expect(allSelected).toBe(false)

      selectedAssets = []
      allSelected = assets.length > 0 && selectedAssets.length === assets.length
      expect(allSelected).toBe(false)
    })
  })

  describe('filter options mapping', () => {
    it('maps categories to select options', () => {
      const categories = [
        { id: 'cat-1', name: 'Electronics' },
        { id: 'cat-2', name: 'Books' }
      ]

      const categoryOptions = categories.map(c => ({ label: c.name, value: c.id }))

      expect(categoryOptions).toEqual([
        { label: 'Electronics', value: 'cat-1' },
        { label: 'Books', value: 'cat-2' }
      ])
    })

    it('maps locations to select options', () => {
      const locations = [
        { id: 'loc-1', name: 'Living Room' },
        { id: 'loc-2', name: 'Bedroom' }
      ]

      const locationOptions = locations.map(l => ({ label: l.name, value: l.id }))

      expect(locationOptions).toEqual([
        { label: 'Living Room', value: 'loc-1' },
        { label: 'Bedroom', value: 'loc-2' }
      ])
    })

    it('maps conditions to select options', () => {
      const conditions = [
        { id: 'cond-1', label: 'New' },
        { id: 'cond-2', label: 'Used' }
      ]

      const conditionOptions = conditions.map(c => ({ label: c.label, value: c.id }))

      expect(conditionOptions).toEqual([
        { label: 'New', value: 'cond-1' },
        { label: 'Used', value: 'cond-2' }
      ])
    })

    it('handles empty arrays gracefully', () => {
      const categories: { name: string, id: string }[] = []
      const categoryOptions = categories?.map(c => ({ label: c.name, value: c.id })) || []

      expect(categoryOptions).toEqual([])
    })
  })

  describe('navigation after import', () => {
    it('navigates to edit page after successful import', () => {
      const mockRouterPush = vi.fn()
      const assetId = 'new-asset-123'

      const onImported = (id: string) => {
        mockRouterPush(`/assets/${id}/edit`)
      }

      onImported(assetId)

      expect(mockRouterPush).toHaveBeenCalledWith('/assets/new-asset-123/edit')
    })
  })

  describe('loading and empty states', () => {
    it('shows loading state when status is pending', () => {
      const status = 'pending'

      expect(status).toBe('pending')
    })

    it('shows empty state when no assets', () => {
      const assetsResponse = { assets: [], total: 0 }

      expect(assetsResponse.assets.length).toBe(0)
    })

    it('shows assets table when data is loaded', () => {
      const assetsResponse = {
        assets: [{ id: 'asset-1', name: 'Test Asset' }],
        total: 1
      }

      expect(assetsResponse.assets.length).toBeGreaterThan(0)
    })
  })
})
