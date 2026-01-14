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
}

// Remove attribute from selection
function removeAttribute(index: number) {
  selectedAttributes.value.splice(index, 1)
  // Update sort orders
  selectedAttributes.value.forEach((a, i) => a.sort_order = i)
}

// Get icon and color for attribute type
function getAttributeStyle(dataType: string): { icon: string; bgColor: string; textColor: string } {
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
  <div class="space-y-8">
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
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div class="flex flex-col gap-2">
          <nav class="flex items-center text-sm font-medium text-mist-500">
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
            <span class="text-mist-950 dark:text-white">Edit</span>
          </nav>
          <div>
            <h1 class="text-3xl font-extrabold tracking-tight text-mist-950 dark:text-white">
              Edit Category
            </h1>
            <p class="text-mist-500 mt-1">
              Modify the structure of "{{ category.name }}".
            </p>
          </div>
        </div>
        <div class="flex items-center gap-3">
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
            @click="saveCategory"
          >
            Save Changes
          </UButton>
        </div>
      </div>

      <!-- Two Column Layout -->
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        <!-- LEFT COLUMN: Identity -->
        <div class="lg:col-span-4 space-y-6">
          <!-- Basic Info Card -->
          <div class="bg-white dark:bg-mist-800 rounded-xl shadow-soft border border-mist-100 dark:border-mist-700 p-6">
            <div class="flex items-center gap-2 mb-6">
              <UIcon
                name="i-lucide-badge"
                class="w-5 h-5 text-attic-500"
              />
              <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                Identity
              </h3>
            </div>
            <div class="space-y-5">
              <div>
                <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                  Category Name
                </label>
                <input
                  v-model="form.name"
                  type="text"
                  placeholder="e.g. Rare Books"
                  class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 font-medium text-mist-950 dark:text-white"
                >
              </div>
              <div>
                <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                  Description
                </label>
                <textarea
                  v-model="form.description"
                  rows="4"
                  maxlength="140"
                  placeholder="What kind of assets belong here?"
                  class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm resize-none text-mist-950 dark:text-white"
                />
                <div class="flex justify-end mt-1">
                  <span class="text-xs text-mist-400">{{ descriptionCount }}/140</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Icon Selection Card -->
          <div class="bg-white dark:bg-mist-800 rounded-xl shadow-soft border border-mist-100 dark:border-mist-700 p-6">
            <div class="flex items-center gap-2 mb-4">
              <UIcon
                name="i-lucide-sparkles"
                class="w-5 h-5 text-attic-500"
              />
              <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                Icon
              </h3>
            </div>
            <div class="max-h-[280px] overflow-y-auto custom-scrollbar pr-1">
              <div class="grid grid-cols-5 gap-3">
                <button
                  v-for="icon in icons"
                  :key="icon"
                  type="button"
                  class="aspect-square rounded-xl flex items-center justify-center transition-all"
                  :class="form.icon === icon
                    ? 'bg-attic-500 text-white ring-2 ring-offset-2 ring-attic-500 dark:ring-offset-mist-800'
                    : 'bg-mist-50 dark:bg-mist-900 text-mist-500 hover:text-attic-500 hover:bg-mist-100 dark:hover:bg-mist-700 border border-transparent hover:border-mist-200 dark:hover:border-mist-600'"
                  @click="form.icon = icon"
                >
                  <UIcon
                    :name="icon"
                    class="w-6 h-6"
                  />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- RIGHT COLUMN: Attribute Schema -->
        <div class="lg:col-span-8">
          <div class="bg-white dark:bg-mist-800 rounded-xl shadow-soft border border-mist-100 dark:border-mist-700 flex flex-col min-h-[500px]">
            <!-- Composer Header -->
            <div class="px-6 py-5 border-b border-mist-100 dark:border-mist-700 flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon
                  name="i-lucide-sliders-horizontal"
                  class="w-5 h-5 text-attic-500"
                />
                <div>
                  <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                    Attribute Schema
                  </h3>
                  <p class="text-xs text-mist-500">
                    Construct the data model for your items.
                  </p>
                </div>
              </div>
              <NuxtLink
                to="/attributes"
                class="text-sm font-semibold text-attic-500 hover:text-attic-600 flex items-center gap-1 px-3 py-1.5 rounded-lg hover:bg-attic-500/5 transition-colors"
              >
                <UIcon
                  name="i-lucide-plus-circle"
                  class="w-4 h-4"
                />
                New Attribute
              </NuxtLink>
            </div>

            <!-- Composer Body: Split View -->
            <div class="flex flex-col lg:flex-row flex-grow min-h-0">
              <!-- Zone 1: Active Attributes -->
              <div class="flex-grow p-6 bg-mist-50 dark:bg-mist-900/50 relative overflow-y-auto custom-scrollbar">
                <div class="relative z-10 space-y-3">
                  <div class="flex items-center justify-between mb-4">
                    <h4 class="text-xs font-bold uppercase tracking-wider text-mist-400">
                      Active Attributes
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
                    class="group bg-white dark:bg-mist-800 p-3 rounded-lg border border-mist-100 dark:border-mist-700 shadow-sm flex items-center gap-4 hover:border-attic-500/50 transition-colors"
                  >
                    <div class="text-mist-300 group-hover:text-mist-500">
                      <UIcon
                        name="i-lucide-grip-vertical"
                        class="w-5 h-5"
                      />
                    </div>
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
                        {{ getAttribute(attr.attribute_id)?.data_type || 'string' }}
                      </p>
                    </div>
                    <div class="flex items-center gap-4 border-l border-mist-100 dark:border-mist-700 pl-4">
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
                    class="h-32 border-2 border-dashed border-mist-300 dark:border-mist-600 rounded-lg flex flex-col items-center justify-center text-mist-400 bg-white/50 dark:bg-mist-800/50"
                  >
                    <UIcon
                      name="i-lucide-list-plus"
                      class="w-8 h-8 mb-2 opacity-50"
                    />
                    <span class="text-sm font-medium">Select attributes from the library</span>
                  </div>

                  <!-- Add more placeholder when has items -->
                  <div
                    v-else
                    class="h-16 border-2 border-dashed border-mist-200 dark:border-mist-700 rounded-lg flex items-center justify-center text-mist-400 bg-white/30 dark:bg-mist-800/30"
                  >
                    <span class="text-xs">Select more from the library</span>
                  </div>
                </div>
              </div>

              <!-- Zone 2: Attribute Library -->
              <div class="w-full lg:w-72 border-t lg:border-t-0 lg:border-l border-mist-100 dark:border-mist-700 bg-white dark:bg-mist-800 flex flex-col">
                <div class="p-4 border-b border-mist-100 dark:border-mist-700">
                  <div class="relative">
                    <UIcon
                      name="i-lucide-search"
                      class="absolute left-3 top-2.5 w-4 h-4 text-mist-400"
                    />
                    <input
                      v-model="attributeSearch"
                      type="text"
                      placeholder="Search attributes..."
                      class="w-full pl-9 pr-4 py-2 rounded-lg bg-mist-50 dark:bg-mist-900 border-none text-sm focus:ring-1 focus:ring-attic-500 text-mist-950 dark:text-white placeholder-mist-400"
                    >
                  </div>
                </div>
                <div class="p-4 overflow-y-auto max-h-[400px] custom-scrollbar space-y-2">
                  <h5 class="text-xs font-bold text-mist-400 uppercase mb-3">
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

                  <!-- Attribute list -->
                  <button
                    v-for="attr in filteredAttributes"
                    :key="attr.id"
                    type="button"
                    class="w-full flex items-center gap-3 p-2 rounded-lg hover:bg-mist-50 dark:hover:bg-mist-700 cursor-pointer group border border-transparent hover:border-mist-200 dark:hover:border-mist-600 transition-all text-left"
                    @click="addAttribute(attr)"
                  >
                    <div
                      class="size-8 rounded flex items-center justify-center"
                      :class="[getAttributeStyle(attr.data_type).bgColor, getAttributeStyle(attr.data_type).textColor]"
                    >
                      <UIcon
                        :name="getAttributeStyle(attr.data_type).icon"
                        class="w-4 h-4"
                      />
                    </div>
                    <div class="flex-grow">
                      <p class="text-sm font-medium text-mist-700 dark:text-mist-200">
                        {{ attr.name }}
                      </p>
                      <p class="text-xs text-mist-400">
                        {{ attr.data_type }}
                      </p>
                    </div>
                    <UIcon
                      name="i-lucide-plus"
                      class="w-4 h-4 text-mist-300 opacity-0 group-hover:opacity-100"
                    />
                  </button>
                </div>
              </div>
            </div>
          </div>
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
