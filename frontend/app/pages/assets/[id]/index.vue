<script setup lang="ts">
import type { Asset, Warranty, Attachment, Category } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

const { data: asset, refresh: refreshAsset } = useApi<Asset>(() => `/api/assets/${route.params.id}`)
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
void refreshAsset // Mark as used
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
  } catch {
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
  } catch {
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
  } catch {
    toast.add({ title: 'Failed to delete asset', color: 'error' })
  }
}

async function downloadAttachment(attachment: Attachment) {
  try {
    const response = await apiFetch<{ url: string }>(`/api/attachments/${attachment.id}`)
    if (response.url) {
      window.open(response.url, '_blank')
    }
  } catch {
    toast.add({ title: 'Failed to get download link', color: 'error' })
  }
}

async function deleteAttachment(attachment: Attachment) {
  if (!confirm(`Delete "${attachment.file_name}"?`)) return
  try {
    await apiFetch(`/api/attachments/${attachment.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Attachment deleted', color: 'success' })
    refreshAttachments()
    // Refresh asset in case we deleted the main image
    if (asset.value?.main_attachment_id === attachment.id) {
      refreshAsset()
    }
  } catch {
    toast.add({ title: 'Failed to delete attachment', color: 'error' })
  }
}

// State for attachment thumbnail URLs
const attachmentUrls = ref<Record<string, string>>({})

// Helper to check if attachment is an image
function isImageAttachment(attachment: Attachment): boolean {
  const imageTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml']
  return attachment.content_type ? imageTypes.includes(attachment.content_type) : false
}

// Load thumbnail URLs for image attachments
watch(attachments, async (atts) => {
  if (!atts) return
  for (const att of atts) {
    if (isImageAttachment(att) && !attachmentUrls.value[att.id]) {
      try {
        const response = await apiFetch<{ url: string }>(`/api/attachments/${att.id}`)
        if (response.url) {
          attachmentUrls.value[att.id] = response.url
        }
      } catch {
        // Ignore errors loading thumbnails
      }
    }
  }
}, { immediate: true })

// Set main image
async function setMainImage(attachment: Attachment) {
  try {
    await apiFetch(`/api/assets/${route.params.id}/main-image/${attachment.id}`, {
      method: 'PUT'
    })
    toast.add({ title: 'Main image updated', color: 'success' })
    refreshAsset()
  } catch {
    toast.add({ title: 'Failed to set main image', color: 'error' })
  }
}

// Clear main image
async function clearMainImage() {
  try {
    await apiFetch(`/api/assets/${route.params.id}/main-image`, {
      method: 'DELETE'
    })
    toast.add({ title: 'Main image cleared', color: 'success' })
    refreshAsset()
  } catch {
    toast.add({ title: 'Failed to clear main image', color: 'error' })
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
    // Refresh asset in case this was auto-set as main image
    refreshAsset()
  } catch {
    toast.add({ title: 'Failed to upload file', color: 'error' })
  } finally {
    uploading.value = false
    input.value = ''
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

function formatDateTime(dateStr?: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatBytes(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function formatCurrency(value?: number) {
  if (value === undefined || value === null) return '-'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD'
  }).format(value)
}

function daysUntilExpiry(endDate?: string) {
  if (!endDate) return null
  const end = new Date(endDate)
  const now = new Date()
  return Math.ceil((end.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

const warrantyStatus = computed((): { color: 'error' | 'warning' | 'success', text: string } | null => {
  if (!warranty.value?.end_date) return null
  const days = daysUntilExpiry(warranty.value.end_date)
  if (days === null) return null
  if (days < 0) return { color: 'error', text: 'Expired' }
  if (days <= 30) return { color: 'warning', text: `${days} days left` }
  return { color: 'success', text: `${days} days left` }
})

// Generate short ID
function getShortId(): string {
  if (!asset.value?.id) return ''
  return `ATC-${asset.value.id.slice(0, 4).toUpperCase()}`
}
</script>

<template>
  <div class="max-w-[1200px] mx-auto">
    <!-- Breadcrumbs -->
    <nav class="flex items-center gap-2 mb-8 text-gray-500 dark:text-gray-400 text-sm font-medium">
      <NuxtLink
        to="/"
        class="hover:text-attic-500 transition-colors"
      >
        Home
      </NuxtLink>
      <UIcon
        name="i-lucide-chevron-right"
        class="w-4 h-4"
      />
      <NuxtLink
        to="/assets"
        class="hover:text-attic-500 transition-colors"
      >
        Assets
      </NuxtLink>
      <UIcon
        name="i-lucide-chevron-right"
        class="w-4 h-4"
      />
      <span class="text-mist-950 dark:text-white">{{ asset?.name || 'Loading...' }}</span>
    </nav>

    <div
      v-if="asset"
      class="grid grid-cols-1 lg:grid-cols-12 gap-10"
    >
      <!-- Left Column: Visual Anchor -->
      <div class="lg:col-span-4 flex flex-col gap-6">
        <!-- Main Image / Placeholder -->
        <div class="relative group">
          <div class="aspect-[3/4] rounded-xl overflow-hidden shadow-2xl bg-white dark:bg-gray-800 ring-1 ring-black/5 flex items-center justify-center">
            <img
              v-if="asset.main_attachment_url"
              :src="asset.main_attachment_url"
              :alt="asset.name"
              class="w-full h-full object-cover"
            >
            <div
              v-else
              class="text-center p-8"
            >
              <UIcon
                name="i-lucide-package"
                class="w-24 h-24 text-gray-200 dark:text-gray-600 mx-auto mb-4"
              />
              <p class="text-sm text-gray-400">
                No image available
              </p>
            </div>
          </div>
          <!-- Clear main image button (shown on hover when image exists) -->
          <button
            v-if="asset.main_attachment_url"
            class="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity bg-black/50 hover:bg-black/70 text-white rounded-full p-2"
            title="Remove main image"
            @click="clearMainImage"
          >
            <UIcon
              name="i-lucide-x"
              class="w-4 h-4"
            />
          </button>
        </div>

        <!-- Asset Intelligence Card -->
        <div class="bg-white dark:bg-gray-800 p-5 rounded-xl border border-gray-200 dark:border-gray-700">
          <div class="flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400 mb-2">
            <UIcon
              name="i-lucide-info"
              class="w-5 h-5 text-attic-500"
            />
            <span>Asset Information</span>
          </div>
          <p class="text-xs leading-relaxed text-gray-600 dark:text-gray-300">
            Asset ID: <span class="font-bold text-attic-500 font-mono">{{ getShortId() }}</span>
          </p>
          <div class="mt-4 pt-4 border-t border-dashed border-gray-200 dark:border-gray-700">
            <p class="text-[11px] uppercase tracking-wider font-bold text-gray-500 dark:text-gray-400 mb-1">
              Last Updated
            </p>
            <p class="text-sm font-medium text-mist-950 dark:text-white">
              {{ formatDateTime(asset.updated_at) }}
            </p>
          </div>
        </div>

        <!-- Quick Actions (Mobile) -->
        <div class="lg:hidden flex gap-3">
          <UButton
            :to="`/assets/${route.params.id}/edit`"
            variant="outline"
            class="flex-1"
            icon="i-lucide-pencil"
          >
            Edit Asset
          </UButton>
          <UButton
            color="error"
            variant="soft"
            icon="i-lucide-trash-2"
            @click="deleteModalOpen = true"
          />
        </div>
      </div>

      <!-- Right Column: Content & Metadata -->
      <div class="lg:col-span-8 space-y-8">
        <!-- Header & Title -->
        <div class="flex flex-col md:flex-row justify-between items-start gap-4">
          <div class="space-y-1">
            <h1 class="text-4xl font-extrabold tracking-tight text-mist-950 dark:text-white">
              {{ asset.name }}
            </h1>
            <p class="text-xl text-attic-500 font-medium">
              {{ asset.category?.name || 'Uncategorized' }}
            </p>
          </div>
          <div class="hidden lg:flex gap-3 shrink-0">
            <UButton
              :to="`/assets/${route.params.id}/edit`"
              variant="outline"
              class="font-bold"
              icon="i-lucide-pencil"
            >
              Edit Asset
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

        <!-- Description (if present) -->
        <section
          v-if="asset.description"
          class="space-y-4"
        >
          <div class="flex items-center justify-between px-2">
            <h3 class="text-xl font-bold tracking-tight text-mist-950 dark:text-white">
              Description
            </h3>
          </div>
          <div class="bg-gray-50 dark:bg-gray-800 p-6 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm">
            <p class="text-mist-950 dark:text-gray-300 leading-relaxed">
              {{ asset.description }}
            </p>
          </div>
        </section>

        <!-- Personal Notes (if present) -->
        <section
          v-if="asset.notes"
          class="space-y-4"
        >
          <div class="flex items-center justify-between px-2">
            <h3 class="text-xl font-bold tracking-tight text-mist-950 dark:text-white">
              Personal Notes
            </h3>
            <NuxtLink
              :to="`/assets/${route.params.id}/edit`"
              class="text-attic-500 text-sm font-bold hover:underline flex items-center gap-1"
            >
              <UIcon
                name="i-lucide-edit"
                class="w-4 h-4"
              />
              Edit Notes
            </NuxtLink>
          </div>
          <div class="bg-amber-50/50 dark:bg-gray-800 p-6 rounded-xl border-l-4 border-attic-500 shadow-sm min-h-[120px]">
            <p class="text-mist-950 dark:text-gray-300 leading-relaxed italic">
              "{{ asset.notes }}"
            </p>
          </div>
        </section>

        <!-- Structured Attributes List -->
        <section class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden shadow-sm">
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50 flex justify-between items-center">
            <h3 class="text-base font-bold flex items-center gap-2 text-mist-950 dark:text-white">
              <UIcon
                name="i-lucide-clipboard-list"
                class="w-5 h-5 text-attic-500"
              />
              Asset Details
            </h3>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 divide-y divide-x-0 md:divide-x md:divide-y-0 divide-gray-100 dark:divide-gray-700">
            <div class="divide-y divide-gray-100 dark:divide-gray-700">
              <div class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Location</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ asset.location?.name || '-' }}</span>
              </div>
              <div class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Condition</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ asset.condition?.label || '-' }}</span>
              </div>
              <div class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Quantity</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ asset.quantity }}</span>
              </div>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-gray-700">
              <div class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Purchase Date</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ formatDate(asset.purchase_at) }}</span>
              </div>
              <div class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Purchase Price</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ formatCurrency(asset.purchase_price) }}</span>
              </div>
              <div class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Created</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ formatDate(asset.created_at) }}</span>
              </div>
            </div>
          </div>
        </section>

        <!-- Category Attributes -->
        <section
          v-if="categoryWithAttrs?.attributes?.length"
          class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden shadow-sm"
        >
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50 flex justify-between items-center">
            <h3 class="text-base font-bold flex items-center gap-2 text-mist-950 dark:text-white">
              <UIcon
                name="i-lucide-sliders-horizontal"
                class="w-5 h-5 text-attic-500"
              />
              {{ categoryWithAttrs.name }} Attributes
            </h3>
            <span class="text-[10px] px-2 py-0.5 bg-attic-500/10 text-attic-500 rounded-full font-bold uppercase tracking-widest">
              Custom Data
            </span>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 divide-y divide-x-0 md:divide-x md:divide-y-0 divide-gray-100 dark:divide-gray-700">
            <div class="divide-y divide-gray-100 dark:divide-gray-700">
              <div
                v-for="ca in categoryWithAttrs.attributes.filter((_, i) => i % 2 === 0)"
                :key="ca.attribute_id"
                class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              >
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ ca.attribute?.name }}</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ getAttributeValue(ca.attribute?.key || '') }}</span>
              </div>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-gray-700">
              <div
                v-for="ca in categoryWithAttrs.attributes.filter((_, i) => i % 2 === 1)"
                :key="ca.attribute_id"
                class="p-5 flex justify-between items-center group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              >
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ ca.attribute?.name }}</span>
                <span class="text-sm font-bold text-mist-950 dark:text-white">{{ getAttributeValue(ca.attribute?.key || '') }}</span>
              </div>
            </div>
          </div>
        </section>

        <!-- Warranty Section -->
        <section class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden shadow-sm">
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50 flex justify-between items-center">
            <h3 class="text-base font-bold flex items-center gap-2 text-mist-950 dark:text-white">
              <UIcon
                name="i-lucide-shield-check"
                class="w-5 h-5 text-attic-500"
              />
              Warranty Information
            </h3>
            <button
              class="text-attic-500 text-sm font-bold hover:underline flex items-center gap-1"
              @click="openWarrantyModal"
            >
              <UIcon
                :name="warranty ? 'i-lucide-pencil' : 'i-lucide-plus'"
                class="w-4 h-4"
              />
              {{ warranty ? 'Edit' : 'Add Warranty' }}
            </button>
          </div>
          <div v-if="warranty">
            <div class="grid grid-cols-1 md:grid-cols-2 divide-y divide-x-0 md:divide-x md:divide-y-0 divide-gray-100 dark:divide-gray-700">
              <div class="divide-y divide-gray-100 dark:divide-gray-700">
                <div class="p-5 flex justify-between items-center">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Provider</span>
                  <span class="text-sm font-bold text-mist-950 dark:text-white">{{ warranty.provider || '-' }}</span>
                </div>
                <div class="p-5 flex justify-between items-center">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Policy Number</span>
                  <span class="text-sm font-bold font-mono text-mist-950 dark:text-white">{{ warranty.policy_number || '-' }}</span>
                </div>
              </div>
              <div class="divide-y divide-gray-100 dark:divide-gray-700">
                <div class="p-5 flex justify-between items-center">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Start Date</span>
                  <span class="text-sm font-bold text-mist-950 dark:text-white">{{ formatDate(warranty.start_date) }}</span>
                </div>
                <div class="p-5 flex justify-between items-center">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">End Date</span>
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-bold text-mist-950 dark:text-white">{{ formatDate(warranty.end_date) }}</span>
                    <UBadge
                      v-if="warrantyStatus"
                      :color="warrantyStatus.color"
                      size="xs"
                    >
                      {{ warrantyStatus.text }}
                    </UBadge>
                  </div>
                </div>
              </div>
            </div>
            <div
              v-if="warranty.notes"
              class="p-5 border-t border-gray-100 dark:border-gray-700"
            >
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400 block mb-1">Notes</span>
              <p class="text-sm text-mist-950 dark:text-white">
                {{ warranty.notes }}
              </p>
            </div>
          </div>
          <div
            v-else
            class="p-8 text-center"
          >
            <UIcon
              name="i-lucide-shield-off"
              class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3"
            />
            <p class="text-gray-500 dark:text-gray-400 mb-4">
              No warranty information
            </p>
            <UButton
              variant="soft"
              size="sm"
              @click="openWarrantyModal"
            >
              Add Warranty
            </UButton>
          </div>
        </section>

        <!-- Attachments Section -->
        <section class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden shadow-sm">
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50 flex justify-between items-center">
            <h3 class="text-base font-bold flex items-center gap-2 text-mist-950 dark:text-white">
              <UIcon
                name="i-lucide-paperclip"
                class="w-5 h-5 text-attic-500"
              />
              Attachments
            </h3>
            <button
              class="text-attic-500 text-sm font-bold hover:underline flex items-center gap-1"
              :disabled="uploading"
              @click="triggerUpload"
            >
              <UIcon
                name="i-lucide-upload"
                class="w-4 h-4"
              />
              Upload File
            </button>
            <input
              ref="fileInput"
              type="file"
              class="hidden"
              @change="handleFileUpload"
            >
          </div>
          <div
            v-if="attachments?.length"
            class="divide-y divide-gray-100 dark:divide-gray-700"
          >
            <div
              v-for="attachment in attachments"
              :key="attachment.id"
              class="p-4 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
            >
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-gray-700 flex items-center justify-center overflow-hidden">
                  <!-- Show thumbnail for images -->
                  <img
                    v-if="isImageAttachment(attachment) && attachmentUrls[attachment.id]"
                    :src="attachmentUrls[attachment.id]"
                    class="w-full h-full object-cover"
                  >
                  <UIcon
                    v-else
                    name="i-lucide-file"
                    class="w-5 h-5 text-gray-400"
                  />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <p class="font-medium text-sm text-mist-950 dark:text-white">
                      {{ attachment.file_name }}
                    </p>
                    <UBadge
                      v-if="asset.main_attachment_id === attachment.id"
                      color="primary"
                      size="xs"
                    >
                      Main Image
                    </UBadge>
                  </div>
                  <p class="text-xs text-gray-500">
                    {{ formatBytes(attachment.file_size) }}
                  </p>
                </div>
              </div>
              <div class="flex gap-1">
                <!-- Set as main image button (only for images) -->
                <UButton
                  v-if="isImageAttachment(attachment) && asset.main_attachment_id !== attachment.id"
                  variant="ghost"
                  color="neutral"
                  icon="i-lucide-image"
                  size="sm"
                  title="Set as main image"
                  @click="setMainImage(attachment)"
                />
                <UButton
                  variant="ghost"
                  color="neutral"
                  icon="i-lucide-download"
                  size="sm"
                  @click="downloadAttachment(attachment)"
                />
                <UButton
                  variant="ghost"
                  color="error"
                  icon="i-lucide-trash-2"
                  size="sm"
                  @click="deleteAttachment(attachment)"
                />
              </div>
            </div>
          </div>
          <div
            v-else
            class="p-8 text-center"
          >
            <UIcon
              name="i-lucide-file-x"
              class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3"
            />
            <p class="text-gray-500 dark:text-gray-400 mb-4">
              No attachments yet
            </p>
            <UButton
              variant="soft"
              size="sm"
              :loading="uploading"
              @click="triggerUpload"
            >
              Upload File
            </UButton>
          </div>
        </section>

        <!-- Asset History -->
        <section class="space-y-4 pb-12">
          <h3 class="text-xl font-bold tracking-tight px-2 text-mist-950 dark:text-white">
            Asset History
          </h3>
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
            <ul class="relative border-l-2 border-gray-200 dark:border-gray-700 ml-8 my-6 space-y-8">
              <!-- History Item: Updated -->
              <li class="relative pl-8">
                <span class="absolute -left-[9px] top-1 h-4 w-4 rounded-full bg-attic-500 ring-4 ring-white dark:ring-gray-800 shadow-sm" />
                <div class="flex flex-col gap-1">
                  <div class="flex items-center justify-between">
                    <p class="text-sm font-bold text-mist-950 dark:text-white">
                      Last Updated
                    </p>
                    <span class="text-xs text-gray-500 dark:text-gray-400 font-medium">{{ formatDateTime(asset.updated_at) }}</span>
                  </div>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    Asset details were modified.
                  </p>
                </div>
              </li>
              <!-- History Item: Created -->
              <li class="relative pl-8">
                <span class="absolute -left-[9px] top-1 h-4 w-4 rounded-full bg-gray-200 dark:bg-gray-600 ring-4 ring-white dark:ring-gray-800 shadow-sm" />
                <div class="flex flex-col gap-1">
                  <div class="flex items-center justify-between">
                    <p class="text-sm font-bold text-mist-950 dark:text-white">
                      Asset Created
                    </p>
                    <span class="text-xs text-gray-500 dark:text-gray-400 font-medium">{{ formatDateTime(asset.created_at) }}</span>
                  </div>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    Initial entry created.
                  </p>
                </div>
              </li>
            </ul>
          </div>
        </section>
      </div>
    </div>

    <!-- Loading State -->
    <div
      v-else
      class="flex items-center justify-center py-24"
    >
      <div class="text-center">
        <UIcon
          name="i-lucide-loader-2"
          class="w-12 h-12 animate-spin text-attic-500 mx-auto mb-4"
        />
        <p class="text-gray-500">
          Loading asset...
        </p>
      </div>
    </div>

    <!-- Delete Asset Modal -->
    <UModal v-model:open="deleteModalOpen">
      <template #content>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-6">
          <div class="flex items-center gap-4 mb-4">
            <div class="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
              <UIcon
                name="i-lucide-alert-triangle"
                class="w-6 h-6 text-red-500"
              />
            </div>
            <div>
              <h3 class="font-bold text-lg text-mist-950 dark:text-white">
                Delete Asset
              </h3>
              <p class="text-sm text-gray-500">
                This action cannot be undone.
              </p>
            </div>
          </div>
          <p class="text-gray-600 dark:text-gray-300 mb-6">
            Are you sure you want to delete "<strong>{{ asset?.name }}</strong>"? All associated data will be permanently removed.
          </p>
          <div class="flex justify-end gap-3">
            <UButton
              variant="ghost"
              @click="deleteModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              color="error"
              @click="deleteAsset"
            >
              Delete Asset
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Warranty Modal -->
    <UModal v-model:open="warrantyModalOpen">
      <template #content>
        <div class="bg-white dark:bg-gray-800 rounded-xl overflow-hidden">
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
            <h3 class="font-bold text-lg text-mist-950 dark:text-white">
              {{ warranty ? 'Edit Warranty' : 'Add Warranty' }}
            </h3>
          </div>

          <form
            class="p-6 space-y-4"
            @submit.prevent="saveWarranty"
          >
            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Provider
              </label>
              <input
                v-model="warrantyForm.provider"
                type="text"
                placeholder="Warranty provider"
                class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm"
              >
            </div>

            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Policy Number
              </label>
              <input
                v-model="warrantyForm.policy_number"
                type="text"
                placeholder="Policy number"
                class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm"
              >
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-2">
                <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Start Date
                </label>
                <input
                  v-model="warrantyForm.start_date"
                  type="date"
                  class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm"
                >
              </div>

              <div class="space-y-2">
                <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  End Date
                </label>
                <input
                  v-model="warrantyForm.end_date"
                  type="date"
                  class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm"
                >
              </div>
            </div>

            <div class="space-y-2">
              <label class="block text-xs font-bold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Notes
              </label>
              <textarea
                v-model="warrantyForm.notes"
                rows="3"
                placeholder="Additional notes"
                class="block w-full rounded-lg border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-mist-950 dark:text-white focus:border-attic-500 focus:ring-attic-500 text-sm py-3 px-4 shadow-sm resize-none"
              />
            </div>
          </form>

          <div class="px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 flex justify-between">
            <UButton
              v-if="warranty"
              variant="ghost"
              color="error"
              @click="deleteWarranty(); warrantyModalOpen = false"
            >
              Delete
            </UButton>
            <div class="flex gap-3 ml-auto">
              <UButton
                variant="ghost"
                @click="warrantyModalOpen = false"
              >
                Cancel
              </UButton>
              <UButton @click="saveWarranty">
                Save
              </UButton>
            </div>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
