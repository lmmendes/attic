<script setup lang="ts">
import type { WarrantyWithAsset } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const { data: warranties, status } = useApi<WarrantyWithAsset[]>('/api/warranties')

// Search
const searchQuery = ref('')

// Pagination
const currentPage = ref(1)
const itemsPerPage = ref(10)

// Get warranty status
function getWarrantyStatus(warranty: WarrantyWithAsset): 'active' | 'expiring' | 'expired' | 'no_date' {
  if (!warranty.end_date) return 'no_date'

  const endDate = new Date(warranty.end_date)
  const now = new Date()
  const daysUntilExpiry = Math.ceil((endDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))

  if (daysUntilExpiry < 0) return 'expired'
  if (daysUntilExpiry <= 30) return 'expiring'
  return 'active'
}

// Days until expiry
function getDaysUntilExpiry(warranty: WarrantyWithAsset): number | null {
  if (!warranty.end_date) return null
  const endDate = new Date(warranty.end_date)
  const now = new Date()
  return Math.ceil((endDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

// Filtered warranties
const filteredWarranties = computed(() => {
  if (!warranties.value) return []
  if (!searchQuery.value.trim()) return warranties.value

  const query = searchQuery.value.toLowerCase()
  return warranties.value.filter(
    w => w.asset_name.toLowerCase().includes(query)
      || (w.provider && w.provider.toLowerCase().includes(query))
  )
})

// Paginated warranties
const paginatedWarranties = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value
  const end = start + itemsPerPage.value
  return filteredWarranties.value.slice(start, end)
})

// Total pages
const totalPages = computed(() => Math.ceil(filteredWarranties.value.length / itemsPerPage.value))

// Reset to page 1 when search changes
watch(searchQuery, () => {
  currentPage.value = 1
})

// Pagination helpers
function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
  }
}

function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
  }
}

// Get style for status
function getStatusStyle(status: string): { icon: string, bgColor: string, textColor: string, borderColor: string, label: string } {
  switch (status) {
    case 'active':
      return {
        icon: 'i-lucide-shield-check',
        bgColor: 'bg-emerald-50 dark:bg-emerald-900/30',
        textColor: 'text-emerald-700 dark:text-emerald-300',
        borderColor: 'border-emerald-100 dark:border-emerald-900/50',
        label: 'Active'
      }
    case 'expiring':
      return {
        icon: 'i-lucide-clock',
        bgColor: 'bg-amber-50 dark:bg-amber-900/30',
        textColor: 'text-amber-700 dark:text-amber-300',
        borderColor: 'border-amber-100 dark:border-amber-900/50',
        label: 'Expiring Soon'
      }
    case 'expired':
      return {
        icon: 'i-lucide-shield-off',
        bgColor: 'bg-red-50 dark:bg-red-900/30',
        textColor: 'text-red-700 dark:text-red-300',
        borderColor: 'border-red-100 dark:border-red-900/50',
        label: 'Expired'
      }
    default:
      return {
        icon: 'i-lucide-help-circle',
        bgColor: 'bg-slate-100 dark:bg-slate-800',
        textColor: 'text-slate-700 dark:text-slate-300',
        borderColor: 'border-slate-200 dark:border-slate-700',
        label: 'No End Date'
      }
  }
}

// Get icon style for warranty based on status
function getWarrantyIcon(warranty: WarrantyWithAsset): { icon: string, bgColor: string, textColor: string } {
  const status = getWarrantyStatus(warranty)
  const style = getStatusStyle(status)
  return { icon: style.icon, bgColor: style.bgColor, textColor: style.textColor }
}

// Format date
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}
</script>

<template>
  <div class="space-y-8">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          Warranties
        </h1>
        <p class="text-mist-500">
          Track and monitor warranty coverage across all your assets.
        </p>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="flex items-center gap-4">
      <div class="relative flex-1 max-w-sm">
        <UIcon
          name="i-lucide-search"
          class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-mist-400"
        />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search warranties..."
          class="w-full pl-10 pr-4 py-2.5 bg-mist-50 dark:bg-mist-800 border border-mist-200 dark:border-mist-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-attic-500/20 focus:border-attic-500 text-mist-950 dark:text-white placeholder-mist-400"
        >
      </div>
    </div>

    <!-- Data Table -->
    <div class="overflow-hidden rounded-xl border border-mist-100 dark:border-mist-700 bg-white dark:bg-mist-800 shadow-sm">
      <!-- Loading State -->
      <div
        v-if="status === 'pending'"
        class="flex items-center justify-center py-20"
      >
        <UIcon
          name="i-lucide-loader-2"
          class="w-8 h-8 text-attic-500 animate-spin"
        />
      </div>

      <!-- Empty State -->
      <div
        v-else-if="!filteredWarranties.length && !searchQuery"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-shield"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No warranties yet
        </h3>
        <p class="text-sm text-mist-500 mb-4 max-w-sm">
          Add warranty information to your assets to track coverage and expiration dates.
        </p>
        <UButton to="/assets">
          View Assets
        </UButton>
      </div>

      <!-- No Results -->
      <div
        v-else-if="!filteredWarranties.length && searchQuery"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <UIcon
          name="i-lucide-search-x"
          class="w-12 h-12 text-mist-300 mb-4"
        />
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No results found
        </h3>
        <p class="text-sm text-mist-500">
          No warranties match "{{ searchQuery }}"
        </p>
      </div>

      <!-- Table -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[700px] border-collapse">
            <thead class="bg-mist-50/50 dark:bg-mist-700/30 border-b border-mist-100 dark:border-mist-700">
              <tr>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Asset
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Provider
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Status
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Expires
                </th>
                <th class="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-mist-500">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-mist-100 dark:divide-mist-700">
              <tr
                v-for="warranty in paginatedWarranties"
                :key="warranty.id"
                class="group hover:bg-mist-50/50 dark:hover:bg-mist-700/30 transition-colors"
              >
                <!-- Asset Name with Icon -->
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <div
                      class="size-9 rounded-full flex items-center justify-center"
                      :class="[getWarrantyIcon(warranty).bgColor, getWarrantyIcon(warranty).textColor]"
                    >
                      <UIcon
                        :name="getWarrantyIcon(warranty).icon"
                        class="w-4 h-4"
                      />
                    </div>
                    <NuxtLink
                      :to="`/assets/${warranty.asset_id}`"
                      class="text-mist-950 dark:text-white text-sm font-semibold hover:text-attic-500 transition-colors"
                    >
                      {{ warranty.asset_name }}
                    </NuxtLink>
                  </div>
                </td>

                <!-- Provider -->
                <td class="px-6 py-4">
                  <span
                    v-if="warranty.provider"
                    class="text-sm text-mist-700 dark:text-mist-300"
                  >
                    {{ warranty.provider }}
                  </span>
                  <span
                    v-else
                    class="text-sm text-mist-300 dark:text-mist-600 italic"
                  >
                    Not specified
                  </span>
                </td>

                <!-- Status Badge -->
                <td class="px-6 py-4">
                  <span
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-semibold border"
                    :class="[getStatusStyle(getWarrantyStatus(warranty)).bgColor, getStatusStyle(getWarrantyStatus(warranty)).textColor, getStatusStyle(getWarrantyStatus(warranty)).borderColor]"
                  >
                    <UIcon
                      :name="getStatusStyle(getWarrantyStatus(warranty)).icon"
                      class="w-3.5 h-3.5"
                    />
                    {{ getStatusStyle(getWarrantyStatus(warranty)).label }}
                  </span>
                </td>

                <!-- Expires -->
                <td class="px-6 py-4">
                  <div class="flex flex-col">
                    <span class="text-sm text-mist-700 dark:text-mist-300">
                      {{ formatDate(warranty.end_date) }}
                    </span>
                    <span
                      v-if="getDaysUntilExpiry(warranty) !== null"
                      class="text-xs text-mist-400"
                    >
                      <template v-if="getDaysUntilExpiry(warranty)! < 0">
                        {{ Math.abs(getDaysUntilExpiry(warranty)!) }} days ago
                      </template>
                      <template v-else-if="getDaysUntilExpiry(warranty) === 0">
                        Expires today
                      </template>
                      <template v-else>
                        {{ getDaysUntilExpiry(warranty) }} days left
                      </template>
                    </span>
                  </div>
                </td>

                <!-- Actions -->
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <NuxtLink
                      :to="`/assets/${warranty.asset_id}`"
                      class="size-8 rounded flex items-center justify-center text-mist-400 hover:text-attic-500 hover:bg-attic-500/10 transition-colors"
                      title="View Asset"
                    >
                      <UIcon
                        name="i-lucide-external-link"
                        class="w-4 h-4"
                      />
                    </NuxtLink>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Footer with Pagination -->
        <div class="px-6 py-3 border-t border-mist-100 dark:border-mist-700 bg-mist-50/50 dark:bg-mist-700/20 flex items-center justify-between">
          <p class="text-xs text-mist-500">
            Showing {{ (currentPage - 1) * itemsPerPage + 1 }}-{{ Math.min(currentPage * itemsPerPage, filteredWarranties.length) }} of {{ filteredWarranties.length }} warranties
            <span v-if="searchQuery && warranties?.length !== filteredWarranties.length">
              (filtered from {{ warranties?.length || 0 }})
            </span>
          </p>
          <div
            v-if="totalPages > 1"
            class="flex items-center gap-2"
          >
            <button
              class="px-3 py-1.5 text-xs font-medium border border-mist-200 dark:border-mist-600 rounded-lg hover:bg-mist-100 dark:hover:bg-mist-700 text-mist-600 dark:text-mist-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              :disabled="currentPage === 1"
              @click="prevPage"
            >
              Prev
            </button>
            <span class="text-xs text-mist-500 px-2">
              Page {{ currentPage }} of {{ totalPages }}
            </span>
            <button
              class="px-3 py-1.5 text-xs font-medium border border-mist-200 dark:border-mist-600 rounded-lg hover:bg-mist-100 dark:hover:bg-mist-700 text-mist-600 dark:text-mist-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              :disabled="currentPage === totalPages"
              @click="nextPage"
            >
              Next
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
