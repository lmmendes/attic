<script setup lang="ts">
import type { Collection, Category, Location, Condition, AssetsResponse, AssetFilters, Asset, Attribute, Tag } from '~/types/api'
import { buildCategoryOptions } from '~/utils/categoryHierarchy'
import { filterFailure, issueLabel } from '~/utils/assetFilters'

const uncategorizedCategoryFilter = 'uncategorized'

definePageMeta({
  middleware: 'auth'
})

const router = useRouter()
const route = useRoute()
const { features } = useFeatures()
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
  collection_id: features.value.collections && typeof route.query.collection_id === 'string' ? route.query.collection_id : undefined,
  q: '',
  category_id: features.value.categories && typeof route.query.category_id === 'string' ? route.query.category_id : undefined,
  location_id: features.value.locations && typeof route.query.location_id === 'string' ? route.query.location_id : undefined,
  condition_id: undefined,
  tag_ids: [],
  tag_match: 'any',
  limit: 24,
  offset: 0
})
const {
  criteria, expression, structured, selected, selectedId, modified, message, issues, busy, routeInvalid, savedLoading, revision,
  savedFilters, savedError, refreshSaved, apply, clear, selectSaved, save, rename, togglePin, remove
} = useAssetFiltering(filters)
const filterModal = useTemplateRef('filterModal')
const renameOpen = ref(false)
const renameName = ref('')
const deleteOpen = ref(false)
const filtersOpen = ref(false)
const savedSearchActions = computed(() => {
  if (!selected.value) return []
  const name = selected.value.name
  return [
    [{ label: `Saved search: ${name}`, type: 'label' as const }],
    [{
      label: selected.value.pinned ? 'Unpin from sidebar' : 'Pin to sidebar',
      icon: selected.value.pinned ? 'i-lucide-pin-off' : 'i-lucide-pin',
      onSelect: () => togglePin()
    }],
    [{
      label: `Edit “${name}”`,
      icon: 'i-lucide-sliders-horizontal',
      onSelect: () => openAdvanced('edit')
    }],
    [{
      label: `Rename “${name}”`,
      icon: 'i-lucide-pencil',
      onSelect: () => {
        renameName.value = name
        renameOpen.value = true
      }
    }],
    [{
      label: `Delete “${name}”`,
      icon: 'i-lucide-trash-2',
      color: 'error' as const,
      onSelect: () => { deleteOpen.value = true }
    }]
  ]
})

const queryString = computed(() => {
  const params = new URLSearchParams()
  if (features.value.collections && filters.collection_id) params.set('collection_id', filters.collection_id)
  if (filters.q) params.set('q', filters.q)
  if (features.value.attributes && filters.attribute_q) params.set('attribute_q', filters.attribute_q)
  if (features.value.categories && filters.category_id) params.set('category_id', filters.category_id)
  if (features.value.locations && filters.location_id) params.set('location_id', filters.location_id)
  if (features.value.conditions && filters.condition_id) params.set('condition_id', filters.condition_id)
  for (const id of filters.tag_ids || []) params.append('tag_id', id)
  if (filters.tag_ids?.length) params.set('tag_match', filters.tag_match || 'any')
  params.set('limit', String(filters.limit))
  params.set('offset', String(filters.offset))
  return params.toString()
})

const disabledCriterion = computed(() => Boolean(
  (filters.attribute_q && !features.value.attributes) || (filters.category_id && !features.value.categories)
  || (filters.location_id && !features.value.locations) || (filters.condition_id && !features.value.conditions)
  || (filters.collection_id && !features.value.collections)
))
const blocked = computed(() => routeInvalid.value || savedLoading.value || disabledCriterion.value || !!(selected.value?.issues?.length && !modified.value))
const structuredSearch = computed(() => structured.value || !!selectedId.value || !!expression.value)
const { data: rawAssetsResponse, status, error, refresh } = useApi<AssetsResponse>(
  () => structuredSearch.value ? '/api/assets/search' : `/api/assets?${queryString.value}`,
  {
    method: computed(() => structuredSearch.value ? 'POST' : 'GET'),
    body: computed(() => structuredSearch.value ? { criteria: criteria.value, limit: filters.limit, offset: filters.offset } : undefined),
    watch: false,
    immediate: !blocked.value
  }
)
const assetsResponse = computed(() => blocked.value || error.value ? null : rawAssetsResponse.value)
const queryFailure = computed(() => error.value ? filterFailure(error.value) : undefined)
watch([queryString, criteria, structuredSearch, blocked], () => {
  if (!blocked.value) void refresh()
}, { deep: true })

const { data: collections } = useApi<Collection[]>('/api/collections', { immediate: features.value.collections })
const collectionOptions = computed(() => collections.value?.map(c => ({ label: c.name, value: c.id, icon: c.icon })) || [])

const { data: categories } = useApi<Category[]>('/api/categories', { immediate: features.value.categories })
const { data: locations } = useApi<Location[]>('/api/locations', { immediate: features.value.locations })
const { data: conditions } = useApi<Condition[]>('/api/conditions', { immediate: features.value.conditions })
const { data: tags } = useApi<Tag[]>('/api/tags')
const tagOptions = computed(() => (tags.value || []).map(tag => ({ label: tag.name, value: tag.id })))
const { data: attributes, error: attributesError, refresh: refreshAttributes } = useApi<Attribute[]>('/api/attributes', { immediate: features.value.attributes })
const visibleAttributes = computed(() => features.value.attributes ? (attributes.value || []).filter(a => features.value.plugins || !a.plugin_id) : [])
const selectableAttributes = computed(() => visibleAttributes.value
  .filter(attribute => attribute.data_type === 'select')
  .filter(attribute => attribute.options?.length)
  .sort((left, right) => left.name.localeCompare(right.name)))
const attributeOptions = computed(() => selectableAttributes.value.map(attribute => ({
  label: attribute.name,
  value: attribute.id,
  icon: 'i-lucide-list-filter'
})))
const selectedAttributeId = ref<string>()
const selectedAttribute = computed(() => selectableAttributes.value.find(attribute => attribute.id === selectedAttributeId.value))
const attributeValueOptions = computed(() => (selectedAttribute.value?.options || [])
  .map(option => ({
    label: option.label,
    value: option.label,
    icon: 'i-lucide-tag'
  }))
  .sort((left, right) => left.label.localeCompare(right.label)))
const advancedOptions = computed(() => ({
  tags: tagOptions.value,
  ...(features.value.collections ? { collections: collectionOptions.value } : {}),
  ...(features.value.categories ? { category: categoryOptions.value } : {}),
  ...(features.value.locations ? { location: locationOptions.value } : {}),
  ...(features.value.conditions ? { condition: conditionOptions.value } : {})
}))

const categoryOptions = computed(() =>
  [
    { label: 'Uncategorized', value: uncategorizedCategoryFilter },
    ...buildCategoryOptions(categories.value || [])
  ]
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

type ActiveFilterKey = 'collection_id' | 'category_id' | 'location_id' | 'condition_id' | 'attribute_q' | 'tag_ids' | 'expression'

const activeFilterChips = computed<{ key: ActiveFilterKey, label: string }[]>(() => {
  const chips: { key: ActiveFilterKey, label: string }[] = []
  const optionName = (items: { id: string, name?: string, label?: string }[] | undefined, id: string) => {
    const item = items?.find(item => item.id === id)
    return item?.name || item?.label || id
  }

  if (filters.collection_id) chips.push({ key: 'collection_id', label: `Collection: ${optionName(collections.value, filters.collection_id)}` })
  if (filters.category_id) {
    const label = filters.category_id === uncategorizedCategoryFilter
      ? 'Uncategorized'
      : optionName(categories.value, filters.category_id)
    chips.push({ key: 'category_id', label: `Category: ${label}` })
  }
  if (filters.location_id) chips.push({ key: 'location_id', label: `Location: ${optionName(locations.value, filters.location_id)}` })
  if (filters.condition_id) chips.push({ key: 'condition_id', label: `Condition: ${optionName(conditions.value, filters.condition_id)}` })
  if (filters.tag_ids?.length) {
    const names = filters.tag_ids.map(id => optionName(tags.value, id)).join(', ')
    chips.push({ key: 'tag_ids', label: `Tags (${filters.tag_match || 'any'}): ${names}` })
  }
  if (filters.attribute_q) {
    const attribute = selectableAttributes.value.find(item => item.options?.some(option => option.label === filters.attribute_q))
    chips.push({ key: 'attribute_q', label: `${attribute?.name || 'Attribute'}: ${filters.attribute_q}` })
  }
  if (expression.value) chips.push({ key: 'expression', label: 'Advanced rules' })
  return chips
})

function removeActiveFilter(key: ActiveFilterKey) {
  if (key === 'expression') {
    const next = { ...criteria.value }
    delete next.expression
    apply(next)
    return
  }
  if (key === 'attribute_q') {
    selectAttributeValue(undefined)
    return
  }
  if (key === 'tag_ids') {
    filters.tag_ids = []
    filters.tag_match = 'any'
    filters.offset = 0
    return
  }
  filters[key] = undefined
  filters.offset = 0
}

const hasActiveFilters = computed(() => Boolean(
  filters.collection_id || filters.q || filters.attribute_q || filters.category_id || filters.location_id || filters.condition_id || filters.tag_ids?.length || expression.value || selectedId.value || routeInvalid.value
))

function clearFilters() {
  clear()
  selectedAttributeId.value = undefined
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

const visiblePages = computed(() => {
  const start = Math.max(1, Math.min(page.value - 1, totalPages.value - 2))
  return [...new Set([1, start, start + 1, start + 2, totalPages.value])]
    .filter(p => p >= 1 && p <= totalPages.value)
    .sort((a, b) => a - b)
})

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
const searchQuery = ref(filters.q || '')
const selectedAttributeValue = ref<string | undefined>(filters.attribute_q || undefined)
const attributeSearchTerm = ref(filters.attribute_q || '')
let searchTimeout: ReturnType<typeof setTimeout> | null = null
watch(searchQuery, (val: string) => {
  if (searchTimeout) clearTimeout(searchTimeout)
  if (val === (filters.q || '')) return
  searchTimeout = setTimeout(() => {
    filters.q = val
    filters.offset = 0
  }, 300)
})
watch(() => filters.q, (value) => {
  searchQuery.value = value || ''
}, { flush: 'sync' })
watch(() => filters.attribute_q, (value) => {
  selectedAttributeValue.value = value || undefined
  attributeSearchTerm.value = value || ''
}, { flush: 'sync' })
watch([selectableAttributes, () => filters.attribute_q], ([definitions, value]) => {
  if (!value) return
  const matchingAttribute = definitions.find(attribute => attribute.options?.some(option => option.label === value))
  selectedAttributeId.value = matchingAttribute?.id
}, { immediate: true })
watch(revision, () => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchQuery.value = filters.q || ''
  selectedAttributeValue.value = filters.attribute_q || undefined
  attributeSearchTerm.value = filters.attribute_q || ''
}, { flush: 'sync' })
onBeforeUnmount(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
})
function selectAttributeValue(value: string | undefined) {
  selectedAttributeValue.value = value
  filters.attribute_q = value
  filters.offset = 0
  attributeSearchTerm.value = value || ''
}
function selectAttribute(attributeId: string | undefined) {
  selectedAttributeId.value = attributeId
  selectAttributeValue(undefined)
}
function openAdvanced(mode: 'advanced' | 'edit' = 'advanced') {
  if (searchTimeout) clearTimeout(searchTimeout)
  filters.q = searchQuery.value
  filterModal.value?.show(mode)
}
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
          v-if="features.plugins"
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
      <div class="flex min-w-0 flex-col gap-2 lg:flex-row lg:items-center">
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
        <div
          v-if="selectedId || savedFilters?.length"
          class="flex w-full shrink-0 gap-1 lg:w-80 2xl:w-auto"
        >
          <USelectMenu
            :model-value="selectedId"
            :items="(savedFilters || []).map(f => ({ label: f.name, value: f.id }))"
            value-key="value"
            placeholder="Saved searches"
            aria-label="Saved searches"
            icon="i-lucide-bookmark"
            class="min-w-0 flex-1 2xl:w-64"
            :disabled="busy || savedLoading"
            @update:model-value="selectSaved($event)"
          />
          <span
            v-if="selected && modified"
            class="self-center rounded-md bg-warning/10 px-2 py-1 text-xs font-semibold text-warning"
          >Modified</span>
          <UDropdownMenu
            v-if="selected"
            :items="savedSearchActions"
            :content="{ align: 'end' }"
          >
            <UButton
              icon="i-lucide-ellipsis"
              color="neutral"
              variant="outline"
              :aria-label="`Manage saved search ${selected.name}`"
              :title="`Manage saved search ${selected.name}`"
              :disabled="busy"
            >
              <span class="hidden 2xl:inline">Manage</span>
            </UButton>
          </UDropdownMenu>
        </div>
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-list-filter"
          class="w-full shrink-0 justify-center font-semibold lg:w-auto"
          :trailing-icon="filtersOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          :aria-expanded="filtersOpen"
          aria-controls="asset-filters"
          aria-label="Filters"
          @click="filtersOpen = !filtersOpen"
        >
          Filters
          <span
            v-if="activeFilterChips.length"
            aria-hidden="true"
            class="rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-extrabold text-primary"
          >{{ activeFilterChips.length }}</span>
        </UButton>
      </div>
      <div
        v-if="activeFilterChips.length || hasActiveFilters"
        class="mt-3 flex flex-wrap items-center gap-2 border-t border-mist-100 pt-3 dark:border-mist-700"
        aria-label="Active filters"
      >
        <UButton
          v-for="chip in activeFilterChips"
          :key="chip.key"
          size="xs"
          color="neutral"
          variant="soft"
          trailing-icon="i-lucide-x"
          :aria-label="`Remove ${chip.label} filter`"
          @click="removeActiveFilter(chip.key)"
        >
          {{ chip.label }}
        </UButton>
        <UButton
          class="ml-auto"
          size="xs"
          variant="link"
          color="neutral"
          @click="clearFilters"
        >
          Clear all
        </UButton>
      </div>
      <div
        v-if="filtersOpen"
        id="asset-filters"
        class="mt-4 border-t border-mist-100 pt-4 dark:border-mist-700"
        aria-label="Filters"
      >
        <div class="mb-3 flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 class="text-sm font-extrabold text-mist-950 dark:text-white">
              Filters
            </h2>
            <p class="text-xs text-muted">
              Narrow the asset list, or combine detailed rules in the search builder.
            </p>
          </div>
          <UButton
            variant="link"
            class="w-fit px-0 font-semibold"
            trailing-icon="i-lucide-arrow-right"
            :disabled="savedLoading"
            @click="openAdvanced()"
          >
            Open search builder
          </UButton>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
          <USelectMenu
            v-if="features.collections"
            v-model="filters.collection_id"
            :items="collectionOptions"
            value-key="value"
            placeholder="Collection"
            aria-label="Filter by collection"
            class="min-w-0"
          />
          <USelectMenu
            v-model="filters.tag_ids"
            :items="tagOptions"
            multiple
            value-key="value"
            placeholder="Tags"
            aria-label="Filter by tags"
            class="min-w-0"
            icon="i-lucide-tags"
          />
          <USelect
            v-if="(filters.tag_ids?.length || 0) > 1"
            v-model="filters.tag_match"
            :items="[{ label: 'Match any tag', value: 'any' }, { label: 'Match all tags', value: 'all' }]"
            value-key="value"
            aria-label="Tag matching"
            class="min-w-0"
          />
          <USelectMenu
            v-if="features.categories"
            v-model="filters.category_id"
            :items="categoryOptions"
            placeholder="Category"
            aria-label="Filter by category"
            class="min-w-0"
            value-key="value"
            icon="i-lucide-folder"
          />
          <USelectMenu
            v-if="features.locations"
            v-model="filters.location_id"
            :items="locationOptions"
            placeholder="Location"
            aria-label="Filter by location"
            class="min-w-0"
            value-key="value"
            icon="i-lucide-map-pin"
          />
          <USelectMenu
            v-if="features.conditions"
            v-model="filters.condition_id"
            :items="conditionOptions"
            placeholder="Condition"
            aria-label="Filter by condition"
            class="min-w-0"
            value-key="value"
            icon="i-lucide-sparkles"
          />
          <USelectMenu
            v-if="features.attributes"
            id="advanced-attribute"
            :model-value="selectedAttributeId"
            :items="attributeOptions"
            value-key="value"
            placeholder="Attribute"
            aria-label="Attribute"
            class="min-w-0"
            @update:model-value="selectAttribute"
          />
          <UInputMenu
            v-if="features.attributes"
            id="advanced-attribute-value"
            v-model:search-term="attributeSearchTerm"
            :model-value="selectedAttributeValue"
            :items="attributeValueOptions"
            value-key="value"
            :placeholder="selectedAttributeId ? 'Attribute value' : 'Choose an attribute first'"
            aria-label="Attribute value"
            icon="i-lucide-search"
            class="min-w-0"
            :disabled="!selectedAttributeId"
            @update:model-value="selectAttributeValue"
          />
        </div>
        <p
          v-if="features.attributes && !selectableAttributes.length && !attributesError"
          class="mt-2 text-xs text-muted"
        >
          No configured attribute values are available. Use the search builder for text, number, date, or boolean attributes.
        </p>
      </div>
      <p
        v-if="message"
        class="mt-3 text-sm text-error"
        role="alert"
      >
        {{ message }}
      </p>
      <ul
        v-if="issues.length"
        class="mt-3 text-sm text-error"
        aria-label="Saved filter issues"
      >
        <li
          v-for="(issue, index) in issues"
          :key="index"
        >
          {{ issueLabel(issue, criteria, visibleAttributes) }}: {{ issue.message }}
        </li>
      </ul>
      <p
        v-if="savedError"
        class="mt-3 text-sm text-error"
        role="alert"
      >
        Could not load saved filters. <UButton
          size="xs"
          variant="link"
          @click="refreshSaved()"
        >
          Retry saved filters
        </UButton>
      </p>
      <p
        v-if="attributesError"
        class="mt-3 text-sm text-error"
        role="alert"
      >
        Could not load attribute definitions. <UButton
          size="xs"
          variant="link"
          @click="refreshAttributes()"
        >
          Retry attributes
        </UButton>
      </p>
    </section>

    <AssetFilterModal
      ref="filterModal"
      :criteria="criteria"
      :selected="selected"
      :attributes="visibleAttributes"
      :options="advancedOptions"
      :features="features"
      :issues="[...issues, ...(queryFailure?.issues || [])]"
      :message="message || queryFailure?.message || ''"
      :busy="busy"
      :save="save"
      @apply="apply"
    />
    <UModal
      v-model:open="renameOpen"
      title="Rename saved search"
      description="Only the name will change, including for searches that need repair."
      :dismissible="!busy"
      :close="!busy"
    >
      <template #body>
        <UInput
          v-model="renameName"
          aria-label="New filter name"
          class="w-full"
        />
        <p
          v-if="message"
          role="alert"
          class="text-error"
        >
          {{ message }}
        </p>
      </template>
      <template #footer>
        <UButton
          variant="ghost"
          color="neutral"
          :disabled="busy"
          @click="renameOpen = false"
        >
          Cancel
        </UButton>
        <UButton
          :loading="busy"
          :disabled="!renameName.trim()"
          @click="rename(renameName).then(ok => { if (ok) renameOpen = false })"
        >
          Rename
        </UButton>
      </template>
    </UModal>
    <UModal
      v-model:open="deleteOpen"
      title="Delete saved search"
      :description="`Delete ${selected?.name || 'this filter'}? Your current search will stay applied.`"
      :dismissible="!busy"
      :close="!busy"
    >
      <template #footer>
        <UButton
          variant="ghost"
          color="neutral"
          :disabled="busy"
          @click="deleteOpen = false"
        >
          Cancel
        </UButton>
        <UButton
          color="error"
          :loading="busy"
          @click="remove().then(ok => { if (ok) deleteOpen = false })"
        >
          Delete
        </UButton>
      </template>
    </UModal>

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
                <th
                  v-if="features.categories"
                  class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted md:table-cell"
                >
                  Category
                </th>
                <th
                  v-if="features.collections"
                  class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted md:table-cell"
                >
                  Collections
                </th>
                <th
                  v-if="features.locations"
                  class="hidden p-3 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted lg:table-cell"
                >
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
              <tr v-if="status === 'pending' || savedLoading">
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
              <tr v-else-if="error || blocked">
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
                      {{ queryFailure?.message || message || 'Open the search builder to remove or repair unavailable criteria.' }}
                    </p>
                    <ul
                      v-if="queryFailure?.issues.length"
                      class="mt-2 text-sm text-error"
                    >
                      <li
                        v-for="(issue, index) in queryFailure.issues"
                        :key="index"
                      >
                        {{ issueLabel(issue, criteria, visibleAttributes) }}: {{ issue.message }}
                      </li>
                    </ul>
                    <UButton
                      v-if="!blocked"
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
                  <div
                    v-if="features.categories || features.locations"
                    class="mt-1.5 flex items-center gap-1.5 md:hidden"
                  >
                    <span
                      v-if="features.categories"
                      class="text-[10px] font-bold text-attic-500"
                    >{{ asset.category?.name || 'Uncategorized' }}</span>
                    <span
                      v-if="features.locations && asset.location?.name"
                      class="text-[10px] text-muted"
                    >· {{ asset.location.name }}</span>
                  </div>
                  <div
                    v-if="features.collections && asset.collections?.length"
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
                  <div
                    v-if="asset.tags?.length"
                    class="mt-2 flex flex-wrap gap-1.5"
                    aria-label="Tags"
                  >
                    <NuxtLink
                      v-for="tag in asset.tags"
                      :key="tag.id"
                      :to="{ path: '/assets', query: { tag_id: tag.id } }"
                      class="rounded-md bg-mist-100 px-2 py-1 text-xs font-semibold text-mist-700 hover:underline dark:bg-mist-700 dark:text-mist-200"
                    >{{ tag.name }}</NuxtLink>
                  </div>
                </td>
                <td
                  v-if="features.categories"
                  class="hidden p-3 md:table-cell"
                >
                  <span class="inline-flex rounded-full bg-attic-50 px-2.5 py-1 text-[10px] font-extrabold text-attic-600 ring-1 ring-attic-100 dark:bg-attic-500/10 dark:text-attic-300 dark:ring-attic-500/20">
                    {{ asset.category?.name || 'Uncategorized' }}
                  </span>
                </td>
                <td
                  v-if="features.collections"
                  class="hidden max-w-64 p-3 md:table-cell"
                >
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
                <td
                  v-if="features.locations"
                  class="hidden p-3 lg:table-cell"
                >
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
            <template
              v-for="(p, index) in visiblePages"
              :key="p"
            >
              <span
                v-if="index > 0 && p - visiblePages[index - 1]! > 1"
                class="px-1 text-muted"
              >...</span>
              <UButton
                :aria-label="`Page ${p}`"
                :aria-current="p === page ? 'page' : undefined"
                :variant="p === page ? 'solid' : 'outline'"
                :color="p === page ? 'primary' : 'neutral'"
                size="sm"
                class="w-9"
                @click="page = p"
              >
                {{ p }}
              </UButton>
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
      v-if="features.plugins"
      v-model:open="importModalOpen"
      @imported="onImported"
    />
  </div>
</template>
