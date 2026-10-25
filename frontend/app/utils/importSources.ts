import type { Plugin } from '~/types/api'

interface ImportProvider {
  id: string
  name: string
  url?: string
  linkLabel?: string
  credentials: string[]
  instruction?: string
  note?: string
}

const providers: Record<string, ImportProvider> = {
  google: {
    id: 'google', name: 'Google Books',
    url: 'https://console.cloud.google.com/apis/credentials', linkLabel: 'Google Cloud Console',
    credentials: ['ATTIC_GOOGLE_BOOKS_API_KEY'],
    instruction: 'Create an API key in a Google Cloud project with the Books API enabled.',
    note: 'Google Books also works without a key, with a lower shared quota.'
  },
  tmdb: {
    id: 'tmdb', name: 'TMDB',
    url: 'https://www.themoviedb.org/settings/api', linkLabel: 'TMDB API settings',
    credentials: ['ATTIC_TMDB_API_KEY'],
    instruction: 'Create a TMDB account and request an API key. Use the API key for v3 requests.',
    note: 'One key enables both Movies and TV Series.'
  },
  bgg: {
    id: 'bgg', name: 'BoardGameGeek',
    url: 'https://boardgamegeek.com/applications', linkLabel: 'BoardGameGeek applications',
    credentials: ['ATTIC_BGG_API_KEY'],
    instruction: 'Register an application. Once approved, create an application token in the applications dashboard.'
  },
  igdb: {
    id: 'igdb', name: 'IGDB',
    url: 'https://dev.twitch.tv/console/apps', linkLabel: 'Twitch Developer Console',
    credentials: ['ATTIC_IGDB_CLIENT_ID', 'ATTIC_IGDB_CLIENT_SECRET'],
    instruction: 'Register a Twitch application and copy its Client ID and Client Secret.',
    note: 'Keep the Client Secret private. Attic exchanges it for a short-lived access token held only in memory.'
  },
  rebrickable: {
    id: 'rebrickable', name: 'Rebrickable',
    url: 'https://rebrickable.com/api/', linkLabel: 'Rebrickable API',
    credentials: ['ATTIC_REBRICKABLE_API_KEY'],
    instruction: 'Create a free Rebrickable account and generate an API key.',
    note: 'One key enables both LEGO Sets and LEGO Minifigures.'
  }
}

const sourceProviders: Record<string, string> = {
  google_books: 'google',
  tmdb_movies: 'tmdb',
  tmdb_series: 'tmdb',
  bgg_boardgames: 'bgg',
  igdb_games: 'igdb',
  rebrickable_sets: 'rebrickable',
  rebrickable_minifigs: 'rebrickable'
}

export function importProvider(plugin: Plugin): ImportProvider {
  return providers[sourceProviders[plugin.id] || ''] || { id: plugin.id, name: plugin.name, credentials: [] }
}

export function importSourceIcon(plugin: Plugin): string {
  const icons: Record<string, string> = {
    google_books: 'i-lucide-book-open',
    tmdb_movies: 'i-lucide-film',
    tmdb_series: 'i-lucide-tv',
    bgg_boardgames: 'i-lucide-dices',
    igdb_games: 'i-lucide-gamepad-2',
    rebrickable_sets: 'i-lucide-brick-wall',
    rebrickable_minifigs: 'i-lucide-person-standing'
  }
  return icons[plugin.id] || 'i-lucide-database-zap'
}

export function compareImportSources(a: Plugin, b: Plugin): number {
  return a.category_name.localeCompare(b.category_name, 'en') || a.id.localeCompare(b.id, 'en')
}

export interface ImportSetupGroup {
  provider: ImportProvider
  sources: Plugin[]
  label: string
}

export function importSetupGroups(plugins: Plugin[]): ImportSetupGroup[] {
  const groups = new Map<string, ImportSetupGroup>()
  for (const source of plugins.filter(plugin => !plugin.enabled).sort(compareImportSources)) {
    const provider = importProvider(source)
    let group = groups.get(provider.id)
    if (!group) {
      group = { provider, sources: [], label: '' }
      groups.set(provider.id, group)
    }
    group.sources.push(source)
    group.label = group.sources.map(source => source.category_name).join(' & ')
  }
  return [...groups.values()].sort((a, b) => compareImportSources(a.sources[0]!, b.sources[0]!))
}
