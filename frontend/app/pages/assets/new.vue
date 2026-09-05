<script setup lang="ts">
import type { Category, Location, Condition } from '~/types/api'
import { isValidAssetQuantity, QUANTITY_VALIDATION_MESSAGE } from '~/utils/assetValidation'

definePageMeta({
  middleware: 'auth'
})

const router = useRouter()
const route = useRoute()
const toast = useToast()
const apiFetch = useApiFetch()

const { data: categories } = useApi<Category[]>('/api/categories')
const { data: locations } = useApi<Location[]>('/api/locations')
const { data: conditions } = useApi<Condition[]>('/api/conditions')

const loading = ref(false)
const selectedCategory = ref<Category | null>(null)
const form = reactive({
  collection_ids: [] as string[],
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

watch(() => route.query.location_id, (locationId) => {
  if (typeof locationId === 'string' && !form.location_id) {
    form.location_id = locationId
  }
}, { immediate: true })

const _categoryOptions = computed(() =>
  categories.value?.map(c => ({ label: c.name, value: c.id, icon: c.icon })) || []
)

interface LocationOption {
  label: string
  value: string
}

interface LocationTreeNode {
  location: Location
  children: LocationTreeNode[]
}

function buildHierarchicalLocationOptions(items: Location[]): LocationOption[] {
  const childrenMap = new Map<string | undefined, Location[]>()
  items.forEach((location) => {
    const parentId = location.parent_id || undefined
    if (!childrenMap.has(parentId)) {
      childrenMap.set(parentId, [])
    }
    childrenMap.get(parentId)!.push(location)
  })

  const buildTree = (parentId: string | undefined): LocationTreeNode[] => {
    const children = childrenMap.get(parentId) || []
    return children
      .sort((a, b) => a.name.localeCompare(b.name))
      .map(location => ({
        location,
        children: buildTree(location.id)
      }))
  }

  const flattenTree = (nodes: LocationTreeNode[], depth = 0): LocationOption[] => {
    return nodes.flatMap((node) => {
      const indent = depth > 0 ? `${'\u00A0'.repeat(depth * 2)}└ ` : ''
      return [
        { label: `${indent}${node.location.name}`, value: node.location.id },
        ...flattenTree(node.children, depth + 1)
      ]
    })
  }

  return flattenTree(buildTree(undefined))
}

const locationOptions = computed(() => [
  { label: 'No location', value: undefined },
  ...(locations.value ? buildHierarchicalLocationOptions(locations.value) : [])
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

function revealInvalidSection(event: Event) {
  (event.currentTarget as HTMLDetailsElement).open = true
}

async function submitForm() {
  if (loading.value) return
  if (!form.name.trim() || !form.category_id) {
    toast.add({ title: 'Name and category are required', color: 'error' })
    return
  }

  if (!isValidAssetQuantity(form.quantity)) {
    toast.add({ title: QUANTITY_VALIDATION_MESSAGE, color: 'error' })
    return
  }

  loading.value = true
  try {
    const payload = {
      name: form.name.trim(),
      description: form.description || undefined,
      category_id: form.category_id,
      location_id: form.location_id || undefined,
      collection_ids: form.collection_ids,
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
</script>

<template>
  <div class="mx-auto max-w-[1040px] space-y-5 pb-24 sm:pb-6">
    <!-- Breadcrumbs & Heading -->
    <div class="space-y-3">
      <nav
        aria-label="Breadcrumb"
        class="flex text-xs"
      >
        <ol class="flex items-center gap-1.5">
          <li>
            <NuxtLink
              to="/"
              class="font-semibold text-muted transition-colors hover:text-attic-500"
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
              class="font-semibold text-muted transition-colors hover:text-attic-500"
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
              class="font-bold text-attic-500"
            >Add New</span>
          </li>
        </ol>
      </nav>

      <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div>
          <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
            New inventory item
          </p>
          <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
            Add an asset
          </h1>
          <p class="mt-1 text-sm text-muted">
            Give it a name and a home. Add more details whenever you like.
          </p>
        </div>
        <div class="hidden items-center gap-2 sm:flex">
          <UButton
            to="/assets"
            variant="ghost"
            color="neutral"
            class="rounded-xl font-semibold"
          >
            Cancel
          </UButton>
          <UButton
            :loading="loading"
            class="rounded-xl font-bold shadow-primary"
            icon="i-lucide-save"
            type="submit"
            form="new-asset-form"
          >
            Save Asset
          </UButton>
        </div>
      </div>
    </div>

    <!-- Main Form Card -->
    <form
      id="new-asset-form"
      class="space-y-5"
      @submit.prevent="submitForm"
    >
      <div class="space-y-5">
        <!-- Section 1: Universal Essentials -->
        <section class="attic-panel space-y-5 rounded-[20px] p-5 sm:p-6">
          <div class="flex items-start gap-3">
            <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-attic-50 text-attic-500 dark:bg-attic-500/10">
              <UIcon
                name="i-lucide-package"
                class="size-4.5"
              />
            </div>
            <div>
              <h2 class="font-extrabold text-mist-950 dark:text-white">
                Basic information
              </h2>
              <p class="text-xs text-muted">
                Name the asset and choose where it belongs.
              </p>
            </div>
          </div>
          <div class="grid grid-cols-1 items-start gap-5 md:grid-cols-12">
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
                  required
                  type="text"
                  placeholder="e.g. Vintage Canon AE-1"
                  class="block w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-mist-950 shadow-sm transition-colors placeholder:text-dimmed focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
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
                Stored in
              </label>
              <USelectMenu
                id="location"
                v-model="form.location_id"
                aria-label="Stored in"
                :items="locationOptions"
                placeholder="Choose a room, shelf, or box"
                value-key="value"
                class="w-full"
                size="lg"
              />
            </div>
          </div>

          <!-- Category Grid -->
          <div class="space-y-3">
            <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Category <span class="text-amber-500">*</span>
            </label>
            <div class="flex flex-wrap gap-2">
              <label
                v-for="cat in categories"
                :key="cat.id"
                class="group relative min-w-36 cursor-pointer"
              >
                <input
                  v-model="form.category_id"
                  type="radio"
                  name="category"
                  required
                  :value="cat.id"
                  class="peer sr-only"
                >
                <div class="flex h-11 items-center gap-2 rounded-xl border border-mist-200 bg-mist-50/50 px-3 pr-8 transition-all hover:border-attic-300 hover:bg-attic-50/50 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-attic-500 peer-checked:border-attic-500 peer-checked:bg-attic-50 peer-checked:ring-2 peer-checked:ring-attic-500/10 dark:border-mist-700 dark:bg-mist-800 dark:peer-checked:bg-attic-500/10">
                  <UIcon
                    name="i-lucide-folder"
                    class="size-4 shrink-0 text-muted transition-colors group-hover:text-attic-500"
                  />
                  <span class="truncate text-xs font-bold text-mist-600 dark:text-mist-300">{{ cat.name }}</span>
                </div>
                <div class="absolute right-2.5 top-3 text-attic-500 opacity-0 transition-opacity peer-checked:opacity-100">
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
          <AssetCollectionsField v-model="form.collection_ids" />
        </section>

        <hr class="hidden">

        <!-- Section 2: Additional Details -->
        <details
          class="attic-panel space-y-5 rounded-[20px] p-5 sm:p-6"
          @invalid.capture="revealInvalidSection"
        >
          <summary class="flex cursor-pointer list-none items-start gap-3 rounded-lg focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-attic-500 [&::-webkit-details-marker]:hidden">
            <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400">
              <UIcon
                name="i-lucide-clipboard-list"
                class="size-4.5"
              />
            </div>
            <div>
              <h2 class="font-extrabold text-mist-950 dark:text-white">
                Details and notes
              </h2>
              <p class="text-xs text-muted">
                Optional context that makes the asset easier to identify later.
              </p>
            </div>
            <UIcon
              name="i-lucide-chevron-down"
              class="ml-auto mt-2 size-4 shrink-0 text-muted transition-transform in-open:rotate-180"
            />
          </summary>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Condition -->
            <div class="space-y-2">
              <label
                for="asset-condition"
                class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
              >
                Condition
              </label>
              <USelectMenu
                id="asset-condition"
                v-model="form.condition_id"
                aria-label="Condition"
                :items="conditionOptions"
                placeholder="Select condition..."
                value-key="value"
                class="w-full"
                size="lg"
              />
            </div>

            <!-- Quantity -->
            <div class="space-y-2">
              <label
                for="asset-quantity"
                class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
              >
                Quantity
              </label>
              <input
                id="asset-quantity"
                v-model.number="form.quantity"
                type="number"
                min="1"
                class="block w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
              >
            </div>
          </div>

          <!-- Description -->
          <div class="space-y-2">
            <label
              for="asset-description"
              class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
            >
              Description
            </label>
            <textarea
              id="asset-description"
              v-model="form.description"
              rows="4"
              placeholder="Product description, features, specifications..."
              class="block w-full resize-none rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm placeholder:text-dimmed focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
            />
          </div>

          <!-- Personal Notes -->
          <div class="space-y-2">
            <label
              for="asset-notes"
              class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
            >
              Personal Notes
            </label>
            <textarea
              id="asset-notes"
              v-model="form.notes"
              rows="3"
              placeholder="Your personal notes: condition details, where you bought it, special memories..."
              class="block w-full resize-none rounded-xl border border-terracotta-100 bg-terracotta-50/40 px-4 py-3 text-sm text-mist-950 shadow-sm placeholder:text-dimmed focus:border-terracotta-300 focus:ring-terracotta-300 dark:border-terracotta-900/40 dark:bg-terracotta-950/10 dark:text-white"
            />
          </div>
        </details>

        <!-- Category Attributes -->
        <template v-if="selectedCategory?.attributes?.length">
          <hr class="hidden">

          <section class="attic-panel space-y-5 rounded-[20px] p-5 sm:p-6">
            <div class="flex items-start gap-3">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-terracotta-50 text-terracotta-500 dark:bg-terracotta-500/10 dark:text-terracotta-300">
                <UIcon
                  name="i-lucide-sliders-horizontal"
                  class="size-4.5"
                />
              </div>
              <div>
                <h2 class="font-extrabold text-mist-950 dark:text-white">
                  {{ selectedCategory.name }} attributes
                </h2>
                <p class="text-xs text-muted">
                  Category-specific information for this asset.
                </p>
              </div>
            </div>

            <div class="max-w-3xl space-y-5">
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
                    rows="4"
                    class="block w-full resize-none rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm placeholder:text-dimmed focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
                    @input="form.attributes[ca.attribute.key] = ($event.target as HTMLTextAreaElement).value"
                  />

                  <!-- Other types: input -->
                  <input
                    v-else
                    v-model="form.attributes[ca.attribute.key]"
                    :type="getInputType(ca.attribute.data_type)"
                    :step="ca.attribute.data_type === 'number' ? 'any' : undefined"
                    :placeholder="`Enter ${ca.attribute.name.toLowerCase()}`"
                    class="block w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm placeholder:text-dimmed focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
                  >
                </template>
              </div>
            </div>
          </section>
        </template>

        <hr class="hidden">

        <!-- Section 3: Purchase Information -->
        <details
          class="attic-panel space-y-5 rounded-[20px] p-5 sm:p-6"
          @invalid.capture="revealInvalidSection"
        >
          <summary class="flex cursor-pointer list-none items-start gap-3 rounded-lg focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-attic-500 [&::-webkit-details-marker]:hidden">
            <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400">
              <UIcon
                name="i-lucide-receipt"
                class="size-4.5"
              />
            </div>
            <div>
              <h2 class="font-extrabold text-mist-950 dark:text-white">
                Purchase information
              </h2>
              <p class="text-xs text-muted">
                Optional cost and purchase records.
              </p>
            </div>
            <UIcon
              name="i-lucide-chevron-down"
              class="ml-auto mt-2 size-4 shrink-0 text-muted transition-transform in-open:rotate-180"
            />
          </summary>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Purchase Date -->
            <div class="space-y-2">
              <label
                for="asset-purchase-at"
                class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
              >
                Purchase Date
              </label>
              <input
                id="asset-purchase-at"
                v-model="form.purchase_at"
                type="date"
                class="block w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
              >
            </div>

            <!-- Purchase Price -->
            <div class="space-y-2">
              <label
                for="asset-purchase-price"
                class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
              >
                Purchase Price
              </label>
              <div class="relative">
                <span class="absolute inset-y-0 left-0 flex items-center pl-4 text-gray-400 pointer-events-none">$</span>
                <input
                  id="asset-purchase-price"
                  v-model.number="form.purchase_price"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="block w-full rounded-xl border border-mist-200 bg-white py-3 pl-8 pr-4 text-sm text-mist-950 shadow-sm placeholder:text-dimmed focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
                >
              </div>
            </div>
          </div>

          <!-- Purchase Notes -->
          <div class="space-y-2">
            <label
              for="asset-purchase-note"
              class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider"
            >
              Purchase Notes
            </label>
            <textarea
              id="asset-purchase-note"
              v-model="form.purchase_note"
              rows="3"
              placeholder="Store, receipt number, warranty info, etc."
              class="block w-full resize-none rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm placeholder:text-dimmed focus:border-attic-500 focus:ring-attic-500 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
            />
          </div>
        </details>
      </div>

      <!-- Footer Action Area -->
      <div class="fixed inset-x-0 bottom-0 z-30 flex items-center justify-between gap-3 border-t border-subtle bg-surface px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:sticky sm:bottom-0 sm:rounded-[18px] sm:border sm:px-4 sm:py-3">
        <div class="text-xs text-muted">
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
            class="min-h-11 rounded-xl shadow-primary"
            icon="i-lucide-check"
          >
            Save Asset
          </UButton>
        </div>
      </div>
    </form>
  </div>
</template>
