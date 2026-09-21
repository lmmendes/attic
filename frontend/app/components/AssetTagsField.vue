<script setup lang="ts">
import type { Tag } from '~/types/api'

const props = defineProps<{ modelValue: string[], tags: Tag[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const error = ref('')
const value = computed({
  get: () => props.modelValue,
  set: (next: string[]) => {
    error.value = next.length > 50 ? 'Assign at most 50 tags.' : ''
    if (!error.value) emit('update:modelValue', next)
  }
})
const items = computed(() => [...new Set([...props.tags.map(tag => tag.name), ...props.modelValue])]
  .sort((left, right) => left.localeCompare(right)))

function create(raw: string) {
  const name = raw.trim()
  if (!name || [...name].length > 100) {
    error.value = 'Tag names must contain 1 to 100 characters.'
    return
  }
  if (props.modelValue.length >= 50) {
    error.value = 'Assign at most 50 tags.'
    return
  }
  const existing = items.value.find(item => item.toLocaleLowerCase() === name.toLocaleLowerCase()) || name
  if (!props.modelValue.some(item => item.toLocaleLowerCase() === existing.toLocaleLowerCase())) {
    emit('update:modelValue', [...props.modelValue, existing])
  }
  error.value = ''
}
</script>

<template>
  <UFormField
    label="Tags"
    description="Choose existing tags or type a new one. New tags are created when you save."
    :error="error || undefined"
  >
    <UInputMenu
      v-model="value"
      multiple
      create-item
      :items="items"
      placeholder="Add tags"
      aria-label="Tags"
      class="w-full"
      @create="create"
    />
  </UFormField>
</template>
