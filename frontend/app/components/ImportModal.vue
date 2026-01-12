<script setup lang="ts">
import type { Plugin, PluginSearchResult, PluginsResponse, PluginSearchResponse, PluginImportResponse } from '~/types/api'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'imported': [assetId: string]
}>()

const toast = useToast()
const apiFetch = useApiFetch()

// State
const step = ref<'select' | 'search' | 'importing'>('select')
const selectedPlugin = ref<Plugin | null>(null)
const searchField = ref('')
const searchQuery = ref('')
const searchResults = ref<PluginSearchResult[]>([])
const searching = ref(false)
const importing = ref(false)

// Load plugins
const { data: pluginsData } = useApi<PluginsResponse>('/api/plugins')

const plugins = computed(() => pluginsData.value?.plugins || [])

// Search field options for selected plugin
const searchFieldOptions = computed(() => {
  if (!selectedPlugin.value) return []
  return selectedPlugin.value.search_fields.map(f => ({
    label: f.label,
    value: f.key
  }))
})

// Reset state when modal closes
watch(() => props.open, (isOpen) => {
  if (!isOpen) {
    step.value = 'select'
    selectedPlugin.value = null
    searchField.value = ''
    searchQuery.value = ''
    searchResults.value = []
  }
})

function selectPlugin(plugin: Plugin) {
  selectedPlugin.value = plugin
  searchField.value = plugin.search_fields[0]?.key || ''
  searchQuery.value = ''
  searchResults.value = []
  step.value = 'search'
}

function goBack() {
  if (step.value === 'search') {
    step.value = 'select'
    selectedPlugin.value = null
    searchResults.value = []
  }
}

async function search() {
  if (!selectedPlugin.value || !searchQuery.value.trim()) return

  const query = searchQuery.value.trim()
  if (query.length < 2) {
    toast.add({ title: 'Please enter at least 2 characters', color: 'warning' })
    return
  }

  searching.value = true
  searchResults.value = []

  try {
    const params = new URLSearchParams({
      field: searchField.value,
      q: query,
      limit: '10'
    })

    const response = await apiFetch<PluginSearchResponse>(
      `/api/plugins/${selectedPlugin.value.id}/search?${params}`
    )

    searchResults.value = response.results || []

    if (searchResults.value.length === 0) {
      toast.add({ title: 'No results found', description: 'Try different search terms', color: 'warning' })
    }
  } catch (err: unknown) {
    const error = err as { data?: { error?: string }, message?: string }
    const message = error?.data?.error || error?.message || 'Search failed'
    toast.add({
      title: 'Search failed',
      description: message.includes('unavailable') ? 'The search service is temporarily unavailable. Please try again.' : message,
      color: 'error'
    })
  } finally {
    searching.value = false
  }
}

async function importItem(result: PluginSearchResult) {
  if (!selectedPlugin.value) return

  importing.value = true
  step.value = 'importing'

  try {
    const response = await apiFetch<PluginImportResponse>(
      `/api/plugins/${selectedPlugin.value.id}/import`,
      {
        method: 'POST',
        body: JSON.stringify({ external_id: result.external_id })
      }
    )

    toast.add({
      title: 'Import successful',
      description: `"${response.asset.name}" has been added to your inventory`,
      color: 'success'
    })

    emit('update:open', false)
    emit('imported', response.asset.id)
  } catch (err: unknown) {
    const error = err as { data?: { error?: string }, message?: string }
    const message = error?.data?.error || error?.message || 'Import failed'

    let description = 'Please try again later'
    if (message.includes('not found')) {
      description = 'This item is no longer available from the source'
    } else if (message.includes('unavailable')) {
      description = 'The external service is temporarily unavailable'
    }

    toast.add({
      title: 'Import failed',
      description,
      color: 'error'
    })
    step.value = 'search'
  } finally {
    importing.value = false
  }
}

function close() {
  emit('update:open', false)
}
</script>

<template>
  <UModal
    :open="open"
    @update:open="$emit('update:open', $event)"
  >
    <template #header>
      <div class="flex items-center gap-2">
        <UButton
          v-if="step === 'search'"
          variant="ghost"
          icon="i-lucide-arrow-left"
          size="sm"
          @click="goBack"
        />
        <h3 class="text-lg font-semibold">
          <template v-if="step === 'select'">
            Import from External Source
          </template>
          <template v-else-if="step === 'search'">
            {{ selectedPlugin?.name }}
          </template>
          <template v-else>
            Importing...
          </template>
        </h3>
      </div>
    </template>

    <template #body>
      <!-- Step 1: Select Plugin -->
      <div
        v-if="step === 'select'"
        class="space-y-3"
      >
        <p class="text-sm text-gray-500 mb-4">
          Choose a source to import from:
        </p>

        <div
          v-for="plugin in plugins"
          :key="plugin.id"
          class="p-4 border rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          @click="selectPlugin(plugin)"
        >
          <div class="flex items-center justify-between">
            <div>
              <h4 class="font-medium">
                {{ plugin.name }}
              </h4>
              <p class="text-sm text-gray-500">
                {{ plugin.description }}
              </p>
            </div>
            <UIcon
              name="i-lucide-chevron-right"
              class="w-5 h-5 text-gray-400"
            />
          </div>
        </div>

        <div
          v-if="plugins.length === 0"
          class="text-center py-8 text-gray-500"
        >
          No import plugins available
        </div>
      </div>

      <!-- Step 2: Search -->
      <div
        v-else-if="step === 'search'"
        class="space-y-4"
      >
        <form
          class="flex gap-2"
          @submit.prevent="search"
        >
          <USelectMenu
            v-model="searchField"
            :items="searchFieldOptions"
            class="w-32"
            value-key="value"
          />
          <UInput
            v-model="searchQuery"
            placeholder="Search..."
            class="flex-1"
            autofocus
          />
          <UButton
            type="submit"
            icon="i-lucide-search"
            :loading="searching"
            :disabled="!searchQuery.trim()"
          >
            Search
          </UButton>
        </form>

        <!-- Results -->
        <div
          v-if="searchResults.length > 0"
          class="space-y-2 max-h-96 overflow-y-auto"
        >
          <div
            v-for="result in searchResults"
            :key="result.external_id"
            class="flex items-center gap-3 p-3 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800"
          >
            <img
              v-if="result.image_url"
              :src="result.image_url"
              :alt="result.title"
              class="w-12 h-16 object-cover rounded"
            >
            <div
              v-else
              class="w-12 h-16 bg-gray-200 dark:bg-gray-700 rounded flex items-center justify-center"
            >
              <UIcon
                name="i-lucide-image-off"
                class="w-6 h-6 text-gray-400"
              />
            </div>

            <div class="flex-1 min-w-0">
              <h4 class="font-medium truncate">
                {{ result.title }}
              </h4>
              <p class="text-sm text-gray-500 truncate">
                {{ result.subtitle }}
              </p>
            </div>

            <UButton
              size="sm"
              @click="importItem(result)"
            >
              Import
            </UButton>
          </div>
        </div>

        <div
          v-else-if="!searching && searchQuery"
          class="text-center py-8 text-gray-500"
        >
          <UIcon
            name="i-lucide-search-x"
            class="w-12 h-12 mx-auto mb-2 opacity-50"
          />
          <p>No results found. Try a different search.</p>
        </div>

        <div
          v-else-if="!searching"
          class="text-center py-8 text-gray-500"
        >
          <UIcon
            name="i-lucide-search"
            class="w-12 h-12 mx-auto mb-2 opacity-50"
          />
          <p>Enter a search term to find items</p>
        </div>
      </div>

      <!-- Step 3: Importing -->
      <div
        v-else-if="step === 'importing'"
        class="flex flex-col items-center justify-center py-12"
      >
        <UIcon
          name="i-lucide-loader-2"
          class="w-12 h-12 animate-spin text-primary mb-4"
        />
        <p class="text-gray-500">
          Importing item...
        </p>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end">
        <UButton
          variant="ghost"
          :disabled="importing"
          @click="close"
        >
          Cancel
        </UButton>
      </div>
    </template>
  </UModal>
</template>
