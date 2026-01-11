<script setup lang="ts">
import type { Asset, Warranty, Attachment, Category } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

const { data: asset, refresh } = useApi<Asset>(() => `/api/assets/${route.params.id}`)
const { data: warranty, refresh: refreshWarranty } = useApi<Warranty>(() => `/api/assets/${route.params.id}/warranty`)
const { data: attachments, refresh: refreshAttachments } = useApi<Attachment[]>(
  () => `/api/assets/${route.params.id}/attachments`
)

// Fetch category with attribute definitions when asset loads
const categoryWithAttrs = ref<Category | null>(null)

watch(() => asset.value?.category_id, async (categoryId) => {
  if (categoryId) {
    try {
      categoryWithAttrs.value = await apiFetch<Category>(`/api/categories/${categoryId}`)
    } catch {
      categoryWithAttrs.value = null
    }
  }
}, { immediate: true })

// Get attribute value formatted for display
function getAttributeValue(key: string): string {
  const value = asset.value?.attributes?.[key]
  if (value === undefined || value === null || value === '') return '-'
  if (typeof value === 'boolean') return value ? 'Yes' : 'No'
  return String(value)
}

const deleteModalOpen = ref(false)
const warrantyModalOpen = ref(false)
const uploadModalOpen = ref(false)
const config = useRuntimeConfig()

// Warranty form
const warrantyForm = reactive({
  provider: '',
  policy_number: '',
  start_date: '',
  end_date: '',
  notes: ''
})

// Initialize warranty form when warranty data loads
watch(warranty, (w) => {
  if (w) {
    warrantyForm.provider = w.provider || ''
    warrantyForm.policy_number = w.policy_number || ''
    warrantyForm.start_date = w.start_date?.split('T')[0] || ''
    warrantyForm.end_date = w.end_date?.split('T')[0] || ''
    warrantyForm.notes = w.notes || ''
  }
}, { immediate: true })

function openWarrantyModal() {
  if (!warranty.value) {
    warrantyForm.provider = ''
    warrantyForm.policy_number = ''
    warrantyForm.start_date = ''
    warrantyForm.end_date = ''
    warrantyForm.notes = ''
  }
  warrantyModalOpen.value = true
}

async function saveWarranty() {
  try {
    const method = warranty.value ? 'PUT' : 'POST'
    await apiFetch(`/api/assets/${route.params.id}/warranty`, {
      method,
      body: JSON.stringify({
        provider: warrantyForm.provider || undefined,
        policy_number: warrantyForm.policy_number || undefined,
        start_date: warrantyForm.start_date || undefined,
        end_date: warrantyForm.end_date || undefined,
        notes: warrantyForm.notes || undefined
      })
    })
    toast.add({ title: 'Warranty saved', color: 'success' })
    warrantyModalOpen.value = false
    refreshWarranty()
  } catch (error) {
    toast.add({ title: 'Failed to save warranty', color: 'error' })
  }
}

async function deleteWarranty() {
  if (!confirm('Delete warranty information?')) return
  try {
    await apiFetch(`/api/assets/${route.params.id}/warranty`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Warranty deleted', color: 'success' })
    refreshWarranty()
  } catch (error) {
    toast.add({ title: 'Failed to delete warranty', color: 'error' })
  }
}

async function deleteAsset() {
  try {
    await apiFetch(`/api/assets/${route.params.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Asset deleted', color: 'success' })
    router.push('/assets')
  } catch (error) {
    toast.add({ title: 'Failed to delete asset', color: 'error' })
  }
}

async function downloadAttachment(attachment: Attachment) {
  try {
    const response = await apiFetch<{ url: string }>(`/api/attachments/${attachment.id}`)
    if (response.url) {
      window.open(response.url, '_blank')
    }
  } catch (error) {
    toast.add({ title: 'Failed to get download link', color: 'error' })
  }
}

async function deleteAttachment(attachment: Attachment) {
  if (!confirm(`Delete "${attachment.filename}"?`)) return
  try {
    await apiFetch(`/api/attachments/${attachment.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Attachment deleted', color: 'success' })
    refreshAttachments()
  } catch (error) {
    toast.add({ title: 'Failed to delete attachment', color: 'error' })
  }
}

const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)

function triggerUpload() {
  fileInput.value?.click()
}

async function handleFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)

    await fetch(`${config.public.apiBase}/api/assets/${route.params.id}/attachments`, {
      method: 'POST',
      body: formData,
      credentials: 'include'
    })

    toast.add({ title: 'File uploaded', color: 'success' })
    refreshAttachments()
  } catch (error) {
    toast.add({ title: 'Failed to upload file', color: 'error' })
  } finally {
    uploading.value = false
    input.value = ''
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString()
}

function formatBytes(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function daysUntilExpiry(endDate?: string) {
  if (!endDate) return null
  const end = new Date(endDate)
  const now = new Date()
  return Math.ceil((end.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

const warrantyStatus = computed(() => {
  if (!warranty.value?.end_date) return null
  const days = daysUntilExpiry(warranty.value.end_date)
  if (days === null) return null
  if (days < 0) return { color: 'error', text: 'Expired' }
  if (days <= 30) return { color: 'warning', text: `${days} days left` }
  return { color: 'success', text: `${days} days left` }
})
</script>

<template>
  <UContainer>
    <div class="py-8">
      <!-- Header -->
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-4">
          <UButton
            to="/assets"
            variant="ghost"
            icon="i-lucide-arrow-left"
          />
          <h1 class="text-2xl font-bold">{{ asset?.name || 'Loading...' }}</h1>
        </div>
        <div class="flex gap-2">
          <UButton
            :to="`/assets/${route.params.id}/edit`"
            variant="soft"
            icon="i-lucide-edit"
          >
            Edit
          </UButton>
          <UButton
            color="error"
            variant="soft"
            icon="i-lucide-trash-2"
            @click="deleteModalOpen = true"
          >
            Delete
          </UButton>
        </div>
      </div>

      <div v-if="asset" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Main Info -->
        <div class="lg:col-span-2 space-y-6">
          <UCard>
            <template #header>
              <h2 class="font-semibold">Details</h2>
            </template>

            <dl class="grid grid-cols-2 gap-4">
              <div>
                <dt class="text-sm text-muted">Category</dt>
                <dd class="font-medium">{{ asset.category?.name || '-' }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted">Location</dt>
                <dd class="font-medium">{{ asset.location?.name || '-' }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted">Condition</dt>
                <dd class="font-medium">{{ asset.condition?.label || '-' }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted">Quantity</dt>
                <dd class="font-medium">{{ asset.quantity }}</dd>
              </div>
              <div class="col-span-2">
                <dt class="text-sm text-muted">Description</dt>
                <dd class="font-medium">{{ asset.description || '-' }}</dd>
              </div>
            </dl>
          </UCard>

          <!-- Attributes -->
          <UCard v-if="categoryWithAttrs?.attributes?.length">
            <template #header>
              <h2 class="font-semibold">{{ categoryWithAttrs.name }} Attributes</h2>
            </template>

            <dl class="grid grid-cols-2 gap-4">
              <div v-for="ca in categoryWithAttrs.attributes" :key="ca.attribute_id">
                <dt class="text-sm text-muted">
                  {{ ca.attribute?.name || ca.attribute_id }}
                  <span v-if="ca.required" class="text-error">*</span>
                </dt>
                <dd class="font-medium">{{ getAttributeValue(ca.attribute?.key || '') }}</dd>
              </div>
            </dl>
          </UCard>

          <!-- Attachments -->
          <UCard>
            <template #header>
              <div class="flex items-center justify-between">
                <h2 class="font-semibold">Attachments</h2>
                <UButton
                  variant="soft"
                  size="sm"
                  icon="i-lucide-upload"
                  :loading="uploading"
                  @click="triggerUpload"
                >
                  Upload
                </UButton>
                <input
                  ref="fileInput"
                  type="file"
                  class="hidden"
                  @change="handleFileUpload"
                >
              </div>
            </template>

            <div v-if="attachments?.length" class="space-y-2">
              <div
                v-for="attachment in attachments"
                :key="attachment.id"
                class="flex items-center justify-between p-3 rounded-lg bg-muted/50"
              >
                <div class="flex items-center gap-3">
                  <UIcon name="i-lucide-file" class="w-5 h-5 text-muted" />
                  <div>
                    <p class="font-medium">{{ attachment.filename }}</p>
                    <p class="text-xs text-muted">{{ formatBytes(attachment.size) }}</p>
                  </div>
                </div>
                <div class="flex gap-1">
                  <UButton
                    variant="ghost"
                    icon="i-lucide-download"
                    size="sm"
                    @click="downloadAttachment(attachment)"
                  />
                  <UButton
                    variant="ghost"
                    icon="i-lucide-trash-2"
                    size="sm"
                    color="error"
                    @click="deleteAttachment(attachment)"
                  />
                </div>
              </div>
            </div>
            <p v-else class="text-muted text-center py-4">
              No attachments yet
            </p>
          </UCard>
        </div>

        <!-- Sidebar -->
        <div class="space-y-6">
          <!-- Warranty -->
          <UCard>
            <template #header>
              <div class="flex items-center justify-between">
                <h2 class="font-semibold">Warranty</h2>
                <UButton
                  variant="ghost"
                  size="sm"
                  :icon="warranty ? 'i-lucide-edit' : 'i-lucide-plus'"
                  @click="openWarrantyModal"
                >
                  {{ warranty ? 'Edit' : 'Add' }}
                </UButton>
              </div>
            </template>

            <div v-if="warranty">
              <dl class="space-y-3">
                <div>
                  <dt class="text-sm text-muted">Provider</dt>
                  <dd class="font-medium">{{ warranty.provider || '-' }}</dd>
                </div>
                <div>
                  <dt class="text-sm text-muted">Policy Number</dt>
                  <dd class="font-medium">{{ warranty.policy_number || '-' }}</dd>
                </div>
                <div>
                  <dt class="text-sm text-muted">Start Date</dt>
                  <dd class="font-medium">{{ formatDate(warranty.start_date) }}</dd>
                </div>
                <div>
                  <dt class="text-sm text-muted">End Date</dt>
                  <dd class="font-medium flex items-center gap-2">
                    {{ formatDate(warranty.end_date) }}
                    <UBadge v-if="warrantyStatus" :color="warrantyStatus.color" size="xs">
                      {{ warrantyStatus.text }}
                    </UBadge>
                  </dd>
                </div>
                <div v-if="warranty.notes">
                  <dt class="text-sm text-muted">Notes</dt>
                  <dd class="font-medium">{{ warranty.notes }}</dd>
                </div>
              </dl>
            </div>
            <p v-else class="text-muted text-center py-4">
              No warranty information
            </p>
          </UCard>

          <!-- Purchase Information -->
          <UCard v-if="asset.purchase_at || asset.purchase_note">
            <template #header>
              <h2 class="font-semibold">Purchase Information</h2>
            </template>

            <dl class="space-y-3">
              <div v-if="asset.purchase_at">
                <dt class="text-sm text-muted">Purchase Date</dt>
                <dd class="font-medium">{{ formatDate(asset.purchase_at) }}</dd>
              </div>
              <div v-if="asset.purchase_note">
                <dt class="text-sm text-muted">Notes</dt>
                <dd class="font-medium">{{ asset.purchase_note }}</dd>
              </div>
            </dl>
          </UCard>

          <!-- Metadata -->
          <UCard>
            <template #header>
              <h2 class="font-semibold">Metadata</h2>
            </template>

            <dl class="space-y-3">
              <div>
                <dt class="text-sm text-muted">Created</dt>
                <dd class="font-medium">{{ formatDate(asset.created_at) }}</dd>
              </div>
              <div>
                <dt class="text-sm text-muted">Last Updated</dt>
                <dd class="font-medium">{{ formatDate(asset.updated_at) }}</dd>
              </div>
            </dl>
          </UCard>
        </div>
      </div>

      <!-- Delete Asset Modal -->
      <UModal v-model:open="deleteModalOpen">
        <template #content>
          <UCard>
            <template #header>
              <h3 class="font-semibold">Delete Asset</h3>
            </template>
            <p>Are you sure you want to delete "{{ asset?.name }}"? This action cannot be undone.</p>
            <template #footer>
              <div class="flex justify-end gap-2">
                <UButton variant="ghost" @click="deleteModalOpen = false">
                  Cancel
                </UButton>
                <UButton color="error" @click="deleteAsset">
                  Delete
                </UButton>
              </div>
            </template>
          </UCard>
        </template>
      </UModal>

      <!-- Warranty Modal -->
      <UModal v-model:open="warrantyModalOpen">
        <template #content>
          <UCard>
            <template #header>
              <h3 class="font-semibold">{{ warranty ? 'Edit Warranty' : 'Add Warranty' }}</h3>
            </template>

            <form class="space-y-4" @submit.prevent="saveWarranty">
              <UFormField label="Provider">
                <UInput v-model="warrantyForm.provider" placeholder="Warranty provider" />
              </UFormField>

              <UFormField label="Policy Number">
                <UInput v-model="warrantyForm.policy_number" placeholder="Policy number" />
              </UFormField>

              <div class="grid grid-cols-2 gap-4">
                <UFormField label="Start Date">
                  <UInput v-model="warrantyForm.start_date" type="date" />
                </UFormField>

                <UFormField label="End Date">
                  <UInput v-model="warrantyForm.end_date" type="date" />
                </UFormField>
              </div>

              <UFormField label="Notes">
                <UTextarea v-model="warrantyForm.notes" placeholder="Additional notes" :rows="3" />
              </UFormField>
            </form>

            <template #footer>
              <div class="flex justify-between">
                <UButton
                  v-if="warranty"
                  variant="ghost"
                  color="error"
                  @click="deleteWarranty(); warrantyModalOpen = false"
                >
                  Delete
                </UButton>
                <div class="flex gap-2 ml-auto">
                  <UButton variant="ghost" @click="warrantyModalOpen = false">
                    Cancel
                  </UButton>
                  <UButton @click="saveWarranty">
                    Save
                  </UButton>
                </div>
              </div>
            </template>
          </UCard>
        </template>
      </UModal>
    </div>
  </UContainer>
</template>
