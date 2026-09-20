import type { Tag } from '~/types/api'

export function tagAssignment(names: string[], available: Tag[]) {
  const byName = new Map(available.map(tag => [tag.name.trim().toLocaleLowerCase(), tag]))
  const seen = new Set<string>()
  const tag_ids: string[] = []
  const new_tag_names: string[] = []
  for (const raw of names) {
    const name = raw.trim()
    const key = name.toLocaleLowerCase()
    if (!name || seen.has(key)) continue
    seen.add(key)
    const existing = byName.get(key)
    if (existing) tag_ids.push(existing.id)
    else new_tag_names.push(name)
  }
  return { tag_ids, new_tag_names }
}
