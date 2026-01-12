<script setup lang="ts">
import type { Asset, AssetStats, Category, Location, Warranty } from '~/types/api'

// No middleware - accessible to all

const { isAuthenticated: loggedIn, login } = useAuth()

// Fetch dashboard data when logged in
const { data: assets } = useApi<{ assets: Asset[], total: number }>('/api/assets?limit=5', {
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

const stats = computed(() => [
  {
    label: 'Total Value',
    value: formatCurrency(assetStats.value?.total_value || 0),
    icon: 'i-lucide-dollar-sign',
    to: '/assets'
  },
  {
    label: 'Total Assets',
    value: assets.value?.total || 0,
    icon: 'i-lucide-box',
    to: '/assets'
  },
  {
    label: 'Categories',
    value: categories.value?.length || 0,
    icon: 'i-lucide-folder-tree',
    to: '/categories'
  },
  {
    label: 'Locations',
    value: locations.value?.length || 0,
    icon: 'i-lucide-map-pin',
    to: '/locations'
  },
  {
    label: 'Expiring Warranties',
    value: expiringWarranties.value?.length || 0,
    icon: 'i-lucide-shield-alert',
    to: '/warranties'
  }
])
</script>

<template>
  <UContainer>
    <!-- Logged out state -->
    <template v-if="!loggedIn">
      <div class="py-24 text-center">
        <UIcon name="i-lucide-archive" class="w-16 h-16 mx-auto text-primary mb-6" />
        <h1 class="text-4xl font-bold mb-4">
          Welcome to Attic
        </h1>
        <p class="text-xl text-muted mb-8 max-w-2xl mx-auto">
          A simple, powerful asset management system for organizations.
          Track your assets, manage warranties, and keep everything organized.
        </p>
        <UButton
          size="xl"
          @click="login()"
        >
          Sign in to get started
        </UButton>
      </div>
    </template>

    <!-- Logged in dashboard -->
    <template v-else>
      <div class="py-8">
        <h1 class="text-2xl font-bold mb-6">
          Dashboard
        </h1>

        <!-- Stats Grid -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
          <NuxtLink
            v-for="stat in stats"
            :key="stat.label"
            :to="stat.to"
          >
            <UCard class="hover:bg-elevated transition-colors">
              <div class="flex items-center gap-4">
                <div class="p-3 rounded-lg bg-primary/10">
                  <UIcon :name="stat.icon" class="w-6 h-6 text-primary" />
                </div>
                <div>
                  <p class="text-2xl font-bold">{{ stat.value }}</p>
                  <p class="text-sm text-muted">{{ stat.label }}</p>
                </div>
              </div>
            </UCard>
          </NuxtLink>
        </div>

        <!-- Recent Assets -->
        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h2 class="font-semibold">Recent Assets</h2>
              <UButton
                to="/assets"
                variant="ghost"
                trailing-icon="i-lucide-arrow-right"
              >
                View all
              </UButton>
            </div>
          </template>

          <UTable
            v-if="assets?.assets?.length"
            :data="assets.assets"
            :columns="[
              { accessorKey: 'name', id: 'name', header: 'Name' },
              { accessorFn: (row: any) => row.category?.name, id: 'category', header: 'Category' },
              { accessorFn: (row: any) => row.location?.name, id: 'location', header: 'Location' },
              { accessorKey: 'quantity', id: 'quantity', header: 'Qty' }
            ]"
          />
          <div v-else class="text-center py-8 text-muted">
            <UIcon name="i-lucide-inbox" class="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>No assets yet. Start by adding your first asset.</p>
            <UButton
              to="/assets/new"
              class="mt-4"
              variant="soft"
            >
              Add Asset
            </UButton>
          </div>
        </UCard>
      </div>
    </template>
  </UContainer>
</template>
