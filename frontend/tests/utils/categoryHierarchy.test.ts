import { describe, expect, it } from 'vitest'
import type { Category } from '../../app/types/api'
import {
  buildCategoryOptions,
  buildCategoryTreeRows,
  getCategoryDescendantIds,
  getInheritedCategoryAttributes
} from '../../app/utils/categoryHierarchy'

function category(id: string, name: string, parentId?: string): Category {
  return {
    id,
    organization_id: 'org',
    name,
    parent_id: parentId,
    created_at: '',
    updated_at: '',
    attributes: []
  }
}

describe('categoryHierarchy', () => {
  it('orders and indents categories by ancestry', () => {
    const categories = [
      category('leaf', 'Ultrabooks', 'child'),
      category('root', 'Hardware'),
      category('child', 'Computers', 'root'),
      category('other', 'Books')
    ]

    expect(buildCategoryOptions(categories).map(option => option.label)).toEqual([
      'Books',
      'Hardware',
      '\u00A0\u00A0└ Computers',
      '\u00A0\u00A0\u00A0\u00A0└ Ultrabooks'
    ])
  })

  it('exposes ancestry and child counts for a tree view', () => {
    const rows = buildCategoryTreeRows([
      category('leaf', 'Ultrabooks', 'child'),
      category('root', 'Hardware'),
      category('child', 'Computers', 'root')
    ])

    expect(rows.map(row => ({
      name: row.category.name,
      depth: row.depth,
      ancestors: row.ancestors.map(ancestor => ancestor.name),
      children: row.childCount
    }))).toEqual([
      { name: 'Hardware', depth: 0, ancestors: [], children: 1 },
      { name: 'Computers', depth: 1, ancestors: ['Hardware'], children: 1 },
      { name: 'Ultrabooks', depth: 2, ancestors: ['Hardware', 'Computers'], children: 0 }
    ])
  })

  it('finds every descendant for safe parent selection', () => {
    const categories = [
      category('root', 'Hardware'),
      category('child', 'Computers', 'root'),
      category('leaf', 'Ultrabooks', 'child'),
      category('other', 'Books')
    ]

    expect([...getCategoryDescendantIds(categories, 'root')].sort()).toEqual(['child', 'leaf', 'root'])
  })

  it('inherits ancestor assignments and lets the nearest ancestor win', () => {
    const root = category('root', 'Hardware')
    root.attributes = [
      { id: 'ca-root', category_id: 'root', attribute_id: 'vendor', required: true, sort_order: 0, created_at: '' }
    ]
    const child = category('child', 'Computers', 'root')
    child.attributes = [
      { id: 'ca-child', category_id: 'child', attribute_id: 'vendor', required: false, sort_order: 1, created_at: '' },
      { id: 'ca-model', category_id: 'child', attribute_id: 'model', required: true, sort_order: 0, created_at: '' }
    ]

    const inherited = getInheritedCategoryAttributes([root, child], 'child')
    expect(inherited.map(assignment => assignment.attribute_id)).toEqual(['vendor', 'model'])
    expect(inherited[0]).toMatchObject({ category_id: 'child', required: false, inherited: true })
  })
})
