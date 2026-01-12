<script setup lang="ts">
import type { Condition } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: conditions, refresh, status } = useApi<Condition[]>('/api/conditions')

const modalOpen = ref(false)
const editingCondition = ref<Condition | null>(null)
const form = reactive({
  code: '',
  label: '',
  description: '',
  sort_order: 0
})

// Auto-generate code from label
function generateCode(label: string): string {
  return label.toUpperCase().replace(/[^A-Z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

watch(() => form.label, (label) => {
  // Only auto-generate code when creating (not editing)
  if (!editingCondition.value) {
    form.code = generateCode(label)
  }
})

function openCreateModal() {
  editingCondition.value = null
  form.code = ''
  form.label = ''
  form.description = ''
  form.sort_order = (conditions.value?.length || 0) + 1
  modalOpen.value = true
}

function openEditModal(condition: Condition) {
  editingCondition.value = condition
  form.code = condition.code
  form.label = condition.label
  form.description = condition.description || ''
  form.sort_order = condition.sort_order
  modalOpen.value = true
}

async function saveCondition() {
  if (!form.code || !form.label) {
    toast.add({ title: 'Code and label are required', color: 'error' })
    return
  }

  try {
    const url = editingCondition.value
      ? `/api/conditions/${editingCondition.value.id}`
      : `/api/conditions`

    await apiFetch(url, {
      method: editingCondition.value ? 'PUT' : 'POST',
      body: JSON.stringify(form)
    })

    toast.add({
      title: editingCondition.value ? 'Condition updated' : 'Condition created',
      color: 'success'
    })
    modalOpen.value = false
    refresh()
  } catch {
    toast.add({ title: 'Failed to save condition', color: 'error' })
  }
}

async function deleteCondition(condition: Condition) {
  if (!confirm(`Delete "${condition.label}"?`)) return

  try {
    await apiFetch(`/api/conditions/${condition.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Condition deleted', color: 'success' })
    refresh()
  } catch {
    toast.add({ title: 'Failed to delete condition', color: 'error' })
  }
}
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold">
          Conditions
        </h1>
        <UButton
          icon="i-lucide-plus"
          @click="openCreateModal"
        >
          Add Condition
        </UButton>
      </div>

      <UCard>
        <UTable
          :data="conditions || []"
          :columns="[
            { accessorKey: 'label', id: 'label', header: 'Label' },
            { accessorKey: 'code', id: 'code', header: 'Code' },
            { accessorKey: 'description', id: 'description', header: 'Description' },
            { accessorKey: 'sort_order', id: 'sort_order', header: 'Order' },
            { id: 'actions', header: '' }
          ]"
          :loading="status === 'pending'"
        >
          <template #label-cell="{ row }">
            <span class="font-medium">{{ row.original.label }}</span>
          </template>
          <template #code-cell="{ row }">
            <code class="text-sm bg-gray-100 dark:bg-gray-800 px-2 py-0.5 rounded">
              {{ row.original.code }}
            </code>
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
                @click="deleteCondition(row.original)"
              />
            </div>
          </template>
        </UTable>
      </UCard>

      <!-- Create/Edit Modal -->
      <UModal v-model:open="modalOpen">
        <template #content>
          <UCard>
            <template #header>
              <h3 class="font-semibold">
                {{ editingCondition ? 'Edit Condition' : 'New Condition' }}
              </h3>
            </template>

            <form
              class="space-y-4"
              @submit.prevent="saveCondition"
            >
              <UFormField
                label="Label"
                required
              >
                <UInput
                  v-model="form.label"
                  placeholder="e.g., New, Good, Fair"
                />
              </UFormField>

              <UFormField
                label="Code"
                required
              >
                <UInput
                  v-model="form.code"
                  placeholder="e.g., NEW, GOOD, FAIR"
                  :disabled="!!editingCondition"
                />
                <template #hint>
                  <span class="text-xs text-gray-500">
                    Unique identifier. Auto-generated from label.
                  </span>
                </template>
              </UFormField>

              <UFormField label="Description">
                <UTextarea
                  v-model="form.description"
                  placeholder="Optional description"
                />
              </UFormField>

              <UFormField label="Sort Order">
                <UInput
                  v-model.number="form.sort_order"
                  type="number"
                  min="1"
                />
              </UFormField>
            </form>

            <template #footer>
              <div class="flex justify-end gap-2">
                <UButton
                  variant="ghost"
                  @click="modalOpen = false"
                >
                  Cancel
                </UButton>
                <UButton @click="saveCondition">
                  {{ editingCondition ? 'Update' : 'Create' }}
                </UButton>
              </div>
            </template>
          </UCard>
        </template>
      </UModal>
    </div>
  </UContainer>
</template>
