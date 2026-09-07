<script setup lang="ts">
import type { Category } from '~/types/api'
import { buildCategoryTreeRows, getInheritedCategoryAttributes } from '~/utils/categoryHierarchy'

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
    const fullCategory = await apiFetch<Category>(`/api/categories/${category.id}?inherited=true`)
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
  if (!categoryAssetCounts.value || !categories.value) return 0
  const activeIds = new Set(categories.value.map(category => category.id))
  return categories.value
    .filter(category => !category.parent_id || !activeIds.has(category.parent_id))
    .reduce((sum, category) => sum + (categoryAssetCounts.value?.[category.id] || 0), 0)
})
const uniqueFields = computed(() => {
  const fieldIds = new Set(
    (categories.value || []).flatMap(category =>
      (category.attributes || []).map(attribute => attribute.attribute_id)
    )
  )
  return fieldIds.size
})

const categoryRows = computed(() => buildCategoryTreeRows(categories.value || []).map((row) => {
  const directAttributeIds = new Set((row.category.attributes || []).map(attribute => attribute.attribute_id))
  const inheritedAttributes = getInheritedCategoryAttributes(categories.value || [], row.category.parent_id)
    .filter(attribute => !directAttributeIds.has(attribute.attribute_id))
  return {
    ...row,
    directFieldCount: directAttributeIds.size,
    inheritedFieldCount: inheritedAttributes.length,
    availableFieldCount: directAttributeIds.size + inheritedAttributes.length
  }
}))

const filteredCategoryRows = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return categoryRows.value

  const matches = new Set(categoryRows.value.filter(row =>
    row.category.name.toLowerCase().includes(query)
    || row.category.description?.toLowerCase().includes(query)
    || [...row.ancestors, row.category].some(category => category.name.toLowerCase().includes(query))
  ).map(row => row.category.id))
  const visible = new Set(matches)
  for (const row of categoryRows.value) {
    if (matches.has(row.category.id)) {
      row.ancestors.forEach(ancestor => visible.add(ancestor.id))
    }
  }
  return categoryRows.value.filter(row => visible.has(row.category.id))
})

const categoryGroups = computed(() => {
  const groups = new Map<string, typeof filteredCategoryRows.value>()
  for (const row of filteredCategoryRows.value) {
    const group = groups.get(row.rootId) || []
    group.push(row)
    groups.set(row.rootId, group)
  }
  return [...groups.values()]
})

function getCategoryName(categoryId: string): string {
  return categories.value?.find(category => category.id === categoryId)?.name || 'ancestor category'
}

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
  <div class="space-y-6 pb-6">
    <!-- Page Header -->
    <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Inventory structure
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Categories
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          Group your home inventory and choose the details to record for each kind of belonging.
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

    <LibrarySummary :items="[{ label: 'Categories', value: totalCategories }, { label: 'Assets', value: totalItems }, { label: 'Unique fields', value: uniqueFields }]" />
    <section class="space-y-6">
      <LibraryToolbar
        v-model="search"
        title="Category library"
        placeholder="Search categories"
        :count="filteredCategoryRows.length"
        :total="totalCategories"
      />
      <div class="flex gap-3 rounded-2xl border border-attic-200 bg-attic-50/70 px-4 py-3 dark:border-attic-800 dark:bg-attic-950/20">
        <UIcon
          name="i-lucide-git-branch"
          class="mt-0.5 size-5 shrink-0 text-attic-500"
        />
        <div>
          <p class="text-sm font-bold text-mist-900 dark:text-white">
            Fields flow down the category tree
          </p>
          <p class="mt-0.5 text-xs leading-5 text-muted">
            A child receives the fields of every category above it. Fields added directly to the child can refine that inherited setup.
          </p>
        </div>
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
            class="w-8 h-8 text-muted"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No categories yet
        </h3>
        <p class="text-sm text-muted mb-4 max-w-sm">
          Create your first category to start organizing your assets.
        </p>
        <UButton to="/categories/new">
          Create Category
        </UButton>
      </div>

      <div
        v-else-if="!filteredCategoryRows.length"
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
        role="tree"
        aria-label="Category inheritance tree"
        class="space-y-4"
      >
        <section
          v-for="group in categoryGroups"
          :key="group[0]?.rootId"
          class="attic-panel overflow-hidden rounded-[20px]"
        >
          <article
            v-for="row in group"
            :key="row.category.id"
            role="treeitem"
            :aria-level="row.depth + 1"
            class="relative border-b border-mist-100 px-4 py-4 last:border-b-0 dark:border-mist-700 sm:px-5"
            :class="row.depth === 0 ? 'bg-mist-50/60 dark:bg-mist-800/50' : 'bg-white dark:bg-mist-800'"
          >
            <div
              class="grid gap-4 sm:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)_auto] sm:items-center"
              :style="{ marginLeft: `min(${Math.min(row.depth, 6) * 4}vw, ${Math.min(row.depth, 6) * 28}px)` }"
            >
              <div class="relative flex min-w-0 items-start gap-3">
                <span
                  v-if="row.depth > 0"
                  aria-hidden="true"
                  class="absolute right-full top-0 mr-2 h-6 w-5 rounded-bl-xl border-b-2 border-l-2 border-attic-200 dark:border-attic-700"
                />
                <div
                  class="flex size-10 shrink-0 items-center justify-center rounded-xl"
                  :class="[getCategoryStyle(row.category).bgColor, getCategoryStyle(row.category).textColor]"
                >
                  <UIcon
                    :name="getCategoryStyle(row.category).icon"
                    class="size-5"
                  />
                </div>
                <div class="min-w-0">
                  <p
                    v-if="row.ancestors.length"
                    class="truncate text-[10px] font-extrabold uppercase tracking-[0.12em] text-attic-500"
                  >
                    {{ row.ancestors.map(ancestor => ancestor.name).join(' / ') }}
                  </p>
                  <h2 class="truncate font-extrabold text-mist-950 dark:text-white">
                    {{ row.category.name }}
                  </h2>
                  <p
                    v-if="row.category.description"
                    class="mt-0.5 line-clamp-2 text-xs text-muted"
                  >
                    {{ row.category.description }}
                  </p>
                  <p
                    v-if="row.childCount"
                    class="mt-1 text-[11px] font-semibold text-muted"
                  >
                    {{ row.childCount }} direct {{ row.childCount === 1 ? 'child' : 'children' }}
                  </p>
                </div>
              </div>

              <div class="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  class="rounded-full bg-mist-100 px-2.5 py-1 text-xs font-bold text-mist-700 transition hover:bg-mist-200 dark:bg-mist-700 dark:text-mist-100 dark:hover:bg-mist-600"
                  :aria-label="'View available fields for ' + row.category.name"
                  @click="viewAttributes(row.category)"
                >
                  {{ row.directFieldCount }} own
                </button>
                <template v-if="row.inheritedFieldCount">
                  <UIcon
                    name="i-lucide-plus"
                    class="size-3 text-muted"
                  />
                  <button
                    type="button"
                    class="rounded-full bg-attic-100 px-2.5 py-1 text-xs font-bold text-attic-700 transition hover:bg-attic-200 dark:bg-attic-900/40 dark:text-attic-300"
                    :aria-label="'View inherited fields for ' + row.category.name"
                    @click="viewAttributes(row.category)"
                  >
                    {{ row.inheritedFieldCount }} inherited
                  </button>
                  <span class="text-xs font-bold text-mist-700 dark:text-mist-200">
                    = {{ row.availableFieldCount }} available
                  </span>
                </template>
              </div>

              <div class="flex flex-wrap items-center gap-1 sm:justify-end">
                <UButton
                  :to="'/assets?category_id=' + encodeURIComponent(row.category.id)"
                  icon="i-lucide-box"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                >
                  {{ getAssetCount(row.category.id) }}
                </UButton>
                <UButton
                  :to="{ path: '/categories/new', query: { parent_id: row.category.id } }"
                  icon="i-lucide-git-branch-plus"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  :aria-label="'Add child category under ' + row.category.name"
                />
                <UButton
                  :to="'/categories/' + row.category.id + '/edit'"
                  icon="i-lucide-pencil"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  :aria-label="'Edit ' + row.category.name"
                />
                <UButton
                  icon="i-lucide-trash-2"
                  variant="ghost"
                  color="error"
                  size="xs"
                  :aria-label="'Delete ' + row.category.name"
                  @click="confirmDelete(row.category)"
                />
              </div>
            </div>
          </article>
        </section>
      </div>
    </section>

    <!-- Delete Confirmation Modal -->
    <UModal
      v-model:open="deleteModalOpen"
      title="Delete Category"
      description="Confirm permanent deletion of this category."
    >
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
              <p class="text-sm text-muted mt-2">
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
    <UModal
      v-model:open="attributesModalOpen"
      :title="`${viewingCategory?.name || 'Category'} fields`"
      :description="`${viewingCategory?.attributes?.length || 0} fields available, including inherited fields`"
    >
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
                  {{ viewingCategory?.name }} fields
                </h3>
                <p class="text-sm text-muted">
                  {{ viewingCategory?.attributes?.length || 0 }} available on assets in this category
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
              <p class="text-sm text-muted">
                No fields are available for this category.
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
                <div class="flex min-w-0 items-center gap-3">
                  <div
                    class="size-8 rounded flex items-center justify-center"
                    :class="[getAttributeStyle(attr.attribute?.data_type || 'string').bgColor, getAttributeStyle(attr.attribute?.data_type || 'string').textColor]"
                  >
                    <UIcon
                      :name="getAttributeStyle(attr.attribute?.data_type || 'string').icon"
                      class="w-4 h-4"
                    />
                  </div>
                  <div class="min-w-0">
                    <p class="truncate font-medium text-mist-950 dark:text-white">
                      {{ attr.attribute?.name || 'Unknown' }}
                    </p>
                    <p
                      v-if="attr.inherited"
                      class="truncate text-[11px] font-semibold text-attic-500"
                    >
                      Inherited from {{ getCategoryName(attr.category_id) }}
                    </p>
                    <p
                      v-else
                      class="text-[11px] text-muted"
                    >
                      Defined here
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-xs text-muted bg-mist-200 dark:bg-mist-600 px-2 py-0.5 rounded">
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
