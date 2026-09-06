<script setup lang="ts">
defineProps<{ items: { label: string, value: number | string }[] }>()
</script>

<template>
  <dl class="attic-panel grid grid-cols-3 divide-x divide-mist-100 rounded-[18px] px-2 py-4 dark:divide-mist-700 sm:px-4">
    <div
      v-for="item in items"
      :key="item.label"
      class="min-w-0 px-3 sm:px-5"
    >
      <dt class="text-[10px] font-extrabold uppercase tracking-wider text-muted">
        {{ item.label }}
      </dt>
      <dd class="mt-1 text-xl font-black text-mist-950 dark:text-white">
        {{ item.value }}
      </dd>
    </div>
  </dl>
</template>
