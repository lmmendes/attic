<script setup lang="ts">
import type { Category, Attribute } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: categories, refresh, status } = useApi<Category[]>('/api/categories')
const { data: attributes } = useApi<Attribute[]>('/api/attributes')

// Fetch asset counts per category (endpoint may not exist yet, so we handle gracefully)
const { data: categoryAssetCounts } = useApi<Record<string, number>>('/api/categories/asset-counts')

interface AttributeSelection {
  attribute_id: string
  required: boolean
  sort_order: number
}

const modalOpen = ref(false)
const editingCategory = ref<Category | null>(null)
const form = reactive({
  name: '',
  description: '',
  parent_id: undefined as string | undefined,
  attributes: [] as AttributeSelection[]
})

// Delete confirmation modal
const deleteModalOpen = ref(false)
const categoryToDelete = ref<Category | null>(null)

// Attributes modal
const attributesModalOpen = ref(false)
const viewingCategory = ref<Category | null>(null)

function openCreateModal() {
  editingCategory.value = null
  form.name = ''
  form.description = ''
  form.parent_id = undefined
  form.attributes = []
  modalOpen.value = true
}

async function openEditModal(category: Category) {
  // Fetch category with attributes
  try {
    const fullCategory = await apiFetch<Category>(`/api/categories/${category.id}`)
    editingCategory.value = fullCategory
    form.name = fullCategory.name
    form.description = fullCategory.description || ''
    form.parent_id = fullCategory.parent_id
    form.attributes = (fullCategory.attributes || []).map((ca, index) => ({
      attribute_id: ca.attribute_id,
      required: ca.required,
      sort_order: ca.sort_order ?? index
    }))
    modalOpen.value = true
  } catch {
    toast.add({ title: 'Failed to load category', color: 'error' })
  }
}

async function viewAttributes(category: Category) {
  try {
    const fullCategory = await apiFetch<Category>(`/api/categories/${category.id}`)
    viewingCategory.value = fullCategory
    attributesModalOpen.value = true
  } catch {
    toast.add({ title: 'Failed to load category attributes', color: 'error' })
  }
}

function addAttribute() {
  if (!attributes.value?.length) return
  const firstUnused = attributes.value.find(
    a => !form.attributes.some(fa => fa.attribute_id === a.id)
  )
  if (firstUnused) {
    form.attributes.push({
      attribute_id: firstUnused.id,
      required: false,
      sort_order: form.attributes.length
    })
  }
}

function removeAttribute(index: number) {
  form.attributes.splice(index, 1)
  // Update sort orders
  form.attributes.forEach((a, i) => a.sort_order = i)
}

function getAttributeName(attributeId: string): string {
  return attributes.value?.find(a => a.id === attributeId)?.name || 'Unknown'
}

const availableAttributes = computed(() => {
  return attributes.value?.filter(
    a => !form.attributes.some(fa => fa.attribute_id === a.id)
  ) || []
})

async function saveCategory() {
  try {
    const url = editingCategory.value
      ? `/api/categories/${editingCategory.value.id}`
      : `/api/categories`

    await apiFetch(url, {
      method: editingCategory.value ? 'PUT' : 'POST',
      body: JSON.stringify({
        name: form.name,
        description: form.description || null,
        parent_id: form.parent_id || null,
        attributes: form.attributes
      })
    })

    toast.add({
      title: editingCategory.value ? 'Category updated' : 'Category created',
      color: 'success'
    })
    modalOpen.value = false
    refresh()
  } catch {
    toast.add({ title: 'Failed to save category', color: 'error' })
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

const parentOptions = computed<{ label: string; value: string | undefined }[]>(() => {
  const options: { label: string; value: string | undefined }[] = [
    { label: 'None (Top Level)', value: undefined }
  ]

  if (!categories.value) return options

  // Exclude current category and its descendants when editing
  const excludeIds = new Set<string>()
  if (editingCategory.value) {
    excludeIds.add(editingCategory.value.id)
    const addDescendants = (parentId: string) => {
      categories.value?.forEach(c => {
        if (c.parent_id === parentId) {
          excludeIds.add(c.id)
          addDescendants(c.id)
        }
      })
    }
    addDescendants(editingCategory.value.id)
  }

  categories.value.forEach(c => {
    if (!excludeIds.has(c.id)) {
      options.push({ label: c.name, value: c.id })
    }
  })

  return options
})

// Stats
const totalCategories = computed(() => categories.value?.length || 0)
const totalItems = computed(() => {
  if (!categoryAssetCounts.value) return 0
  return Object.values(categoryAssetCounts.value).reduce((sum, count) => sum + count, 0)
})
const avgAttributes = computed(() => {
  if (!categories.value?.length) return 0
  const total = categories.value.reduce((sum, c) => sum + (c.attributes?.length || 0), 0)
  return (total / categories.value.length).toFixed(1)
})

// Get asset count for a category
function getAssetCount(categoryId: string): number {
  return categoryAssetCounts.value?.[categoryId] || 0
}

// Get icon and color based on category name
function getCategoryStyle(category: Category): { icon: string; bgColor: string; textColor: string } {
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
</script>

<template>
  <div class="space-y-8">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          Asset Categories
        </h1>
        <p class="text-mist-500">
          Organize and classify your home belongings with custom schemas.
        </p>
      </div>
      <UButton
        size="lg"
        icon="i-lucide-plus-circle"
        class="shadow-lg shadow-attic-500/20"
        @click="openCreateModal"
      >
        Create New Category
      </UButton>
    </div>

    <!-- Stats Overview -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- Total Categories -->
      <div class="bg-white dark:bg-mist-800 p-6 rounded-2xl border border-mist-100 dark:border-mist-700 shadow-sm">
        <p class="text-sm font-medium text-mist-500 mb-2 uppercase tracking-wider">
          Total Categories
        </p>
        <div class="flex items-end justify-between">
          <span class="text-3xl font-black text-mist-950 dark:text-white">{{ totalCategories }}</span>
        </div>
      </div>

      <!-- Total Items Tracked -->
      <div class="bg-white dark:bg-mist-800 p-6 rounded-2xl border border-mist-100 dark:border-mist-700 shadow-sm">
        <p class="text-sm font-medium text-mist-500 mb-2 uppercase tracking-wider">
          Total Items Tracked
        </p>
        <div class="flex items-end justify-between">
          <span class="text-3xl font-black text-mist-950 dark:text-white">{{ totalItems }}</span>
        </div>
      </div>

      <!-- Avg Attributes -->
      <div class="bg-white dark:bg-mist-800 p-6 rounded-2xl border border-mist-100 dark:border-mist-700 shadow-sm">
        <p class="text-sm font-medium text-mist-500 mb-2 uppercase tracking-wider">
          Avg. Attributes
        </p>
        <div class="flex items-end justify-between">
          <span class="text-3xl font-black text-mist-950 dark:text-white">{{ avgAttributes }}</span>
          <span class="text-mist-500 text-xs">Per category</span>
        </div>
      </div>
    </div>

    <!-- Main Category Table -->
    <div class="bg-white dark:bg-mist-800 rounded-2xl border border-mist-100 dark:border-mist-700 overflow-hidden shadow-sm">
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
        <UButton @click="openCreateModal">
          Create Category
        </UButton>
      </div>

      <!-- Table -->
      <div
        v-else
        class="overflow-x-auto"
      >
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-mist-50 dark:bg-mist-700/50">
              <th class="px-6 py-4 text-xs font-black uppercase tracking-widest text-mist-500">
                Category Name
              </th>
              <th class="px-6 py-4 text-xs font-black uppercase tracking-widest text-mist-500">
                Description
              </th>
              <th class="px-6 py-4 text-xs font-black uppercase tracking-widest text-mist-500">
                Attributes
              </th>
              <th class="px-6 py-4 text-xs font-black uppercase tracking-widest text-mist-500">
                Items
              </th>
              <th class="px-6 py-4 text-xs font-black uppercase tracking-widest text-mist-500 text-right">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-mist-100 dark:divide-mist-700">
            <tr
              v-for="category in categories"
              :key="category.id"
              class="group hover:bg-mist-50/50 dark:hover:bg-mist-700/30 transition-colors"
            >
              <!-- Category Name with Icon -->
              <td class="px-6 py-5">
                <div class="flex items-center gap-3">
                  <div
                    class="size-10 rounded-lg flex items-center justify-center"
                    :class="[getCategoryStyle(category).bgColor, getCategoryStyle(category).textColor]"
                  >
                    <UIcon
                      :name="getCategoryStyle(category).icon"
                      class="w-5 h-5"
                    />
                  </div>
                  <span class="font-bold text-lg text-mist-950 dark:text-white">
                    {{ category.name }}
                  </span>
                </div>
              </td>

              <!-- Description -->
              <td class="px-6 py-5 text-mist-500 text-sm max-w-xs">
                <span class="line-clamp-2">
                  {{ category.description || 'No description' }}
                </span>
              </td>

              <!-- Attributes Button -->
              <td class="px-6 py-5">
                <button
                  class="flex items-center gap-1.5 px-3 py-1.5 bg-attic-500/10 text-attic-500 text-xs font-bold rounded-full hover:bg-attic-500/20 transition-colors"
                  @click="viewAttributes(category)"
                >
                  <UIcon
                    name="i-lucide-list"
                    class="w-3.5 h-3.5"
                  />
                  {{ category.attributes?.length || 0 }} Attributes
                </button>
              </td>

              <!-- Items Count -->
              <td class="px-6 py-5">
                <NuxtLink
                  :to="`/assets?category_id=${category.id}`"
                  class="text-sm font-semibold px-2.5 py-1 bg-mist-100 dark:bg-mist-700 rounded-lg hover:bg-mist-200 dark:hover:bg-mist-600 transition-colors"
                >
                  {{ getAssetCount(category.id) }} Items
                </NuxtLink>
              </td>

              <!-- Actions -->
              <td class="px-6 py-5 text-right">
                <div class="flex items-center justify-end gap-2">
                  <button
                    class="p-2 text-mist-500 hover:text-attic-500 hover:bg-attic-500/10 rounded-lg transition-all"
                    title="Edit Category"
                    @click="openEditModal(category)"
                  >
                    <UIcon
                      name="i-lucide-edit"
                      class="w-5 h-5"
                    />
                  </button>
                  <button
                    class="p-2 text-mist-500 hover:text-red-500 hover:bg-red-500/10 rounded-lg transition-all"
                    title="Delete Category"
                    @click="confirmDelete(category)"
                  >
                    <UIcon
                      name="i-lucide-trash-2"
                      class="w-5 h-5"
                    />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Tip Section -->
    <div class="p-4 bg-attic-500/5 rounded-xl border border-attic-500/10 flex items-start gap-3">
      <UIcon
        name="i-lucide-lightbulb"
        class="w-5 h-5 text-attic-500 flex-shrink-0 mt-0.5"
      />
      <p class="text-sm text-attic-600 dark:text-attic-400 leading-relaxed">
        <strong>Tip:</strong> You can define mandatory attributes (like Purchase Price or Serial Number) for each category. These will be automatically prompted whenever you add a new item to that category.
      </p>
    </div>

    <!-- Create/Edit Modal -->
    <UModal v-model:open="modalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl max-w-lg w-full">
          <div class="p-6 border-b border-mist-100 dark:border-mist-700">
            <h3 class="text-lg font-bold text-mist-950 dark:text-white">
              {{ editingCategory ? 'Edit Category' : 'New Category' }}
            </h3>
          </div>

          <form
            class="p-6 space-y-4"
            @submit.prevent="saveCategory"
          >
            <!-- Name -->
            <div>
              <label class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5">
                Name <span class="text-red-500">*</span>
              </label>
              <input
                v-model="form.name"
                type="text"
                required
                placeholder="Category name"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white placeholder-mist-400 focus:ring-2 focus:ring-attic-500 focus:border-transparent"
              >
            </div>

            <!-- Description -->
            <div>
              <label class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5">
                Description
              </label>
              <textarea
                v-model="form.description"
                rows="3"
                placeholder="Optional description"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white placeholder-mist-400 focus:ring-2 focus:ring-attic-500 focus:border-transparent resize-none"
              />
            </div>

            <!-- Parent Category -->
            <div>
              <label class="block text-sm font-medium text-mist-700 dark:text-mist-300 mb-1.5">
                Parent Category
              </label>
              <select
                :value="form.parent_id ?? ''"
                class="w-full bg-mist-50 dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-4 py-2.5 text-sm text-mist-950 dark:text-white focus:ring-2 focus:ring-attic-500 focus:border-transparent"
                @change="form.parent_id = ($event.target as HTMLSelectElement).value || undefined"
              >
                <option
                  v-for="opt in parentOptions"
                  :key="opt.value ?? 'none'"
                  :value="opt.value ?? ''"
                >
                  {{ opt.label }}
                </option>
              </select>
            </div>

            <!-- Attributes Section -->
            <div class="border-t border-mist-100 dark:border-mist-700 pt-4 mt-4">
              <div class="flex items-center justify-between mb-3">
                <label class="block text-sm font-medium text-mist-700 dark:text-mist-300">
                  Attributes
                </label>
                <button
                  v-if="availableAttributes.length > 0"
                  type="button"
                  class="text-xs font-bold text-attic-500 hover:text-attic-600 flex items-center gap-1"
                  @click="addAttribute"
                >
                  <UIcon
                    name="i-lucide-plus"
                    class="w-3.5 h-3.5"
                  />
                  Add Attribute
                </button>
              </div>

              <div
                v-if="form.attributes.length === 0"
                class="text-sm text-mist-500 py-3 px-4 bg-mist-50 dark:bg-mist-700/50 rounded-lg"
              >
                No attributes assigned. Click "Add Attribute" to assign attributes.
              </div>

              <div
                v-else
                class="space-y-2"
              >
                <div
                  v-for="(attr, index) in form.attributes"
                  :key="attr.attribute_id"
                  class="flex items-center gap-3 p-3 bg-mist-50 dark:bg-mist-700/50 rounded-lg"
                >
                  <select
                    :value="attr.attribute_id"
                    class="flex-1 bg-white dark:bg-mist-700 border border-mist-200 dark:border-mist-600 rounded-lg px-3 py-2 text-sm text-mist-950 dark:text-white focus:ring-2 focus:ring-attic-500 focus:border-transparent"
                    @change="attr.attribute_id = ($event.target as HTMLSelectElement).value"
                  >
                    <option :value="attr.attribute_id">
                      {{ getAttributeName(attr.attribute_id) }}
                    </option>
                    <option
                      v-for="a in availableAttributes"
                      :key="a.id"
                      :value="a.id"
                    >
                      {{ a.name }}
                    </option>
                  </select>
                  <label class="flex items-center gap-2 text-sm text-mist-700 dark:text-mist-300 cursor-pointer">
                    <input
                      v-model="attr.required"
                      type="checkbox"
                      class="rounded border-mist-300 text-attic-500 focus:ring-attic-500"
                    >
                    Required
                  </label>
                  <button
                    type="button"
                    class="p-1.5 text-mist-400 hover:text-red-500 hover:bg-red-500/10 rounded transition-all"
                    @click="removeAttribute(index)"
                  >
                    <UIcon
                      name="i-lucide-trash-2"
                      class="w-4 h-4"
                    />
                  </button>
                </div>
              </div>

              <p
                v-if="attributes?.length === 0"
                class="text-sm text-mist-500 mt-3"
              >
                <NuxtLink
                  to="/attributes"
                  class="text-attic-500 hover:underline font-medium"
                >
                  Create attributes
                </NuxtLink>
                first to assign them to categories.
              </p>
            </div>
          </form>

          <div class="p-6 border-t border-mist-100 dark:border-mist-700 flex justify-end gap-3">
            <UButton
              variant="ghost"
              color="neutral"
              @click="modalOpen = false"
            >
              Cancel
            </UButton>
            <UButton @click="saveCategory">
              {{ editingCategory ? 'Save Changes' : 'Create Category' }}
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

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
                  <UIcon
                    name="i-lucide-hash"
                    class="w-4 h-4 text-mist-400"
                  />
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
              variant="soft"
              @click="viewingCategory && openEditModal(viewingCategory); attributesModalOpen = false"
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
