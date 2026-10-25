<script setup lang="ts">
import type { AssetAttributeValue } from '~/types/api'
import { urlAttributeValue } from '~/utils/urlAttributes'

const props = defineProps<{ required?: boolean, name: string }>()
const model = defineModel<AssetAttributeValue>()
const value = computed(() => urlAttributeValue(model.value))
function update(field: 'url' | 'label', text: string) {
  const next = { url: value.value?.url || '', label: value.value?.label || '', [field]: text }
  model.value = next.url || next.label ? next : undefined
}
</script>

<template>
  <div class="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
    <UInput
      :model-value="value?.url || ''"
      type="url"
      placeholder="https://paperless.example/documents/123/"
      :aria-label="`${props.name} URL`"
      :required="required || !!value?.label"
      class="min-w-0 w-full"
      @update:model-value="update('url', String($event))"
    />
    <UInput
      :model-value="value?.label || ''"
      placeholder="Link text (optional)"
      :aria-label="`${props.name} link text (optional)`"
      class="min-w-0 w-full"
      @update:model-value="update('label', String($event))"
    />
  </div>
</template>
