<script setup lang="ts">
import type { Category, Location, Condition, AssetsResponse, AssetFilters } from '~/types/api'

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
  limit: 20,
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

interface AssetRow {
  category?: { name: string }
  location?: { name: string }
  condition?: { label: string }
}

const columns = [
  { accessorKey: 'name', id: 'name', header: 'Name' },
  { accessorFn: (row: AssetRow) => row.category?.name, id: 'category', header: 'Category' },
  { accessorFn: (row: AssetRow) => row.location?.name, id: 'location', header: 'Location' },
  { accessorFn: (row: AssetRow) => row.condition?.label, id: 'condition', header: 'Condition' },
  { accessorKey: 'quantity', id: 'quantity', header: 'Qty' },
  { id: 'actions', header: '' }
]

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
  get: () => Math.floor((filters.offset ?? 0) / (filters.limit ?? 20)) + 1,
  set: (val: number) => {
    filters.offset = (val - 1) * (filters.limit ?? 20)
  }
})

const totalPages = computed(() =>
  Math.ceil((assetsResponse.value?.total || 0) / (filters.limit ?? 20))
)
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold">
          Assets
        </h1>
        <div class="flex gap-2">
          <UButton
            variant="outline"
            icon="i-lucide-download"
            @click="importModalOpen = true"
          >
            Import
          </UButton>
          <UButton
            to="/assets/new"
            icon="i-lucide-plus"
          >
            Add Asset
          </UButton>
        </div>
      </div>

      <!-- Filters -->
      <UCard class="mb-6">
        <div class="flex flex-wrap gap-4">
          <UInput
            v-model="filters.q"
            placeholder="Search assets..."
            icon="i-lucide-search"
            class="w-64"
          />
          <USelectMenu
            v-model="filters.category_id"
            :items="categoryOptions"
            placeholder="All Categories"
            class="w-48"
            value-key="value"
          />
          <USelectMenu
            v-model="filters.location_id"
            :items="locationOptions"
            placeholder="All Locations"
            class="w-48"
            value-key="value"
          />
          <USelectMenu
            v-model="filters.condition_id"
            :items="conditionOptions"
            placeholder="All Conditions"
            class="w-48"
            value-key="value"
          />
          <UButton
            variant="ghost"
            color="neutral"
            @click="clearFilters"
          >
            Clear
          </UButton>
        </div>
      </UCard>

      <!-- Assets Table -->
      <UCard>
        <UTable
          :data="assetsResponse?.assets || []"
          :columns="columns"
          :loading="status === 'pending'"
        >
          <template #name-cell="{ row }">
            <NuxtLink
              :to="`/assets/${row.original.id}`"
              class="text-primary hover:underline font-medium"
            >
              {{ row.original.name }}
            </NuxtLink>
          </template>
          <template #actions-cell="{ row }">
            <UButton
              :to="`/assets/${row.original.id}`"
              variant="ghost"
              icon="i-lucide-eye"
              size="sm"
            />
          </template>
        </UTable>

        <template #footer>
          <div class="flex items-center justify-between">
            <p class="text-sm text-muted">
              Showing {{ assetsResponse?.assets?.length || 0 }} of {{ assetsResponse?.total || 0 }} assets
            </p>
            <UPagination
              v-if="totalPages > 1"
              v-model:page="page"
              :total="assetsResponse?.total || 0"
              :items-per-page="filters.limit"
            />
          </div>
        </template>
      </UCard>

      <!-- Import Modal -->
      <ImportModal
        v-model:open="importModalOpen"
        @imported="onImported"
      />
    </div>
  </UContainer>
</template>
