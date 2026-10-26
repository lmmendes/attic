<script setup lang="ts">
import { safeAttributeURL, urlAttributeValue } from '~/utils/urlAttributes'

const props = defineProps<{ value: unknown }>()
const destination = computed(() => safeAttributeURL(props.value))
const text = computed(() => {
  const link = urlAttributeValue(props.value)
  if (link) return link.label?.trim() || link.url || '-'
  if (props.value === undefined || props.value === null || props.value === '') return '-'
  return typeof props.value === 'string' ? props.value : JSON.stringify(props.value)
})
</script>

<template>
  <a
    v-if="destination"
    :href="destination"
    target="_blank"
    rel="noopener noreferrer"
    class="min-w-0 break-all text-sm font-bold text-attic-600 underline dark:text-attic-400"
  >{{ text }}</a>
  <span
    v-else
    class="min-w-0 break-all text-sm font-bold text-mist-950 dark:text-white"
  >{{ text }}</span>
</template>
