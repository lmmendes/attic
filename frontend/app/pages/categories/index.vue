<script setup lang="ts">
import type { Category } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: categories, refresh, status } = useApi<Category[]>('/api/categories')

// Fetch asset counts per category (endpoint may not exist yet, so we handle gracefully)
const { data: categoryAssetCounts } = useApi<Record<string, number>>('/api/categories/asset-counts')

// Delete confirmation modal
const deleteModalOpen = ref(false)
const categoryToDelete = ref<Category | null>(null)

// Attributes modal
const attributesModalOpen = ref(false)
const viewingCategory = ref<Category | null>(null)
const search = ref('')

async function viewAttributes(category: Category) {
  try {
    const fullCategory = await apiFetch<Category>(`/api/categories/${category.id}`)
    viewingCategory.value = fullCategory
    attributesModalOpen.value = true
  } catch {
    toast.add({ title: 'Failed to load category attributes', color: 'error' })
  }
}

function confirmDelete(category: Category) {
  categoryToDelete.value = category
  deleteModalOpen.value = true
}

async function deleteCategory() {
  if (!categoryToDelete.value) return

  try {
    await apiFetch(`/api/categories/${categoryToDelete.value.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Category deleted', color: 'success' })
    deleteModalOpen.value = false
    categoryToDelete.value = null
    refresh()
  } catch {
    toast.add({ title: 'Failed to delete category', color: 'error' })
  }
}

// Stats
const totalCategories = computed(() => categories.value?.length || 0)
const totalItems = computed(() => {
  if (!categoryAssetCounts.value) return 0
  return Object.values(categoryAssetCounts.value).reduce((sum, count) => sum + count, 0)
})
const uniqueFields = computed(() => {
  const fieldIds = new Set(
    (categories.value || []).flatMap(category =>
      (category.attributes || []).map(attribute => attribute.attribute_id)
    )
  )
  return fieldIds.size
})

const filteredCategories = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return categories.value || []

  return (categories.value || []).filter(category =>
    category.name.toLowerCase().includes(query)
    || category.description?.toLowerCase().includes(query)
  )
})

// Get asset count for a category
function getAssetCount(categoryId: string): number {
  return categoryAssetCounts.value?.[categoryId] || 0
}

// Get icon and color based on category name or icon field
function getCategoryStyle(category: Category): { icon: string, bgColor: string, textColor: string } {
  // Use saved icon if available
  if (category.icon) {
    return { icon: category.icon, bgColor: 'bg-attic-100 dark:bg-attic-900/30', textColor: 'text-attic-600 dark:text-attic-400' }
  }

  const name = category.name.toLowerCase()

  if (name.includes('electronic') || name.includes('computer') || name.includes('tech')) {
    return { icon: 'i-lucide-laptop', bgColor: 'bg-orange-100 dark:bg-orange-900/30', textColor: 'text-orange-600 dark:text-orange-400' }
  }
  if (name.includes('book') || name.includes('library')) {
    return { icon: 'i-lucide-book-open', bgColor: 'bg-blue-100 dark:bg-blue-900/30', textColor: 'text-blue-600 dark:text-blue-400' }
  }
  if (name.includes('movie') || name.includes('blu-ray') || name.includes('dvd') || name.includes('media')) {
    return { icon: 'i-lucide-film', bgColor: 'bg-purple-100 dark:bg-purple-900/30', textColor: 'text-purple-600 dark:text-purple-400' }
  }
  if (name.includes('furniture') || name.includes('chair') || name.includes('table')) {
    return { icon: 'i-lucide-armchair', bgColor: 'bg-amber-100 dark:bg-amber-900/30', textColor: 'text-amber-600 dark:text-amber-400' }
  }
  if (name.includes('kitchen') || name.includes('appliance')) {
    return { icon: 'i-lucide-chef-hat', bgColor: 'bg-red-100 dark:bg-red-900/30', textColor: 'text-red-600 dark:text-red-400' }
  }
  if (name.includes('clothing') || name.includes('apparel') || name.includes('fashion')) {
    return { icon: 'i-lucide-shirt', bgColor: 'bg-pink-100 dark:bg-pink-900/30', textColor: 'text-pink-600 dark:text-pink-400' }
  }
  if (name.includes('tool') || name.includes('hardware')) {
    return { icon: 'i-lucide-wrench', bgColor: 'bg-slate-100 dark:bg-slate-900/30', textColor: 'text-slate-600 dark:text-slate-400' }
  }
  if (name.includes('sport') || name.includes('fitness') || name.includes('exercise')) {
    return { icon: 'i-lucide-dumbbell', bgColor: 'bg-green-100 dark:bg-green-900/30', textColor: 'text-green-600 dark:text-green-400' }
  }
  if (name.includes('art') || name.includes('decor') || name.includes('decoration')) {
    return { icon: 'i-lucide-palette', bgColor: 'bg-indigo-100 dark:bg-indigo-900/30', textColor: 'text-indigo-600 dark:text-indigo-400' }
  }
  if (name.includes('game') || name.includes('toy')) {
    return { icon: 'i-lucide-gamepad-2', bgColor: 'bg-cyan-100 dark:bg-cyan-900/30', textColor: 'text-cyan-600 dark:text-cyan-400' }
  }

  // Default
  return { icon: 'i-lucide-tag', bgColor: 'bg-attic-100 dark:bg-attic-900/30', textColor: 'text-attic-600 dark:text-attic-400' }
}

// Get attribute style by type
function getAttributeStyle(dataType: string): { icon: string, bgColor: string, textColor: string } {
  switch (dataType) {
    case 'string':
      return { icon: 'i-lucide-type', bgColor: 'bg-blue-50 dark:bg-blue-900/20', textColor: 'text-blue-600 dark:text-blue-400' }
    case 'number':
      return { icon: 'i-lucide-hash', bgColor: 'bg-amber-50 dark:bg-amber-900/20', textColor: 'text-amber-600 dark:text-amber-400' }
    case 'boolean':
      return { icon: 'i-lucide-toggle-left', bgColor: 'bg-green-50 dark:bg-green-900/20', textColor: 'text-green-600 dark:text-green-400' }
    case 'text':
      return { icon: 'i-lucide-align-left', bgColor: 'bg-purple-50 dark:bg-purple-900/20', textColor: 'text-purple-600 dark:text-purple-400' }
    case 'date':
      return { icon: 'i-lucide-calendar', bgColor: 'bg-red-50 dark:bg-red-900/20', textColor: 'text-red-600 dark:text-red-400' }
    default:
      return { icon: 'i-lucide-circle', bgColor: 'bg-gray-50 dark:bg-gray-900/20', textColor: 'text-gray-600 dark:text-gray-400' }
  }
}
</script>

<template>
  <div class="space-y-5 pb-6">
    <!-- Page Header -->
    <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Inventory structure
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Categories
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-mist-500">
          Define how your collection is grouped and which details each asset records.
        </p>
      </div>
      <UButton
        to="/categories/new"
        icon="i-lucide-plus"
        class="rounded-xl font-bold shadow-primary"
      >
        New category
      </UButton>
    </div>

    <!-- Stats Overview -->
    <div class="attic-panel grid grid-cols-3 divide-x divide-mist-100 rounded-[18px] px-2 py-3 dark:divide-mist-700 sm:px-4">
      <!-- Total Categories -->
      <div class="px-3 sm:px-5">
        <p class="text-[10px] font-extrabold uppercase tracking-wider text-mist-400">
          Categories
        </p>
        <p class="mt-1 text-xl font-black text-mist-950 dark:text-white">
          {{ totalCategories }}
        </p>
      </div>

      <!-- Total Items Tracked -->
      <div class="px-3 sm:px-5">
        <p class="text-[10px] font-extrabold uppercase tracking-wider text-mist-400">
          Assets
        </p>
        <p class="mt-1 text-xl font-black text-mist-950 dark:text-white">
          {{ totalItems }}
        </p>
      </div>

      <!-- Unique Fields -->
      <div class="px-3 sm:px-5">
        <p class="text-[10px] font-extrabold uppercase tracking-wider text-mist-400">
          Unique fields
        </p>
        <p class="mt-1 text-xl font-black text-mist-950 dark:text-white">
          {{ uniqueFields }}
        </p>
      </div>
    </div>

    <!-- Main Category Table -->
    <div class="attic-panel overflow-hidden rounded-[20px]">
      <div class="flex flex-col gap-3 border-b border-mist-100 px-4 py-4 dark:border-mist-700 sm:flex-row sm:items-center sm:justify-between sm:px-5">
        <div>
          <h2 class="font-extrabold text-mist-950 dark:text-white">
            Category library
          </h2>
          <p class="text-xs text-mist-500">
            Open a schema, browse its assets, or adjust its fields.
          </p>
        </div>
        <UInput
          v-model="search"
          icon="i-lucide-search"
          placeholder="Search categories"
          class="w-full sm:w-64"
          size="lg"
        />
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
        v-else-if="!categories?.length"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-tag"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No categories yet
        </h3>
        <p class="text-sm text-mist-500 mb-4 max-w-sm">
          Create your first category to start organizing your assets.
        </p>
        <UButton to="/categories/new">
          Create Category
        </UButton>
      </div>

      <div
        v-else-if="!filteredCategories.length"
        class="px-4 py-14 text-center"
      >
        <UIcon
          name="i-lucide-search-x"
          class="mx-auto mb-3 size-7 text-mist-300"
        />
        <p class="font-bold text-mist-700 dark:text-mist-200">
          No matching categories
        </p>
        <button
          type="button"
          class="mt-1 text-sm font-semibold text-attic-500 hover:text-attic-600"
          @click="search = ''"
        >
          Clear search
        </button>
      </div>

      <div
        v-else
        class="grid gap-px bg-mist-100 dark:bg-mist-700 sm:grid-cols-2 xl:grid-cols-3"
      >
        <article
          v-for="category in filteredCategories"
          :key="category.id"
          class="group flex min-h-48 flex-col bg-white p-5 transition-colors hover:bg-mist-50/70 dark:bg-mist-800 dark:hover:bg-mist-800/70"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3">
              <div
                class="flex size-10 shrink-0 items-center justify-center rounded-xl"
                :class="[getCategoryStyle(category).bgColor, getCategoryStyle(category).textColor]"
              >
                <UIcon
                  :name="getCategoryStyle(category).icon"
                  class="size-5"
                />
              </div>
              <div class="min-w-0">
                <h3 class="truncate font-extrabold text-mist-950 dark:text-white">
                  {{ category.name }}
                </h3>
                <p class="text-xs text-mist-400">
                  {{ getAssetCount(category.id) }} {{ getAssetCount(category.id) === 1 ? 'asset' : 'assets' }}
                </p>
              </div>
            </div>
            <button
              type="button"
              class="rounded-lg p-2 text-mist-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
              :aria-label="`Delete ${category.name}`"
              @click="confirmDelete(category)"
            >
              <UIcon
                name="i-lucide-trash-2"
                class="size-4"
              />
            </button>
          </div>

          <p class="mt-4 line-clamp-2 min-h-10 text-sm leading-5 text-mist-500">
            {{ category.description || 'No description yet.' }}
          </p>

          <div class="mt-auto flex items-center justify-between gap-3 pt-4">
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-lg bg-attic-50 px-2.5 py-1.5 text-xs font-bold text-attic-600 transition-colors hover:bg-attic-100 dark:bg-attic-500/10 dark:text-attic-300"
              @click="viewAttributes(category)"
            >
              <UIcon
                name="i-lucide-list-checks"
                class="size-3.5"
              />
              {{ category.attributes?.length || 0 }} {{ (category.attributes?.length || 0) === 1 ? 'field' : 'fields' }}
            </button>
            <div class="flex items-center gap-1">
              <UButton
                :to="`/assets?category_id=${category.id}`"
                variant="ghost"
                color="neutral"
                size="sm"
                icon="i-lucide-package-open"
              >
                Assets
              </UButton>
              <UButton
                :to="`/categories/${category.id}/edit`"
                variant="soft"
                size="sm"
                icon="i-lucide-pencil"
              >
                Edit
              </UButton>
            </div>
          </div>
        </article>
      </div>
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
                Delete Category
              </h3>
              <p class="text-sm text-mist-500 mt-2">
                Are you sure you want to delete <strong>{{ categoryToDelete?.name }}</strong>? This action cannot be undone.
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
              @click="deleteCategory"
            >
              Delete
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Attributes View Modal -->
    <UModal v-model:open="attributesModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl max-w-md w-full">
          <div class="p-6 border-b border-mist-100 dark:border-mist-700">
            <div class="flex items-center gap-3">
              <div
                v-if="viewingCategory"
                class="size-10 rounded-lg flex items-center justify-center"
                :class="[getCategoryStyle(viewingCategory).bgColor, getCategoryStyle(viewingCategory).textColor]"
              >
                <UIcon
                  :name="getCategoryStyle(viewingCategory).icon"
                  class="w-5 h-5"
                />
              </div>
              <div>
                <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                  {{ viewingCategory?.name }} Attributes
                </h3>
                <p class="text-sm text-mist-500">
                  {{ viewingCategory?.attributes?.length || 0 }} attributes defined
                </p>
              </div>
            </div>
          </div>

          <div class="p-6">
            <div
              v-if="!viewingCategory?.attributes?.length"
              class="text-center py-6"
            >
              <UIcon
                name="i-lucide-list"
                class="w-10 h-10 text-mist-300 mx-auto mb-3"
              />
              <p class="text-sm text-mist-500">
                No attributes defined for this category.
              </p>
            </div>

            <div
              v-else
              class="space-y-2"
            >
              <div
                v-for="attr in viewingCategory.attributes"
                :key="attr.id"
                class="flex items-center justify-between p-3 bg-mist-50 dark:bg-mist-700/50 rounded-lg"
              >
                <div class="flex items-center gap-3">
                  <div
                    class="size-8 rounded flex items-center justify-center"
                    :class="[getAttributeStyle(attr.attribute?.data_type || 'string').bgColor, getAttributeStyle(attr.attribute?.data_type || 'string').textColor]"
                  >
                    <UIcon
                      :name="getAttributeStyle(attr.attribute?.data_type || 'string').icon"
                      class="w-4 h-4"
                    />
                  </div>
                  <span class="font-medium text-mist-950 dark:text-white">
                    {{ attr.attribute?.name || 'Unknown' }}
                  </span>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-xs text-mist-500 bg-mist-200 dark:bg-mist-600 px-2 py-0.5 rounded">
                    {{ attr.attribute?.data_type || 'string' }}
                  </span>
                  <span
                    v-if="attr.required"
                    class="text-xs font-bold text-red-500 bg-red-500/10 px-2 py-0.5 rounded"
                  >
                    Required
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div class="p-6 border-t border-mist-100 dark:border-mist-700 flex justify-between">
            <UButton
              v-if="viewingCategory"
              :to="`/categories/${viewingCategory.id}/edit`"
              variant="soft"
              @click="attributesModalOpen = false"
            >
              Edit Attributes
            </UButton>
            <UButton
              variant="ghost"
              color="neutral"
              @click="attributesModalOpen = false"
            >
              Close
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
