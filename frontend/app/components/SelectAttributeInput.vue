<script setup lang="ts">
import type { Attribute } from '~/types/api'

const props = defineProps<{ attribute: Attribute, modelValue?: unknown, required?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string | string[] | undefined] }>()
const single = computed(() => typeof props.modelValue === 'string' && props.modelValue ? props.modelValue : undefined)
const multiple = computed(() => Array.isArray(props.modelValue) ? props.modelValue.filter((v): v is string => typeof v === 'string') : [])
function remove(value: string) {
  emit('update:modelValue', multiple.value.filter(v => v !== value))
}
function label(value: string) {
  return props.attribute.options?.find(o => o.value === value)?.label || value
}
</script>

<template>
  <div class="space-y-2">
    <USelectMenu
      v-if="attribute.selection_mode === 'multiple'"
      :id="`attr-${attribute.key}`"
      :model-value="multiple"
      :items="attribute.options || []"
      value-key="value"
      multiple
      :aria-label="attribute.name"
      :placeholder="`Select ${attribute.name.toLowerCase()}`"
      class="w-full"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <USelectMenu
      v-else
      :id="`attr-${attribute.key}`"
      :model-value="single"
      :items="attribute.options || []"
      value-key="value"
      :aria-label="attribute.name"
      :placeholder="`Select ${attribute.name.toLowerCase()}`"
      class="w-full"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <div
      v-if="attribute.selection_mode === 'multiple'"
      class="flex flex-wrap gap-2"
    >
      <UButton
        v-for="value in multiple"
        :key="value"
        size="xs"
        variant="soft"
        trailing-icon="i-lucide-x"
        :aria-label="`Remove ${label(value)}`"
        @click="remove(value)"
      >
        {{ label(value) }}
      </UButton>
    </div>
    <UButton
      v-if="!required && (single || multiple.length)"
      size="xs"
      variant="link"
      @click="emit('update:modelValue', undefined)"
    >
      Clear selection
    </UButton>
  </div>
</template>
