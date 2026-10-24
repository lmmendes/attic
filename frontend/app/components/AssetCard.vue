<script setup lang="ts">
import type { Asset } from '~/types/api'

defineProps<{ asset: Asset }>()

const formatRelativeTime = (dateString: string) => {
  const diffDays = Math.floor((Date.now() - new Date(dateString).getTime()) / 86400000)
  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  if (diffDays < 7) return `${diffDays} days ago`
  const weeks = Math.floor(diffDays / 7)
  return `${weeks} week${weeks > 1 ? 's' : ''} ago`
}
</script>

<template>
  <NuxtLink
    :to="`/assets/${asset.id}`"
    class="attic-panel attic-panel-interactive group flex overflow-hidden rounded-[18px] sm:block"
  >
    <div class="relative w-20 shrink-0 overflow-hidden bg-gradient-to-br sm:aspect-[2/1] sm:w-auto from-attic-50 to-mist-100 dark:from-mist-700 dark:to-mist-800">
      <img
        v-if="asset.main_attachment_url"
        :src="asset.main_attachment_url"
        :alt="asset.name"
        class="size-full object-cover transition duration-300 group-hover:scale-[1.03]"
      >
      <div
        v-else
        class="flex size-full items-center justify-center"
      >
        <div class="flex size-14 items-center justify-center rounded-2xl bg-white/80 text-attic-500 dark:text-attic-300 shadow-sm dark:bg-mist-700">
          <UIcon
            name="i-lucide-package"
            class="size-6"
          />
        </div>
      </div>
      <span class="absolute left-3 top-3 hidden rounded-full sm:inline bg-white/90 px-2.5 py-1 text-[10px] font-extrabold text-mist-600 shadow-sm backdrop-blur dark:bg-mist-900/85 dark:text-mist-300">
        {{ asset.category?.name || 'Uncategorized' }}
      </span>
    </div>
    <div class="min-w-0 flex-1 p-3.5">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <h3 class="line-clamp-2 font-extrabold text-mist-950 dark:text-white">{{ asset.name }}</h3>
          <p class="mt-1 flex items-center gap-1.5 truncate text-xs text-muted">
            <UIcon
              name="i-lucide-map-pin"
              class="size-3.5 shrink-0"
            />{{ asset.location?.name || 'No location' }}
          </p>
          <div
            class="mt-2 flex min-w-0 items-center gap-1.5 text-[11px] text-muted"
            :title="asset.collections?.length ? asset.collections.map(collection => collection.name).join(', ') : 'No collections'"
          >
            <UIcon
              name="i-lucide-library"
              class="size-3.5 shrink-0"
            />
            <template v-if="asset.collections?.length">
              <span class="truncate rounded-md bg-attic-50 px-1.5 py-0.5 font-semibold text-attic-600 dark:bg-attic-500/10 dark:text-attic-300">
                {{ asset.collections[0]?.name }}
              </span>
              <span
                v-if="asset.collections.length > 1"
                class="shrink-0 font-bold text-attic-500 dark:text-attic-300"
              >
                +{{ asset.collections.length - 1 }}
              </span>
            </template>
            <span v-else>No collections</span>
          </div>
        </div>
        <UIcon
          name="i-lucide-arrow-up-right"
          class="mt-0.5 size-4 shrink-0 text-mist-300 transition group-hover:text-attic-500 dark:text-attic-300"
        />
      </div>
      <p class="mt-2.5 text-[11px] font-semibold text-muted">{{ formatRelativeTime(asset.updated_at) }}</p>
    </div>
  </NuxtLink>
</template>
