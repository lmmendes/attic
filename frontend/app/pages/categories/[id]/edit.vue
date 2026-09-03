<script setup lang="ts">
import type { Category, Attribute } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

const categoryId = computed(() => route.params.id as string)

// Fetch category and attributes
const { data: category, status: categoryStatus } = useApi<Category>(
  () => `/api/categories/${categoryId.value}`
)
const { data: attributes } = useApi<Attribute[]>('/api/attributes')

// Form state
const form = reactive({
  name: '',
  description: '',
  icon: 'i-lucide-tag',
  parent_id: undefined as string | undefined
})

// Attribute selection
interface AttributeSelection {
  attribute_id: string
  required: boolean
  sort_order: number
}

const selectedAttributes = ref<AttributeSelection[]>([])
const draggedAttributeId = ref<string | null>(null)

// Initialize form when category loads
watch(category, (cat) => {
  if (cat) {
    form.name = cat.name
    form.description = cat.description || ''
    form.icon = cat.icon || 'i-lucide-tag'
    form.parent_id = cat.parent_id
    selectedAttributes.value = (cat.attributes || []).map((ca, index) => ({
      attribute_id: ca.attribute_id,
      required: ca.required,
      sort_order: ca.sort_order ?? index
    }))
  }
}, { immediate: true })

// Search query for attribute library
const attributeSearch = ref('')

// Available icons - expanded list
const icons = [
  // Media & Entertainment
  'i-lucide-book-open',
  'i-lucide-headphones',
  'i-lucide-gamepad-2',
  'i-lucide-film',
  'i-lucide-music',
  'i-lucide-tv',
  'i-lucide-disc-3',
  'i-lucide-radio',
  // Electronics & Tech
  'i-lucide-laptop',
  'i-lucide-camera',
  'i-lucide-smartphone',
  'i-lucide-tablet',
  'i-lucide-monitor',
  'i-lucide-printer',
  'i-lucide-cpu',
  'i-lucide-hard-drive',
  // Clothing & Accessories
  'i-lucide-shirt',
  'i-lucide-watch',
  'i-lucide-glasses',
  'i-lucide-gem',
  // Home & Furniture
  'i-lucide-armchair',
  'i-lucide-lamp',
  'i-lucide-bed',
  'i-lucide-sofa',
  // Tools & Equipment
  'i-lucide-wrench',
  'i-lucide-hammer',
  'i-lucide-drill',
  'i-lucide-scissors',
  // Sports & Fitness
  'i-lucide-dumbbell',
  'i-lucide-bike',
  'i-lucide-footprints',
  // Art & Creative
  'i-lucide-palette',
  'i-lucide-brush',
  'i-lucide-pen-tool',
  'i-lucide-image',
  // Transport
  'i-lucide-car',
  'i-lucide-plane',
  'i-lucide-ship',
  // Kitchen & Dining
  'i-lucide-chef-hat',
  'i-lucide-utensils',
  'i-lucide-coffee',
  'i-lucide-wine',
  // Other
  'i-lucide-box',
  'i-lucide-archive',
  'i-lucide-briefcase',
  'i-lucide-gift',
  'i-lucide-heart',
  'i-lucide-star',
  'i-lucide-tag'
]

// Character count for description
const descriptionCount = computed(() => form.description.length)

// Filter available attributes (not already selected)
const availableAttributes = computed(() => {
  if (!attributes.value) return []
  return attributes.value.filter(
    a => !selectedAttributes.value.some(sa => sa.attribute_id === a.id)
  )
})

// Filtered attributes based on search
const filteredAttributes = computed(() => {
  if (!attributeSearch.value.trim()) return availableAttributes.value
  const query = attributeSearch.value.toLowerCase()
  return availableAttributes.value.filter(
    a => a.name.toLowerCase().includes(query)
  )
})

// Get attribute by ID
function getAttribute(id: string): Attribute | undefined {
  return attributes.value?.find(a => a.id === id)
}

// Add attribute to selection
function addAttribute(attr: Attribute) {
  selectedAttributes.value.push({
    attribute_id: attr.id,
    required: false,
    sort_order: selectedAttributes.value.length
  })
  syncSelectedAttributeSortOrder()
}

// Remove attribute from selection
function removeAttribute(index: number) {
  selectedAttributes.value.splice(index, 1)
  syncSelectedAttributeSortOrder()
}

function handleAttributeDragStart(attributeID: string) {
  draggedAttributeId.value = attributeID
}

function handleAttributeDragEnd() {
  draggedAttributeId.value = null
}

function handleAttributeDrop(targetIndex: number) {
  if (!draggedAttributeId.value) {
    return
  }

  const sourceIndex = selectedAttributes.value.findIndex(
    attr => attr.attribute_id === draggedAttributeId.value
  )
  if (sourceIndex === -1 || sourceIndex === targetIndex) {
    draggedAttributeId.value = null
    return
  }

  const [movedAttribute] = selectedAttributes.value.splice(sourceIndex, 1)
  if (!movedAttribute) {
    draggedAttributeId.value = null
    return
  }

  selectedAttributes.value.splice(targetIndex, 0, movedAttribute)
  draggedAttributeId.value = null
  syncSelectedAttributeSortOrder()
}

function syncSelectedAttributeSortOrder() {
  selectedAttributes.value.forEach((attribute, index) => {
    attribute.sort_order = index
  })
}

// Get icon and color for attribute type
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

// Saving state
const saving = ref(false)

// Save category
async function saveCategory() {
  if (!form.name.trim()) {
    toast.add({ title: 'Please enter a category name', color: 'error' })
    return
  }

  saving.value = true
  try {
    await apiFetch(`/api/categories/${categoryId.value}`, {
      method: 'PUT',
      body: JSON.stringify({
        name: form.name,
        description: form.description || null,
        icon: form.icon,
        parent_id: form.parent_id || null,
        attributes: selectedAttributes.value
      })
    })

    toast.add({ title: 'Category updated successfully', color: 'success' })
    router.push('/categories')
  } catch {
    toast.add({ title: 'Failed to update category', color: 'error' })
  } finally {
    saving.value = false
  }
}

// Cancel and go back
function cancel() {
  router.push('/categories')
}
</script>

<template>
  <div class="mx-auto max-w-[1040px] space-y-5 pb-6">
    <!-- Loading State -->
    <div
      v-if="categoryStatus === 'pending'"
      class="flex items-center justify-center py-20"
    >
      <UIcon
        name="i-lucide-loader-2"
        class="w-8 h-8 text-attic-500 animate-spin"
      />
    </div>

    <template v-else-if="category">
      <!-- Breadcrumbs & Header -->
      <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div class="space-y-3">
          <nav class="flex items-center text-xs font-semibold text-mist-500">
            <NuxtLink
              to="/"
              class="hover:text-attic-500 transition-colors"
            >
              Home
            </NuxtLink>
            <span class="mx-2 text-mist-300 dark:text-mist-600">/</span>
            <NuxtLink
              to="/categories"
              class="hover:text-attic-500 transition-colors"
            >
              Categories
            </NuxtLink>
            <span class="mx-2 text-mist-300 dark:text-mist-600">/</span>
            <span class="font-bold text-attic-500">Edit</span>
          </nav>
          <div>
            <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
              Category editor
            </p>
            <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
              Edit {{ category.name }}
            </h1>
            <p class="mt-1 text-sm text-mist-500">
              Set its identity and choose which details assets in this category should record.
            </p>
          </div>
        </div>
        <div class="hidden items-center gap-2 sm:flex">
          <UButton
            variant="ghost"
            color="neutral"
            class="rounded-xl font-semibold"
            @click="cancel"
          >
            Cancel
          </UButton>
          <UButton
            icon="i-lucide-save"
            :loading="saving"
            class="rounded-xl font-bold shadow-primary"
            @click="saveCategory"
          >
            Save Changes
          </UButton>
        </div>
      </div>

      <div class="space-y-5">
        <!-- LEFT COLUMN: Identity -->
        <div class="space-y-5">
          <!-- Basic Info Card -->
          <section class="attic-panel rounded-[20px] p-5 sm:p-6">
            <div class="mb-5 flex items-start gap-3">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-attic-50 text-attic-500 dark:bg-attic-500/10">
                <UIcon
                  name="i-lucide-badge"
                  class="size-4.5"
                />
              </div>
              <div>
                <h2 class="font-extrabold text-mist-950 dark:text-white">
                  Identity
                </h2>
                <p class="text-xs text-mist-500">
                  Give the category a clear, recognizable name.
                </p>
              </div>
            </div>
            <div class="max-w-3xl space-y-5">
              <div>
                <label class="mb-2 block text-xs font-bold uppercase tracking-wider text-mist-500">
                  Category Name
                </label>
                <input
                  v-model="form.name"
                  type="text"
                  placeholder="e.g. Rare Books"
                  class="w-full rounded-xl border border-mist-200 bg-white px-4 py-3 font-medium text-mist-950 shadow-sm outline-none transition-all placeholder:text-mist-400 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
                >
              </div>
              <div>
                <label class="mb-2 block text-xs font-bold uppercase tracking-wider text-mist-500">
                  Description
                </label>
                <textarea
                  v-model="form.description"
                  rows="4"
                  maxlength="140"
                  placeholder="What kind of assets belong here?"
                  class="w-full resize-none rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm outline-none transition-all placeholder:text-mist-400 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
                />
                <div class="flex justify-end mt-1">
                  <span class="text-xs text-mist-400">{{ descriptionCount }}/140</span>
                </div>
              </div>
            </div>
          </section>

          <!-- Icon Selection Card -->
          <section class="attic-panel rounded-[20px] p-5 sm:p-6">
            <div class="mb-4 flex items-start gap-3">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-terracotta-50 text-terracotta-500 dark:bg-terracotta-500/10">
                <UIcon
                  name="i-lucide-sparkles"
                  class="size-4.5"
                />
              </div>
              <div>
                <h2 class="font-extrabold text-mist-950 dark:text-white">
                  Icon
                </h2>
                <p class="text-xs text-mist-500">
                  Make the category easy to spot across your collection.
                </p>
              </div>
            </div>
            <div class="max-h-40 overflow-y-auto pr-1 custom-scrollbar">
              <div class="grid grid-cols-7 gap-2 sm:grid-cols-10 md:grid-cols-12">
                <button
                  v-for="icon in icons"
                  :key="icon"
                  type="button"
                  class="flex aspect-square items-center justify-center rounded-xl transition-all"
                  :class="form.icon === icon
                    ? 'bg-attic-500 text-white ring-2 ring-offset-2 ring-attic-500 dark:ring-offset-mist-800'
                    : 'bg-mist-50 dark:bg-mist-900 text-mist-500 hover:text-attic-500 hover:bg-mist-100 dark:hover:bg-mist-700 border border-transparent hover:border-mist-200 dark:hover:border-mist-600'"
                  @click="form.icon = icon"
                >
                  <UIcon
                    :name="icon"
                    class="size-5"
                  />
                </button>
              </div>
            </div>
          </section>
        </div>

        <!-- Attribute Schema -->
        <div>
          <section class="attic-panel flex min-h-[500px] flex-col overflow-hidden rounded-[20px]">
            <!-- Composer Header -->
            <div class="flex flex-col gap-3 border-b border-mist-100 px-5 py-5 dark:border-mist-700 sm:flex-row sm:items-center sm:justify-between sm:px-6">
              <div class="flex items-start gap-3">
                <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400">
                  <UIcon
                    name="i-lucide-sliders-horizontal"
                    class="size-4.5"
                  />
                </div>
                <div>
                  <h2 class="font-extrabold text-mist-950 dark:text-white">
                    Asset fields
                  </h2>
                  <p class="text-xs text-mist-500">
                    Choose the information shown when adding or editing an asset.
                  </p>
                </div>
              </div>
              <NuxtLink
                to="/attributes"
                class="flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm font-semibold text-attic-500 transition-colors hover:bg-attic-500/5 hover:text-attic-600"
              >
                <UIcon
                  name="i-lucide-plus-circle"
                  class="w-4 h-4"
                />
                Create field
              </NuxtLink>
            </div>

            <div class="flex min-h-0 flex-grow flex-col">
              <!-- Zone 1: Active Attributes -->
              <div class="relative flex-grow bg-mist-50/70 p-5 dark:bg-mist-900/40 sm:p-6">
                <div class="relative z-10 space-y-3">
                  <div class="flex items-center justify-between mb-4">
                    <h4 class="text-xs font-bold uppercase tracking-wider text-mist-400">
                      Selected fields
                    </h4>
                    <span
                      v-if="selectedAttributes.length > 0"
                      class="text-xs text-mist-400"
                    >
                      {{ selectedAttributes.length }} selected
                    </span>
                  </div>

                  <!-- Selected Attributes -->
                  <div
                    v-for="(attr, index) in selectedAttributes"
                    :key="attr.attribute_id"
                    class="group flex items-center gap-3 rounded-xl border border-mist-100 bg-white p-3 shadow-sm transition-colors hover:border-attic-500/50 dark:border-mist-700 dark:bg-mist-800 sm:gap-4"
                    @dragover.prevent
                    @drop.prevent="handleAttributeDrop(index)"
                  >
                    <button
                      type="button"
                      draggable="true"
                      class="text-mist-300 group-hover:text-mist-500 cursor-grab active:cursor-grabbing"
                      @dragstart="handleAttributeDragStart(attr.attribute_id)"
                      @dragend="handleAttributeDragEnd"
                    >
                      <UIcon
                        name="i-lucide-grip-vertical"
                        class="w-5 h-5"
                      />
                    </button>
                    <div
                      class="flex items-center justify-center size-10 rounded-md"
                      :class="[getAttributeStyle(getAttribute(attr.attribute_id)?.data_type || 'string').bgColor, getAttributeStyle(getAttribute(attr.attribute_id)?.data_type || 'string').textColor]"
                    >
                      <UIcon
                        :name="getAttributeStyle(getAttribute(attr.attribute_id)?.data_type || 'string').icon"
                        class="w-5 h-5"
                      />
                    </div>
                    <div class="flex-grow">
                      <p class="font-bold text-sm text-mist-950 dark:text-white">
                        {{ getAttribute(attr.attribute_id)?.name || 'Unknown' }}
                      </p>
                      <p class="text-xs text-mist-500">
                        {{ getAttribute(attr.attribute_id)?.key || 'unknown_key' }} · {{ getAttribute(attr.attribute_id)?.data_type || 'string' }}
                      </p>
                    </div>
                    <div class="flex items-center gap-3 border-l border-mist-100 pl-3 dark:border-mist-700 sm:gap-4 sm:pl-4">
                      <div class="flex flex-col items-end">
                        <span class="text-[10px] font-semibold uppercase tracking-wider text-mist-400 mb-1">
                          Required
                        </span>
                        <label class="relative inline-flex items-center cursor-pointer">
                          <input
                            v-model="attr.required"
                            type="checkbox"
                            class="sr-only peer"
                          >
                          <div class="w-9 h-5 bg-mist-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-attic-500 rounded-full peer dark:bg-mist-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-mist-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-mist-600 peer-checked:bg-attic-500" />
                        </label>
                      </div>
                      <button
                        type="button"
                        class="text-mist-400 hover:text-red-500 transition-colors p-1"
                        @click="removeAttribute(index)"
                      >
                        <UIcon
                          name="i-lucide-trash-2"
                          class="w-5 h-5"
                        />
                      </button>
                    </div>
                  </div>

                  <!-- Empty State / Drop Placeholder -->
                  <div
                    v-if="selectedAttributes.length === 0"
                    class="flex h-32 flex-col items-center justify-center rounded-xl border-2 border-dashed border-mist-300 bg-white/50 text-mist-400 dark:border-mist-600 dark:bg-mist-800/50"
                  >
                    <UIcon
                      name="i-lucide-list-plus"
                      class="w-8 h-8 mb-2 opacity-50"
                    />
                    <span class="text-sm font-medium">Add fields from the library below</span>
                  </div>

                  <!-- Add more placeholder when has items -->
                  <div
                    v-else
                    class="flex h-12 items-center justify-center rounded-xl border border-dashed border-mist-200 bg-white/30 text-mist-400 dark:border-mist-700 dark:bg-mist-800/30"
                  >
                    <span class="text-xs">Drag to reorder · toggle required fields on the right</span>
                  </div>
                </div>
              </div>

              <!-- Attribute Library -->
              <div class="border-t border-mist-100 bg-white dark:border-mist-700 dark:bg-mist-800">
                <div class="flex flex-col gap-3 border-b border-mist-100 p-4 dark:border-mist-700 sm:flex-row sm:items-center sm:justify-between sm:px-6">
                  <div>
                    <h4 class="text-xs font-bold uppercase tracking-wider text-mist-500">
                      Field library
                    </h4>
                    <p class="text-xs text-mist-400">
                      Select a field to add it to this category.
                    </p>
                  </div>
                  <UInput
                    v-model="attributeSearch"
                    icon="i-lucide-search"
                    placeholder="Search fields"
                    class="w-full sm:w-64"
                  />
                </div>
                <div class="max-h-[360px] overflow-y-auto p-4 custom-scrollbar sm:p-6">
                  <h5 class="mb-3 text-xs font-bold uppercase text-mist-400">
                    Available ({{ filteredAttributes.length }})
                  </h5>

                  <!-- No attributes message -->
                  <div
                    v-if="!attributes?.length"
                    class="text-center py-6"
                  >
                    <UIcon
                      name="i-lucide-list"
                      class="w-8 h-8 text-mist-300 mx-auto mb-2"
                    />
                    <p class="text-sm text-mist-500 mb-2">
                      No attributes created yet
                    </p>
                    <NuxtLink
                      to="/attributes"
                      class="text-xs text-attic-500 hover:underline font-medium"
                    >
                      Create attributes first
                    </NuxtLink>
                  </div>

                  <!-- No results -->
                  <div
                    v-else-if="filteredAttributes.length === 0"
                    class="text-center py-6"
                  >
                    <p class="text-sm text-mist-500">
                      {{ attributeSearch ? 'No matching attributes' : 'All attributes selected' }}
                    </p>
                  </div>

                  <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    <button
                      v-for="attr in filteredAttributes"
                      :key="attr.id"
                      type="button"
                      class="group flex w-full items-center gap-3 rounded-xl border border-mist-100 p-3 text-left transition-all hover:border-attic-200 hover:bg-attic-50/50 dark:border-mist-700 dark:hover:border-attic-500/30 dark:hover:bg-attic-500/5"
                      @click="addAttribute(attr)"
                    >
                      <div
                        class="flex size-8 shrink-0 items-center justify-center rounded-lg"
                        :class="[getAttributeStyle(attr.data_type).bgColor, getAttributeStyle(attr.data_type).textColor]"
                      >
                        <UIcon
                          :name="getAttributeStyle(attr.data_type).icon"
                          class="size-4"
                        />
                      </div>
                      <div class="min-w-0 flex-grow">
                        <p class="truncate text-sm font-bold text-mist-700 dark:text-mist-200">
                          {{ attr.name }}
                        </p>
                        <p class="truncate text-xs text-mist-400">
                          {{ attr.key }} · {{ attr.data_type }}
                        </p>
                      </div>
                      <UIcon
                        name="i-lucide-plus"
                        class="size-4 shrink-0 text-attic-500"
                      />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>

      <div class="attic-panel flex flex-col items-center justify-between gap-3 rounded-[18px] px-4 py-3 sm:flex-row">
        <p class="text-xs text-mist-400">
          Changes apply to future asset edits immediately.
        </p>
        <div class="flex items-center gap-2">
          <UButton
            variant="ghost"
            color="neutral"
            @click="cancel"
          >
            Cancel
          </UButton>
          <UButton
            icon="i-lucide-save"
            :loading="saving"
            class="rounded-xl shadow-primary"
            @click="saveCategory"
          >
            Save changes
          </UButton>
        </div>
      </div>
    </template>

    <!-- Not Found -->
    <div
      v-else
      class="flex flex-col items-center justify-center py-20"
    >
      <UIcon
        name="i-lucide-alert-circle"
        class="w-12 h-12 text-mist-400 mb-4"
      />
      <h2 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
        Category not found
      </h2>
      <p class="text-mist-500 mb-4">
        The category you're looking for doesn't exist.
      </p>
      <UButton
        to="/categories"
        variant="soft"
      >
        Back to Categories
      </UButton>
    </div>
  </div>
</template>
