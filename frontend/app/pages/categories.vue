<script setup lang="ts">
import type { Category, Attribute } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: categories, refresh, status } = useApi<Category[]>('/api/categories')
const { data: attributes } = useApi<Attribute[]>('/api/attributes')

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
  } catch (error) {
    toast.add({ title: 'Failed to load category', color: 'error' })
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
  } catch (error) {
    toast.add({ title: 'Failed to save category', color: 'error' })
  }
}

async function deleteCategory(category: Category) {
  if (!confirm(`Delete "${category.name}"?`)) return

  try {
    await apiFetch(`/api/categories/${category.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Category deleted', color: 'success' })
    refresh()
  } catch (error) {
    toast.add({ title: 'Failed to delete category', color: 'error' })
  }
}

const parentOptions = computed(() => [
  { label: 'None (Top Level)', value: undefined },
  ...(categories.value?.map(c => ({ label: c.name, value: c.id })) || [])
])
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold">Categories</h1>
        <UButton
          icon="i-lucide-plus"
          @click="openCreateModal"
        >
          Add Category
        </UButton>
      </div>

      <UCard>
        <UTable
          :data="categories || []"
          :columns="[
            { accessorKey: 'name', id: 'name', header: 'Name' },
            { accessorKey: 'description', id: 'description', header: 'Description' },
            { id: 'actions', header: '' }
          ]"
          :loading="status === 'pending'"
        >
          <template #name-cell="{ row }">
            <div class="flex items-center gap-2">
              <UIcon
                v-if="row.original.icon"
                :name="row.original.icon"
                class="w-4 h-4"
              />
              <span class="font-medium">{{ row.original.name }}</span>
            </div>
          </template>
          <template #actions-cell="{ row }">
            <div class="flex gap-1">
              <UButton
                variant="ghost"
                icon="i-lucide-edit"
                size="sm"
                @click="openEditModal(row.original)"
              />
              <UButton
                variant="ghost"
                icon="i-lucide-trash-2"
                size="sm"
                color="error"
                @click="deleteCategory(row.original)"
              />
            </div>
          </template>
        </UTable>
      </UCard>

      <!-- Create/Edit Modal -->
      <UModal
        v-model:open="modalOpen"
        :title="editingCategory ? 'Edit Category' : 'New Category'"
      >
        <template #body>
          <div class="space-y-4">
            <UFormField label="Name" required>
              <UInput v-model="form.name" placeholder="Category name" />
            </UFormField>

            <UFormField label="Description">
              <UTextarea v-model="form.description" placeholder="Optional description" />
            </UFormField>

            <UFormField label="Parent Category">
              <USelectMenu
                v-model="form.parent_id"
                :items="parentOptions"
                placeholder="Select parent category"
                value-key="value"
              />
            </UFormField>

            <!-- Attributes Section -->
            <div>
              <div class="flex items-center justify-between mb-2">
                <label class="block text-sm font-medium">Attributes</label>
                <UButton
                  v-if="availableAttributes.length > 0"
                  size="xs"
                  variant="ghost"
                  icon="i-lucide-plus"
                  @click="addAttribute"
                >
                  Add
                </UButton>
              </div>

              <div v-if="form.attributes.length === 0" class="text-sm text-gray-500 py-2">
                No attributes assigned. Click "Add" to assign attributes.
              </div>

              <div v-else class="space-y-2">
                <div
                  v-for="(attr, index) in form.attributes"
                  :key="attr.attribute_id"
                  class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-800 rounded"
                >
                  <USelectMenu
                    v-model="attr.attribute_id"
                    :items="[
                      { label: getAttributeName(attr.attribute_id), value: attr.attribute_id },
                      ...availableAttributes.map(a => ({ label: a.name, value: a.id }))
                    ]"
                    class="flex-1"
                    value-key="value"
                  />
                  <UCheckbox
                    v-model="attr.required"
                    label="Required"
                  />
                  <UButton
                    variant="ghost"
                    icon="i-lucide-trash-2"
                    size="xs"
                    color="error"
                    @click="removeAttribute(index)"
                  />
                </div>
              </div>

              <p v-if="attributes?.length === 0" class="text-sm text-gray-500 mt-2">
                <NuxtLink to="/attributes" class="text-primary hover:underline">
                  Create attributes
                </NuxtLink>
                first to assign them to categories.
              </p>
            </div>
          </div>
        </template>

        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="modalOpen = false">
              Cancel
            </UButton>
            <UButton @click="saveCategory">
              {{ editingCategory ? 'Save' : 'Create' }}
            </UButton>
          </div>
        </template>
      </UModal>
    </div>
  </UContainer>
</template>
