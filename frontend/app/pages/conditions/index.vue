<script setup lang="ts">
import type { Condition } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: conditions, refresh, status } = useApi<Condition[]>('/api/conditions')

// Search
const searchQuery = ref('')

// Pagination
const currentPage = ref(1)
const itemsPerPage = ref(10)

// Filtered conditions
const filteredConditions = computed(() => {
  if (!conditions.value) return []
  if (!searchQuery.value.trim()) return conditions.value
  const query = searchQuery.value.toLowerCase()
  return conditions.value.filter(
    c => c.label.toLowerCase().includes(query) || c.code.toLowerCase().includes(query)
  )
})

// Paginated conditions
const paginatedConditions = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value
  const end = start + itemsPerPage.value
  return filteredConditions.value.slice(start, end)
})

// Total pages
const totalPages = computed(() => Math.ceil(filteredConditions.value.length / itemsPerPage.value))

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

// Delete confirmation modal
const deleteModalOpen = ref(false)
const conditionToDelete = ref<Condition | null>(null)

function confirmDelete(condition: Condition) {
  conditionToDelete.value = condition
  deleteModalOpen.value = true
}

async function deleteCondition() {
  if (!conditionToDelete.value) return

  try {
    await apiFetch(`/api/conditions/${conditionToDelete.value.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Condition deleted', color: 'success' })
    deleteModalOpen.value = false
    conditionToDelete.value = null
    refresh()
  } catch {
    toast.add({ title: 'Failed to delete condition', color: 'error' })
  }
}

// Get style for condition based on label/code
function getConditionStyle(condition: Condition): { icon: string; bgColor: string; textColor: string; badgeBg: string; badgeText: string } {
  const label = condition.label.toLowerCase()
  const code = condition.code.toLowerCase()

  if (label.includes('new') || label.includes('mint') || label.includes('excellent') || code.includes('new')) {
    return {
      icon: 'i-lucide-sparkles',
      bgColor: 'bg-emerald-50 dark:bg-emerald-900/20',
      textColor: 'text-emerald-600 dark:text-emerald-400',
      badgeBg: 'bg-emerald-100 dark:bg-emerald-900/30',
      badgeText: 'text-emerald-700 dark:text-emerald-300'
    }
  }
  if (label.includes('good') || label.includes('great') || code.includes('good')) {
    return {
      icon: 'i-lucide-thumbs-up',
      bgColor: 'bg-blue-50 dark:bg-blue-900/20',
      textColor: 'text-blue-600 dark:text-blue-400',
      badgeBg: 'bg-blue-100 dark:bg-blue-900/30',
      badgeText: 'text-blue-700 dark:text-blue-300'
    }
  }
  if (label.includes('fair') || label.includes('average') || code.includes('fair')) {
    return {
      icon: 'i-lucide-minus',
      bgColor: 'bg-amber-50 dark:bg-amber-900/20',
      textColor: 'text-amber-600 dark:text-amber-400',
      badgeBg: 'bg-amber-100 dark:bg-amber-900/30',
      badgeText: 'text-amber-700 dark:text-amber-300'
    }
  }
  if (label.includes('poor') || label.includes('bad') || label.includes('damaged') || code.includes('poor')) {
    return {
      icon: 'i-lucide-alert-triangle',
      bgColor: 'bg-orange-50 dark:bg-orange-900/20',
      textColor: 'text-orange-600 dark:text-orange-400',
      badgeBg: 'bg-orange-100 dark:bg-orange-900/30',
      badgeText: 'text-orange-700 dark:text-orange-300'
    }
  }
  if (label.includes('broken') || label.includes('repair') || label.includes('needs') || code.includes('broken')) {
    return {
      icon: 'i-lucide-wrench',
      bgColor: 'bg-red-50 dark:bg-red-900/20',
      textColor: 'text-red-600 dark:text-red-400',
      badgeBg: 'bg-red-100 dark:bg-red-900/30',
      badgeText: 'text-red-700 dark:text-red-300'
    }
  }

  // Default style
  return {
    icon: 'i-lucide-circle',
    bgColor: 'bg-slate-50 dark:bg-slate-900/20',
    textColor: 'text-slate-600 dark:text-slate-400',
    badgeBg: 'bg-slate-100 dark:bg-slate-900/30',
    badgeText: 'text-slate-700 dark:text-slate-300'
  }
}
</script>

<template>
  <div class="space-y-8">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          Conditions
        </h1>
        <p class="text-mist-500">
          Define the physical state or quality levels for your assets.
        </p>
      </div>
      <UButton
        to="/conditions/new"
        icon="i-lucide-plus"
        class="h-11 px-6 font-bold shadow-lg shadow-attic-500/20"
      >
        Add Condition
      </UButton>
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
          placeholder="Search conditions..."
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
        v-else-if="!filteredConditions.length && !searchQuery"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-heart-pulse"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No conditions yet
        </h3>
        <p class="text-sm text-mist-500 mb-4 max-w-sm">
          Create your first condition to start tracking the quality of your assets.
        </p>
        <UButton to="/conditions/new">
          Create Condition
        </UButton>
      </div>

      <!-- No Results -->
      <div
        v-else-if="!filteredConditions.length && searchQuery"
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
          No conditions match "{{ searchQuery }}"
        </p>
      </div>

      <!-- Table -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[600px] border-collapse">
            <thead class="bg-mist-50/50 dark:bg-mist-700/30 border-b border-mist-100 dark:border-mist-700">
              <tr>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Condition
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Code
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Description
                </th>
                <th class="px-6 py-4 text-center text-xs font-bold uppercase tracking-wider text-mist-500">
                  Order
                </th>
                <th class="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-mist-500">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-mist-100 dark:divide-mist-700">
              <tr
                v-for="condition in paginatedConditions"
                :key="condition.id"
                class="group hover:bg-mist-50/50 dark:hover:bg-mist-700/30 transition-colors"
              >
                <!-- Condition Name with Icon -->
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <div
                      class="size-9 rounded-full flex items-center justify-center"
                      :class="[getConditionStyle(condition).bgColor, getConditionStyle(condition).textColor]"
                    >
                      <UIcon
                        :name="getConditionStyle(condition).icon"
                        class="w-4 h-4"
                      />
                    </div>
                    <span class="text-mist-950 dark:text-white text-sm font-semibold">
                      {{ condition.label }}
                    </span>
                  </div>
                </td>

                <!-- Code Badge -->
                <td class="px-6 py-4">
                  <code class="text-xs bg-mist-100 dark:bg-mist-700 text-mist-600 dark:text-mist-300 px-2 py-1 rounded font-mono">
                    {{ condition.code }}
                  </code>
                </td>

                <!-- Description -->
                <td class="px-6 py-4">
                  <span
                    v-if="condition.description"
                    class="text-sm text-mist-500 line-clamp-1"
                  >
                    {{ condition.description }}
                  </span>
                  <span
                    v-else
                    class="text-sm text-mist-300 dark:text-mist-600 italic"
                  >
                    No description
                  </span>
                </td>

                <!-- Sort Order -->
                <td class="px-6 py-4 text-center">
                  <span
                    class="inline-flex items-center justify-center size-7 rounded-full text-xs font-bold"
                    :class="[getConditionStyle(condition).badgeBg, getConditionStyle(condition).badgeText]"
                  >
                    {{ condition.sort_order }}
                  </span>
                </td>

                <!-- Actions -->
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <NuxtLink
                      :to="`/conditions/${condition.id}/edit`"
                      class="size-8 rounded flex items-center justify-center text-mist-400 hover:text-attic-500 hover:bg-attic-500/10 transition-colors"
                      title="Edit"
                    >
                      <UIcon
                        name="i-lucide-edit"
                        class="w-4 h-4"
                      />
                    </NuxtLink>
                    <button
                      class="size-8 rounded flex items-center justify-center text-mist-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                      title="Delete"
                      @click="confirmDelete(condition)"
                    >
                      <UIcon
                        name="i-lucide-trash-2"
                        class="w-4 h-4"
                      />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Footer with Pagination -->
        <div class="px-6 py-3 border-t border-mist-100 dark:border-mist-700 bg-mist-50/50 dark:bg-mist-700/20 flex items-center justify-between">
          <p class="text-xs text-mist-500">
            Showing {{ (currentPage - 1) * itemsPerPage + 1 }}-{{ Math.min(currentPage * itemsPerPage, filteredConditions.length) }} of {{ filteredConditions.length }} conditions
            <span v-if="searchQuery && conditions?.length !== filteredConditions.length">
              (filtered from {{ conditions?.length || 0 }})
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

    <!-- Delete Confirmation Modal -->
    <UModal v-model:open="deleteModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl p-6 max-w-md">
          <div class="flex items-start gap-4">
            <div class="p-3 bg-red-100 dark:bg-red-900/30 rounded-full">
              <UIcon
                name="i-lucide-alert-triangle"
                class="w-6 h-6 text-red-600 dark:text-red-400"
              />
            </div>
            <div class="flex-1">
              <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                Delete Condition
              </h3>
              <p class="text-sm text-mist-500 mt-2">
                Are you sure you want to delete <strong>{{ conditionToDelete?.label }}</strong>? This may affect assets using this condition.
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
              @click="deleteCondition"
            >
              Delete
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
