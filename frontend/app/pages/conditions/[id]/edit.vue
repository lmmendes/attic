<script setup lang="ts">
import type { Condition } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

const conditionId = computed(() => route.params.id as string)

// Fetch existing condition
const { data: condition, status } = useApi<Condition>(`/api/conditions/${conditionId.value}`)

// Form state
const form = reactive({
  label: '',
  code: '',
  description: '',
  sort_order: 1
})

// Populate form when condition loads
watch(condition, (item) => {
  if (item) {
    form.label = item.label
    form.code = item.code
    form.description = item.description || ''
    form.sort_order = item.sort_order
  }
}, { immediate: true })

// Saving state
const saving = ref(false)

// Character count for description
const descriptionCount = computed(() => form.description.length)

// Save condition
async function saveCondition() {
  if (!form.label.trim()) {
    toast.add({ title: 'Please enter a condition label', color: 'error' })
    return
  }
  if (!form.code.trim()) {
    toast.add({ title: 'Please enter a condition code', color: 'error' })
    return
  }

  saving.value = true
  try {
    await apiFetch(`/api/conditions/${conditionId.value}`, {
      method: 'PUT',
      body: JSON.stringify({
        label: form.label,
        code: form.code,
        description: form.description || null,
        sort_order: form.sort_order
      })
    })

    toast.add({ title: 'Condition updated successfully', color: 'success' })
    router.push('/conditions')
  } catch {
    toast.add({ title: 'Failed to update condition', color: 'error' })
  } finally {
    saving.value = false
  }
}

// Cancel and go back
function cancel() {
  router.push('/conditions')
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
        Condition not found
      </h2>
      <p class="text-mist-500 mb-4">
        The condition you're looking for doesn't exist or has been deleted.
      </p>
      <UButton
        to="/conditions"
        variant="ghost"
      >
        Back to Conditions
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
              to="/conditions"
              class="hover:text-attic-500 transition-colors"
            >
              Conditions
            </NuxtLink>
            <span class="mx-2 text-mist-300 dark:text-mist-600">/</span>
            <span class="text-mist-950 dark:text-white">Edit</span>
          </nav>
          <div>
            <h1 class="text-3xl font-extrabold tracking-tight text-mist-950 dark:text-white">
              Edit Condition
            </h1>
            <p class="text-mist-500 mt-1">
              Update the details of "{{ condition?.label }}".
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
            @click="saveCondition"
          >
            Save Changes
          </UButton>
        </div>
      </div>

      <!-- Form Card -->
      <div class="max-w-2xl">
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-soft border border-mist-100 dark:border-mist-700 p-6">
          <div class="space-y-6">
            <!-- Label Field -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Condition Label
              </label>
              <input
                v-model="form.label"
                type="text"
                placeholder="e.g. Like New"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 font-medium text-mist-950 dark:text-white"
              >
              <p class="text-xs text-mist-400 mt-1">
                The display name shown when selecting a condition.
              </p>
            </div>

            <!-- Code Field (disabled for edit) -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Condition Code
              </label>
              <input
                v-model="form.code"
                type="text"
                disabled
                class="w-full px-4 py-3 rounded-lg bg-mist-100 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 outline-none font-mono text-sm text-mist-500 dark:text-mist-400 uppercase cursor-not-allowed"
              >
              <p class="text-xs text-mist-400 mt-1">
                The code cannot be changed after creation.
              </p>
            </div>

            <!-- Description Field -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Description
                <span class="font-normal text-mist-400">(optional)</span>
              </label>
              <textarea
                v-model="form.description"
                rows="3"
                maxlength="200"
                placeholder="Describe what this condition means..."
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm resize-none text-mist-950 dark:text-white"
              />
              <div class="flex justify-end mt-1">
                <span class="text-xs text-mist-400">{{ descriptionCount }}/200</span>
              </div>
            </div>

            <!-- Sort Order Field -->
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Sort Order
              </label>
              <input
                v-model.number="form.sort_order"
                type="number"
                min="1"
                class="w-32 px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all text-sm text-mist-950 dark:text-white"
              >
              <p class="text-xs text-mist-400 mt-1">
                Controls the display order. Lower numbers appear first.
              </p>
            </div>
          </div>
        </div>

        <!-- Info Box -->
        <div class="mt-6 p-4 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800/50">
          <div class="flex gap-3">
            <UIcon
              name="i-lucide-info"
              class="w-5 h-5 text-amber-500 shrink-0 mt-0.5"
            />
            <div>
              <p class="text-sm font-semibold text-amber-700 dark:text-amber-300">
                Note
              </p>
              <p class="text-sm text-amber-600 dark:text-amber-400 mt-1">
                Changing the label or sort order will update how this condition appears across all assets using it.
              </p>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
