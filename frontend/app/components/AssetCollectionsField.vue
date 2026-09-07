<script setup lang="ts">
import type { Collection } from '~/types/api'

const model = defineModel<string[]>({ required: true })
const { data, status, error, refresh } = useApi<Collection[]>('/api/collections')
const options = computed(() => data.value?.map(c => ({ label: c.name, value: c.id, icon: c.icon })) || [])
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-end justify-between gap-2">
      <div>
        <label
          for="asset-collections"
          class="block text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400"
        >
          Collections <span class="normal-case font-medium text-muted">(optional)</span>
        </label>
        <p class="mt-1 text-xs text-muted">
          Group this asset with others, such as PS5 games or furniture. Choose as many as you like.
        </p>
      </div>
      <NuxtLink
        to="/collections"
        target="_blank"
        rel="noopener noreferrer"
        class="text-xs font-bold text-attic-500 hover:text-attic-600 hover:underline"
      >
        Manage collections
      </NuxtLink>
    </div>
    <div
      v-if="error"
      role="alert"
      class="text-sm text-error"
    >
      Collections could not be loaded. <UButton
        variant="link"
        @click="refresh()"
      >
        Try again
      </UButton>
    </div>
    <USelectMenu
      v-else
      id="asset-collections"
      v-model="model"
      multiple
      :items="options"
      :loading="status === 'pending'"
      :disabled="status === 'pending'"
      value-key="value"
      placeholder="Search and choose a collection"
      icon="i-lucide-library"
      aria-label="Collections"
      class="w-full"
      size="lg"
    />
    <div class="flex items-center justify-end gap-3 text-xs text-muted">
      <UButton
        v-if="model.length"
        color="neutral"
        variant="link"
        size="xs"
        @click="model = []"
      >
        Clear selections
      </UButton>
    </div>
    <p
      v-if="status === 'success' && !data?.length"
      class="text-xs text-muted"
    >
      No collections yet. Create one from Manage collections, then refresh this list.
    </p>
    <UButton
      color="neutral"
      variant="link"
      size="xs"
      icon="i-lucide-refresh-cw"
      @click="refresh()"
    >
      Refresh collections
    </UButton>
  </div>
</template>
