<script setup lang="ts">
import type { Collection, Category, Location, Condition, AssetsResponse, AssetFilters, Asset } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const router = useRouter()
const route = useRoute()
const importModalOpen = ref(false)
const searchContainer = ref<HTMLElement | null>(null)

onMounted(async () => {
  if (route.query.focus !== 'search') return

  await nextTick()
  searchContainer.value?.querySelector<HTMLInputElement>('input')?.focus()
})

function onImported(assetId: string) {
  // Navigate to the newly imported asset's edit page
  router.push(`/assets/${assetId}/edit`)
}

const filters = reactive<AssetFilters>({
  collection_id: typeof route.query.collection_id === 'string' ? route.query.collection_id : undefined,
  q: '',
  category_id: undefined,
  location_id: typeof route.query.location_id === 'string' ? route.query.location_id : undefined,
  condition_id: undefined,
  limit: 24,
  offset: 0
})

const queryString = computed(() => {
  const params = new URLSearchParams()
  if (filters.collection_id) params.set('collection_id', filters.collection_id)
  if (filters.q) params.set('q', filters.q)
  if (filters.category_id) params.set('category_id', filters.category_id)
  if (filters.location_id) params.set('location_id', filters.location_id)
  if (filters.condition_id) params.set('condition_id', filters.condition_id)
  params.set('limit', String(filters.limit))
  params.set('offset', String(filters.offset))
  return params.toString()
})

const { data: assetsResponse, status, error, refresh } = useApi<AssetsResponse>(
  () => `/api/assets?${queryString.value}`
)

const { data: collections } = useApi<Collection[]>('/api/collections')
const collectionOptions = computed(() => collections.value?.map(c => ({ label: c.name, value: c.id, icon: c.icon })) || [])
watch(() => route.query.collection_id, (id) => {
  filters.collection_id = typeof id === 'string' ? id : undefined
  filters.offset = 0
})
watch(() => filters.collection_id, (id) => {
  filters.offset = 0
  if (id !== route.query.collection_id) router.replace({ query: { ...route.query, collection_id: id } })
})

const { data: categories } = useApi<Category[]>('/api/categories')
const { data: locations } = useApi<Location[]>('/api/locations')
const { data: conditions } = useApi<Condition[]>('/api/conditions')

const categoryOptions = computed(() =>
  categories.value?.map(c => ({ label: c.name, value: c.id })) || []
)

interface LocationTreeNode {
  location: Location
  children: LocationTreeNode[]
}

function buildLocationOptions(items: Location[]): { label: string, value: string }[] {
  const childrenMap = new Map<string | undefined, Location[]>()
  items.forEach((location) => {
    const parentId = location.parent_id || undefined
    if (!childrenMap.has(parentId)) {
      childrenMap.set(parentId, [])
    }
    childrenMap.get(parentId)!.push(location)
  })

  const buildTree = (parentId: string | undefined): LocationTreeNode[] => {
    const children = childrenMap.get(parentId) || []
    return children
      .sort((a, b) => a.name.localeCompare(b.name))
      .map(location => ({
        location,
        children: buildTree(location.id)
      }))
  }

  const flattenTree = (nodes: LocationTreeNode[], depth = 0): { label: string, value: string }[] => {
    return nodes.flatMap((node) => {
      const indent = depth > 0 ? `${'\u00A0'.repeat(depth * 2)}└ ` : ''
      return [
        { label: `${indent}${node.location.name}`, value: node.location.id },
        ...flattenTree(node.children, depth + 1)
      ]
    })
  }

  return flattenTree(buildTree(undefined))
}

const locationOptions = computed(() =>
  locations.value ? buildLocationOptions(locations.value) : []
)

const conditionOptions = computed(() =>
  conditions.value?.map(c => ({ label: c.label, value: c.id })) || []
)

const hasActiveFilters = computed(() => Boolean(
  filters.collection_id || filters.q || filters.category_id || filters.location_id || filters.condition_id
))

function clearFilters() {
  searchQuery.value = ''
  filters.collection_id = undefined
  filters.q = ''
  filters.category_id = undefined
  filters.location_id = undefined
  filters.condition_id = undefined
  filters.offset = 0
}

const page = computed({
  get: () => Math.floor((filters.offset ?? 0) / (filters.limit ?? 24)) + 1,
  set: (val: number) => {
    filters.offset = (val - 1) * (filters.limit ?? 24)
  }
})

const totalPages = computed(() =>
  Math.ceil((assetsResponse.value?.total || 0) / (filters.limit ?? 24))
)

// Generate short ID from asset ID
function getShortId(asset: Asset): string {
  return `ATC-${asset.id.slice(0, 4).toUpperCase()}`
}

// Get location breadcrumb
function _getLocationPath(asset: Asset): string[] {
  if (!asset.location?.name) return []
  return [asset.location.name]
}

// Debounced search
const searchQuery = ref('')
let searchTimeout: ReturnType<typeof setTimeout> | null = null
watch(searchQuery, (val: string) => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    filters.q = val
    filters.offset = 0
  }, 300)
})
</script>

<template>
  <div class="flex min-h-full flex-col gap-5 pb-6">
    <!-- Page Header -->
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Inventory
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          All Assets
        </h1>
        <p class="text-sm text-muted">
          Manage and track everything you keep and care for.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          variant="outline"
          color="neutral"
          class="rounded-xl font-bold"
          icon="i-lucide-puzzle"
          @click="importModalOpen = true"
        >
          Import
        </UButton>
        <UButton
          to="/assets/new"
          class="rounded-xl font-bold shadow-primary"
          icon="i-lucide-plus"
        >
          Add Asset
        </UButton>
      </div>
    </header>

    <!-- Filters Bar -->
    <section class="attic-panel rounded-[18px] p-3 sm:p-4">
      <div class="flex flex-col gap-3 2xl:flex-row 2xl:items-center 2xl:justify-between">
        <div
          ref="searchContainer"
          class="min-w-0 flex-1"
        >
          <UInput
            v-model="searchQuery"
            placeholder="Search by name, tag, or serial number..."
            icon="i-lucide-search"
            size="lg"
            class="w-full"
          />
        </div>
        <div class="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap sm:items-center 2xl:justify-end">
          <USelectMenu
            v-model="filters.collection_id"
            :items="collectionOptions"
            value-key="value"
            placeholder="Collection"
            aria-label="Filter by collection"
            class="min-w-0 sm:w-44"
          />
          <USelectMenu
            v-model="filters.category_id"
            :items="categoryOptions"
            placeholder="Category"
            class="min-w-0 sm:w-40"
            value-key="value"
            icon="i-lucide-folder"
          />
          <USelectMenu
            v-model="filters.location_id"
            :items="locationOptions"
            placeholder="Location"
            class="min-w-0 sm:w-40"
            value-key="value"
            icon="i-lucide-map-pin"
          />
          <USelectMenu
            v-model="filters.condition_id"
            :items="conditionOptions"
            placeholder="Condition"
            class="min-w-0 sm:w-36"
            value-key="value"
            icon="i-lucide-sparkles"
          />
          <UButton
            v-if="hasActiveFilters"
            variant="ghost"
            color="neutral"
            icon="i-lucide-x"
            @click="clearFilters"
          >
            Clear
          </UButton>
        </div>
      </div>
    </section>

    <!-- Assets Table -->
    <div class="flex-1">
      <div class="attic-panel overflow-hidden rounded-[20px]">
        <div class="flex items-center justify-between border-b border-mist-100 px-4 py-4 dark:border-mist-700 sm:px-5">
          <div>
            <h2 class="font-extrabold text-mist-950 dark:text-white">
              Asset inventory
            </h2>
            <p class="text-xs text-muted">
              Browse, filter, and manage every item in your collection.
            </p>
          </div>
          <span class="rounded-lg bg-mist-50 px-2.5 py-1 text-xs font-bold text-muted dark:bg-mist-700/60">
            {{ assetsResponse?.total || 0 }} shown
          </span>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="border-b border-mist-200/80 bg-mist-50/70 dark:border-mist-700 dark:bg-mist-800/80">
                <th class="w-16 p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted">
                  Item
                </th>
                <th class="p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted">
                  Asset
                </th>
                <th class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted md:table-cell">
                  Category
                </th>
                <th class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted md:table-cell">
                  Collections
                </th>
                <th class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted lg:table-cell">
                  Location
                </th>
                <th class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted xl:table-cell">
                  ID
                </th>
                <th class="w-16 p-3 text-right text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted">
                  <span class="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-mist-100 dark:divide-mist-800">
              <!-- Loading State -->
              <tr v-if="status === 'pending'">
                <td
                  colspan="7"
                  class="p-8 text-center"
                >
                  <div class="flex items-center justify-center gap-2 text-gray-400">
                    <UIcon
                      name="i-lucide-loader-2"
                      class="w-5 h-5 animate-spin"
                    />
                    <span>Loading assets...</span>
                  </div>
                </td>
              </tr>

              <!-- Error State -->
              <tr v-else-if="error">
                <td
                  colspan="7"
                  class="p-10 text-center"
                >
                  <div class="flex flex-col items-center">
                    <UIcon
                      name="i-lucide-circle-alert"
                      class="mb-3 size-10 text-red-400"
                    />
                    <p class="font-bold text-mist-950 dark:text-white">
                      Could not load assets
                    </p>
                    <p class="mt-1 text-sm text-muted">
                      Check your connection and try again.
                    </p>
                    <UButton
                      class="mt-4"
                      variant="soft"
                      icon="i-lucide-refresh-cw"
                      @click="refresh()"
                    >
                      Try again
                    </UButton>
                  </div>
                </td>
              </tr>

              <!-- Empty State -->
              <tr v-else-if="!assetsResponse?.assets?.length">
                <td
                  colspan="7"
                  class="p-12 text-center"
                >
                  <UIcon
                    name="i-lucide-inbox"
                    class="w-12 h-12 mx-auto mb-4 text-gray-300"
                  />
                  <p class="text-gray-500 mb-4">
                    {{ hasActiveFilters ? 'No assets match these filters' : 'No assets yet' }}
                  </p>
                  <UButton
                    v-if="hasActiveFilters"
                    variant="soft"
                    @click="clearFilters"
                  >
                    Clear filters
                  </UButton>
                  <UButton
                    v-else
                    to="/assets/new"
                    variant="soft"
                  >
                    Add your first asset
                  </UButton>
                </td>
              </tr>

              <!-- Asset Rows -->
              <tr
                v-for="asset in assetsResponse?.assets"
                v-else
                :key="asset.id"
                class="group transition-colors hover:bg-attic-50/55 dark:hover:bg-attic-500/5"
              >
                <td class="p-3">
                  <div class="flex size-11 items-center justify-center overflow-hidden rounded-xl border border-mist-100 bg-gradient-to-br from-attic-50 to-mist-100 dark:border-mist-700 dark:from-mist-700 dark:to-mist-800">
                    <img
                      v-if="asset.main_attachment_url"
                      :src="asset.main_attachment_url"
                      :alt="asset.name"
                      class="w-full h-full object-cover"
                    >
                    <UIcon
                      v-else
                      name="i-lucide-package"
                      class="w-6 h-6 text-gray-300 dark:text-gray-500"
                    />
                  </div>
                </td>
                <td class="max-w-[360px] p-3">
                  <NuxtLink
                    :to="`/assets/${asset.id}`"
                    class="text-sm font-bold text-mist-950 transition-colors hover:text-attic-500 focus-visible:text-attic-500 dark:text-white"
                  >
                    {{ asset.name }}
                  </NuxtLink>
                  <p class="mt-0.5 truncate text-xs text-muted">
                    {{ asset.description || `${asset.quantity} ${asset.quantity === 1 ? 'item' : 'items'} in inventory` }}
                  </p>
                  <div class="mt-1.5 flex items-center gap-1.5 md:hidden">
                    <span
                      v-if="asset.category?.name"
                      class="text-[10px] font-bold text-attic-500"
                    >{{ asset.category.name }}</span>
                    <span
                      v-if="asset.location?.name"
                      class="text-[10px] text-muted"
                    >· {{ asset.location.name }}</span>
                  </div>
                  <div
                    v-if="asset.collections?.length"
                    class="mt-2 flex flex-wrap gap-1.5 md:hidden"
                    aria-label="Collections"
                  >
                    <NuxtLink
                      v-for="collection in asset.collections"
                      :key="collection.id"
                      :to="{ path: '/assets', query: { ...route.query, collection_id: collection.id } }"
                      class="rounded-md bg-attic-50 px-2 py-1 text-xs font-semibold text-attic-600 hover:underline dark:bg-attic-500/10 dark:text-attic-300"
                    >{{ collection.name }}</NuxtLink>
                  </div>
                </td>
                <td class="hidden p-3 md:table-cell">
                  <span
                    v-if="asset.category?.name"
                    class="inline-flex rounded-full bg-attic-50 px-2.5 py-1 text-[10px] font-extrabold text-attic-600 ring-1 ring-attic-100 dark:bg-attic-500/10 dark:text-attic-300 dark:ring-attic-500/20"
                  >
                    {{ asset.category.name }}
                  </span>
                  <span
                    v-else
                    class="text-xs text-gray-400"
                  >—</span>
                </td>
                <td class="hidden max-w-64 p-3 md:table-cell">
                  <div
                    v-if="asset.collections?.length"
                    class="flex flex-wrap gap-1.5"
                  >
                    <NuxtLink
                      v-for="collection in asset.collections"
                      :key="collection.id"
                      :to="{ path: '/assets', query: { ...route.query, collection_id: collection.id } }"
                      class="inline-flex max-w-full items-center gap-1.5 rounded-md bg-attic-50 px-2 py-1 text-xs font-semibold text-attic-600 hover:underline dark:bg-attic-500/10 dark:text-attic-300"
                    >
                      <UIcon
                        :name="collection.icon"
                        class="size-3.5 shrink-0"
                      />
                      <span class="break-words">{{ collection.name }}</span>
                    </NuxtLink>
                  </div>
                  <span
                    v-else
                    class="text-xs text-muted"
                    aria-label="No collections"
                  >—</span>
                </td>
                <td class="hidden p-3 lg:table-cell">
                  <div
                    v-if="asset.location?.name"
                    class="flex items-center gap-1.5 text-muted"
                  >
                    <UIcon
                      name="i-lucide-map-pin"
                      class="size-3.5"
                    />
                    <span class="text-xs font-medium">{{ asset.location.name }}</span>
                  </div>
                  <span
                    v-else
                    class="text-xs text-gray-400"
                  >—</span>
                </td>
                <td class="hidden p-3 xl:table-cell">
                  <span class="text-xs font-mono font-semibold text-gray-400">
                    {{ getShortId(asset) }}
                  </span>
                </td>
                <td
                  class="p-3 text-right"
                  @click.stop
                >
                  <UDropdownMenu
                    :items="[
                      [
                        { label: 'View', icon: 'i-lucide-eye', onSelect: () => $router.push(`/assets/${asset.id}`) },
                        { label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => $router.push(`/assets/${asset.id}/edit`) }
                      ]
                    ]"
                  >
                    <UButton
                      variant="ghost"
                      color="neutral"
                      icon="i-lucide-more-horizontal"
                      size="sm"
                      :aria-label="`Actions for ${asset.name}`"
                      :title="`Actions for ${asset.name}`"
                    />
                  </UDropdownMenu>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Footer / Pagination -->
        <footer class="flex flex-col gap-3 border-t border-mist-100 bg-mist-50/50 px-4 py-3 dark:border-mist-700 dark:bg-mist-700/20 sm:flex-row sm:items-center sm:justify-between sm:px-5">
          <p class="text-xs font-medium text-muted">
            <template v-if="assetsResponse?.total">
              Showing {{ ((page - 1) * (filters.limit ?? 24)) + 1 }}–{{ Math.min(page * (filters.limit ?? 24), assetsResponse.total) }} of {{ assetsResponse.total }} assets
            </template>
            <template v-else>
              No assets to show
            </template>
          </p>
          <div
            v-if="totalPages > 1"
            class="flex items-center gap-2"
          >
            <UButton
              variant="outline"
              color="neutral"
              icon="i-lucide-chevron-left"
              size="sm"
              :disabled="page <= 1"
              @click="page--"
            />
            <template v-for="p in Math.min(totalPages, 5)">
              <UButton
                v-if="p <= 3 || p === totalPages || p === page"
                :key="p"
                :variant="p === page ? 'solid' : 'outline'"
                :color="p === page ? 'primary' : 'neutral'"
                size="sm"
                class="w-9"
                @click="page = p"
              >
                {{ p }}
              </UButton>
              <span
                v-else-if="p === 4 && totalPages > 5"
                :key="`ellipsis-${p}`"
                class="px-1 text-muted"
              >...</span>
            </template>
            <UButton
              variant="outline"
              color="neutral"
              icon="i-lucide-chevron-right"
              size="sm"
              :disabled="page >= totalPages"
              @click="page++"
            />
          </div>
        </footer>
      </div>
    </div>

    <!-- Import Modal -->
    <ImportModal
      v-model:open="importModalOpen"
      @imported="onImported"
    />
  </div>
</template>
