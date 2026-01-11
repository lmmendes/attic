<script setup lang="ts">
import type { Location } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const apiFetch = useApiFetch()

const { data: locations, refresh, status } = useApi<Location[]>('/api/locations')

const modalOpen = ref(false)
const editingLocation = ref<Location | null>(null)
const form = reactive({
  name: '',
  description: '',
  parent_id: undefined as string | undefined
})

function openCreateModal() {
  editingLocation.value = null
  form.name = ''
  form.description = ''
  form.parent_id = undefined
  modalOpen.value = true
}

function openEditModal(location: Location) {
  editingLocation.value = location
  form.name = location.name
  form.description = location.description || ''
  form.parent_id = location.parent_id
  modalOpen.value = true
}

async function saveLocation() {
  try {
    const url = editingLocation.value
      ? `/api/locations/${editingLocation.value.id}`
      : `/api/locations`

    await apiFetch(url, {
      method: editingLocation.value ? 'PUT' : 'POST',
      body: JSON.stringify(form)
    })

    toast.add({
      title: editingLocation.value ? 'Location updated' : 'Location created',
      color: 'success'
    })
    modalOpen.value = false
    refresh()
  } catch (error) {
    toast.add({ title: 'Failed to save location', color: 'error' })
  }
}

async function deleteLocation(location: Location) {
  if (!confirm(`Delete "${location.name}"?`)) return

  try {
    await apiFetch(`/api/locations/${location.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Location deleted', color: 'success' })
    refresh()
  } catch (error) {
    toast.add({ title: 'Failed to delete location', color: 'error' })
  }
}

const parentOptions = computed(() => [
  { label: 'None (Top Level)', value: undefined },
  ...(locations.value?.map(l => ({ label: l.name, value: l.id })) || [])
])

// Build a map of location id to location for quick lookup
const locationMap = computed(() => {
  const map = new Map<string, Location>()
  locations.value?.forEach(l => map.set(l.id, l))
  return map
})

// Get the full path/breadcrumb for a location
function getLocationPath(location: Location): string[] {
  const path: string[] = []
  let current: Location | undefined = location

  // Traverse up the tree, collecting names
  while (current) {
    path.unshift(current.name)
    if (current.parent_id) {
      current = locationMap.value.get(current.parent_id)
    } else {
      break
    }
  }

  return path
}
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold">Locations</h1>
        <UButton
          icon="i-lucide-plus"
          @click="openCreateModal"
        >
          Add Location
        </UButton>
      </div>

      <UCard>
        <UTable
          :data="locations || []"
          :columns="[
            { accessorKey: 'name', id: 'name', header: 'Location' },
            { accessorKey: 'description', id: 'description', header: 'Description' },
            { id: 'actions', header: '' }
          ]"
          :loading="status === 'pending'"
        >
          <template #name-cell="{ row }">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-map-pin" class="w-4 h-4 text-muted flex-shrink-0" />
              <div class="flex items-center gap-1 text-sm">
                <template v-for="(segment, index) in getLocationPath(row.original)" :key="index">
                  <span
                    :class="index === getLocationPath(row.original).length - 1 ? 'font-medium' : 'text-muted'"
                  >
                    {{ segment }}
                  </span>
                  <UIcon
                    v-if="index < getLocationPath(row.original).length - 1"
                    name="i-lucide-chevron-right"
                    class="w-3 h-3 text-muted"
                  />
                </template>
              </div>
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
                @click="deleteLocation(row.original)"
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
                {{ editingLocation ? 'Edit Location' : 'New Location' }}
              </h3>
            </template>

            <form class="space-y-4" @submit.prevent="saveLocation">
              <UFormField label="Name" required>
                <UInput v-model="form.name" placeholder="Location name" />
              </UFormField>

              <UFormField label="Description">
                <UTextarea v-model="form.description" placeholder="Optional description" />
              </UFormField>

              <UFormField label="Parent Location">
                <USelectMenu
                  v-model="form.parent_id"
                  :items="parentOptions"
                  placeholder="Select parent location"
                  value-key="value"
                />
              </UFormField>
            </form>

            <template #footer>
              <div class="flex justify-end gap-2">
                <UButton variant="ghost" @click="modalOpen = false">
                  Cancel
                </UButton>
                <UButton @click="saveLocation">
                  {{ editingLocation ? 'Update' : 'Create' }}
                </UButton>
              </div>
            </template>
          </UCard>
        </template>
      </UModal>
    </div>
  </UContainer>
</template>
