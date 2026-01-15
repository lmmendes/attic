<script setup lang="ts">
import type { Category, Location, Condition, AssetsResponse, AssetFilters, Asset } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const router = useRouter()
const importModalOpen = ref(false)

function onImported(assetId: string) {
  // Navigate to the newly imported asset's edit page
  router.push(`/assets/${assetId}/edit`)
}

const filters = reactive<AssetFilters>({
  q: '',
  category_id: undefined,
  location_id: undefined,
  condition_id: undefined,
  limit: 24,
  offset: 0
})

const queryString = computed(() => {
  const params = new URLSearchParams()
  if (filters.q) params.set('q', filters.q)
  if (filters.category_id) params.set('category_id', filters.category_id)
  if (filters.location_id) params.set('location_id', filters.location_id)
  if (filters.condition_id) params.set('condition_id', filters.condition_id)
  params.set('limit', String(filters.limit))
  params.set('offset', String(filters.offset))
  return params.toString()
})

const { data: assetsResponse, status } = useApi<AssetsResponse>(
  () => `/api/assets?${queryString.value}`
)

const { data: categories } = useApi<Category[]>('/api/categories')
const { data: locations } = useApi<Location[]>('/api/locations')
const { data: conditions } = useApi<Condition[]>('/api/conditions')

const categoryOptions = computed(() =>
  categories.value?.map(c => ({ label: c.name, value: c.id })) || []
)

const locationOptions = computed(() =>
  locations.value?.map(l => ({ label: l.name, value: l.id })) || []
)

const conditionOptions = computed(() =>
  conditions.value?.map(c => ({ label: c.label, value: c.id })) || []
)

function clearFilters() {
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
function getLocationPath(asset: Asset): string[] {
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

// Selected assets for bulk actions
const selectedAssets = ref<string[]>([])
const allSelected = computed({
  get: () => assetsResponse.value?.assets?.length
    ? selectedAssets.value.length === assetsResponse.value.assets.length
    : false,
  set: (val: boolean) => {
    if (val && assetsResponse.value?.assets) {
      selectedAssets.value = assetsResponse.value.assets.map(a => a.id)
    } else {
      selectedAssets.value = []
    }
  }
})

function toggleAssetSelection(assetId: string) {
  const index = selectedAssets.value.indexOf(assetId)
  if (index === -1) {
    selectedAssets.value.push(assetId)
  } else {
    selectedAssets.value.splice(index, 1)
  }
}
</script>

<template>
  <div class="flex flex-col min-h-full">
    <!-- Page Header -->
    <header class="bg-white dark:bg-mist-900 border-b border-gray-100 dark:border-gray-800 px-0 py-6 sticky top-0 z-10">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div class="flex flex-col gap-1">
          <h2 class="text-mist-950 dark:text-white text-3xl font-extrabold tracking-tight">
            All Assets
          </h2>
          <p class="text-mist-500 dark:text-gray-400 text-sm font-medium">
            Manage and track {{ assetsResponse?.total || 0 }} items across your inventory
          </p>
        </div>
        <div class="flex items-center gap-3">
          <UButton
            variant="outline"
            class="h-11 px-5 font-bold"
            icon="i-lucide-puzzle"
            @click="importModalOpen = true"
          >
            Import via Plugin
          </UButton>
          <UButton
            to="/assets/new"
            class="h-11 px-6 font-bold shadow-lg shadow-attic-500/20"
            icon="i-lucide-plus"
          >
            Add Asset
          </UButton>
        </div>
      </div>
    </header>

    <!-- Filters Bar -->
    <div class="py-4 bg-white/50 dark:bg-mist-900/50 backdrop-blur-sm border-b border-gray-100 dark:border-gray-800">
      <div class="flex flex-wrap items-center gap-4">
        <div class="flex-1 min-w-[300px]">
          <UInput
            v-model="searchQuery"
            placeholder="Search assets by name, tag, or serial number..."
            icon="i-lucide-search"
            size="lg"
            class="w-full"
          />
        </div>
        <div class="flex items-center gap-3">
          <USelectMenu
            v-model="filters.category_id"
            :items="categoryOptions"
            placeholder="Category"
            class="w-40"
            value-key="value"
            icon="i-lucide-folder"
          />
          <USelectMenu
            v-model="filters.location_id"
            :items="locationOptions"
            placeholder="Location"
            class="w-40"
            value-key="value"
            icon="i-lucide-map-pin"
          />
          <div class="h-8 w-px bg-gray-200 dark:bg-gray-700 mx-1" />
          <UButton
            v-if="filters.q || filters.category_id || filters.location_id || filters.condition_id"
            variant="ghost"
            color="neutral"
            icon="i-lucide-x"
            @click="clearFilters"
          >
            Clear
          </UButton>
        </div>
      </div>
    </div>

    <!-- Assets Table -->
    <div class="py-6 flex-1">
      <div class="bg-white dark:bg-gray-800/50 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="bg-gray-50 dark:bg-gray-800/80 border-b border-gray-100 dark:border-gray-700">
                <th class="p-4 w-12">
                  <div class="flex items-center justify-center">
                    <input
                      v-model="allSelected"
                      type="checkbox"
                      class="rounded border-gray-300 dark:border-gray-600 text-attic-500 focus:ring-attic-500 h-4 w-4"
                    >
                  </div>
                </th>
                <th class="p-4 text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-20">
                  Thumbnail
                </th>
                <th class="p-4 text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Asset Name
                </th>
                <th class="p-4 text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Category
                </th>
                <th class="p-4 text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Location
                </th>
                <th class="p-4 text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  ID
                </th>
                <th class="p-4 text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-20 text-right">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
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
                    No assets found
                  </p>
                  <UButton
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
                class="hover:bg-gray-50/80 dark:hover:bg-gray-800/40 transition-colors group cursor-pointer"
                @click="$router.push(`/assets/${asset.id}`)"
              >
                <td
                  class="p-4"
                  @click.stop
                >
                  <div class="flex items-center justify-center">
                    <input
                      type="checkbox"
                      :checked="selectedAssets.includes(asset.id)"
                      class="rounded border-gray-300 dark:border-gray-600 text-attic-500 focus:ring-attic-500 h-4 w-4"
                      @change="toggleAssetSelection(asset.id)"
                    >
                  </div>
                </td>
                <td class="p-4">
                  <div class="size-12 rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-700 border border-gray-100 dark:border-gray-600 flex items-center justify-center">
                    <UIcon
                      name="i-lucide-package"
                      class="w-6 h-6 text-gray-300 dark:text-gray-500"
                    />
                  </div>
                </td>
                <td class="p-4">
                  <p class="text-sm font-bold text-mist-950 dark:text-white group-hover:text-attic-500 transition-colors">
                    {{ asset.name }}
                  </p>
                </td>
                <td class="p-4">
                  <span
                    v-if="asset.category?.name"
                    class="inline-flex px-2 py-0.5 rounded text-[10px] font-bold text-attic-500 bg-attic-500/10 uppercase tracking-wider"
                  >
                    {{ asset.category.name }}
                  </span>
                  <span
                    v-else
                    class="text-xs text-gray-400"
                  >—</span>
                </td>
                <td class="p-4">
                  <div
                    v-if="asset.location?.name"
                    class="flex items-center gap-1.5 text-gray-500 dark:text-gray-400"
                  >
                    <span class="text-xs font-medium">{{ asset.location.name }}</span>
                  </div>
                  <span
                    v-else
                    class="text-xs text-gray-400"
                  >—</span>
                </td>
                <td class="p-4">
                  <span class="text-xs font-mono font-semibold text-gray-400">
                    {{ getShortId(asset) }}
                  </span>
                </td>
                <td
                  class="p-4 text-right"
                  @click.stop
                >
                  <UDropdownMenu
                    :items="[
                      [
                        { label: 'View', icon: 'i-lucide-eye', click: () => $router.push(`/assets/${asset.id}`) },
                        { label: 'Edit', icon: 'i-lucide-pencil', click: () => $router.push(`/assets/${asset.id}/edit`) }
                      ]
                    ]"
                  >
                    <UButton
                      variant="ghost"
                      color="neutral"
                      icon="i-lucide-more-horizontal"
                      size="sm"
                    />
                  </UDropdownMenu>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Footer / Pagination -->
    <footer class="py-6 border-t border-gray-100 dark:border-gray-800 bg-white/30 dark:bg-mist-900/30">
      <div class="flex items-center justify-between">
        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">
          Showing {{ ((page - 1) * (filters.limit ?? 24)) + 1 }} to {{ Math.min(page * (filters.limit ?? 24), assetsResponse?.total || 0) }} of {{ assetsResponse?.total || 0 }} assets
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
              class="px-1 text-gray-400"
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
      </div>
    </footer>

    <!-- Import Modal -->
    <ImportModal
      v-model:open="importModalOpen"
      @imported="onImported"
    />
  </div>
</template>
