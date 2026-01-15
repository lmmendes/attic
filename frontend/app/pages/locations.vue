<script setup lang="ts">
import type { Location, Asset } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: locations, refresh, status } = useApi<Location[]>('/api/locations')

// Selected location state
const selectedLocation = ref<Location | null>(null)
const searchQuery = ref('')

// Track expanded nodes in the tree
const expandedNodes = ref<Set<string>>(new Set())

// Fetch assets for selected location
const locationAssetsUrl = computed(() =>
  selectedLocation.value ? `/api/assets?location_id=${selectedLocation.value.id}&limit=20` : ''
)
const { data: locationAssets, refresh: refreshAssets } = useApi<{ assets: Asset[], total: number }>(
  () => locationAssetsUrl.value,
  { immediate: false, watch: false }
)

// Watch selected location to fetch assets
watch(selectedLocation, (loc) => {
  if (loc) {
    refreshAssets()
  }
})

// Modal state
const modalOpen = ref(false)
const editingLocation = ref<Location | null>(null)
const form = reactive({
  name: '',
  description: '',
  parent_id: undefined as string | undefined
})

// Delete confirmation modal
const deleteModalOpen = ref(false)
const locationToDelete = ref<Location | null>(null)

function openCreateModal(parentId?: string) {
  editingLocation.value = null
  form.name = ''
  form.description = ''
  form.parent_id = parentId
  modalOpen.value = true
}

function openEditModal(location: Location) {
  editingLocation.value = location
  form.name = location.name
  form.description = location.description || ''
  form.parent_id = location.parent_id
  modalOpen.value = true
}

async function saveLocation() {
  try {
    const url = editingLocation.value
      ? `/api/locations/${editingLocation.value.id}`
      : `/api/locations`

    await apiFetch(url, {
      method: editingLocation.value ? 'PUT' : 'POST',
      body: JSON.stringify(form)
    })

    toast.add({
      title: editingLocation.value ? 'Location updated' : 'Location created',
      color: 'success'
    })
    modalOpen.value = false
    refresh()
  } catch {
    toast.add({ title: 'Failed to save location', color: 'error' })
  }
}

function confirmDelete(location: Location) {
  locationToDelete.value = location
  deleteModalOpen.value = true
}

async function deleteLocation() {
  if (!locationToDelete.value) return

  try {
    await apiFetch(`/api/locations/${locationToDelete.value.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Location deleted', color: 'success' })
    deleteModalOpen.value = false

    // Clear selection if deleted location was selected
    if (selectedLocation.value?.id === locationToDelete.value.id) {
      selectedLocation.value = null
    }

    locationToDelete.value = null
    refresh()
  } catch {
    toast.add({ title: 'Failed to delete location', color: 'error' })
  }
}

// Parent options for the select (exclude the currently editing location and its children)
const parentOptions = computed<{ label: string; value: string | undefined }[]>(() => {
  const options: { label: string; value: string | undefined }[] = [
    { label: 'None (Top Level)', value: undefined }
  ]

  if (!locations.value) return options

  // Get IDs to exclude (current location and all its descendants)
  const excludeIds = new Set<string>()
  if (editingLocation.value) {
    excludeIds.add(editingLocation.value.id)
    // Add all descendants
    const addDescendants = (parentId: string) => {
      locations.value?.forEach(l => {
        if (l.parent_id === parentId) {
          excludeIds.add(l.id)
          addDescendants(l.id)
        }
      })
    }
    addDescendants(editingLocation.value.id)
  }

  locations.value.forEach(l => {
    if (!excludeIds.has(l.id)) {
      options.push({ label: l.name, value: l.id })
    }
  })

  return options
})

// Build a map of location id to location for quick lookup
const locationMap = computed(() => {
  const map = new Map<string, Location>()
  locations.value?.forEach(l => map.set(l.id, l))
  return map
})

// Build hierarchical tree structure
interface TreeNode {
  location: Location
  children: TreeNode[]
  level: number
}

const locationTree = computed<TreeNode[]>(() => {
  if (!locations.value) return []

  // Build a map of parent_id to children
  const childrenMap = new Map<string | undefined, Location[]>()
  locations.value.forEach(l => {
    const parentId = l.parent_id || undefined
    if (!childrenMap.has(parentId)) {
      childrenMap.set(parentId, [])
    }
    childrenMap.get(parentId)!.push(l)
  })

  // Recursively build tree
  const buildTree = (parentId: string | undefined, level: number): TreeNode[] => {
    const children = childrenMap.get(parentId) || []
    return children
      .sort((a, b) => a.name.localeCompare(b.name))
      .map(loc => ({
        location: loc,
        children: buildTree(loc.id, level + 1),
        level
      }))
  }

  return buildTree(undefined, 0)
})

// Filter tree based on search
const filteredTree = computed<TreeNode[]>(() => {
  if (!searchQuery.value.trim()) return locationTree.value

  const query = searchQuery.value.toLowerCase()

  const filterTree = (nodes: TreeNode[]): TreeNode[] => {
    return nodes.reduce<TreeNode[]>((acc, node) => {
      const matchesSearch = node.location.name.toLowerCase().includes(query)
      const filteredChildren = filterTree(node.children)

      if (matchesSearch || filteredChildren.length > 0) {
        acc.push({
          ...node,
          children: filteredChildren
        })
      }

      return acc
    }, [])
  }

  return filterTree(locationTree.value)
})

// Get children of a location
const getChildren = (locationId: string): Location[] => {
  return locations.value?.filter(l => l.parent_id === locationId) || []
}

// Get the full path/breadcrumb for a location
function getLocationPath(location: Location): Location[] {
  const path: Location[] = []
  let current: Location | undefined = location

  while (current) {
    path.unshift(current)
    if (current.parent_id) {
      current = locationMap.value.get(current.parent_id)
    } else {
      break
    }
  }

  return path
}

// Toggle expanded state
function toggleExpanded(locationId: string) {
  if (expandedNodes.value.has(locationId)) {
    expandedNodes.value.delete(locationId)
  } else {
    expandedNodes.value.add(locationId)
  }
}

// Select a location
function selectLocation(location: Location) {
  selectedLocation.value = location
  // Auto-expand parent nodes
  const path = getLocationPath(location)
  path.forEach(l => expandedNodes.value.add(l.id))
}

// Collapse all nodes
function collapseAll() {
  expandedNodes.value.clear()
}

// Expand all nodes
function expandAll() {
  locations.value?.forEach(l => expandedNodes.value.add(l.id))
}

// Check if a node has children
function hasChildren(locationId: string): boolean {
  return locations.value?.some(l => l.parent_id === locationId) || false
}

// Format currency
const formatCurrency = (value: number) => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0
  }).format(value)
}

// Calculate total value of assets in location
const totalValue = computed(() => {
  if (!locationAssets.value?.assets) return 0
  return locationAssets.value.assets.reduce((sum, asset) => sum + (asset.purchase_price || 0), 0)
})

// Get icon for location based on name
function getLocationIcon(location: Location): string {
  const name = location.name.toLowerCase()
  if (name.includes('bedroom')) return 'i-lucide-bed'
  if (name.includes('living')) return 'i-lucide-sofa'
  if (name.includes('kitchen')) return 'i-lucide-utensils'
  if (name.includes('bathroom')) return 'i-lucide-bath'
  if (name.includes('garage')) return 'i-lucide-car'
  if (name.includes('attic')) return 'i-lucide-archive'
  if (name.includes('basement')) return 'i-lucide-home'
  if (name.includes('office')) return 'i-lucide-briefcase'
  if (name.includes('closet')) return 'i-lucide-door-open'
  if (name.includes('storage')) return 'i-lucide-box'
  if (name.includes('home') || name.includes('house')) return 'i-lucide-home'
  return 'i-lucide-map-pin'
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          Locations
        </h1>
        <p class="text-mist-500">
          Organize where your belongings are stored with a hierarchical structure.
        </p>
      </div>
      <UButton
        icon="i-lucide-plus"
        class="h-11 px-6 font-bold shadow-lg shadow-attic-500/20"
        @click="openCreateModal()"
      >
        Add Location
      </UButton>
    </div>

    <!-- Two Panel Layout -->
    <div class="flex flex-col lg:flex-row gap-4 min-h-[calc(100vh-14rem)] lg:h-[calc(100vh-14rem)]">
      <!-- Left Panel: Location Tree -->
    <section class="flex flex-col w-full lg:w-[340px] xl:w-[380px] min-h-[400px] lg:min-h-0 bg-white dark:bg-mist-800 rounded-2xl shadow-soft border border-mist-100 dark:border-mist-700 overflow-hidden flex-shrink-0">
      <!-- Tree Header -->
      <div class="p-4 border-b border-mist-100 dark:border-mist-700 flex items-center justify-between bg-white dark:bg-mist-800 sticky top-0 z-10">
        <h2 class="text-lg font-bold text-mist-950 dark:text-white">
          Hierarchy
        </h2>
        <div class="flex gap-1">
          <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-chevrons-down-up"
            size="sm"
            title="Collapse All"
            @click="collapseAll"
          />
          <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-plus-square"
            size="sm"
            title="Add Root Location"
            @click="openCreateModal()"
          />
        </div>
      </div>

      <!-- Search Filter -->
      <div class="px-4 py-2 bg-white dark:bg-mist-800">
        <div class="relative">
          <UIcon
            name="i-lucide-search"
            class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-mist-400"
          />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Filter locations..."
            class="w-full bg-mist-50 dark:bg-mist-700 border-none rounded-lg py-2 pl-9 pr-4 text-sm focus:ring-1 focus:ring-attic-500 placeholder-mist-400 text-mist-950 dark:text-white"
          >
        </div>
      </div>

      <!-- Tree Content -->
      <div class="flex-1 overflow-y-auto p-2 custom-scrollbar">
        <!-- Loading State -->
        <div
          v-if="status === 'pending'"
          class="flex items-center justify-center py-12"
        >
          <UIcon
            name="i-lucide-loader-2"
            class="w-6 h-6 text-attic-500 animate-spin"
          />
        </div>

        <!-- Empty State -->
        <div
          v-else-if="!filteredTree.length"
          class="flex flex-col items-center justify-center py-12 px-4 text-center"
        >
          <div class="size-12 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-3">
            <UIcon
              name="i-lucide-map-pin"
              class="w-6 h-6 text-mist-400"
            />
          </div>
          <p class="text-sm text-mist-500 mb-3">
            {{ searchQuery ? 'No locations found' : 'No locations yet' }}
          </p>
          <UButton
            v-if="!searchQuery"
            size="sm"
            @click="openCreateModal()"
          >
            Add Location
          </UButton>
        </div>

        <!-- Tree Nodes -->
        <template v-else>
          <LocationTreeNode
            v-for="node in filteredTree"
            :key="node.location.id"
            :node="node"
            :selected-id="selectedLocation?.id"
            :expanded-nodes="expandedNodes"
            :get-icon="getLocationIcon"
            :has-children="hasChildren"
            @select="selectLocation"
            @toggle="toggleExpanded"
            @add-child="openCreateModal"
          />
        </template>
      </div>
    </section>

    <!-- Right Panel: Details & Assets -->
    <section class="flex-1 flex flex-col min-h-[500px] lg:min-h-0 bg-white dark:bg-mist-800 rounded-2xl shadow-soft border border-mist-100 dark:border-mist-700 overflow-hidden">
      <!-- No Selection State -->
      <div
        v-if="!selectedLocation"
        class="flex-1 flex flex-col items-center justify-center p-8 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-map-pin"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          Select a Location
        </h3>
        <p class="text-sm text-mist-500 max-w-sm">
          Choose a location from the hierarchy to view its details and assets.
        </p>
      </div>

      <!-- Location Details -->
      <template v-else>
        <!-- Breadcrumbs & Actions Header -->
        <div class="px-6 py-4 border-b border-mist-100 dark:border-mist-700 flex flex-wrap gap-4 items-center justify-between bg-white/80 dark:bg-mist-800/95 backdrop-blur-sm sticky top-0 z-20">
          <div class="flex items-center gap-2 text-sm text-mist-500 overflow-hidden">
            <template
              v-for="(loc, index) in getLocationPath(selectedLocation)"
              :key="loc.id"
            >
              <button
                v-if="index < getLocationPath(selectedLocation).length - 1"
                class="hover:text-attic-500 cursor-pointer transition-colors whitespace-nowrap"
                @click="selectLocation(loc)"
              >
                {{ loc.name }}
              </button>
              <span
                v-else
                class="font-bold text-mist-950 dark:text-white whitespace-nowrap"
              >
                {{ loc.name }}
              </span>
              <UIcon
                v-if="index < getLocationPath(selectedLocation).length - 1"
                name="i-lucide-chevron-right"
                class="w-4 h-4 flex-shrink-0"
              />
            </template>
          </div>
          <div class="flex gap-3">
            <UButton
              variant="outline"
              color="neutral"
              icon="i-lucide-edit"
              @click="openEditModal(selectedLocation)"
            >
              <span class="hidden sm:inline">Edit Details</span>
            </UButton>
            <UButton
              icon="i-lucide-plus"
              :to="`/assets/new?location_id=${selectedLocation.id}`"
            >
              Add Asset
            </UButton>
          </div>
        </div>

        <!-- Main Detail Content -->
        <div class="flex-1 overflow-y-auto custom-scrollbar">
          <div class="p-6 md:p-8 max-w-6xl mx-auto space-y-8">
            <!-- Location Info Card -->
            <div class="flex flex-col md:flex-row gap-8 items-start">
              <div class="flex-1 space-y-4">
                <div>
                  <h1 class="text-3xl font-bold text-mist-950 dark:text-white tracking-tight flex items-center gap-3">
                    <span class="p-2 bg-attic-500/10 rounded-xl text-attic-500">
                      <UIcon
                        :name="getLocationIcon(selectedLocation)"
                        class="w-8 h-8"
                      />
                    </span>
                    {{ selectedLocation.name }}
                  </h1>
                </div>
                <p
                  v-if="selectedLocation.description"
                  class="text-mist-500 leading-relaxed max-w-2xl"
                >
                  {{ selectedLocation.description }}
                </p>
                <p
                  v-else
                  class="text-mist-400 italic"
                >
                  No description provided
                </p>

                <!-- Stats Row -->
                <div class="flex gap-6 pt-2">
                  <div class="flex items-baseline gap-2">
                    <span class="text-2xl font-bold text-mist-950 dark:text-white">{{ locationAssets?.total || 0 }}</span>
                    <span class="text-sm font-medium text-mist-500 uppercase tracking-wider">Assets</span>
                  </div>
                  <div class="w-px h-8 bg-mist-200 dark:bg-mist-600" />
                  <div class="flex items-baseline gap-2">
                    <span class="text-2xl font-bold text-mist-950 dark:text-white">{{ formatCurrency(totalValue) }}</span>
                    <span class="text-sm font-medium text-mist-500 uppercase tracking-wider">Total Value</span>
                  </div>
                </div>
              </div>

              <!-- Sub-locations List -->
              <div
                v-if="getChildren(selectedLocation.id).length > 0"
                class="w-full md:w-72 bg-mist-50 dark:bg-mist-700/50 rounded-xl p-4 border border-mist-100 dark:border-mist-600 self-stretch flex flex-col"
              >
                <div class="flex items-center justify-between mb-3">
                  <h3 class="text-xs font-bold uppercase text-mist-500 tracking-wider">
                    Sub-locations
                  </h3>
                  <button
                    class="text-attic-500 hover:text-attic-600 text-xs font-bold flex items-center gap-1 transition-colors"
                    @click="openCreateModal(selectedLocation.id)"
                  >
                    <UIcon
                      name="i-lucide-plus-circle"
                      class="w-3.5 h-3.5"
                    />
                    ADD
                  </button>
                </div>
                <div class="space-y-2 flex-1 overflow-y-auto max-h-[160px] custom-scrollbar">
                  <button
                    v-for="child in getChildren(selectedLocation.id)"
                    :key="child.id"
                    class="w-full flex items-center gap-3 p-2 rounded-lg bg-white dark:bg-mist-800 shadow-sm border border-mist-100 dark:border-mist-600 hover:border-attic-500/50 transition-colors group text-left"
                    @click="selectLocation(child)"
                  >
                    <UIcon
                      :name="getLocationIcon(child)"
                      class="w-4.5 h-4.5 text-mist-400 group-hover:text-attic-500"
                    />
                    <span class="text-sm font-medium text-mist-950 dark:text-white flex-1 truncate">
                      {{ child.name }}
                    </span>
                  </button>
                </div>
              </div>

              <!-- No Sub-locations: Add button -->
              <div
                v-else
                class="w-full md:w-72 bg-mist-50 dark:bg-mist-700/50 rounded-xl p-4 border border-dashed border-mist-200 dark:border-mist-600 flex flex-col items-center justify-center text-center min-h-[120px]"
              >
                <UIcon
                  name="i-lucide-folder-plus"
                  class="w-6 h-6 text-mist-400 mb-2"
                />
                <p class="text-xs text-mist-500 mb-2">
                  No sub-locations yet
                </p>
                <UButton
                  size="xs"
                  variant="soft"
                  @click="openCreateModal(selectedLocation.id)"
                >
                  Add Sub-location
                </UButton>
              </div>
            </div>

            <div class="border-t border-mist-100 dark:border-mist-700" />

            <!-- Asset Grid Section -->
            <div>
              <div class="flex items-center justify-between mb-6">
                <h2 class="text-lg font-bold text-mist-950 dark:text-white">
                  Assets in this location
                </h2>
                <NuxtLink
                  v-if="(locationAssets?.total || 0) > 0"
                  :to="`/assets?location_id=${selectedLocation.id}`"
                  class="text-sm font-medium text-attic-500 hover:text-attic-600 hover:underline"
                >
                  View All
                </NuxtLink>
              </div>

              <!-- Assets Grid -->
              <div
                v-if="locationAssets?.assets?.length"
                class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4"
              >
                <!-- Asset Cards -->
                <NuxtLink
                  v-for="asset in locationAssets.assets.slice(0, 7)"
                  :key="asset.id"
                  :to="`/assets/${asset.id}`"
                  class="group bg-white dark:bg-mist-700 rounded-xl border border-mist-100 dark:border-mist-600 shadow-sm hover:shadow-md hover:border-attic-500/30 transition-all cursor-pointer overflow-hidden flex flex-col"
                >
                  <!-- Asset Image Placeholder -->
                  <div class="relative h-32 overflow-hidden bg-mist-100 dark:bg-mist-800 flex items-center justify-center">
                    <UIcon
                      name="i-lucide-package"
                      class="w-10 h-10 text-mist-300 dark:text-mist-500 group-hover:scale-110 transition-transform"
                    />
                    <div
                      v-if="asset.purchase_price"
                      class="absolute top-2 right-2 bg-white/90 dark:bg-black/80 backdrop-blur px-2 py-0.5 rounded-full text-[10px] font-bold text-mist-950 dark:text-white shadow-sm"
                    >
                      {{ formatCurrency(asset.purchase_price) }}
                    </div>
                  </div>
                  <div class="p-4 flex flex-col flex-1">
                    <div class="flex justify-between items-start mb-1">
                      <h3 class="font-bold text-mist-950 dark:text-white text-sm line-clamp-1 group-hover:text-attic-500 transition-colors">
                        {{ asset.name }}
                      </h3>
                      <span
                        v-if="asset.condition"
                        class="shrink-0 text-[10px] bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 px-1.5 py-0.5 rounded font-medium ml-2"
                      >
                        {{ asset.condition.label }}
                      </span>
                    </div>
                    <p
                      v-if="asset.description"
                      class="text-xs text-mist-500 mb-3 line-clamp-2"
                    >
                      {{ asset.description }}
                    </p>
                    <div class="mt-auto flex items-center gap-2 pt-2 border-t border-mist-50 dark:border-mist-600">
                      <UIcon
                        name="i-lucide-tag"
                        class="w-4 h-4 text-mist-400"
                      />
                      <span class="text-[10px] uppercase font-bold text-mist-500">
                        {{ asset.category?.name || 'Uncategorized' }}
                      </span>
                    </div>
                  </div>
                </NuxtLink>

                <!-- Add Asset Card -->
                <NuxtLink
                  :to="`/assets/new?location_id=${selectedLocation.id}`"
                  class="group bg-mist-50 dark:bg-mist-700/50 rounded-xl border-2 border-dashed border-mist-200 dark:border-mist-600 hover:border-attic-500 hover:bg-attic-500/5 transition-all cursor-pointer flex flex-col items-center justify-center min-h-[220px] text-center p-4"
                >
                  <div class="size-12 rounded-full bg-white dark:bg-mist-700 shadow-sm flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                    <UIcon
                      name="i-lucide-plus"
                      class="w-6 h-6 text-attic-500"
                    />
                  </div>
                  <h3 class="font-bold text-mist-950 dark:text-white text-sm">
                    Add Item Here
                  </h3>
                  <p class="text-xs text-mist-500 mt-1">
                    Place a new asset in {{ selectedLocation.name }}
                  </p>
                </NuxtLink>
              </div>

              <!-- Empty Assets State -->
              <div
                v-else
                class="bg-mist-50 dark:bg-mist-700/50 rounded-xl border border-dashed border-mist-200 dark:border-mist-600 p-8 text-center"
              >
                <div class="size-12 rounded-full bg-white dark:bg-mist-700 shadow-sm flex items-center justify-center mx-auto mb-3">
                  <UIcon
                    name="i-lucide-package"
                    class="w-6 h-6 text-mist-400"
                  />
                </div>
                <h3 class="font-bold text-mist-950 dark:text-white text-sm mb-1">
                  No assets in this location
                </h3>
                <p class="text-xs text-mist-500 mb-4">
                  Start by adding your first asset to {{ selectedLocation.name }}
                </p>
                <UButton
                  size="sm"
                  :to="`/assets/new?location_id=${selectedLocation.id}`"
                >
                  Add Asset
                </UButton>
              </div>
            </div>

            <!-- Danger Zone -->
            <div class="border-t border-mist-100 dark:border-mist-700 pt-8">
              <div class="bg-red-50 dark:bg-red-900/10 rounded-xl border border-red-200 dark:border-red-800/50 p-4">
                <div class="flex items-start gap-3">
                  <div class="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
                    <UIcon
                      name="i-lucide-trash-2"
                      class="w-5 h-5 text-red-600 dark:text-red-400"
                    />
                  </div>
                  <div class="flex-1">
                    <h4 class="text-sm font-bold text-red-800 dark:text-red-300">
                      Delete Location
                    </h4>
                    <p class="text-xs text-red-600 dark:text-red-400 mt-1">
                      Permanently remove this location. This action cannot be undone.
                    </p>
                  </div>
                  <UButton
                    variant="soft"
                    color="error"
                    size="sm"
                    @click="confirmDelete(selectedLocation)"
                  >
                    Delete
                  </UButton>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </section>
    </div>

    <!-- Create/Edit Modal -->
    <UModal v-model:open="modalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl">
          <div class="p-6 border-b border-mist-100 dark:border-mist-700">
            <h3 class="text-lg font-bold text-mist-950 dark:text-white">
              {{ editingLocation ? 'Edit Location' : 'New Location' }}
            </h3>
          </div>

          <form
            class="p-6 space-y-4"
            @submit.prevent="saveLocation"
          >
            <div>
              <label class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5">
                Name <span class="text-red-500">*</span>
              </label>
              <input
                v-model="form.name"
                type="text"
                required
                placeholder="Location name"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white placeholder-mist-400 focus:ring-2 focus:ring-attic-500 focus:border-transparent"
              >
            </div>

            <div>
              <label class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5">
                Description
              </label>
              <textarea
                v-model="form.description"
                rows="3"
                placeholder="Optional description"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white placeholder-mist-400 focus:ring-2 focus:ring-attic-500 focus:border-transparent resize-none"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5">
                Parent Location
              </label>
              <select
                :value="form.parent_id ?? ''"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white focus:ring-2 focus:ring-attic-500 focus:border-transparent"
                @change="form.parent_id = ($event.target as HTMLSelectElement).value || undefined"
              >
                <option
                  v-for="opt in parentOptions"
                  :key="opt.value ?? 'none'"
                  :value="opt.value ?? ''"
                >
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </form>

          <div class="p-6 border-t border-mist-100 dark:border-mist-700 flex justify-end gap-3">
            <UButton
              variant="ghost"
              color="neutral"
              @click="modalOpen = false"
            >
              Cancel
            </UButton>
            <UButton @click="saveLocation">
              {{ editingLocation ? 'Update' : 'Create' }}
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Delete Confirmation Modal -->
    <UModal v-model:open="deleteModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl p-6">
          <div class="flex items-start gap-4">
            <div class="p-3 bg-red-100 dark:bg-red-900/30 rounded-full">
              <UIcon
                name="i-lucide-alert-triangle"
                class="w-6 h-6 text-red-600 dark:text-red-400"
              />
            </div>
            <div class="flex-1">
              <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                Delete Location
              </h3>
              <p class="text-sm text-mist-500 mt-2">
                Are you sure you want to delete <strong>{{ locationToDelete?.name }}</strong>? This action cannot be undone.
              </p>
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <UButton
              variant="ghost"
              color="neutral"
              @click="deleteModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              color="error"
              @click="deleteLocation"
            >
              Delete
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
