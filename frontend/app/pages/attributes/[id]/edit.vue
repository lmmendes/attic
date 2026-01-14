<script setup lang="ts">
import type { Attribute } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

const attributeId = computed(() => route.params.id as string)

// Fetch existing attribute
const { data: attribute, status } = useApi<Attribute>(`/api/attributes/${attributeId.value}`)

// Form state
const form = reactive({
  name: '',
  key: '',
  data_type: 'string'
})

// Populate form when attribute loads
watch(attribute, (attr) => {
  if (attr) {
    form.name = attr.name
    form.key = attr.key
    form.data_type = attr.data_type
  }
}, { immediate: true })

// Data types available
const dataTypes = [
  { value: 'string', label: 'String', icon: 'i-lucide-type', description: 'Short text, names, or identifiers' },
  { value: 'text', label: 'Text', icon: 'i-lucide-align-left', description: 'Long form text or descriptions' },
  { value: 'number', label: 'Number', icon: 'i-lucide-hash', description: 'Numeric values' },
  { value: 'boolean', label: 'Boolean', icon: 'i-lucide-toggle-left', description: 'True/false values' },
  { value: 'date', label: 'Date', icon: 'i-lucide-calendar', description: 'Date values' }
]

// Saving state
const saving = ref(false)

// Get style for selected type
function getTypeStyle(type: string): { bgColor: string; textColor: string; borderColor: string } {
  switch (type) {
    case 'string':
      return {
        bgColor: 'bg-slate-100 dark:bg-slate-800',
        textColor: 'text-slate-700 dark:text-slate-300',
        borderColor: 'border-slate-200 dark:border-slate-700'
      }
    case 'text':
      return {
        bgColor: 'bg-indigo-50 dark:bg-indigo-900/30',
        textColor: 'text-indigo-700 dark:text-indigo-300',
        borderColor: 'border-indigo-200 dark:border-indigo-900/50'
      }
    case 'number':
      return {
        bgColor: 'bg-orange-50 dark:bg-orange-900/30',
        textColor: 'text-orange-700 dark:text-orange-300',
        borderColor: 'border-orange-200 dark:border-orange-900/50'
      }
    case 'boolean':
      return {
        bgColor: 'bg-green-50 dark:bg-green-900/30',
        textColor: 'text-green-700 dark:text-green-300',
        borderColor: 'border-green-200 dark:border-green-900/50'
      }
    case 'date':
      return {
        bgColor: 'bg-purple-50 dark:bg-purple-900/30',
        textColor: 'text-purple-700 dark:text-purple-300',
        borderColor: 'border-purple-200 dark:border-purple-900/50'
      }
    default:
      return {
        bgColor: 'bg-gray-100 dark:bg-gray-800',
        textColor: 'text-gray-700 dark:text-gray-300',
        borderColor: 'border-gray-200 dark:border-gray-700'
      }
  }
}

// Save attribute
async function saveAttribute() {
  if (!form.name.trim()) {
    toast.add({ title: 'Please enter an attribute name', color: 'error' })
    return
  }
  if (!form.key.trim()) {
    toast.add({ title: 'Please enter an attribute key', color: 'error' })
    return
  }

  saving.value = true
  try {
    await apiFetch(`/api/attributes/${attributeId.value}`, {
      method: 'PUT',
      body: JSON.stringify({
        name: form.name,
        key: form.key,
        data_type: form.data_type
      })
    })

    toast.add({ title: 'Attribute updated successfully', color: 'success' })
    router.push('/attributes')
  } catch {
    toast.add({ title: 'Failed to update attribute', color: 'error' })
  } finally {
    saving.value = false
  }
}

// Cancel and go back
function cancel() {
  router.push('/attributes')
}
</script>

<template>
  <div class="space-y-8">
    <!-- Loading State -->
    <div
      v-if="status === 'pending'"
      class="flex items-center justify-center py-20"
    >
      <UIcon
        name="i-lucide-loader-2"
        class="w-8 h-8 text-attic-500 animate-spin"
      />
    </div>

    <!-- Error State -->
    <div
      v-else-if="status === 'error'"
      class="flex flex-col items-center justify-center py-20"
    >
      <UIcon
        name="i-lucide-alert-circle"
        class="w-12 h-12 text-red-500 mb-4"
      />
      <h2 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
        Attribute not found
      </h2>
      <p class="text-mist-500 mb-4">
        The attribute you're looking for doesn't exist or has been deleted.
      </p>
      <UButton
        to="/attributes"
        variant="ghost"
      >
        Back to Attributes
      </UButton>
    </div>

    <!-- Form Content -->
    <template v-else>
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
              to="/attributes"
              class="hover:text-attic-500 transition-colors"
            >
              Attributes
            </NuxtLink>
            <span class="mx-2 text-mist-300 dark:text-mist-600">/</span>
            <span class="text-mist-950 dark:text-white">Edit</span>
          </nav>
          <div>
            <h1 class="text-3xl font-extrabold tracking-tight text-mist-950 dark:text-white">
              Edit Attribute
            </h1>
            <p class="text-mist-500 mt-1">
              Update the details of "{{ attribute?.name }}".
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
            @click="saveAttribute"
          >
            Save Changes
          </UButton>
        </div>
      </div>

      <!-- Form Card -->
      <div class="max-w-2xl">
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-soft border border-mist-100 dark:border-mist-700 p-6">
          <div class="space-y-6">
            <!-- Name Field -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Attribute Name
              </label>
              <input
                v-model="form.name"
                type="text"
                placeholder="e.g. Purchase Date"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 font-medium text-mist-950 dark:text-white"
              >
              <p class="text-xs text-mist-400 mt-1">
                This is the display name shown in forms and lists.
              </p>
            </div>

            <!-- Key Field -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Attribute Key
              </label>
              <input
                v-model="form.key"
                type="text"
                placeholder="e.g. purchase_date"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 font-mono text-sm text-mist-950 dark:text-white"
              >
              <p class="text-xs text-mist-400 mt-1">
                A unique identifier for this attribute.
              </p>
            </div>

            <!-- Data Type Field -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-3">
                Data Type
              </label>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <button
                  v-for="type in dataTypes"
                  :key="type.value"
                  type="button"
                  class="flex items-start gap-3 p-4 rounded-lg border-2 transition-all text-left"
                  :class="form.data_type === type.value
                    ? [getTypeStyle(type.value).bgColor, getTypeStyle(type.value).borderColor, 'ring-2 ring-offset-2', `ring-${type.value === 'string' ? 'slate' : type.value === 'text' ? 'indigo' : type.value === 'number' ? 'orange' : type.value === 'boolean' ? 'green' : 'purple'}-500/50`]
                    : 'border-mist-200 dark:border-mist-600 hover:border-mist-300 dark:hover:border-mist-500 bg-white dark:bg-mist-900'"
                  @click="form.data_type = type.value"
                >
                  <div
                    class="size-10 rounded-lg flex items-center justify-center shrink-0"
                    :class="form.data_type === type.value
                      ? [getTypeStyle(type.value).textColor]
                      : 'bg-mist-100 dark:bg-mist-700 text-mist-500'"
                  >
                    <UIcon
                      :name="type.icon"
                      class="w-5 h-5"
                    />
                  </div>
                  <div>
                    <p
                      class="font-bold text-sm"
                      :class="form.data_type === type.value ? getTypeStyle(type.value).textColor : 'text-mist-950 dark:text-white'"
                    >
                      {{ type.label }}
                    </p>
                    <p class="text-xs text-mist-500 mt-0.5">
                      {{ type.description }}
                    </p>
                  </div>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Warning Box -->
        <div class="mt-6 p-4 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800/50">
          <div class="flex gap-3">
            <UIcon
              name="i-lucide-alert-triangle"
              class="w-5 h-5 text-amber-500 shrink-0 mt-0.5"
            />
            <div>
              <p class="text-sm font-semibold text-amber-700 dark:text-amber-300">
                Caution
              </p>
              <p class="text-sm text-amber-600 dark:text-amber-400 mt-1">
                Changing the data type may affect existing asset data that uses this attribute. Make sure to review any assets using this attribute after making changes.
              </p>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
