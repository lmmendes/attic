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
const selectedType = ref('all')

const dataTypes = [
  { value: 'all', label: 'All fields', icon: 'i-lucide-layers-3' },
  { value: 'string', label: 'Short text', icon: 'i-lucide-type' },
  { value: 'text', label: 'Long text', icon: 'i-lucide-align-left' },
  { value: 'number', label: 'Number', icon: 'i-lucide-hash' },
  { value: 'boolean', label: 'Yes / No', icon: 'i-lucide-toggle-left' },
  { value: 'date', label: 'Date', icon: 'i-lucide-calendar' }
]

// Pagination
const currentPage = ref(1)
const itemsPerPage = ref(10)

// Filtered attributes
const filteredAttributes = computed(() => {
  if (!attributes.value) return []
  const query = searchQuery.value.trim().toLowerCase()

  return attributes.value.filter((attribute) => {
    const matchesSearch = !query
      || attribute.name.toLowerCase().includes(query)
      || attribute.key.toLowerCase().includes(query)
    const matchesType = selectedType.value === 'all' || attribute.data_type === selectedType.value
    return matchesSearch && matchesType
  })
})

function getTypeCount(type: string): number {
  if (type === 'all') return attributes.value?.length || 0
  return attributes.value?.filter(attribute => attribute.data_type === type).length || 0
}

// Paginated attributes
const paginatedAttributes = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value
  const end = start + itemsPerPage.value
  return filteredAttributes.value.slice(start, end)
})

// Total pages
const totalPages = computed(() => Math.ceil(filteredAttributes.value.length / itemsPerPage.value))

// Reset to page 1 when search changes
watch([searchQuery, selectedType], () => {
  currentPage.value = 1
})

watch(totalPages, (pages) => {
  if (currentPage.value > Math.max(1, pages)) {
    currentPage.value = Math.max(1, pages)
  }
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
function getTypeStyle(type: string): { icon: string, bgColor: string, textColor: string, borderColor: string, label: string } {
  switch (type) {
    case 'string':
      return {
        icon: 'i-lucide-type',
        bgColor: 'bg-slate-100 dark:bg-slate-800',
        textColor: 'text-slate-700 dark:text-slate-300',
        borderColor: 'border-slate-200 dark:border-slate-700',
        label: 'Short text'
      }
    case 'text':
      return {
        icon: 'i-lucide-align-left',
        bgColor: 'bg-indigo-50 dark:bg-indigo-900/30',
        textColor: 'text-indigo-700 dark:text-indigo-300',
        borderColor: 'border-indigo-100 dark:border-indigo-900/50',
        label: 'Long text'
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
function getAttributeIcon(attr: Attribute): { icon: string, bgColor: string, textColor: string } {
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
  <div class="space-y-5 pb-6">
    <!-- Page Header -->
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Schema library
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Fields
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          Reusable details that categories can add to their asset forms.
        </p>
      </div>
      <UButton
        to="/attributes/new"
        icon="i-lucide-plus"
        class="rounded-xl font-bold shadow-primary"
      >
        New field
      </UButton>
    </header>

    <section class="attic-panel rounded-[18px] p-3 sm:p-4">
      <div class="flex flex-col gap-3 2xl:flex-row 2xl:items-center 2xl:justify-between">
        <div class="flex min-w-0 flex-1 gap-2 overflow-x-auto pb-1 2xl:pb-0">
          <button
            v-for="type in dataTypes"
            :key="type.value"
            type="button"
            class="flex shrink-0 items-center gap-2 rounded-xl px-3 py-2 text-xs font-bold transition-colors"
            :class="selectedType === type.value
              ? 'bg-attic-500 text-white shadow-sm'
              : 'bg-mist-50 text-muted hover:bg-attic-50 hover:text-attic-600 dark:bg-mist-700/60 dark:text-mist-300'"
            @click="selectedType = type.value"
          >
            <UIcon
              :name="type.icon"
              class="size-3.5"
            />
            {{ type.label }}
            <span
              class="rounded-md px-1.5 py-0.5 text-[10px]"
              :class="selectedType === type.value ? 'bg-white/15 text-white' : 'bg-white text-muted dark:bg-mist-800'"
            >
              {{ getTypeCount(type.value) }}
            </span>
          </button>
        </div>
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Search name or key"
          class="w-full shrink-0 2xl:w-64"
          size="lg"
        />
      </div>
    </section>

    <!-- Data Table -->
    <section class="attic-panel overflow-hidden rounded-[20px]">
      <div class="flex items-center justify-between border-b border-mist-100 px-4 py-4 dark:border-mist-700 sm:px-5">
        <div>
          <h2 class="font-extrabold text-mist-950 dark:text-white">
            Field library
          </h2>
          <p class="text-xs text-muted">
            Names are for people; keys are used to store the data.
          </p>
        </div>
        <span class="rounded-lg bg-mist-50 px-2.5 py-1 text-xs font-bold text-muted dark:bg-mist-700/60">
          {{ filteredAttributes.length }} shown
        </span>
      </div>
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
        v-else-if="!attributes?.length"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-list"
            class="w-8 h-8 text-muted"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No fields yet
        </h3>
        <p class="text-sm text-muted mb-4 max-w-sm">
          Create your first reusable field, then add it to one or more categories.
        </p>
        <UButton to="/attributes/new">
          Create field
        </UButton>
      </div>

      <!-- No Results -->
      <div
        v-else-if="!filteredAttributes.length"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <UIcon
          name="i-lucide-search-x"
          class="w-12 h-12 text-mist-300 mb-4"
        />
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No results found
        </h3>
        <p class="text-sm text-muted">
          No fields match these filters.
        </p>
        <button
          type="button"
          class="mt-2 text-sm font-semibold text-attic-500 hover:text-attic-600"
          @click="searchQuery = ''; selectedType = 'all'"
        >
          Clear filters
        </button>
      </div>

      <template v-else>
        <div class="divide-y divide-mist-100 dark:divide-mist-700">
          <article
            v-for="attr in paginatedAttributes"
            :key="attr.id"
            class="grid gap-3 px-4 py-4 transition-colors hover:bg-mist-50/60 dark:hover:bg-mist-700/20 sm:grid-cols-[minmax(0,1.25fr)_minmax(150px,.7fr)_auto] sm:items-center sm:px-5"
          >
            <div class="flex min-w-0 items-center gap-3">
              <div
                class="flex size-10 shrink-0 items-center justify-center rounded-xl"
                :class="[getAttributeIcon(attr).bgColor, getAttributeIcon(attr).textColor]"
              >
                <UIcon
                  :name="getAttributeIcon(attr).icon"
                  class="size-4.5"
                />
              </div>
              <div class="min-w-0">
                <h3 class="truncate text-sm font-extrabold text-mist-950 dark:text-white">
                  {{ attr.name }}
                </h3>
                <code class="mt-0.5 block truncate font-mono text-xs text-muted">
                  {{ attr.key }}
                </code>
              </div>
            </div>

            <div class="flex items-center justify-between gap-3 pl-[52px] sm:justify-start sm:pl-0">
              <span
                class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-bold"
                :class="[getTypeStyle(attr.data_type).bgColor, getTypeStyle(attr.data_type).textColor, getTypeStyle(attr.data_type).borderColor]"
              >
                <UIcon
                  :name="getTypeStyle(attr.data_type).icon"
                  class="size-3.5"
                />
                {{ getTypeStyle(attr.data_type).label }}
              </span>
            </div>

            <div class="flex items-center justify-end gap-1 border-t border-mist-100 pt-3 dark:border-mist-700 sm:border-0 sm:pt-0">
              <UButton
                :to="`/attributes/${attr.id}/edit`"
                variant="soft"
                size="sm"
                icon="i-lucide-pencil"
              >
                Edit
              </UButton>
              <button
                type="button"
                class="flex size-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
                :aria-label="`Delete ${attr.name}`"
                @click="confirmDelete(attr)"
              >
                <UIcon
                  name="i-lucide-trash-2"
                  class="size-4"
                />
              </button>
            </div>
          </article>
        </div>

        <!-- Footer with Pagination -->
        <div class="flex flex-col gap-3 border-t border-mist-100 bg-mist-50/50 px-4 py-3 dark:border-mist-700 dark:bg-mist-700/20 sm:flex-row sm:items-center sm:justify-between sm:px-5">
          <p class="text-xs text-muted">
            Showing {{ (currentPage - 1) * itemsPerPage + 1 }}–{{ Math.min(currentPage * itemsPerPage, filteredAttributes.length) }} of {{ filteredAttributes.length }} fields
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
            <span class="text-xs text-muted px-2">
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
    </section>

    <!-- Delete Confirmation Modal -->
    <UModal
      v-model:open="deleteModalOpen"
      title="Delete field"
      description="Confirm deletion of this field and review the impact on categories that use it."
    >
      <template #content>
        <div class="max-w-md rounded-[20px] bg-white p-6 shadow-xl dark:bg-mist-800">
          <div class="flex items-start gap-4">
            <div class="p-3 bg-red-100 dark:bg-red-900/30 rounded-full">
              <UIcon
                name="i-lucide-alert-triangle"
                class="w-6 h-6 text-red-600 dark:text-red-400"
              />
            </div>
            <div class="flex-1">
              <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                Delete field
              </h3>
              <p class="text-sm text-muted mt-2">
                Are you sure you want to delete <strong>{{ attributeToDelete?.name }}</strong>? Categories using this field may be affected.
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
