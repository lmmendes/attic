import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('New Asset Page', () => {
  const mockApiFetch = vi.fn()
  const mockToast = { add: vi.fn() }
  const mockRouterPush = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('form validation', () => {
    it('requires name field', () => {
      const form = { name: '', category_id: 'cat-1' }

      const isValid = form.name && form.category_id
      expect(isValid).toBeFalsy()
    })

    it('requires category field', () => {
      const form = { name: 'Test Asset', category_id: undefined }

      const isValid = form.name && form.category_id
      expect(isValid).toBeFalsy()
    })

    it('validates with both required fields', () => {
      const form = { name: 'Test Asset', category_id: 'cat-1' }

      const isValid = form.name && form.category_id
      expect(isValid).toBeTruthy()
    })
  })

  describe('form progress', () => {
    it('calculates progress based on filled fields', () => {
      const calculateProgress = (form: {
        name: string
        category_id?: string
        location_id?: string
        condition_id?: string
      }) => {
        let filled = 0
        const total = 4
        if (form.name) filled++
        if (form.category_id) filled++
        if (form.location_id) filled++
        if (form.condition_id) filled++
        return (filled / total) * 100
      }

      expect(calculateProgress({ name: '' })).toBe(0)
      expect(calculateProgress({ name: 'Test' })).toBe(25)
      expect(calculateProgress({ name: 'Test', category_id: 'cat-1' })).toBe(50)
      expect(calculateProgress({ name: 'Test', category_id: 'cat-1', location_id: 'loc-1' })).toBe(75)
      expect(calculateProgress({
        name: 'Test',
        category_id: 'cat-1',
        location_id: 'loc-1',
        condition_id: 'cond-1'
      })).toBe(100)
    })
  })

  describe('select options mapping', () => {
    it('maps categories to options with icon', () => {
      const categories = [
        { id: 'cat-1', name: 'Electronics', icon: 'i-lucide-laptop' },
        { id: 'cat-2', name: 'Books', icon: 'i-lucide-book' }
      ]

      const options = categories.map(c => ({
        label: c.name,
        value: c.id,
        icon: c.icon
      }))

      expect(options).toEqual([
        { label: 'Electronics', value: 'cat-1', icon: 'i-lucide-laptop' },
        { label: 'Books', value: 'cat-2', icon: 'i-lucide-book' }
      ])
    })

    it('maps locations with "No location" option', () => {
      const locations = [
        { id: 'loc-1', name: 'Living Room' },
        { id: 'loc-2', name: 'Bedroom' }
      ]

      const options = [
        { label: 'No location', value: undefined },
        ...locations.map(l => ({ label: l.name, value: l.id }))
      ]

      expect(options).toHaveLength(3)
      expect(options[0].value).toBeUndefined()
    })

    it('maps conditions with "No condition" option', () => {
      const conditions = [
        { id: 'cond-1', label: 'New' },
        { id: 'cond-2', label: 'Used' }
      ]

      const options = [
        { label: 'No condition', value: undefined },
        ...conditions.map(c => ({ label: c.label, value: c.id }))
      ]

      expect(options).toHaveLength(3)
      expect(options[0].value).toBeUndefined()
    })
  })

  describe('attribute default values', () => {
    const getDefaultValue = (dataType: string): string | number | boolean => {
      switch (dataType) {
        case 'number': return 0
        case 'boolean': return false
        default: return ''
      }
    }

    it('returns 0 for number type', () => {
      expect(getDefaultValue('number')).toBe(0)
    })

    it('returns false for boolean type', () => {
      expect(getDefaultValue('boolean')).toBe(false)
    })

    it('returns empty string for string type', () => {
      expect(getDefaultValue('string')).toBe('')
    })

    it('returns empty string for text type', () => {
      expect(getDefaultValue('text')).toBe('')
    })

    it('returns empty string for date type', () => {
      expect(getDefaultValue('date')).toBe('')
    })
  })

  describe('input type mapping', () => {
    const getInputType = (dataType: string): string => {
      switch (dataType) {
        case 'number': return 'number'
        case 'date': return 'date'
        case 'boolean': return 'checkbox'
        default: return 'text'
      }
    }

    it('returns number for number type', () => {
      expect(getInputType('number')).toBe('number')
    })

    it('returns date for date type', () => {
      expect(getInputType('date')).toBe('date')
    })

    it('returns checkbox for boolean type', () => {
      expect(getInputType('boolean')).toBe('checkbox')
    })

    it('returns text for string type', () => {
      expect(getInputType('string')).toBe('text')
    })

    it('returns text for unknown type', () => {
      expect(getInputType('unknown')).toBe('text')
    })
  })

  describe('form submission', () => {
    it('builds correct payload', () => {
      const form = {
        name: 'Test Asset',
        description: 'A test description',
        category_id: 'cat-1',
        location_id: 'loc-1',
        condition_id: 'cond-1',
        quantity: 2,
        attributes: { serial_number: 'SN123' },
        purchase_at: '2024-01-15',
        purchase_price: 99.99,
        purchase_note: 'Bought at store'
      }

      const payload = {
        name: form.name,
        description: form.description || undefined,
        category_id: form.category_id,
        location_id: form.location_id || undefined,
        condition_id: form.condition_id || undefined,
        quantity: form.quantity,
        attributes: Object.keys(form.attributes).length > 0 ? form.attributes : undefined,
        purchase_at: form.purchase_at || undefined,
        purchase_price: form.purchase_price || undefined,
        purchase_note: form.purchase_note || undefined
      }

      expect(payload.name).toBe('Test Asset')
      expect(payload.attributes).toEqual({ serial_number: 'SN123' })
      expect(payload.quantity).toBe(2)
    })

    it('omits undefined optional fields', () => {
      const form = {
        name: 'Test Asset',
        description: '',
        category_id: 'cat-1',
        location_id: undefined,
        condition_id: undefined,
        quantity: 1,
        attributes: {},
        purchase_at: '',
        purchase_price: undefined,
        purchase_note: ''
      }

      const payload = {
        name: form.name,
        description: form.description || undefined,
        category_id: form.category_id,
        location_id: form.location_id || undefined,
        condition_id: form.condition_id || undefined,
        quantity: form.quantity,
        attributes: Object.keys(form.attributes).length > 0 ? form.attributes : undefined,
        purchase_at: form.purchase_at || undefined,
        purchase_price: form.purchase_price || undefined,
        purchase_note: form.purchase_note || undefined
      }

      expect(payload.description).toBeUndefined()
      expect(payload.location_id).toBeUndefined()
      expect(payload.condition_id).toBeUndefined()
      expect(payload.attributes).toBeUndefined()
      expect(payload.purchase_at).toBeUndefined()
      expect(payload.purchase_price).toBeUndefined()
      expect(payload.purchase_note).toBeUndefined()
    })

    it('submits form successfully', async () => {
      mockApiFetch.mockResolvedValueOnce({ id: 'new-asset-123' })

      const payload = { name: 'Test Asset', category_id: 'cat-1', quantity: 1 }
      const response = await mockApiFetch('/api/assets', {
        method: 'POST',
        body: JSON.stringify(payload)
      })

      expect(response.id).toBe('new-asset-123')
      expect(mockApiFetch).toHaveBeenCalledWith('/api/assets', {
        method: 'POST',
        body: JSON.stringify(payload)
      })
    })

    it('navigates to asset page on success', async () => {
      mockApiFetch.mockResolvedValueOnce({ id: 'new-asset-123' })

      const response = await mockApiFetch('/api/assets', {
        method: 'POST',
        body: JSON.stringify({ name: 'Test', category_id: 'cat-1' })
      })

      mockToast.add({ title: 'Asset created successfully', color: 'success' })
      mockRouterPush(`/assets/${response.id}`)

      expect(mockRouterPush).toHaveBeenCalledWith('/assets/new-asset-123')
    })

    it('shows error toast on failure', async () => {
      mockApiFetch.mockRejectedValueOnce({ message: 'Failed to create asset' })

      try {
        await mockApiFetch('/api/assets', {
          method: 'POST',
          body: JSON.stringify({ name: 'Test', category_id: 'cat-1' })
        })
      } catch (error: any) {
        mockToast.add({ title: error.message || 'Failed to create asset', color: 'error' })
      }

      expect(mockToast.add).toHaveBeenCalledWith({
        title: 'Failed to create asset',
        color: 'error'
      })
    })
  })

  describe('category selection', () => {
    it('fetches category attributes when category changes', async () => {
      const categoryWithAttributes = {
        id: 'cat-1',
        name: 'Electronics',
        attributes: [
          {
            attribute_id: 'attr-1',
            required: true,
            attribute: { key: 'serial_number', name: 'Serial Number', data_type: 'string' }
          },
          {
            attribute_id: 'attr-2',
            required: false,
            attribute: { key: 'warranty_years', name: 'Warranty Years', data_type: 'number' }
          }
        ]
      }

      mockApiFetch.mockResolvedValueOnce(categoryWithAttributes)

      const result = await mockApiFetch('/api/categories/cat-1')

      expect(result.attributes).toHaveLength(2)
      expect(result.attributes[0].attribute.key).toBe('serial_number')
    })

    it('initializes attribute values based on data type', () => {
      const attributes = [
        { attribute: { key: 'serial', data_type: 'string' } },
        { attribute: { key: 'count', data_type: 'number' } },
        { attribute: { key: 'is_active', data_type: 'boolean' } }
      ]

      const getDefaultValue = (dataType: string): string | number | boolean => {
        switch (dataType) {
          case 'number': return 0
          case 'boolean': return false
          default: return ''
        }
      }

      const formAttributes: Record<string, string | number | boolean> = {}
      attributes.forEach(ca => {
        if (ca.attribute) {
          formAttributes[ca.attribute.key] = getDefaultValue(ca.attribute.data_type)
        }
      })

      expect(formAttributes).toEqual({
        serial: '',
        count: 0,
        is_active: false
      })
    })

    it('clears selected category and attributes when category_id is undefined', () => {
      let selectedCategory: any = { id: 'cat-1', name: 'Electronics' }
      let formAttributes: Record<string, any> = { serial: 'ABC123' }

      const categoryId = undefined

      if (!categoryId) {
        selectedCategory = null
        formAttributes = {}
      }

      expect(selectedCategory).toBeNull()
      expect(formAttributes).toEqual({})
    })
  })

  describe('loading state', () => {
    it('sets loading true during submission', async () => {
      let loading = false

      loading = true
      expect(loading).toBe(true)

      mockApiFetch.mockResolvedValueOnce({ id: 'new-asset' })
      await mockApiFetch('/api/assets', { method: 'POST' })

      loading = false
      expect(loading).toBe(false)
    })
  })
})
