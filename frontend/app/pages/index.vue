<script setup lang="ts">
import type { Asset, AssetStats, Category, Collection, Location, Warranty } from '~/types/api'
import { getDashboardUrls } from '~/utils/dashboardUrls'

definePageMeta({ middleware: 'auth' })

const { user } = useAuth()
const { data: categories } = useApi<Category[]>('/api/categories')
const { data: locations } = useApi<Location[]>('/api/locations')
const { data: collections, error: collectionsError } = useApi<Collection[]>('/api/collections')
const { data: expiringWarranties } = useApi<Warranty[]>('/api/warranties/expiring?days=30')

const selectedLocationId = ref('all')

interface LocationTreeNode {
  location: Location
  children: LocationTreeNode[]
}

function buildLocationOptions(items: Location[]): { label: string, value: string }[] {
  const childrenMap = new Map<string | undefined, Location[]>()
  items.forEach((location) => {
    const parentId = location.parent_id || undefined
    if (!childrenMap.has(parentId)) childrenMap.set(parentId, [])
    childrenMap.get(parentId)!.push(location)
  })

  const buildTree = (parentId: string | undefined): LocationTreeNode[] =>
    (childrenMap.get(parentId) || [])
      .sort((a, b) => a.name.localeCompare(b.name))
      .map(location => ({ location, children: buildTree(location.id) }))

  const flattenTree = (nodes: LocationTreeNode[], depth = 0): { label: string, value: string }[] =>
    nodes.flatMap((node) => {
      const indent = depth > 0 ? `${'\u00A0'.repeat(depth * 2)}└ ` : ''
      return [
        { label: `${indent}${node.location.name}`, value: node.location.id },
        ...flattenTree(node.children, depth + 1)
      ]
    })

  return [
    { label: 'All locations', value: 'all' },
    ...flattenTree(buildTree(undefined))
  ]
}

const locationOptions = computed(() => buildLocationOptions(locations.value || []))
const selectedLocation = computed(() =>
  locations.value?.find(location => location.id === selectedLocationId.value)
)
const dashboardUrls = computed(() => getDashboardUrls(selectedLocationId.value))
const assetsUrl = computed(() => dashboardUrls.value.assets)
const assetStatsUrl = computed(() => dashboardUrls.value.stats)
const assetsPageUrl = computed(() => dashboardUrls.value.inventory)

const { data: assets } = useApi<{ assets: Asset[], total: number }>(() => assetsUrl.value)
const { data: assetStats } = useApi<AssetStats>(() => assetStatsUrl.value)

const formatCurrency = (value: number) => new Intl.NumberFormat('en-US', {
  style: 'currency', currency: 'USD', minimumFractionDigits: 0, maximumFractionDigits: 0
}).format(value)

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
})

const userName = computed(() => {
  if (user.value?.name) return user.value.name.split(' ')[0]
  return user.value?.email?.split('@')[0] || 'there'
})

const currentDate = new Intl.DateTimeFormat('en-US', {
  weekday: 'long', month: 'long', day: 'numeric'
}).format(new Date())

const formatRelativeTime = (dateString: string) => {
  const diffDays = Math.floor((Date.now() - new Date(dateString).getTime()) / 86400000)
  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  if (diffDays < 7) return `${diffDays} days ago`
  const weeks = Math.floor(diffDays / 7)
  return `${weeks} week${weeks > 1 ? 's' : ''} ago`
}

const overviewMetrics = computed(() => [
  { label: 'Assets', value: assets.value?.total || 0, icon: 'i-lucide-package', to: assetsPageUrl.value },
  { label: 'Locations', value: locations.value?.length || 0, icon: 'i-lucide-map-pin', to: '/locations' },
  { label: 'Expiring', value: expiringWarranties.value?.length || 0, icon: 'i-lucide-shield-alert', to: '/warranties' }
])

const quickLinks = computed(() => [
  {
    label: 'Collections', description: collectionsError.value ? 'Manage your collections' : `${collections.value?.length || 0} shared collections`,
    icon: 'i-lucide-library', to: '/collections',
    iconClass: 'bg-attic-100 text-attic-600 dark:bg-attic-500/15 dark:text-attic-300'
  },
  {
    label: 'Browse assets', description: `${assets.value?.total || 0} items catalogued`,
    icon: 'i-lucide-package-search', to: assetsPageUrl.value,
    iconClass: 'bg-attic-100 text-attic-600 dark:bg-attic-500/15 dark:text-attic-300'
  },
  {
    label: 'Locations', description: `${locations.value?.length || 0} storage spaces`,
    icon: 'i-lucide-map-pinned', to: '/locations',
    iconClass: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400'
  },
  {
    label: 'Categories', description: `${categories.value?.length || 0} ways to organize`,
    icon: 'i-lucide-shapes', to: '/categories',
    iconClass: 'bg-terracotta-100 text-terracotta-600 dark:bg-terracotta-500/10 dark:text-terracotta-300'
  },
  {
    label: 'Warranties',
    description: expiringWarranties.value?.length ? `${expiringWarranties.value.length} need attention` : 'Everything is up to date',
    icon: 'i-lucide-shield-check', to: '/warranties',
    iconClass: 'bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400'
  }
])
</script>

<template>
  <div class="grid grid-cols-1 gap-5 pb-20 xl:grid-cols-[minmax(0,1.6fr)_minmax(340px,0.8fr)] xl:pb-6">
    <header class="xl:col-span-2 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="mb-1.5 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500 dark:text-attic-300">
          {{ currentDate }}
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          {{ greeting }}, {{ userName }}.
        </h1>
        <p class="mt-1 text-sm text-muted">
          Your belongings, and where to find them.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          to="/assets?focus=search"
          color="neutral"
          variant="outline"
          icon="i-lucide-search"
          class="rounded-xl"
        >
          Find an asset
        </UButton>
        <UButton
          to="/assets/new"
          icon="i-lucide-plus"
          class="rounded-xl shadow-primary"
        >
          Add asset
        </UButton>
      </div>
    </header>

    <div class="relative isolate sm:min-h-[250px] overflow-hidden rounded-[24px] bg-gradient-to-br from-attic-500 via-attic-600 to-[#174AE8] p-5 text-white shadow-primary sm:p-6">
      <div class="pointer-events-none absolute -right-24 -top-32 size-80 rounded-full border-[42px] border-white/5" />
      <div class="pointer-events-none absolute -bottom-32 left-1/3 size-64 rounded-full bg-white/5 blur-2xl" />
      <div class="relative flex h-full flex-col justify-between gap-4 sm:gap-6">
        <div class="flex items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <div class="flex size-9 items-center justify-center rounded-xl border border-white/15 bg-white/10 backdrop-blur-sm">
              <UIcon
                name="i-lucide-chart-no-axes-combined"
                class="size-5"
              />
            </div>
            <div>
              <p class="text-[10px] font-extrabold uppercase tracking-[0.18em] text-white/85">
                Inventory overview
              </p>
              <USelectMenu
                v-model="selectedLocationId"
                :items="locationOptions"
                value-key="value"
                aria-label="Filter dashboard by location"
                variant="none"
                size="xs"
                class="-ml-1 mt-0.5 w-auto max-w-44"
                :ui="{
                  base: 'min-h-0 gap-1 rounded-md px-1 py-0.5 text-sm font-bold text-white/90 hover:bg-white/10 hover:text-white focus-visible:ring-2 focus-visible:ring-white/50',
                  value: 'truncate',
                  trailingIcon: 'size-3.5 text-white/85'
                }"
              />
            </div>
          </div>
          <NuxtLink
            :to="assetsPageUrl"
            aria-label="Open all assets"
            class="flex size-9 items-center justify-center rounded-xl border border-white/15 bg-white/10 transition hover:bg-white/20"
          >
            <UIcon
              name="i-lucide-arrow-up-right"
              class="size-4"
            />
          </NuxtLink>
        </div>
        <div>
          <p class="text-xs font-bold uppercase tracking-[0.14em] text-white/85">
            Total purchase value
          </p>
          <p class="mt-1 text-3xl font-black tracking-[-0.05em] sm:text-4xl">
            {{ formatCurrency(assetStats?.total_value || 0) }}
          </p>
        </div>
        <div class="grid grid-cols-3 overflow-hidden rounded-2xl border border-white/15 bg-white/10 backdrop-blur-sm">
          <NuxtLink
            v-for="(metric, index) in overviewMetrics"
            :key="metric.label"
            :to="metric.to"
            class="group flex min-w-0 flex-wrap items-center justify-center gap-x-2 gap-y-1 px-2 py-3 text-center transition hover:bg-white/10 sm:px-4"
            :class="index > 0 ? 'border-l border-white/15' : ''"
          >
            <UIcon
              :name="metric.icon"
              class="size-4 text-white/75 transition group-hover:text-white"
            />
            <span class="text-base font-black">{{ metric.value }}</span>
            <span class="w-full text-xs font-medium text-white/85 sm:w-auto">{{ metric.label }}</span>
          </NuxtLink>
        </div>
      </div>
    </div>

    <section class="xl:col-span-2">
      <div class="mb-3 flex items-end justify-between gap-4">
        <div>
          <p class="text-xs font-extrabold uppercase tracking-[0.14em] text-muted">
            Your inventory
          </p>
          <h2 class="text-lg font-extrabold text-mist-950 dark:text-white md:text-xl">
            Recently updated
          </h2>
        </div>
        <NuxtLink
          :to="assetsPageUrl"
          class="group flex items-center gap-1.5 text-sm font-bold text-attic-500 dark:text-attic-300 hover:text-attic-700"
        >
          View all <UIcon
            name="i-lucide-arrow-right"
            class="size-4 transition group-hover:translate-x-0.5"
          />
        </NuxtLink>
      </div>

      <div
        v-if="assets?.assets?.length"
        class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
      >
        <NuxtLink
          v-for="asset in assets.assets"
          :key="asset.id"
          :to="`/assets/${asset.id}`"
          class="attic-panel attic-panel-interactive group flex overflow-hidden rounded-[18px] sm:block"
        >
          <div class="relative w-20 shrink-0 overflow-hidden bg-gradient-to-br sm:aspect-[2/1] sm:w-auto from-attic-50 to-mist-100 dark:from-mist-700 dark:to-mist-800">
            <img
              v-if="asset.main_attachment_url"
              :src="asset.main_attachment_url"
              :alt="asset.name"
              class="size-full object-cover transition duration-300 group-hover:scale-[1.03]"
            >
            <div
              v-else
              class="flex size-full items-center justify-center"
            >
              <div class="flex size-14 items-center justify-center rounded-2xl bg-white/80 text-attic-500 dark:text-attic-300 shadow-sm dark:bg-mist-700">
                <UIcon
                  name="i-lucide-package"
                  class="size-6"
                />
              </div>
            </div>
            <span class="absolute left-3 top-3 hidden rounded-full sm:inline bg-white/90 px-2.5 py-1 text-[10px] font-extrabold text-mist-600 shadow-sm backdrop-blur dark:bg-mist-900/85 dark:text-mist-300">
              {{ asset.category?.name || 'Uncategorized' }}
            </span>
          </div>
          <div class="min-w-0 flex-1 p-3.5">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h3 class="line-clamp-2 font-extrabold text-mist-950 dark:text-white">{{ asset.name }}</h3>
                <p class="mt-1 flex items-center gap-1.5 truncate text-xs text-muted">
                  <UIcon
                    name="i-lucide-map-pin"
                    class="size-3.5 shrink-0"
                  />{{ asset.location?.name || 'No location' }}
                </p>
                <div
                  class="mt-2 flex min-w-0 items-center gap-1.5 text-[11px] text-muted"
                  :title="asset.collections?.length ? asset.collections.map(collection => collection.name).join(', ') : 'No collections'"
                >
                  <UIcon
                    name="i-lucide-library"
                    class="size-3.5 shrink-0"
                  />
                  <template v-if="asset.collections?.length">
                    <span class="truncate rounded-md bg-attic-50 px-1.5 py-0.5 font-semibold text-attic-600 dark:bg-attic-500/10 dark:text-attic-300">
                      {{ asset.collections[0]?.name }}
                    </span>
                    <span
                      v-if="asset.collections.length > 1"
                      class="shrink-0 font-bold text-attic-500 dark:text-attic-300"
                    >
                      +{{ asset.collections.length - 1 }}
                    </span>
                  </template>
                  <span v-else>No collections</span>
                </div>
              </div>
              <UIcon
                name="i-lucide-arrow-up-right"
                class="mt-0.5 size-4 shrink-0 text-mist-300 transition group-hover:text-attic-500 dark:text-attic-300"
              />
            </div>
            <p class="mt-2.5 text-[11px] font-semibold text-muted">{{ formatRelativeTime(asset.updated_at) }}</p>
          </div>
        </NuxtLink>
      </div>

      <div
        v-else-if="selectedLocation"
        class="attic-panel rounded-[24px] px-6 py-14 text-center"
      >
        <div class="mx-auto flex size-14 items-center justify-center rounded-2xl bg-attic-50 text-attic-500 dark:text-attic-300 dark:bg-attic-500/10">
          <UIcon
            name="i-lucide-map-pin-off"
            class="size-6"
          />
        </div>
        <h3 class="mt-4 font-extrabold text-mist-950 dark:text-white">
          No assets in {{ selectedLocation.name }}
        </h3>
        <p class="mx-auto mt-1 max-w-sm text-sm text-muted">
          Try another location or view the full inventory.
        </p>
        <UButton
          variant="outline"
          color="neutral"
          icon="i-lucide-layers"
          class="mt-5 rounded-xl"
          @click="selectedLocationId = 'all'"
        >
          View all locations
        </UButton>
      </div>

      <div
        v-else
        class="attic-panel rounded-[24px] px-6 py-14 text-center"
      >
        <div class="mx-auto flex size-14 items-center justify-center rounded-2xl bg-attic-50 text-attic-500 dark:text-attic-300 dark:bg-attic-500/10">
          <UIcon
            name="i-lucide-package-open"
            class="size-6"
          />
        </div>
        <h3 class="mt-4 font-extrabold text-mist-950 dark:text-white">
          Your inventory starts here
        </h3>
        <p class="mx-auto mt-1 max-w-sm text-sm text-muted">
          Add your first asset and Attic will keep the details organized.
        </p>
        <UButton
          to="/assets/new"
          icon="i-lucide-plus"
          class="mt-5 rounded-xl"
        >
          Add your first asset
        </UButton>
      </div>
    </section>

    <NuxtLink
      v-if="(expiringWarranties?.length || 0) > 0"
      to="/warranties"
      class="xl:col-span-2 group flex flex-col gap-4 rounded-[22px] border border-amber-200 bg-amber-50 p-5 transition hover:border-amber-300 dark:border-amber-800/50 dark:bg-amber-900/10 sm:flex-row sm:items-center"
    >
      <div class="flex size-11 shrink-0 items-center justify-center rounded-xl bg-amber-100 text-amber-600 dark:bg-amber-500/15 dark:text-amber-400">
        <UIcon
          name="i-lucide-shield-alert"
          class="size-5"
        />
      </div>
      <div class="flex-1">
        <p class="font-extrabold text-mist-950 dark:text-white">A warranty check is due</p>
        <p class="mt-0.5 text-sm text-mist-600 dark:text-mist-400">
          {{ expiringWarranties?.length }} {{ expiringWarranties?.length === 1 ? 'warranty expires' : 'warranties expire' }} in the next 30 days.
        </p>
      </div>
      <span class="flex items-center gap-1.5 text-sm font-bold text-amber-700 dark:text-amber-400">
        Review coverage <UIcon
          name="i-lucide-arrow-right"
          class="size-4 transition group-hover:translate-x-0.5"
        />
      </span>
    </NuxtLink>
    <div class="attic-panel xl:col-start-2 xl:row-start-2 rounded-[22px] p-4">
      <div class="mb-2 flex items-center justify-between">
        <h2 class="text-lg font-extrabold text-mist-950 dark:text-white">
          Quick access
        </h2>
        <UIcon
          name="i-lucide-sparkles"
          class="size-5 text-terracotta-400"
        />
      </div>
      <div class="grid gap-1 sm:grid-cols-2 xl:grid-cols-1">
        <NuxtLink
          v-for="link in quickLinks"
          :key="link.to"
          :to="link.to"
          class="group flex min-h-11 items-center gap-2.5 rounded-xl px-2 py-1 transition hover:bg-mist-50 dark:hover:bg-mist-700/50"
        >
          <div
            class="flex size-8 shrink-0 items-center justify-center rounded-lg"
            :class="link.iconClass"
          >
            <UIcon
              :name="link.icon"
              class="size-4"
            />
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-sm font-bold text-mist-950 dark:text-white">{{ link.label }}</p>
            <p class="truncate text-xs text-muted">{{ link.description }}</p>
          </div>
          <UIcon
            name="i-lucide-chevron-right"
            class="size-4 text-mist-300 transition group-hover:translate-x-0.5 group-hover:text-attic-500 dark:text-attic-300"
          />
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
