<script setup lang="ts">
import type { AttributeOption } from '~/types/api'

const options = defineModel<AttributeOption[]>({ default: () => [] })
const newLabel = ref('')
const newValue = ref('')
const editedValue = ref(false)
const drafts = reactive<Record<string, { label: string, value: string }>>({})
watch(newLabel, (label) => {
  if (!editedValue.value) newValue.value = label.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_|_$/g, '')
})
watch(options, (items) => {
  for (const o of items) drafts[o.id] = { label: o.label, value: o.value }
}, { immediate: true, deep: true })
function addOption() {
  if (!newLabel.value.trim() || !newValue.value.trim()) return
  const option = { id: crypto.randomUUID(), label: newLabel.value.trim(), value: newValue.value.trim(), sort_order: options.value.length }
  options.value = [...options.value, option]
  newLabel.value = ''
  newValue.value = ''
  editedValue.value = false
}
function editOption(option: AttributeOption, field: 'label' | 'value', value: string | number) {
  drafts[option.id]![field] = String(value)
  options.value = options.value.map(o => o.id === option.id ? { ...o, [field]: String(value) } : o)
}
function deleteOption(option: AttributeOption) {
  options.value = options.value.filter(o => o.id !== option.id)
}
function move(index: number, direction: number) {
  const items = [...options.value]
  const other = index + direction
  if (other < 0 || other >= items.length) return
  ;[items[index], items[other]] = [items[other]!, items[index]!]
  options.value = items.map((o, sort_order) => ({ ...o, sort_order }))
}
</script>

<template>
  <section class="space-y-4">
    <h2 class="font-semibold">
      Options
    </h2>
    <p class="text-sm text-muted">
      Option changes are saved when you save the attribute.
    </p>
    <div
      v-for="(option, index) in options"
      :key="option.id"
      class="space-y-2 rounded-lg border border-default p-3"
    >
      <div
        v-if="drafts[option.id]"
        class="grid gap-2 sm:grid-cols-2"
      >
        <UInput
          :model-value="drafts[option.id]!.label"
          :aria-label="`Label for ${option.label}`"
          placeholder="Label"
          @update:model-value="editOption(option, 'label', $event)"
        />
        <UInput
          :model-value="drafts[option.id]!.value"
          :aria-label="`Stored value for ${option.label}`"
          placeholder="Stored value"
          @update:model-value="editOption(option, 'value', $event)"
        />
      </div>
      <div class="flex flex-wrap gap-2">
        <UButton
          size="xs"
          color="error"
          variant="soft"
          icon="i-lucide-trash-2"
          :aria-label="`Delete ${option.label || 'option'}`"
          title="Delete option"
          @click="deleteOption(option)"
        />
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          :disabled="index === 0"
          :aria-label="`Move ${option.label} up`"
          @click="move(index, -1)"
        >
          ↑
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          :disabled="index === options.length - 1"
          :aria-label="`Move ${option.label} down`"
          @click="move(index, 1)"
        >
          ↓
        </UButton>
      </div>
    </div>
    <div class="grid gap-2 sm:grid-cols-2">
      <UInput
        v-model="newLabel"
        aria-label="New option label"
        placeholder="Option label"
      />
      <UInput
        v-model="newValue"
        aria-label="New option stored value"
        placeholder="Stored value"
        @update:model-value="editedValue = true"
      />
    </div>
    <UButton
      :disabled="!newLabel.trim() || !newValue.trim()"
      icon="i-lucide-plus"
      @click="addOption"
    >
      Add option
    </UButton>
  </section>
</template>
