<script setup lang="ts">
import type { Asset, AssetsResponse } from '~/types/api'

const props = defineProps<{
  label: string
  excludeSubtreeOf?: string
  excludeAncestorsOf?: string
  showLocation?: boolean
}>()
const emit = defineEmits<{ resolved: [asset: Asset | null], valid: [valid: boolean] }>()
const model = defineModel<string | null>({ default: null })
const apiFetch = useApiFetch()
const search = ref('')
const results = ref<Asset[]>([])
const selectedAsset = ref<Asset | null>(null)
const pending = ref(false)
const resolving = ref(false)
const error = ref('')
const selectionError = ref('')
const total = ref(0)
let searchRequest = 0
let selectionRequest = 0
let timer: ReturnType<typeof setTimeout> | undefined

async function loadMore(reset = false) {
  const request = ++searchRequest
  pending.value = true
  error.value = ''
  const params = new URLSearchParams({ q: search.value, limit: '20', offset: String(reset ? 0 : results.value.length) })
  if (props.excludeSubtreeOf) params.set('exclude_subtree_of', props.excludeSubtreeOf)
  if (props.excludeAncestorsOf) params.set('exclude_ancestors_of', props.excludeAncestorsOf)
  try {
    const response = await apiFetch<AssetsResponse>(`/api/assets?${params}`)
    if (request !== searchRequest) return
    results.value = reset ? response.assets : [...results.value, ...response.assets]
    total.value = response.total
  } catch {
    if (request === searchRequest) error.value = 'Assets could not be loaded.'
  } finally {
    if (request === searchRequest) pending.value = false
  }
}

async function resolveSelection() {
  const request = ++selectionRequest
  const id = model.value
  selectedAsset.value = null
  selectionError.value = ''
  emit('resolved', null)
  emit('valid', !id)
  resolving.value = Boolean(id)
  if (!id) return
  try {
    const asset = await apiFetch<Asset>(`/api/assets/${id}`)
    if (request !== selectionRequest) return
    selectedAsset.value = asset
    emit('resolved', asset)
    emit('valid', true)
  } catch {
    if (request === selectionRequest) selectionError.value = 'Selected asset is unavailable. Choose another asset or clear the selection.'
  } finally {
    if (request === selectionRequest) resolving.value = false
  }
}

watch(model, resolveSelection, { immediate: true })
watch(() => [search.value, props.excludeSubtreeOf, props.excludeAncestorsOf], () => {
  clearTimeout(timer)
  ++searchRequest
  results.value = []
  timer = setTimeout(() => loadMore(true), 250)
})
onBeforeUnmount(() => {
  clearTimeout(timer)
  ++searchRequest
  ++selectionRequest
})

const value = computed({ get: () => model.value || undefined, set: (id: string | undefined) => {
  model.value = id || null
} })
const options = computed(() => {
  const assets = new Map(results.value.map(asset => [asset.id, asset]))
  if (selectedAsset.value) assets.set(selectedAsset.value.id, selectedAsset.value)
  return [...assets.values()].map(asset => ({
    value: asset.id,
    label: asset.parent ? `${asset.name} — inside ${asset.parent.name}` : asset.name
  }))
})
</script>

<template>
  <div class="space-y-2">
    <label class="block text-xs font-bold text-muted">{{ label }}</label>
    <USelectMenu
      v-model="value"
      v-model:search-term="search"
      :aria-label="label"
      :items="options"
      :loading="pending || resolving"
      value-key="value"
      ignore-filter
      placeholder="Search and choose an asset"
      class="w-full"
      @update:open="open => open && loadMore(true)"
    />
    <div class="flex gap-3">
      <UButton
        v-if="model"
        type="button"
        variant="link"
        size="xs"
        @click="model = null"
      >
        Clear {{ label.toLowerCase() }}
      </UButton>
      <UButton
        v-if="results.length < total"
        type="button"
        variant="link"
        size="xs"
        :disabled="pending"
        @click="loadMore()"
      >
        Load more assets
      </UButton>
    </div>
    <p
      v-if="selectedAsset?.parent && label !== 'Inside'"
      class="text-xs text-muted"
    >
      Saving will move this asset out of {{ selectedAsset.parent.name }}.
    </p>
    <p
      v-if="showLocation && model"
      class="text-sm text-muted"
    >
      Location: {{ selectedAsset?.location?.name || 'Unset' }}. Contents move with their parent.
    </p>
    <p
      v-if="selectionError || error"
      role="alert"
      class="text-sm text-error"
    >
      {{ selectionError || error }}
    </p>
    <UButton
      v-if="selectionError || error"
      type="button"
      variant="link"
      size="xs"
      @click="selectionError ? resolveSelection() : loadMore(true)"
    >
      Try again
    </UButton>
  </div>
</template>
