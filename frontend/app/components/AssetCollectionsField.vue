<script setup lang="ts">
import type { Collection } from '~/types/api'

const model = defineModel<string[]>({ required: true })
const { data, status, error, refresh } = useApi<Collection[]>('/api/collections')
const options = computed(() => data.value?.map(c => ({ label: c.name, value: c.id, icon: c.icon })) || [])
</script>

<template>
  <div class="space-y-2">
    <label
      for="asset-collections"
      class="block text-sm font-semibold"
    >Collections <span class="font-normal text-muted">(optional)</span></label>
    <p class="text-xs text-muted">
      Group this asset with others, such as PS5 games or furniture. Choose as many as you like.
    </p>
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
      placeholder="Choose collections"
      aria-label="Collections"
      class="w-full"
    />
    <div class="flex items-center justify-between gap-3 text-xs text-muted">
      <NuxtLink
        to="/collections"
        target="_blank"
        class="underline"
      >Manage collections (opens in a new tab)</NuxtLink>
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
