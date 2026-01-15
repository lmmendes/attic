<script setup lang="ts">
import type { Condition } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const router = useRouter()
const toast = useToast()
const apiFetch = useApiFetch()

// Get existing conditions to determine next sort order
const { data: conditions } = useApi<Condition[]>('/api/conditions')

// Form state
const form = reactive({
  label: '',
  code: '',
  description: '',
  sort_order: 1
})

// Auto-generate code from label
const isCodeManuallyEdited = ref(false)

function generateCode(label: string): string {
  return label.toUpperCase().replace(/[^A-Z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

watch(() => form.label, (newLabel) => {
  if (!isCodeManuallyEdited.value) {
    form.code = generateCode(newLabel)
  }
})

function onCodeInput() {
  isCodeManuallyEdited.value = true
}

// Set default sort order when conditions load
watch(conditions, (items) => {
  if (items && items.length > 0) {
    form.sort_order = Math.max(...items.map(c => c.sort_order)) + 1
  }
}, { immediate: true })

// Saving state
const saving = ref(false)

// Character count for description
const descriptionCount = computed(() => form.description.length)

// Preset conditions for quick selection
const presets = [
  { label: 'New', code: 'NEW', icon: 'i-lucide-sparkles', color: 'emerald' },
  { label: 'Like New', code: 'LIKE_NEW', icon: 'i-lucide-star', color: 'teal' },
  { label: 'Good', code: 'GOOD', icon: 'i-lucide-thumbs-up', color: 'blue' },
  { label: 'Fair', code: 'FAIR', icon: 'i-lucide-minus', color: 'amber' },
  { label: 'Poor', code: 'POOR', icon: 'i-lucide-alert-triangle', color: 'orange' },
  { label: 'For Parts', code: 'FOR_PARTS', icon: 'i-lucide-wrench', color: 'red' }
]

function applyPreset(preset: typeof presets[0]) {
  form.label = preset.label
  form.code = preset.code
  isCodeManuallyEdited.value = true
}

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
    await apiFetch('/api/conditions', {
      method: 'POST',
      body: JSON.stringify({
        label: form.label,
        code: form.code,
        description: form.description || null,
        sort_order: form.sort_order
      })
    })

    toast.add({ title: 'Condition created successfully', color: 'success' })
    router.push('/conditions')
  } catch {
    toast.add({ title: 'Failed to create condition', color: 'error' })
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
          <span class="text-mist-950 dark:text-white">New</span>
        </nav>
        <div>
          <h1 class="text-3xl font-extrabold tracking-tight text-mist-950 dark:text-white">
            Create Condition
          </h1>
          <p class="text-mist-500 mt-1">
            Define a new quality level to describe the state of your assets.
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
          Save Condition
        </UButton>
      </div>
    </div>

    <!-- Form Card -->
    <div class="max-w-2xl">
      <!-- Quick Presets -->
      <div class="mb-6">
        <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-3">
          Quick Start
        </label>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in presets"
            :key="preset.code"
            type="button"
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium border transition-all"
            :class="form.code === preset.code
              ? `bg-${preset.color}-100 dark:bg-${preset.color}-900/30 border-${preset.color}-300 dark:border-${preset.color}-700 text-${preset.color}-700 dark:text-${preset.color}-300`
              : 'bg-white dark:bg-mist-800 border-mist-200 dark:border-mist-600 text-mist-600 dark:text-mist-400 hover:border-mist-300 dark:hover:border-mist-500'"
            @click="applyPreset(preset)"
          >
            <UIcon
              :name="preset.icon"
              class="w-4 h-4"
            />
            {{ preset.label }}
          </button>
        </div>
        <p class="text-xs text-mist-400 mt-2">
          Click a preset to quick-fill the form, or create a custom condition below.
        </p>
      </div>

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

          <!-- Code Field -->
          <div>
            <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
              Condition Code
            </label>
            <input
              v-model="form.code"
              type="text"
              placeholder="e.g. LIKE_NEW"
              class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 font-mono text-sm text-mist-950 dark:text-white uppercase"
              @input="onCodeInput"
            >
            <p class="text-xs text-mist-400 mt-1">
              A unique uppercase identifier. Auto-generated from label.
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
      <div class="mt-6 p-4 rounded-lg bg-attic-50 dark:bg-attic-900/20 border border-attic-200 dark:border-attic-800/50">
        <div class="flex gap-3">
          <UIcon
            name="i-lucide-lightbulb"
            class="w-5 h-5 text-attic-500 shrink-0 mt-0.5"
          />
          <div>
            <p class="text-sm font-semibold text-attic-700 dark:text-attic-300">
              Pro Tip
            </p>
            <p class="text-sm text-attic-600 dark:text-attic-400 mt-1">
              Common condition scales range from "New" to "For Parts/Repair". You can customize these to match your needs, such as "Sealed" for collectibles or "Restored" for antiques.
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
