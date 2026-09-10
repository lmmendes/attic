<script setup lang="ts">
import type { Category } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const categoryId = computed(() => route.params.id as string)
const { data: category, status: categoryStatus } = useApi<Category>(
  () => `/api/categories/${categoryId.value}`
)
</script>

<template>
  <div
    v-if="categoryStatus === 'pending'"
    class="flex items-center justify-center py-20"
  >
    <UIcon
      name="i-lucide-loader-2"
      class="w-8 h-8 text-attic-500 animate-spin"
    />
  </div>

  <CategoryEditor
    v-else-if="category"
    :category="category"
  />

  <div
    v-else
    class="flex flex-col items-center justify-center py-20"
  >
    <UIcon
      name="i-lucide-alert-circle"
      class="w-12 h-12 text-muted mb-4"
    />
    <h2 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
      Category not found
    </h2>
    <p class="text-muted mb-4">
      The category you're looking for doesn't exist.
    </p>
    <UButton
      to="/categories"
      variant="soft"
    >
      Back to Categories
    </UButton>
  </div>
</template>
