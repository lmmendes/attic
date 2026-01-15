<script setup lang="ts">
import type { Attribute } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: attributes, refresh, status } = useApi<Attribute[]>('/api/attributes')

// Search
const searchQuery = ref('')

// Pagination
const currentPage = ref(1)
const itemsPerPage = ref(10)

// Filtered attributes
const filteredAttributes = computed(() => {
  if (!attributes.value) return []
  if (!searchQuery.value.trim()) return attributes.value
  const query = searchQuery.value.toLowerCase()
  return attributes.value.filter(
    a => a.name.toLowerCase().includes(query) || a.key.toLowerCase().includes(query)
  )
})

// Paginated attributes
const paginatedAttributes = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value
  const end = start + itemsPerPage.value
  return filteredAttributes.value.slice(start, end)
})

// Total pages
const totalPages = computed(() => Math.ceil(filteredAttributes.value.length / itemsPerPage.value))

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
const attributeToDelete = ref<Attribute | null>(null)

function confirmDelete(attribute: Attribute) {
  attributeToDelete.value = attribute
  deleteModalOpen.value = true
}

async function deleteAttribute() {
  if (!attributeToDelete.value) return

  try {
    await apiFetch(`/api/attributes/${attributeToDelete.value.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Attribute deleted', color: 'success' })
    deleteModalOpen.value = false
    attributeToDelete.value = null
    refresh()
  } catch {
    toast.add({ title: 'Failed to delete attribute', color: 'error' })
  }
}

// Get style for data type
function getTypeStyle(type: string): { icon: string; bgColor: string; textColor: string; borderColor: string; label: string } {
  switch (type) {
    case 'string':
      return {
        icon: 'i-lucide-type',
        bgColor: 'bg-slate-100 dark:bg-slate-800',
        textColor: 'text-slate-700 dark:text-slate-300',
        borderColor: 'border-slate-200 dark:border-slate-700',
        label: 'String'
      }
    case 'text':
      return {
        icon: 'i-lucide-align-left',
        bgColor: 'bg-indigo-50 dark:bg-indigo-900/30',
        textColor: 'text-indigo-700 dark:text-indigo-300',
        borderColor: 'border-indigo-100 dark:border-indigo-900/50',
        label: 'Text'
      }
    case 'number':
      return {
        icon: 'i-lucide-hash',
        bgColor: 'bg-orange-50 dark:bg-orange-900/30',
        textColor: 'text-orange-700 dark:text-orange-300',
        borderColor: 'border-orange-100 dark:border-orange-900/50',
        label: 'Number'
      }
    case 'boolean':
      return {
        icon: 'i-lucide-toggle-left',
        bgColor: 'bg-green-50 dark:bg-green-900/30',
        textColor: 'text-green-700 dark:text-green-300',
        borderColor: 'border-green-100 dark:border-green-900/50',
        label: 'Boolean'
      }
    case 'date':
      return {
        icon: 'i-lucide-calendar',
        bgColor: 'bg-purple-50 dark:bg-purple-900/30',
        textColor: 'text-purple-700 dark:text-purple-300',
        borderColor: 'border-purple-100 dark:border-purple-900/50',
        label: 'Date'
      }
    default:
      return {
        icon: 'i-lucide-circle',
        bgColor: 'bg-gray-100 dark:bg-gray-800',
        textColor: 'text-gray-700 dark:text-gray-300',
        borderColor: 'border-gray-200 dark:border-gray-700',
        label: type
      }
  }
}

// Get icon for attribute based on name or type
function getAttributeIcon(attr: Attribute): { icon: string; bgColor: string; textColor: string } {
  const name = attr.name.toLowerCase()

  if (name.includes('isbn') || name.includes('serial') || name.includes('code')) {
    return { icon: 'i-lucide-tag', bgColor: 'bg-blue-50 dark:bg-blue-900/20', textColor: 'text-blue-600 dark:text-blue-400' }
  }
  if (name.includes('weight') || name.includes('dimension') || name.includes('size')) {
    return { icon: 'i-lucide-scale', bgColor: 'bg-orange-50 dark:bg-orange-900/20', textColor: 'text-orange-600 dark:text-orange-400' }
  }
  if (name.includes('insur') || name.includes('verified') || name.includes('warranty')) {
    return { icon: 'i-lucide-shield-check', bgColor: 'bg-green-50 dark:bg-green-900/20', textColor: 'text-green-600 dark:text-green-400' }
  }
  if (name.includes('year') || name.includes('date') || name.includes('purchased')) {
    return { icon: 'i-lucide-calendar', bgColor: 'bg-purple-50 dark:bg-purple-900/20', textColor: 'text-purple-600 dark:text-purple-400' }
  }
  if (name.includes('color') || name.includes('colour')) {
    return { icon: 'i-lucide-palette', bgColor: 'bg-pink-50 dark:bg-pink-900/20', textColor: 'text-pink-600 dark:text-pink-400' }
  }
  if (name.includes('brand') || name.includes('manufacturer')) {
    return { icon: 'i-lucide-building-2', bgColor: 'bg-cyan-50 dark:bg-cyan-900/20', textColor: 'text-cyan-600 dark:text-cyan-400' }
  }
  if (name.includes('model')) {
    return { icon: 'i-lucide-box', bgColor: 'bg-amber-50 dark:bg-amber-900/20', textColor: 'text-amber-600 dark:text-amber-400' }
  }

  // Default based on type
  const typeStyle = getTypeStyle(attr.data_type)
  return { icon: typeStyle.icon, bgColor: typeStyle.bgColor, textColor: typeStyle.textColor }
}
</script>

<template>
  <div class="space-y-8">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          Custom Attributes
        </h1>
        <p class="text-mist-500">
          Define the specific data points you want to track for your home inventory.
        </p>
      </div>
      <UButton
        to="/attributes/new"
        icon="i-lucide-plus"
        class="h-11 px-6 font-bold shadow-lg shadow-attic-500/20"
      >
        Add Attribute
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
          placeholder="Search attributes..."
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
        v-else-if="!filteredAttributes.length && !searchQuery"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-list"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No attributes yet
        </h3>
        <p class="text-sm text-mist-500 mb-4 max-w-sm">
          Create your first attribute to start defining custom data points for your assets.
        </p>
        <UButton to="/attributes/new">
          Create Attribute
        </UButton>
      </div>

      <!-- No Results -->
      <div
        v-else-if="!filteredAttributes.length && searchQuery"
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
          No attributes match "{{ searchQuery }}"
        </p>
      </div>

      <!-- Table -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[600px] border-collapse">
            <thead class="bg-mist-50/50 dark:bg-mist-700/30 border-b border-mist-100 dark:border-mist-700">
              <tr>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Attribute Name
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Data Type
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Key
                </th>
                <th class="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-mist-500">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-mist-100 dark:divide-mist-700">
              <tr
                v-for="attr in paginatedAttributes"
                :key="attr.id"
                class="group hover:bg-mist-50/50 dark:hover:bg-mist-700/30 transition-colors"
              >
                <!-- Attribute Name with Icon -->
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <div
                      class="size-9 rounded-full flex items-center justify-center"
                      :class="[getAttributeIcon(attr).bgColor, getAttributeIcon(attr).textColor]"
                    >
                      <UIcon
                        :name="getAttributeIcon(attr).icon"
                        class="w-4 h-4"
                      />
                    </div>
                    <span class="text-mist-950 dark:text-white text-sm font-semibold">
                      {{ attr.name }}
                    </span>
                  </div>
                </td>

                <!-- Data Type Badge -->
                <td class="px-6 py-4">
                  <span
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-semibold border"
                    :class="[getTypeStyle(attr.data_type).bgColor, getTypeStyle(attr.data_type).textColor, getTypeStyle(attr.data_type).borderColor]"
                  >
                    <UIcon
                      :name="getTypeStyle(attr.data_type).icon"
                      class="w-3.5 h-3.5"
                    />
                    {{ getTypeStyle(attr.data_type).label }}
                  </span>
                </td>

                <!-- Key -->
                <td class="px-6 py-4">
                  <code class="text-xs bg-mist-100 dark:bg-mist-700 text-mist-600 dark:text-mist-300 px-2 py-1 rounded font-mono">
                    {{ attr.key }}
                  </code>
                </td>

                <!-- Actions -->
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <NuxtLink
                      :to="`/attributes/${attr.id}/edit`"
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
                      @click="confirmDelete(attr)"
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
            Showing {{ (currentPage - 1) * itemsPerPage + 1 }}-{{ Math.min(currentPage * itemsPerPage, filteredAttributes.length) }} of {{ filteredAttributes.length }} attributes
            <span v-if="searchQuery && attributes?.length !== filteredAttributes.length">
              (filtered from {{ attributes?.length || 0 }})
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
                Delete Attribute
              </h3>
              <p class="text-sm text-mist-500 mt-2">
                Are you sure you want to delete <strong>{{ attributeToDelete?.name }}</strong>? This may affect categories using this attribute.
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
              @click="deleteAttribute"
            >
              Delete
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
