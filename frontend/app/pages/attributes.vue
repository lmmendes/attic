<script setup lang="ts">
import type { Attribute, AttributeDataType } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: attributes, refresh, status } = useApi<Attribute[]>('/api/attributes')

const modalOpen = ref(false)
const editingAttribute = ref<Attribute | null>(null)
const form = reactive({
  name: '',
  key: '',
  data_type: 'string' as AttributeDataType
})

const dataTypeOptions = [
  { label: 'Text (short)', value: 'string' },
  { label: 'Text (long)', value: 'text' },
  { label: 'Number', value: 'number' },
  { label: 'Yes/No', value: 'boolean' },
  { label: 'Date', value: 'date' }
]

// Auto-generate key from name
function generateKey(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

watch(() => form.name, (name) => {
  // Only auto-generate key when creating (not editing)
  if (!editingAttribute.value) {
    form.key = generateKey(name)
  }
})

function openCreateModal() {
  editingAttribute.value = null
  form.name = ''
  form.key = ''
  form.data_type = 'string'
  modalOpen.value = true
}

function openEditModal(attribute: Attribute) {
  editingAttribute.value = attribute
  form.name = attribute.name
  form.key = attribute.key
  form.data_type = attribute.data_type
  modalOpen.value = true
}

async function saveAttribute() {
  if (!form.name || !form.key) {
    toast.add({ title: 'Name and key are required', color: 'error' })
    return
  }

  try {
    const url = editingAttribute.value
      ? `/api/attributes/${editingAttribute.value.id}`
      : `/api/attributes`

    await apiFetch(url, {
      method: editingAttribute.value ? 'PUT' : 'POST',
      body: JSON.stringify({
        name: form.name,
        key: form.key,
        data_type: form.data_type
      })
    })

    toast.add({
      title: editingAttribute.value ? 'Attribute updated' : 'Attribute created',
      color: 'success'
    })
    modalOpen.value = false
    refresh()
  } catch {
    toast.add({ title: 'Failed to save attribute', color: 'error' })
  }
}

async function deleteAttribute(attribute: Attribute) {
  if (!confirm(`Delete "${attribute.name}"? This may affect categories using this attribute.`)) return

  try {
    await apiFetch(`/api/attributes/${attribute.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Attribute deleted', color: 'success' })
    refresh()
  } catch {
    toast.add({ title: 'Failed to delete attribute', color: 'error' })
  }
}

function formatDataType(type: string): string {
  const labels: Record<string, string> = {
    string: 'Text (short)',
    text: 'Text (long)',
    number: 'Number',
    boolean: 'Yes/No',
    date: 'Date'
  }
  return labels[type] || type
}
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-2xl font-bold">
            Attributes
          </h1>
          <p class="text-sm text-gray-500 mt-1">
            Define reusable attributes that can be assigned to categories
          </p>
        </div>
        <UButton
          icon="i-lucide-plus"
          @click="openCreateModal"
        >
          Add Attribute
        </UButton>
      </div>

      <UCard>
        <UTable
          :data="attributes || []"
          :columns="[
            { accessorKey: 'name', id: 'name', header: 'Name' },
            { accessorKey: 'key', id: 'key', header: 'Key' },
            { accessorKey: 'data_type', id: 'data_type', header: 'Type' },
            { id: 'actions', header: '' }
          ]"
          :loading="status === 'pending'"
        >
          <template #name-cell="{ row }">
            <span class="font-medium">{{ row.original.name }}</span>
          </template>
          <template #key-cell="{ row }">
            <code class="text-sm bg-gray-100 dark:bg-gray-800 px-2 py-0.5 rounded">
              {{ row.original.key }}
            </code>
          </template>
          <template #data_type-cell="{ row }">
            <UBadge variant="subtle">
              {{ formatDataType(row.original.data_type) }}
            </UBadge>
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
                @click="deleteAttribute(row.original)"
              />
            </div>
          </template>
        </UTable>
      </UCard>

      <!-- Create/Edit Modal -->
      <UModal
        v-model:open="modalOpen"
        :title="editingAttribute ? 'Edit Attribute' : 'New Attribute'"
      >
        <template #body>
          <div class="space-y-4">
            <UFormField
              label="Name"
              required
            >
              <UInput
                v-model="form.name"
                placeholder="e.g., Serial Number"
              />
            </UFormField>

            <UFormField
              label="Key"
              required
            >
              <UInput
                v-model="form.key"
                placeholder="e.g., serial_number"
                :disabled="!!editingAttribute"
              />
              <template #hint>
                <span class="text-xs text-gray-500">
                  Unique identifier used in the database. Cannot be changed after creation.
                </span>
              </template>
            </UFormField>

            <UFormField
              label="Data Type"
              required
            >
              <USelectMenu
                v-model="form.data_type"
                :items="dataTypeOptions"
                placeholder="Select data type"
                value-key="value"
              />
            </UFormField>
          </div>
        </template>

        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton
              variant="ghost"
              @click="modalOpen = false"
            >
              Cancel
            </UButton>
            <UButton @click="saveAttribute">
              {{ editingAttribute ? 'Save' : 'Create' }}
            </UButton>
          </div>
        </template>
      </UModal>
    </div>
  </UContainer>
</template>
