<script setup lang="ts">
defineProps<{ title: string, placeholder: string, count: number, total: number }>()
const search = defineModel<string>({ required: true })
</script>

<template>
  <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
    <div>
      <h2 class="font-extrabold text-mist-950 dark:text-white">
        {{ title }}
      </h2>
      <p
        class="mt-1 text-xs text-muted"
        role="status"
      >
        Showing {{ count }} of {{ total }}
      </p>
    </div>
    <UInput
      v-model="search"
      icon="i-lucide-search"
      :placeholder="placeholder"
      :aria-label="placeholder"
      class="w-full sm:w-72"
      size="lg"
    />
  </div>
</template>
