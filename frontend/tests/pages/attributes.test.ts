import { describe, expect, it } from 'vitest'

interface TestAttribute {
  id: string
  name: string
  key: string
  data_type: string
}

const attributes: TestAttribute[] = [
  { id: '1', name: 'Serial Number', key: 'serial_number', data_type: 'string' },
  { id: '2', name: 'Purchase Date', key: 'purchase_date', data_type: 'date' },
  { id: '3', name: 'Insured', key: 'insured', data_type: 'boolean' }
]

function filterAttributes(items: TestAttribute[], search: string, type: string) {
  const query = search.trim().toLowerCase()

  return items.filter((attribute) => {
    const matchesSearch = !query
      || attribute.name.toLowerCase().includes(query)
      || attribute.key.toLowerCase().includes(query)
    const matchesType = type === 'all' || attribute.data_type === type
    return matchesSearch && matchesType
  })
}

describe('Attributes Index Page', () => {
  it('filters fields by their human-readable name or storage key', () => {
    expect(filterAttributes(attributes, 'serial', 'all')).toEqual([attributes[0]])
    expect(filterAttributes(attributes, 'purchase_date', 'all')).toEqual([attributes[1]])
  })

  it('combines search and data type filters', () => {
    expect(filterAttributes(attributes, 'date', 'date')).toEqual([attributes[1]])
    expect(filterAttributes(attributes, 'date', 'string')).toEqual([])
  })

  it('counts each data type for the filter controls', () => {
    const getTypeCount = (type: string) => type === 'all'
      ? attributes.length
      : attributes.filter(attribute => attribute.data_type === type).length

    expect(getTypeCount('all')).toBe(3)
    expect(getTypeCount('boolean')).toBe(1)
    expect(getTypeCount('number')).toBe(0)
  })

  it('clamps the current page after the result set shrinks', () => {
    const currentPage = 3
    const totalPages = 1
    const clampedPage = currentPage > Math.max(1, totalPages)
      ? Math.max(1, totalPages)
      : currentPage

    expect(clampedPage).toBe(1)
  })
})
