<script setup lang="ts">
import type { Asset, AssetsResponse } from '~/types/api'

const props = defineProps<{ assetId: string, editing?: boolean }>()
const emit = defineEmits<{ changed: [] }>()
const added = defineModel<Asset[]>('added', { default: () => [] })
const removed = defineModel<string[]>('removed', { default: () => [] })
const apiFetch = useApiFetch()
const toast = useToast()
const offset = ref(0)
const selected = ref<string | null>(null)
const selectedAsset = ref<Asset | null>(null)
const busy = ref(false)
const pickerOpen = ref(false)
const contentsUrl = computed(() => `/api/assets?${new URLSearchParams({ parent_id: props.assetId, limit: '10', offset: String(offset.value) })}`)
const { data, status, error, refresh } = useApi<AssetsResponse>(() => contentsUrl.value)

async function linkSelected() {
  const asset = selectedAsset.value
  if (!asset || busy.value) return
  if (props.editing) {
    removed.value = removed.value.filter(id => id !== asset.id)
    if (asset.parent_id !== props.assetId && !added.value.some(item => item.id === asset.id)) added.value = [...added.value, asset]
    selected.value = null
    pickerOpen.value = false
    return
  }
  busy.value = true
  try {
    await apiFetch(`/api/assets/${asset.id}/parent`, { method: 'PATCH', body: JSON.stringify({ parent_id: props.assetId }) })
    selected.value = null
    pickerOpen.value = false
    await refresh()
    emit('changed')
  } catch (err: unknown) {
    const failure = err as { data?: { error?: string } }
    toast.add({ title: failure.data?.error || 'Failed to link asset', color: 'error' })
  } finally { busy.value = false }
}

function remove(asset: Asset) {
  if (added.value.some(item => item.id === asset.id)) added.value = added.value.filter(item => item.id !== asset.id)
  else if (!removed.value.includes(asset.id)) removed.value = [...removed.value, asset.id]
}

watch(() => props.assetId, () => {
  offset.value = 0
  selected.value = null
  pickerOpen.value = false
})
</script>

<template>
  <section
    class="attic-panel overflow-hidden rounded-[20px] text-default"
    aria-label="Contents"
  >
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-mist-100 bg-mist-50/60 px-5 py-3.5 dark:border-mist-700 dark:bg-mist-800/50">
      <div>
        <h3 class="flex items-center gap-2 text-base font-bold text-default">
          <UIcon
            name="i-lucide-boxes"
            class="size-5 text-attic-500"
            aria-hidden="true"
          />
          Contents
          <span
            v-if="data?.total"
            class="rounded-md bg-mist-100 px-2 py-0.5 text-xs font-medium text-muted dark:bg-mist-800"
          >{{ data.total }}</span>
        </h3>
      </div>
      <div class="flex flex-wrap gap-2">
        <UButton
          v-if="!editing"
          :to="{ path: '/assets/new', query: { parent_id: assetId } }"
          size="sm"
          variant="soft"
          icon="i-lucide-plus"
        >
          Add asset
        </UButton>
        <UButton
          type="button"
          size="sm"
          variant="ghost"
          icon="i-lucide-link"
          :aria-expanded="pickerOpen"
          :aria-controls="`link-content-${assetId}`"
          @click="pickerOpen = !pickerOpen"
        >
          Link existing asset
        </UButton>
      </div>
    </div>
    <div class="space-y-4 p-4 sm:p-5">
      <p
        v-if="editing"
        class="text-xs text-muted"
      >
        Contents changes apply when you save this asset.
      </p>
      <div
        v-if="pickerOpen"
        :id="`link-content-${assetId}`"
        class="space-y-3 rounded-xl border border-mist-200 bg-mist-50/50 p-4 dark:border-mist-700 dark:bg-mist-800/50"
      >
        <AssetPicker
          v-model="selected"
          label="Content asset"
          :exclude-ancestors-of="assetId"
          @resolved="selectedAsset = $event"
        />
        <div class="flex flex-wrap gap-2">
          <UButton
            type="button"
            :disabled="!selectedAsset || busy"
            :loading="busy"
            @click="linkSelected"
          >
            {{ editing ? 'Stage addition' : 'Link asset' }}
          </UButton>
          <UButton
            type="button"
            variant="ghost"
            color="neutral"
            :disabled="busy"
            @click="pickerOpen = false; selected = null"
          >
            Cancel
          </UButton>
        </div>
      </div>
      <div
        v-if="error"
        role="alert"
      >
        Contents could not be loaded. <UButton
          type="button"
          variant="link"
          @click="refresh()"
        >
          Try again
        </UButton>
      </div>
      <p
        v-else-if="status === 'pending'"
        role="status"
      >
        Loading contents…
      </p>
      <template v-else>
        <p
          v-if="!data?.total && !added.length"
          class="text-sm text-muted"
        >
          No contents yet. Add a new asset or link an existing one.
        </p>
        <ul class="space-y-2">
          <AssetContentsNode
            v-for="child in data?.assets"
            :key="child.id"
            :asset="child"
            :editing="editing"
            :removed="removed.includes(child.id)"
            @remove="remove(child)"
            @undo="removed = removed.filter(id => id !== child.id)"
          />
          <li
            v-for="child in added"
            :key="child.id"
            class="flex items-center justify-between gap-3"
          >
            <span>{{ child.name }} <span class="text-sm text-muted">Will add on save</span></span>
            <UButton
              type="button"
              variant="link"
              :aria-label="`Remove ${child.name}`"
              @click="remove(child)"
            >
              Remove
            </UButton>
          </li>
        </ul>
        <div
          v-if="(data?.total || 0) > 10"
          class="flex items-center justify-between"
        >
          <UButton
            type="button"
            variant="ghost"
            :disabled="offset === 0"
            @click="offset -= 10"
          >
            Previous contents
          </UButton>
          <span class="text-xs text-muted">{{ offset + 1 }}–{{ Math.min(offset + 10, data?.total || 0) }} of {{ data?.total }}</span>
          <UButton
            type="button"
            variant="ghost"
            :disabled="offset + 10 >= (data?.total || 0)"
            @click="offset += 10"
          >
            Next contents
          </UButton>
        </div>
      </template>
    </div>
  </section>
</template>
