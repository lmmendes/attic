<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
await navigateTo('/profile?tab=api-keys', { replace: true })
</script>

<template>
  <span class="sr-only">Opening your API keys…</span>
</template>
