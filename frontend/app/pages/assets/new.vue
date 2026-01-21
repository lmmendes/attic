<script setup lang="ts">
import type { Category, Location, Condition } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

const { data: categories } = useApi<Category[]>('/api/categories')
const { data: locations } = useApi<Location[]>('/api/locations')
const { data: conditions } = useApi<Condition[]>('/api/conditions')

const loading = ref(false)
const selectedCategory = ref<Category | null>(null)
const form = reactive({
  name: '',
  description: '',
  category_id: undefined as string | undefined,
  location_id: undefined as string | undefined,
  condition_id: undefined as string | undefined,
  quantity: 1,
  attributes: {} as Record<string, string | number | boolean>,
  purchase_at: '',
  purchase_price: undefined as number | undefined,
  purchase_note: '',
  notes: ''
})

const _categoryOptions = computed(() =>
  categories.value?.map(c => ({ label: c.name, value: c.id, icon: c.icon })) || []
)

const locationOptions = computed(() => [
  { label: 'No location', value: undefined },
  ...(locations.value?.map(l => ({ label: l.name, value: l.id })) || [])
])

const conditionOptions = computed(() => [
  { label: 'No condition', value: undefined },
  ...(conditions.value?.map(c => ({ label: c.label, value: c.id })) || [])
])

// Fetch category with attributes when category changes
watch(() => form.category_id, async (categoryId) => {
  if (categoryId) {
    try {
      selectedCategory.value = await apiFetch<Category>(`/api/categories/${categoryId}`)
      // Initialize attribute values
      form.attributes = {}
      selectedCategory.value?.attributes?.forEach((ca) => {
        if (ca.attribute) {
          form.attributes[ca.attribute.key] = getDefaultValue(ca.attribute.data_type)
        }
      })
    } catch {
      selectedCategory.value = null
    }
  } else {
    selectedCategory.value = null
    form.attributes = {}
  }
})

function getDefaultValue(dataType: string): string | number | boolean {
  switch (dataType) {
    case 'number': return 0
    case 'boolean': return false
    default: return ''
  }
}

function getInputType(dataType: string): string {
  switch (dataType) {
    case 'number': return 'number'
    case 'date': return 'date'
    case 'boolean': return 'checkbox'
    default: return 'text'
  }
}

async function submitForm() {
  if (!form.name || !form.category_id) {
    toast.add({ title: 'Name and category are required', color: 'error' })
    return
  }

  loading.value = true
  try {
    const payload = {
      name: form.name,
      description: form.description || undefined,
      category_id: form.category_id,
      location_id: form.location_id || undefined,
      condition_id: form.condition_id || undefined,
      quantity: form.quantity,
      attributes: Object.keys(form.attributes).length > 0 ? form.attributes : undefined,
      purchase_at: form.purchase_at || undefined,
      purchase_price: form.purchase_price || undefined,
      purchase_note: form.purchase_note || undefined,
      notes: form.notes || undefined
    }

    const response = await apiFetch<{ id: string }>(`/api/assets`, {
      method: 'POST',
      body: JSON.stringify(payload)
    })

    toast.add({ title: 'Asset created successfully', color: 'success' })
    router.push(`/assets/${response.id}`)
  } catch (err: unknown) {
    const error = err as { message?: string }
    toast.add({ title: error?.message || 'Failed to create asset', color: 'error' })
  } finally {
    loading.value = false
  }
}

// Calculate form progress
const formProgress = computed(() => {
  let filled = 0
  const total = 4 // name, category, location, condition
  if (form.name) filled++
  if (form.category_id) filled++
  if (form.location_id) filled++
  if (form.condition_id) filled++
  return (filled / total) * 100
})
</script>

<template>
  <div class="max-w-4xl mx-auto space-y-8">
    <!-- Breadcrumbs & Heading -->
    <div class="space-y-4">
      <nav
        aria-label="Breadcrumb"
        class="flex"
      >
        <ol class="flex items-center space-x-2">
          <li>
            <NuxtLink
              to="/"
              class="text-gray-500 hover:text-attic-500 dark:text-gray-400 transition-colors text-sm font-medium"
            >
              Dashboard
            </NuxtLink>
          </li>
          <li>
            <span class="text-gray-300 dark:text-gray-600">/</span>
          </li>
          <li>
            <NuxtLink
              to="/assets"
              class="text-gray-500 hover:text-attic-500 dark:text-gray-400 transition-colors text-sm font-medium"
            >
              Assets
            </NuxtLink>
          </li>
          <li>
            <span class="text-gray-300 dark:text-gray-600">/</span>
          </li>
          <li>
            <span
              aria-current="page"
              class="text-attic-500 font-medium text-sm"
            >Add New</span>
          </li>
        </ol>
      </nav>

      <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h2 class="text-3xl font-extrabold text-mist-950 dark:text-white tracking-tight">
            Catalog New Item
          </h2>
          <p class="text-gray-500 dark:text-gray-400 mt-1">
            Add a new asset to your inventory collection.
          </p>
        </div>
        <div class="flex items-center gap-3">
          <UButton
            to="/assets"
            variant="ghost"
            color="neutral"
            class="font-medium"
          >
            Cancel
          </UButton>
          <UButton
            :loading="loading"
            class="shadow-lg shadow-attic-500/20 font-bold"
            icon="i-lucide-save"
            @click="submitForm"
          >
            Save Asset
          </UButton>
        </div>
      </div>
    </div>

    <!-- Main Form Card -->
    <form
      class="bg-white dark:bg-mist-800 rounded-xl shadow-xl shadow-gray-200/50 dark:shadow-none border border-gray-100 dark:border-gray-700 overflow-hidden"
      @submit.prevent="submitForm"
    >
      <!-- Progress Indicator -->
      <div class="h-1 w-full bg-gray-100 dark:bg-gray-700">
        <div
          class="h-full bg-attic-500 rounded-r-full transition-all duration-500"
          :style="{ width: `${formProgress}%` }"
        />
      </div>

      <div class="p-6 sm:p-10 space-y-10">
        <!-- Section 1: Universal Essentials -->
        <section class="space-y-6">
          <div class="grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
            <!-- Name -->
            <div class="md:col-span-8 space-y-2">
              <label
                for="asset-name"
                class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
              >
                Asset Name <span class="text-amber-500">*</span>
              </label>
              <div class="relative group">
                <input
                  id="asset-name"
                  v-model="form.name"
                  type="text"
                  placeholder="e.g. Vintage Canon AE-1"
                  class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-800/50 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 sm:text-lg py-3 px-4 shadow-sm transition-colors placeholder:text-gray-400"
                  autofocus
                >
                <div class="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none opacity-0 group-focus-within:opacity-100 transition-opacity">
                  <UIcon
                    name="i-lucide-pencil"
                    class="w-5 h-5 text-attic-500"
                  />
                </div>
              </div>
            </div>

            <!-- Location -->
            <div class="md:col-span-4 space-y-2">
              <label
                for="location"
                class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
              >
                Location
              </label>
              <USelectMenu
                id="location"
                v-model="form.location_id"
                :items="locationOptions"
                placeholder="Select a space..."
                value-key="value"
                class="w-full"
                size="lg"
              />
            </div>
          </div>

          <!-- Category Grid -->
          <div class="space-y-3 pt-2">
            <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Category <span class="text-amber-500">*</span>
            </label>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
              <label
                v-for="cat in categories"
                :key="cat.id"
                class="cursor-pointer group relative"
              >
                <input
                  v-model="form.category_id"
                  type="radio"
                  name="category"
                  :value="cat.id"
                  class="peer sr-only"
                >
                <div class="flex flex-col items-center justify-center p-4 rounded-lg border-2 border-gray-100 dark:border-gray-700 hover:border-attic-500/50 dark:hover:border-attic-500/50 transition-all bg-white dark:bg-gray-800 h-28 gap-2 peer-checked:border-attic-500 peer-checked:bg-attic-50 dark:peer-checked:bg-attic-900/20">
                  <UIcon
                    name="i-lucide-folder"
                    class="w-8 h-8 text-gray-400 peer-checked:text-attic-500 group-hover:text-attic-500 transition-colors"
                  />
                  <span class="text-xs font-bold text-gray-600 dark:text-gray-300 text-center truncate w-full">{{ cat.name }}</span>
                </div>
                <div class="absolute top-2 right-2 opacity-0 peer-checked:opacity-100 text-attic-500 transition-opacity">
                  <UIcon
                    name="i-lucide-check-circle"
                    class="w-4 h-4"
                  />
                </div>
              </label>
            </div>
            <p
              v-if="!categories?.length"
              class="text-sm text-gray-400"
            >
              No categories available. <NuxtLink
                to="/categories"
                class="text-attic-500 hover:underline"
              >Create one first</NuxtLink>.
            </p>
          </div>
        </section>

        <hr class="border-t border-gray-100 dark:border-gray-700">

        <!-- Section 2: Additional Details -->
        <section class="space-y-6">
          <h3 class="text-lg font-bold text-mist-950 dark:text-white flex items-center gap-2">
            <UIcon
              name="i-lucide-clipboard-list"
              class="w-5 h-5 text-attic-500"
            />
            Additional Details
          </h3>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Condition -->
            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Condition
              </label>
              <USelectMenu
                v-model="form.condition_id"
                :items="conditionOptions"
                placeholder="Select condition..."
                value-key="value"
                class="w-full"
              />
            </div>

            <!-- Quantity -->
            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Quantity
              </label>
              <input
                v-model.number="form.quantity"
                type="number"
                min="1"
                class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm"
              >
            </div>
          </div>

          <!-- Description -->
          <div class="space-y-2">
            <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Description
            </label>
            <textarea
              v-model="form.description"
              rows="3"
              placeholder="Product description, features, specifications..."
              class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm resize-none placeholder:text-gray-400"
            />
          </div>

          <!-- Personal Notes -->
          <div class="space-y-2">
            <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Personal Notes
            </label>
            <textarea
              v-model="form.notes"
              rows="3"
              placeholder="Your personal notes: condition details, where you bought it, special memories..."
              class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm resize-none placeholder:text-gray-400"
            />
          </div>
        </section>

        <!-- Category Attributes -->
        <template v-if="selectedCategory?.attributes?.length">
          <hr class="border-t border-gray-100 dark:border-gray-700">

          <section class="space-y-6">
            <h3 class="text-lg font-bold text-mist-950 dark:text-white flex items-center gap-2">
              <UIcon
                name="i-lucide-sliders-horizontal"
                class="w-5 h-5 text-attic-500"
              />
              {{ selectedCategory.name }} Attributes
            </h3>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div
                v-for="ca in selectedCategory.attributes"
                :key="ca.attribute_id"
                class="space-y-2"
              >
                <template v-if="ca.attribute">
                  <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    {{ ca.attribute.name }}
                    <span
                      v-if="ca.required"
                      class="text-amber-500"
                    >*</span>
                  </label>

                  <!-- Boolean type: checkbox -->
                  <div
                    v-if="ca.attribute.data_type === 'boolean'"
                    class="flex items-center gap-2 py-2"
                  >
                    <input
                      :id="`attr-${ca.attribute.key}`"
                      v-model="form.attributes[ca.attribute.key]"
                      type="checkbox"
                      class="rounded border-gray-300 dark:border-gray-600 text-attic-500 focus:ring-attic-500"
                    >
                    <label
                      :for="`attr-${ca.attribute.key}`"
                      class="text-sm text-gray-600 dark:text-gray-300"
                    >
                      {{ ca.attribute.name }}
                    </label>
                  </div>

                  <!-- Text (long) type: textarea -->
                  <textarea
                    v-else-if="ca.attribute.data_type === 'text'"
                    :value="String(form.attributes[ca.attribute.key] ?? '')"
                    :placeholder="`Enter ${ca.attribute.name.toLowerCase()}`"
                    rows="3"
                    class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm resize-none placeholder:text-gray-400"
                    @input="form.attributes[ca.attribute.key] = ($event.target as HTMLTextAreaElement).value"
                  />

                  <!-- Other types: input -->
                  <input
                    v-else
                    v-model="form.attributes[ca.attribute.key]"
                    :type="getInputType(ca.attribute.data_type)"
                    :step="ca.attribute.data_type === 'number' ? 'any' : undefined"
                    :placeholder="`Enter ${ca.attribute.name.toLowerCase()}`"
                    class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm placeholder:text-gray-400"
                  >
                </template>
              </div>
            </div>
          </section>
        </template>

        <div
          v-else-if="form.category_id && !selectedCategory?.attributes?.length"
          class="text-center py-4 text-gray-400"
        >
          <p>This category has no custom attributes.</p>
        </div>

        <hr class="border-t border-gray-100 dark:border-gray-700">

        <!-- Section 3: Purchase Information -->
        <section class="space-y-6">
          <h3 class="text-lg font-bold text-mist-950 dark:text-white flex items-center gap-2">
            <UIcon
              name="i-lucide-receipt"
              class="w-5 h-5 text-attic-500"
            />
            Purchase Information
          </h3>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Purchase Date -->
            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Purchase Date
              </label>
              <input
                v-model="form.purchase_at"
                type="date"
                class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm"
              >
            </div>

            <!-- Purchase Price -->
            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Purchase Price
              </label>
              <div class="relative">
                <span class="absolute inset-y-0 left-0 flex items-center pl-4 text-gray-400 pointer-events-none">$</span>
                <input
                  v-model.number="form.purchase_price"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 pl-8 pr-4 shadow-sm placeholder:text-gray-400"
                >
              </div>
            </div>
          </div>

          <!-- Purchase Notes -->
          <div class="space-y-2">
            <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Purchase Notes
            </label>
            <textarea
              v-model="form.purchase_note"
              rows="3"
              placeholder="Store, receipt number, warranty info, etc."
              class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm resize-none placeholder:text-gray-400"
            />
          </div>
        </section>
      </div>

      <!-- Footer Action Area -->
      <div class="bg-gray-50 dark:bg-gray-800/50 px-6 sm:px-10 py-4 border-t border-gray-100 dark:border-gray-700 flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="text-xs text-gray-400 dark:text-gray-500">
          <span class="text-amber-500">*</span> Required fields
        </div>
        <div class="flex items-center gap-3">
          <UButton
            to="/assets"
            variant="ghost"
            color="neutral"
          >
            Cancel
          </UButton>
          <UButton
            type="submit"
            :loading="loading"
            class="shadow-lg shadow-attic-500/20"
            icon="i-lucide-save"
          >
            Save Asset
          </UButton>
        </div>
      </div>
    </form>

    <!-- Tip -->
    <div class="flex justify-center pb-8">
      <div class="max-w-lg text-center">
        <p class="text-sm text-gray-400 dark:text-gray-500 flex items-center justify-center gap-2">
          <UIcon
            name="i-lucide-lightbulb"
            class="w-4 h-4"
          />
          Tip: You can import assets using plugins from the Assets page.
        </p>
      </div>
    </div>
  </div>
</template>
