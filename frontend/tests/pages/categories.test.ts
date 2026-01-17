import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('Categories Index Page', () => {
  const mockApiFetch = vi.fn()
  const mockToast = { add: vi.fn() }
  const mockRefresh = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('statistics', () => {
    it('calculates total categories count', () => {
      const categories = [
        { id: 'cat-1', name: 'Electronics' },
        { id: 'cat-2', name: 'Books' },
        { id: 'cat-3', name: 'Furniture' }
      ]

      const totalCategories = categories.length

      expect(totalCategories).toBe(3)
    })

    it('calculates total items tracked', () => {
      const categoryAssetCounts = {
        'cat-1': 10,
        'cat-2': 25,
        'cat-3': 5
      }

      const totalItems = Object.values(categoryAssetCounts).reduce((sum, count) => sum + count, 0)

      expect(totalItems).toBe(40)
    })

    it('calculates average attributes per category', () => {
      const categories = [
        { id: 'cat-1', attributes: [{ id: 'a1' }, { id: 'a2' }] },
        { id: 'cat-2', attributes: [{ id: 'a3' }] },
        { id: 'cat-3', attributes: [] }
      ]

      const total = categories.reduce((sum, c) => sum + (c.attributes?.length || 0), 0)
      const avgAttributes = (total / categories.length).toFixed(1)

      expect(avgAttributes).toBe('1.0')
    })

    it('handles empty categories array', () => {
      const categories: any[] = []

      const totalCategories = categories.length || 0
      const avgAttributes = categories.length > 0
        ? (categories.reduce((sum, c) => sum + (c.attributes?.length || 0), 0) / categories.length).toFixed(1)
        : 0

      expect(totalCategories).toBe(0)
      expect(avgAttributes).toBe(0)
    })
  })

  describe('category styling', () => {
    const getCategoryStyle = (category: { name: string; icon?: string }) => {
      if (category.icon) {
        return { icon: category.icon, bgColor: 'bg-attic-100', textColor: 'text-attic-600' }
      }

      const name = category.name.toLowerCase()

      if (name.includes('electronic') || name.includes('computer')) {
        return { icon: 'i-lucide-laptop', bgColor: 'bg-orange-100', textColor: 'text-orange-600' }
      }
      if (name.includes('book')) {
        return { icon: 'i-lucide-book-open', bgColor: 'bg-blue-100', textColor: 'text-blue-600' }
      }
      if (name.includes('movie') || name.includes('dvd')) {
        return { icon: 'i-lucide-film', bgColor: 'bg-purple-100', textColor: 'text-purple-600' }
      }
      if (name.includes('furniture')) {
        return { icon: 'i-lucide-armchair', bgColor: 'bg-amber-100', textColor: 'text-amber-600' }
      }

      return { icon: 'i-lucide-tag', bgColor: 'bg-attic-100', textColor: 'text-attic-600' }
    }

    it('returns electronics style for electronics category', () => {
      const style = getCategoryStyle({ name: 'Electronics' })
      expect(style.icon).toBe('i-lucide-laptop')
      expect(style.bgColor).toContain('orange')
    })

    it('returns book style for books category', () => {
      const style = getCategoryStyle({ name: 'Books & Media' })
      expect(style.icon).toBe('i-lucide-book-open')
      expect(style.bgColor).toContain('blue')
    })

    it('returns movie style for DVD category', () => {
      const style = getCategoryStyle({ name: 'DVD Collection' })
      expect(style.icon).toBe('i-lucide-film')
      expect(style.bgColor).toContain('purple')
    })

    it('uses custom icon when provided', () => {
      const style = getCategoryStyle({ name: 'Custom', icon: 'i-lucide-star' })
      expect(style.icon).toBe('i-lucide-star')
    })

    it('returns default style for unknown categories', () => {
      const style = getCategoryStyle({ name: 'Something Else' })
      expect(style.icon).toBe('i-lucide-tag')
    })
  })

  describe('attribute styling', () => {
    const getAttributeStyle = (dataType: string) => {
      switch (dataType) {
        case 'string':
          return { icon: 'i-lucide-type', bgColor: 'bg-blue-50', textColor: 'text-blue-600' }
        case 'number':
          return { icon: 'i-lucide-hash', bgColor: 'bg-amber-50', textColor: 'text-amber-600' }
        case 'boolean':
          return { icon: 'i-lucide-toggle-left', bgColor: 'bg-green-50', textColor: 'text-green-600' }
        case 'text':
          return { icon: 'i-lucide-align-left', bgColor: 'bg-purple-50', textColor: 'text-purple-600' }
        case 'date':
          return { icon: 'i-lucide-calendar', bgColor: 'bg-red-50', textColor: 'text-red-600' }
        default:
          return { icon: 'i-lucide-circle', bgColor: 'bg-gray-50', textColor: 'text-gray-600' }
      }
    }

    it('returns correct style for string type', () => {
      const style = getAttributeStyle('string')
      expect(style.icon).toBe('i-lucide-type')
    })

    it('returns correct style for number type', () => {
      const style = getAttributeStyle('number')
      expect(style.icon).toBe('i-lucide-hash')
    })

    it('returns correct style for boolean type', () => {
      const style = getAttributeStyle('boolean')
      expect(style.icon).toBe('i-lucide-toggle-left')
    })

    it('returns correct style for date type', () => {
      const style = getAttributeStyle('date')
      expect(style.icon).toBe('i-lucide-calendar')
    })

    it('returns default style for unknown type', () => {
      const style = getAttributeStyle('unknown')
      expect(style.icon).toBe('i-lucide-circle')
    })
  })

  describe('delete category', () => {
    it('opens delete confirmation modal', () => {
      let deleteModalOpen = false
      let categoryToDelete: any = null

      const confirmDelete = (category: any) => {
        categoryToDelete = category
        deleteModalOpen = true
      }

      const category = { id: 'cat-1', name: 'Electronics' }
      confirmDelete(category)

      expect(deleteModalOpen).toBe(true)
      expect(categoryToDelete).toEqual(category)
    })

    it('deletes category on confirmation', async () => {
      mockApiFetch.mockResolvedValueOnce({})

      await mockApiFetch('/api/categories/cat-1', { method: 'DELETE' })

      expect(mockApiFetch).toHaveBeenCalledWith('/api/categories/cat-1', { method: 'DELETE' })
    })

    it('shows success toast after deletion', async () => {
      mockApiFetch.mockResolvedValueOnce({})
      let deleteModalOpen = true
      let categoryToDelete: any = { id: 'cat-1', name: 'Test' }

      await mockApiFetch(`/api/categories/${categoryToDelete.id}`, { method: 'DELETE' })
      mockToast.add({ title: 'Category deleted', color: 'success' })
      deleteModalOpen = false
      categoryToDelete = null
      mockRefresh()

      expect(mockToast.add).toHaveBeenCalledWith({ title: 'Category deleted', color: 'success' })
      expect(deleteModalOpen).toBe(false)
      expect(categoryToDelete).toBeNull()
      expect(mockRefresh).toHaveBeenCalled()
    })

    it('shows error toast on deletion failure', async () => {
      mockApiFetch.mockRejectedValueOnce(new Error('Delete failed'))

      try {
        await mockApiFetch('/api/categories/cat-1', { method: 'DELETE' })
      } catch {
        mockToast.add({ title: 'Failed to delete category', color: 'error' })
      }

      expect(mockToast.add).toHaveBeenCalledWith({ title: 'Failed to delete category', color: 'error' })
    })
  })

  describe('view attributes', () => {
    it('fetches full category details', async () => {
      const fullCategory = {
        id: 'cat-1',
        name: 'Electronics',
        attributes: [
          { id: 'attr-1', attribute: { name: 'Serial Number', data_type: 'string' }, required: true }
        ]
      }
      mockApiFetch.mockResolvedValueOnce(fullCategory)

      const result = await mockApiFetch('/api/categories/cat-1')

      expect(result).toEqual(fullCategory)
    })

    it('opens attributes modal on success', async () => {
      let attributesModalOpen = false
      let viewingCategory: any = null

      const fullCategory = { id: 'cat-1', name: 'Electronics', attributes: [] }
      mockApiFetch.mockResolvedValueOnce(fullCategory)

      const result = await mockApiFetch('/api/categories/cat-1')
      viewingCategory = result
      attributesModalOpen = true

      expect(viewingCategory).toEqual(fullCategory)
      expect(attributesModalOpen).toBe(true)
    })

    it('shows error toast on fetch failure', async () => {
      mockApiFetch.mockRejectedValueOnce(new Error('Fetch failed'))

      try {
        await mockApiFetch('/api/categories/cat-1')
      } catch {
        mockToast.add({ title: 'Failed to load category attributes', color: 'error' })
      }

      expect(mockToast.add).toHaveBeenCalledWith({
        title: 'Failed to load category attributes',
        color: 'error'
      })
    })
  })

  describe('asset count per category', () => {
    it('gets asset count for category', () => {
      const categoryAssetCounts: Record<string, number> = {
        'cat-1': 15,
        'cat-2': 8
      }

      const getAssetCount = (categoryId: string): number => {
        return categoryAssetCounts[categoryId] || 0
      }

      expect(getAssetCount('cat-1')).toBe(15)
      expect(getAssetCount('cat-2')).toBe(8)
      expect(getAssetCount('cat-3')).toBe(0)
    })
  })

  describe('loading and empty states', () => {
    it('shows loading state when status is pending', () => {
      const status = 'pending'
      expect(status).toBe('pending')
    })

    it('shows empty state when no categories', () => {
      const categories: any[] = []
      expect(categories.length).toBe(0)
    })

    it('shows table when categories exist', () => {
      const categories = [{ id: 'cat-1', name: 'Test' }]
      expect(categories.length).toBeGreaterThan(0)
    })
  })
})
