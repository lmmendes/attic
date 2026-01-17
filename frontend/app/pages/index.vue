<script setup lang="ts">
import type { Asset, AssetStats, Category, Location, Warranty } from '~/types/api'

// No middleware - accessible to all

const { isAuthenticated: loggedIn, user } = useAuth()

// Fetch dashboard data when logged in
const { data: assets } = useApi<{ assets: Asset[], total: number }>('/api/assets?limit=4', {
  immediate: loggedIn.value
})

const { data: assetStats } = useApi<AssetStats>('/api/assets/stats', {
  immediate: loggedIn.value
})

const { data: categories } = useApi<Category[]>('/api/categories', {
  immediate: loggedIn.value
})

const { data: locations } = useApi<Location[]>('/api/locations', {
  immediate: loggedIn.value
})

const { data: expiringWarranties } = useApi<Warranty[]>('/api/warranties/expiring?days=30', {
  immediate: loggedIn.value
})

const formatCurrency = (value: number) => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0
  }).format(value)
}

// Get greeting based on time of day
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
})

const userName = computed(() => {
  if (user.value?.name) {
    return user.value.name.split(' ')[0]
  }
  return user.value?.email?.split('@')[0] || 'there'
})

// Format relative time
const formatRelativeTime = (dateString: string) => {
  const date = new Date(dateString)
  const now = new Date()
  const diffDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return 'Added today'
  if (diffDays === 1) return 'Added yesterday'
  if (diffDays < 7) return `Added ${diffDays} days ago`
  return `Added ${Math.floor(diffDays / 7)} week${Math.floor(diffDays / 7) > 1 ? 's' : ''} ago`
}
</script>

<template>
  <div>
    <!-- Logged out state -->
    <div
      v-if="!loggedIn"
      class="py-24 text-center max-w-xl mx-auto"
    >
      <div class="bg-attic-500/10 rounded-2xl p-4 w-20 h-20 mx-auto mb-6 flex items-center justify-center">
        <UIcon
          name="i-lucide-archive"
          class="w-10 h-10 text-attic-500"
        />
      </div>
      <h1 class="text-4xl font-bold text-mist-950 dark:text-white mb-4">
        Welcome to Attic
      </h1>
      <p class="text-lg text-mist-500 mb-8">
        A simple, powerful asset management system for organizations.
        Track your assets, manage warranties, and keep everything organized.
      </p>
      <UButton
        size="xl"
        to="/login"
      >
        Sign in to get started
      </UButton>
    </div>

    <!-- Logged in dashboard -->
    <div
      v-else
      class="flex flex-col gap-8"
    >
      <!-- Welcome Section -->
      <div>
        <h2 class="text-2xl font-bold text-mist-950 dark:text-white">
          {{ greeting }}, {{ userName }}
        </h2>
        <p class="text-mist-500 mt-1">
          Here is what's happening in your Attic today.
        </p>
      </div>

      <!-- Stats Grid -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <!-- Total Assets -->
        <NuxtLink
          to="/assets"
          class="bg-white dark:bg-mist-800 p-6 rounded-xl shadow-card border border-mist-100 dark:border-mist-700 flex flex-col justify-between h-32 relative overflow-hidden group hover:shadow-soft transition-all"
        >
          <div class="absolute right-0 top-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <UIcon
              name="i-lucide-box"
              class="w-16 h-16 text-attic-500"
            />
          </div>
          <p class="text-mist-500 font-medium text-sm">Total Assets</p>
          <div class="flex items-baseline gap-2">
            <span class="text-3xl font-bold text-mist-950 dark:text-white">{{ assets?.total || 0 }}</span>
          </div>
        </NuxtLink>

        <!-- Total Value -->
        <NuxtLink
          to="/assets"
          class="bg-white dark:bg-mist-800 p-6 rounded-xl shadow-card border border-mist-100 dark:border-mist-700 flex flex-col justify-between h-32 relative overflow-hidden group hover:shadow-soft transition-all"
        >
          <div class="absolute right-0 top-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <UIcon
              name="i-lucide-banknote"
              class="w-16 h-16 text-attic-500"
            />
          </div>
          <p class="text-mist-500 font-medium text-sm">Total Value</p>
          <div class="flex items-baseline gap-2">
            <span class="text-3xl font-bold text-mist-950 dark:text-white">{{ formatCurrency(assetStats?.total_value || 0) }}</span>
          </div>
        </NuxtLink>

        <!-- Expiring Warranties -->
        <NuxtLink
          to="/warranties"
          class="bg-white dark:bg-mist-800 p-6 rounded-xl shadow-card border border-mist-100 dark:border-mist-700 flex flex-col justify-between h-32 relative overflow-hidden group hover:shadow-soft transition-all"
        >
          <div class="absolute right-0 top-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <UIcon
              name="i-lucide-shield-alert"
              class="w-16 h-16 text-amber-500"
            />
          </div>
          <p class="text-mist-500 font-medium text-sm">Expiring Warranties</p>
          <div class="flex items-baseline gap-2">
            <span class="text-3xl font-bold text-mist-950 dark:text-white">{{ expiringWarranties?.length || 0 }}</span>
            <span
              v-if="(expiringWarranties?.length || 0) > 0"
              class="text-xs text-amber-600 bg-amber-500/10 px-2 py-0.5 rounded-full font-bold"
            >
              within 30 days
            </span>
          </div>
        </NuxtLink>
      </div>

      <!-- Main Layout Grid (2 Columns) -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <!-- Left Column: Recent Assets -->
        <div class="lg:col-span-2 flex flex-col gap-6">
          <div class="flex items-center justify-between">
            <h3 class="text-lg font-bold text-mist-950 dark:text-white">
              Recently Added
            </h3>
            <NuxtLink
              to="/assets"
              class="text-sm font-medium text-attic-500 hover:text-attic-700 hover:underline"
            >
              View All
            </NuxtLink>
          </div>

          <div
            v-if="assets?.assets?.length"
            class="grid grid-cols-1 sm:grid-cols-2 gap-4"
          >
            <NuxtLink
              v-for="asset in assets.assets"
              :key="asset.id"
              :to="`/assets/${asset.id}`"
              class="bg-white dark:bg-mist-800 p-4 rounded-xl shadow-card border border-mist-100 dark:border-mist-700 flex gap-4 hover:shadow-soft transition-all cursor-pointer group"
            >
              <div class="w-20 h-20 rounded-lg bg-mist-100 dark:bg-mist-700 flex-shrink-0 flex items-center justify-center">
                <UIcon
                  name="i-lucide-package"
                  class="w-8 h-8 text-mist-400 group-hover:text-attic-500 transition-colors"
                />
              </div>
              <div class="flex flex-col justify-center min-w-0">
                <h4 class="font-bold text-mist-950 dark:text-white truncate">
                  {{ asset.name }}
                </h4>
                <p class="text-sm text-mist-500 mb-2 truncate">
                  {{ asset.category?.name || 'Uncategorized' }}
                  <span v-if="asset.location?.name"> &bull; {{ asset.location.name }}</span>
                </p>
                <span class="text-xs text-mist-500 bg-mist-200 dark:bg-mist-700 px-2 py-1 rounded w-fit">
                  {{ formatRelativeTime(asset.created_at) }}
                </span>
              </div>
            </NuxtLink>
          </div>

          <!-- Empty state -->
          <div
            v-else
            class="bg-white dark:bg-mist-800 p-8 rounded-xl shadow-card border border-mist-100 dark:border-mist-700 text-center"
          >
            <UIcon
              name="i-lucide-inbox"
              class="w-12 h-12 mx-auto mb-4 text-mist-300"
            />
            <p class="text-mist-500 mb-4">
              No assets yet. Start by adding your first asset.
            </p>
            <UButton
              to="/assets/new"
              variant="soft"
            >
              Add Asset
            </UButton>
          </div>
        </div>

        <!-- Right Column: Quick Stats -->
        <div class="flex flex-col gap-6">
          <div class="flex items-center justify-between">
            <h3 class="text-lg font-bold text-mist-950 dark:text-white">
              Quick Overview
            </h3>
          </div>

          <div class="bg-white dark:bg-mist-800 rounded-xl shadow-card border border-mist-100 dark:border-mist-700 p-4">
            <div class="flex flex-col gap-1">
              <!-- Categories -->
              <NuxtLink
                to="/categories"
                class="flex items-center gap-2 p-2 hover:bg-mist-100 dark:hover:bg-mist-700 rounded cursor-pointer group"
              >
                <UIcon
                  name="i-lucide-folder-tree"
                  class="w-5 h-5 text-mist-500 group-hover:text-attic-500"
                />
                <span class="font-medium text-sm text-mist-950 dark:text-white">Categories</span>
                <span class="ml-auto text-xs font-mono text-mist-500 bg-mist-200 dark:bg-mist-700 px-1.5 py-0.5 rounded">
                  {{ categories?.length || 0 }}
                </span>
              </NuxtLink>

              <!-- Locations -->
              <NuxtLink
                to="/locations"
                class="flex items-center gap-2 p-2 hover:bg-mist-100 dark:hover:bg-mist-700 rounded cursor-pointer group"
              >
                <UIcon
                  name="i-lucide-map-pin"
                  class="w-5 h-5 text-mist-500 group-hover:text-attic-500"
                />
                <span class="font-medium text-sm text-mist-950 dark:text-white">Locations</span>
                <span class="ml-auto text-xs font-mono text-mist-500 bg-mist-200 dark:bg-mist-700 px-1.5 py-0.5 rounded">
                  {{ locations?.length || 0 }}
                </span>
              </NuxtLink>

              <!-- Warranties expiring -->
              <NuxtLink
                to="/warranties"
                class="flex items-center gap-2 p-2 hover:bg-mist-100 dark:hover:bg-mist-700 rounded cursor-pointer group"
              >
                <UIcon
                  name="i-lucide-shield-check"
                  class="w-5 h-5 text-mist-500 group-hover:text-attic-500"
                />
                <span class="font-medium text-sm text-mist-950 dark:text-white">Warranties</span>
                <div class="ml-auto flex items-center gap-2">
                  <span
                    v-if="(expiringWarranties?.length || 0) > 0"
                    class="w-2 h-2 rounded-full bg-amber-500"
                  />
                  <span class="text-xs font-mono text-mist-500 bg-mist-200 dark:bg-mist-700 px-1.5 py-0.5 rounded">
                    {{ expiringWarranties?.length || 0 }}
                  </span>
                </div>
              </NuxtLink>
            </div>
          </div>

          <!-- Quick actions -->
          <div class="bg-gradient-to-br from-mist-200 to-white dark:from-mist-800 dark:to-mist-700 p-4 rounded-xl border border-mist-200 dark:border-mist-600">
            <div class="flex items-start gap-3">
              <div class="p-2 bg-white dark:bg-mist-600 rounded-lg shadow-sm">
                <UIcon
                  name="i-lucide-plus-circle"
                  class="w-5 h-5 text-attic-500"
                />
              </div>
              <div>
                <h4 class="text-sm font-bold text-mist-950 dark:text-white">
                  Quick Actions
                </h4>
                <p class="text-xs text-mist-500 mt-1 leading-relaxed">
                  Add a new asset, category, or location to keep your inventory organized.
                </p>
                <div class="flex gap-2 mt-3 flex-wrap">
                  <UButton
                    to="/assets/new"
                    size="xs"
                  >
                    New Asset
                  </UButton>
                  <UButton
                    to="/categories"
                    size="xs"
                    color="neutral"
                    variant="soft"
                  >
                    Categories
                  </UButton>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
