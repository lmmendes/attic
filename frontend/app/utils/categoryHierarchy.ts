import type { Category, CategoryAttribute } from '~/types/api'

export interface CategoryOption {
  label: string
  value: string
  icon?: string
}

export interface CategoryTreeRow {
  category: Category
  ancestors: Category[]
  depth: number
  rootId: string
  childCount: number
}

export function buildCategoryTreeRows(categories: Category[], excludedIds = new Set<string>()): CategoryTreeRow[] {
  const byId = new Map(categories.map(category => [category.id, category]))
  const children = new Map<string | undefined, Category[]>()
  for (const category of categories) {
    const parentId = category.parent_id && byId.has(category.parent_id)
      ? category.parent_id
      : undefined
    const siblings = children.get(parentId) || []
    siblings.push(category)
    children.set(parentId, siblings)
  }

  const rows: CategoryTreeRow[] = []
  const visited = new Set<string>()
  function visit(category: Category, ancestors: Category[]) {
    if (excludedIds.has(category.id) || visited.has(category.id)) return
    visited.add(category.id)
    rows.push({
      category,
      ancestors,
      depth: ancestors.length,
      rootId: ancestors[0]?.id || category.id,
      childCount: (children.get(category.id) || []).filter(child => !excludedIds.has(child.id)).length
    })
    for (const child of [...(children.get(category.id) || [])].sort((a, b) => a.name.localeCompare(b.name))) {
      visit(child, [...ancestors, category])
    }
  }

  for (const root of [...(children.get(undefined) || [])].sort((a, b) => a.name.localeCompare(b.name))) {
    visit(root, [])
  }
  // Cycles cannot normally be persisted, but keep malformed legacy data visible.
  for (const category of [...categories].sort((a, b) => a.name.localeCompare(b.name))) {
    visit(category, [])
  }
  return rows
}

export function getCategoryDescendantIds(categories: Category[], categoryId: string): Set<string> {
  const descendants = new Set<string>([categoryId])
  let changed = true
  while (changed) {
    changed = false
    for (const category of categories) {
      if (category.parent_id && descendants.has(category.parent_id) && !descendants.has(category.id)) {
        descendants.add(category.id)
        changed = true
      }
    }
  }
  return descendants
}

export function buildCategoryOptions(categories: Category[], excludedIds = new Set<string>()): CategoryOption[] {
  return buildCategoryTreeRows(categories, excludedIds).map(({ category, depth }) => {
    const indent = depth > 0 ? `${'\u00A0'.repeat(depth * 2)}└ ` : ''
    return { label: `${indent}${category.name}`, value: category.id, icon: category.icon }
  })
}

export function getInheritedCategoryAttributes(categories: Category[], parentId?: string): CategoryAttribute[] {
  if (!parentId) return []
  const byId = new Map(categories.map(category => [category.id, category]))
  const ancestry: Category[] = []
  const visited = new Set<string>()
  let current = byId.get(parentId)
  while (current && !visited.has(current.id)) {
    ancestry.unshift(current)
    visited.add(current.id)
    current = current.parent_id ? byId.get(current.parent_id) : undefined
  }

  const closestAssignment = new Map<string, CategoryAttribute>()
  for (const category of ancestry) {
    for (const assignment of category.attributes || []) {
      closestAssignment.set(assignment.attribute_id, { ...assignment, inherited: true })
    }
  }
  return [...closestAssignment.values()]
}
