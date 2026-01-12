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
  purchase_note: ''
})

const categoryOptions = computed(() =>
  categories.value?.map(c => ({ label: c.name, value: c.id })) || []
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
      purchase_note: form.purchase_note || undefined
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
  <UContainer>
    <div class="py-8 max-w-2xl mx-auto">
      <div class="flex items-center gap-4 mb-6">
        <UButton
          to="/assets"
          variant="ghost"
          icon="i-lucide-arrow-left"
        />
        <h1 class="text-2xl font-bold">
          New Asset
        </h1>
      </div>

      <form @submit.prevent="submitForm">
        <UCard>
          <div class="space-y-6">
            <!-- Basic Info -->
            <div class="space-y-4">
              <h3 class="font-medium text-lg">
                Basic Information
              </h3>

              <UFormField
                label="Name"
                required
              >
                <UInput
                  v-model="form.name"
                  placeholder="Enter asset name"
                  autofocus
                />
              </UFormField>

              <UFormField label="Description">
                <UTextarea
                  v-model="form.description"
                  placeholder="Optional description"
                  :rows="3"
                />
              </UFormField>

              <div class="grid grid-cols-2 gap-4">
                <UFormField
                  label="Category"
                  required
                >
                  <USelectMenu
                    v-model="form.category_id"
                    :items="categoryOptions"
                    placeholder="Select category"
                    value-key="value"
                  />
                </UFormField>

                <UFormField label="Quantity">
                  <UInput
                    v-model.number="form.quantity"
                    type="number"
                    min="1"
                  />
                </UFormField>
              </div>

              <div class="grid grid-cols-2 gap-4">
                <UFormField label="Location">
                  <USelectMenu
                    v-model="form.location_id"
                    :items="locationOptions"
                    placeholder="Select location"
                    value-key="value"
                  />
                </UFormField>

                <UFormField label="Condition">
                  <USelectMenu
                    v-model="form.condition_id"
                    :items="conditionOptions"
                    placeholder="Select condition"
                    value-key="value"
                  />
                </UFormField>
              </div>
            </div>

            <!-- Category Attributes -->
            <div
              v-if="selectedCategory?.attributes?.length"
              class="space-y-4"
            >
              <USeparator />
              <h3 class="font-medium text-lg">
                {{ selectedCategory.name }} Attributes
              </h3>

              <div class="space-y-3">
                <div
                  v-for="ca in selectedCategory.attributes"
                  :key="ca.attribute_id"
                >
                  <UFormField
                    v-if="ca.attribute"
                    :label="ca.attribute.name"
                    :required="ca.required"
                  >
                    <!-- Boolean type: checkbox -->
                    <UCheckbox
                      v-if="ca.attribute.data_type === 'boolean'"
                      :model-value="form.attributes[ca.attribute.key] as boolean"
                      :label="ca.attribute.name"
                      @update:model-value="form.attributes[ca.attribute.key] = $event"
                    />
                    <!-- Text (long) type: textarea -->
                    <UTextarea
                      v-else-if="ca.attribute.data_type === 'text'"
                      :model-value="form.attributes[ca.attribute.key] as string"
                      :placeholder="`Enter ${ca.attribute.name.toLowerCase()}`"
                      :rows="3"
                      @update:model-value="form.attributes[ca.attribute.key] = $event"
                    />
                    <!-- Other types: input -->
                    <UInput
                      v-else
                      v-model="form.attributes[ca.attribute.key]"
                      :type="getInputType(ca.attribute.data_type)"
                      :placeholder="`Enter ${ca.attribute.name.toLowerCase()}`"
                    />
                  </UFormField>
                </div>
              </div>
            </div>

            <div
              v-else-if="form.category_id"
              class="text-center py-4 text-muted"
            >
              <p>This category has no custom attributes.</p>
            </div>

            <!-- Purchase Information -->
            <div class="space-y-4">
              <USeparator />
              <h3 class="font-medium text-lg">
                Purchase Information
              </h3>

              <div class="grid grid-cols-2 gap-4">
                <UFormField label="Purchase Date">
                  <UInput
                    v-model="form.purchase_at"
                    type="date"
                  />
                </UFormField>

                <UFormField label="Purchase Price">
                  <UInput
                    v-model.number="form.purchase_price"
                    type="number"
                    step="0.01"
                    min="0"
                    placeholder="0.00"
                  >
                    <template #leading>
                      <span class="text-muted">$</span>
                    </template>
                  </UInput>
                </UFormField>
              </div>

              <UFormField label="Purchase Notes">
                <UTextarea
                  v-model="form.purchase_note"
                  placeholder="Store, receipt number, etc."
                  :rows="3"
                />
              </UFormField>
            </div>
          </div>

          <template #footer>
            <div class="flex justify-end gap-2">
              <UButton
                to="/assets"
                variant="ghost"
              >
                Cancel
              </UButton>
              <UButton
                :loading="loading"
                @click="submitForm"
              >
                Create Asset
              </UButton>
            </div>
          </template>
        </UCard>
      </form>
    </div>
  </UContainer>
</template>
