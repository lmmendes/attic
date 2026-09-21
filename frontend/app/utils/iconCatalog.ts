export interface LucideIconCatalogEntry {
  name: string
  aliases: string[]
}

export const iconPickerPageSize = 60

export function filterIconCatalog(
  catalog: LucideIconCatalogEntry[],
  query: string
): LucideIconCatalogEntry[] {
  const tokens = normalizeIconText(query).split(' ').filter(Boolean)
  if (!tokens.length) return catalog

  return catalog.filter((icon) => {
    const searchText = normalizeIconText([
      icon.name.replace(/^i-lucide-/, ''),
      ...icon.aliases
    ].join(' '))
    return tokens.every(token => searchText.includes(token))
  })
}

function normalizeIconText(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, ' ')
}
