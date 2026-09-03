<script setup lang="ts">
import type { Location, Asset } from '~/types/api'
import { getIconLabel } from '~/utils/iconLabel'
import { getLocationNameError } from '~/utils/locationValidation'

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
  parent_id: undefined as string | undefined,
  icon: undefined as string | undefined
})
const nameError = ref('')

// Delete confirmation modal
const deleteModalOpen = ref(false)
const locationToDelete = ref<Location | null>(null)

interface TreeNode {
  location: Location
  children: TreeNode[]
  level: number
}

interface ParentOption {
  label: string
  value: string | undefined
}

const locationIcons = [
  'i-lucide-map-pin',
  'i-lucide-home',
  'i-lucide-archive',
  'i-lucide-box',
  'i-lucide-bed',
  'i-lucide-sofa',
  'i-lucide-utensils',
  'i-lucide-bath',
  'i-lucide-car',
  'i-lucide-briefcase',
  'i-lucide-door-open',
  'i-lucide-warehouse'
]

function openCreateModal(parentId?: string) {
  editingLocation.value = null
  form.name = ''
  form.description = ''
  form.parent_id = parentId
  form.icon = undefined
  nameError.value = ''
  modalOpen.value = true
}

function openEditModal(location: Location) {
  editingLocation.value = location
  form.name = location.name
  form.description = location.description || ''
  form.parent_id = location.parent_id
  form.icon = location.icon
  nameError.value = ''
  modalOpen.value = true
}

async function saveLocation() {
  nameError.value = getLocationNameError(form.name) || ''
  if (nameError.value) return

  try {
    const url = editingLocation.value
      ? `/api/locations/${editingLocation.value.id}`
      : `/api/locations`

    await apiFetch(url, {
      method: editingLocation.value ? 'PUT' : 'POST',
      body: JSON.stringify({ ...form, name: form.name.trim() })
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

// Build a map of location id to location for quick lookup
const locationMap = computed(() => {
  const map = new Map<string, Location>()
  locations.value?.forEach(l => map.set(l.id, l))
  return map
})

// Build hierarchical tree structure
function buildTree(nodes: Location[]): TreeNode[] {
  const childrenMap = new Map<string | undefined, Location[]>()
  nodes.forEach((l) => {
    const parentId = l.parent_id || undefined
    if (!childrenMap.has(parentId)) {
      childrenMap.set(parentId, [])
    }
    childrenMap.get(parentId)!.push(l)
  })

  const buildTreeRecursive = (parentId: string | undefined, level: number): TreeNode[] => {
    const children = childrenMap.get(parentId) || []
    return children
      .sort((a, b) => a.name.localeCompare(b.name))
      .map(loc => ({
        location: loc,
        children: buildTreeRecursive(loc.id, level + 1),
        level
      }))
  }

  return buildTreeRecursive(undefined, 0)
}

const locationTree = computed<TreeNode[]>(() => {
  if (!locations.value) return []
  return buildTree(locations.value)
})

function buildParentOptions(nodes: TreeNode[], excludeIds: Set<string> = new Set(), depth = 0): ParentOption[] {
  return nodes.flatMap((node) => {
    if (excludeIds.has(node.location.id)) {
      return []
    }

    const indent = depth > 0 ? `${'\u00A0'.repeat(depth * 2)}└ ` : ''
    return [
      {
        label: `${indent}${node.location.name}`,
        value: node.location.id
      },
      ...buildParentOptions(node.children, excludeIds, depth + 1)
    ]
  })
}

// Parent options for the select (exclude the currently editing location and its descendants)
const parentOptions = computed<ParentOption[]>(() => {
  const options: ParentOption[] = [
    { label: 'None (Top Level)', value: undefined }
  ]

  if (!locations.value) return options

  const excludeIds = new Set<string>()
  if (editingLocation.value) {
    excludeIds.add(editingLocation.value.id)
    const addDescendants = (parentId: string) => {
      locations.value?.forEach((l) => {
        if (l.parent_id === parentId) {
          excludeIds.add(l.id)
          addDescendants(l.id)
        }
      })
    }
    addDescendants(editingLocation.value.id)
  }

  return options.concat(buildParentOptions(locationTree.value, excludeIds))
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

watch(searchQuery, (query) => {
  if (query.trim()) expandAll()
})

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

// Get icon for location based on explicit icon or fallback by name
function getLocationIcon(location: Location): string {
  if (location.icon) return location.icon

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
  <div class="space-y-5 pb-6">
    <!-- Page Header -->
    <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Storage map
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Locations
        </h1>
        <p class="mt-1 text-sm text-mist-500">
          Browse {{ locations?.length || 0 }} spaces and see what is stored in each one.
        </p>
      </div>
      <UButton
        icon="i-lucide-plus"
        class="rounded-xl font-bold shadow-primary"
        @click="openCreateModal()"
      >
        Add Location
      </UButton>
    </div>

    <!-- Two Panel Layout -->
    <div class="grid min-h-[calc(100vh-11.5rem)] grid-cols-1 gap-4 lg:h-[calc(100vh-11.5rem)] lg:grid-cols-[300px_minmax(0,1fr)]">
      <!-- Left Panel: Location Tree -->
      <section class="attic-panel flex min-h-[360px] flex-col overflow-hidden rounded-[20px] lg:min-h-0">
        <!-- Tree Header -->
        <div class="flex items-center justify-between border-b border-mist-100 px-4 py-3 dark:border-mist-700">
          <div>
            <h2 class="text-sm font-extrabold text-mist-950 dark:text-white">
              Hierarchy
            </h2>
            <p class="text-[11px] text-mist-400">
              Select a space to inspect it
            </p>
          </div>
          <div class="flex gap-1">
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-chevrons-up-down"
              size="sm"
              title="Expand All"
              @click="expandAll"
            />
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
        <div class="border-b border-mist-100 px-3 py-3 dark:border-mist-700">
          <div class="relative">
            <UIcon
              name="i-lucide-search"
              class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-mist-400"
            />
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Filter locations..."
              class="w-full rounded-xl border border-mist-200 bg-mist-50 py-2 pl-9 pr-4 text-sm text-mist-950 placeholder-mist-400 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 dark:border-mist-700 dark:bg-mist-800 dark:text-white"
            >
          </div>
        </div>

        <!-- Tree Content -->
        <div class="custom-scrollbar flex-1 overflow-y-auto p-2.5">
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
      <section class="attic-panel flex min-h-[500px] min-w-0 flex-col overflow-hidden rounded-[20px] lg:min-h-0">
        <!-- No Selection State -->
        <div
          v-if="!selectedLocation"
          class="flex-1 flex flex-col items-center justify-center p-8 text-center"
        >
          <div class="mb-4 flex size-14 items-center justify-center rounded-2xl bg-attic-50 text-attic-500 dark:bg-attic-500/10">
            <UIcon
              name="i-lucide-map-pin"
              class="size-6"
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
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-mist-100 px-4 py-3 dark:border-mist-700 sm:px-5">
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
            <div class="flex gap-2">
              <UButton
                variant="outline"
                color="neutral"
                icon="i-lucide-edit"
                @click="openEditModal(selectedLocation)"
              >
                <span class="hidden sm:inline">Edit</span>
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
            <div class="mx-auto max-w-6xl space-y-6 p-4 sm:p-5 lg:p-6">
              <!-- Location Info Card -->
              <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
                <div class="relative isolate min-h-48 overflow-hidden rounded-[22px] bg-gradient-to-br from-attic-500 via-attic-600 to-[#174AE8] p-5 text-white shadow-primary sm:p-6">
                  <div class="pointer-events-none absolute -right-14 -top-20 size-52 rounded-full border-[30px] border-white/5" />
                  <div class="relative space-y-4">
                    <div>
                      <h1 class="flex items-center gap-3 text-2xl font-extrabold tracking-[-0.03em] text-white sm:text-3xl">
                        <span class="rounded-xl border border-white/15 bg-white/10 p-2.5 text-white">
                          <UIcon
                            :name="getLocationIcon(selectedLocation)"
                            class="size-6"
                          />
                        </span>
                        {{ selectedLocation.name }}
                      </h1>
                    </div>
                    <p
                      v-if="selectedLocation.description"
                      class="max-w-2xl text-sm leading-relaxed text-white/75"
                    >
                      {{ selectedLocation.description }}
                    </p>
                    <p
                      v-else
                      class="text-sm text-white/55"
                    >
                      No description provided
                    </p>

                    <!-- Stats Row -->
                    <div class="flex gap-5 pt-2">
                      <div class="flex items-baseline gap-2">
                        <span class="text-2xl font-black text-white">{{ locationAssets?.total || 0 }}</span>
                        <span class="text-[10px] font-bold uppercase tracking-wider text-white/60">Assets</span>
                      </div>
                      <div class="h-8 w-px bg-white/20" />
                      <div class="flex items-baseline gap-2">
                        <span class="text-2xl font-black text-white">{{ formatCurrency(totalValue) }}</span>
                        <span class="text-[10px] font-bold uppercase tracking-wider text-white/60">Total value</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Sub-locations List -->
                <div
                  v-if="getChildren(selectedLocation.id).length > 0"
                  class="flex min-h-48 w-full flex-col rounded-[20px] border border-mist-200 bg-mist-50 p-4 dark:border-mist-700 dark:bg-mist-800"
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
                  class="flex min-h-48 w-full flex-col items-center justify-center rounded-[20px] border border-dashed border-mist-200 bg-mist-50 p-4 text-center dark:border-mist-700 dark:bg-mist-800"
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

              <!-- Asset Grid Section -->
              <div>
                <div class="mb-4 flex items-center justify-between">
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
                  class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
                >
                  <!-- Asset Cards -->
                  <NuxtLink
                    v-for="asset in locationAssets.assets.slice(0, 7)"
                    :key="asset.id"
                    :to="`/assets/${asset.id}`"
                    class="attic-panel attic-panel-interactive group flex cursor-pointer flex-col overflow-hidden rounded-[18px]"
                  >
                    <!-- Asset Image Placeholder -->
                    <div class="relative flex aspect-[2/1] items-center justify-center overflow-hidden bg-gradient-to-br from-attic-50 to-mist-100 dark:from-mist-700 dark:to-mist-800">
                      <img
                        v-if="asset.main_attachment_url"
                        :src="asset.main_attachment_url"
                        :alt="asset.name"
                        class="size-full object-cover transition duration-300 group-hover:scale-[1.03]"
                      >
                      <UIcon
                        v-else
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
                    class="group flex min-h-48 cursor-pointer flex-col items-center justify-center rounded-[18px] border border-dashed border-mist-200 bg-mist-50 p-4 text-center transition-all hover:border-attic-500 hover:bg-attic-50 dark:border-mist-600 dark:bg-mist-700/50"
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
              <div class="border-t border-mist-100 pt-5 dark:border-mist-700">
                <div class="rounded-xl border border-red-100 bg-red-50/60 p-3 dark:border-red-900/40 dark:bg-red-900/10">
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
    <UModal
      v-model:open="modalOpen"
      :title="editingLocation ? 'Edit Location' : 'New Location'"
      description="Enter the location details and choose how it appears in the location tree."
    >
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
              <label
                for="location-name"
                class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5"
              >
                Name <span class="text-red-500">*</span>
              </label>
              <input
                id="location-name"
                v-model="form.name"
                type="text"
                required
                placeholder="Location name"
                :aria-invalid="!!nameError"
                :aria-describedby="nameError ? 'location-name-error' : undefined"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white placeholder-mist-400 focus:ring-2 focus:ring-attic-500 focus:border-transparent"
              >
              <p
                v-if="nameError"
                id="location-name-error"
                class="mt-1.5 text-sm text-red-600 dark:text-red-400"
              >
                {{ nameError }}
              </p>
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
                Icon
              </label>
              <div class="grid grid-cols-6 gap-2">
                <button
                  v-for="icon in locationIcons"
                  :key="icon"
                  type="button"
                  :aria-label="`${getIconLabel(icon)} icon`"
                  :title="`${getIconLabel(icon)} icon`"
                  :aria-pressed="form.icon === icon"
                  class="h-10 rounded-lg border transition-all flex items-center justify-center"
                  :class="form.icon === icon
                    ? 'bg-attic-500 text-white border-attic-500'
                    : 'bg-mist-50 dark:bg-mist-700 text-mist-500 dark:text-mist-300 border-mist-200 dark:border-mist-600 hover:border-attic-400 hover:text-attic-500'"
                  @click="form.icon = icon"
                >
                  <UIcon
                    :name="icon"
                    class="w-4 h-4"
                  />
                </button>
              </div>
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
    <UModal
      v-model:open="deleteModalOpen"
      title="Delete Location"
      description="Confirm permanent deletion of this location."
    >
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
