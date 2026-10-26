<script setup lang="ts">
import type { Asset } from '~/types/api'

withDefaults(defineProps<{
  asset: Asset
  layout?: 'responsive' | 'grid' | 'list'
  priceLabel?: string
  updatedLabel?: string
  showDescription?: boolean
}>(), { layout: 'responsive' })
const { features } = useFeatures()
</script>

<template>
  <article class="attic-panel attic-panel-interactive group flex min-w-0 flex-col overflow-hidden rounded-[18px]">
    <NuxtLink
      :to="`/assets/${asset.id}`"
      :aria-label="asset.name"
      class="min-w-0 flex-1"
      :class="layout === 'grid' ? 'flex flex-col' : layout === 'list' ? 'flex' : 'flex sm:flex-col'"
    >
      <div
        class="relative flex shrink-0 items-center justify-center overflow-hidden bg-gradient-to-br from-attic-50 to-mist-100 dark:from-mist-700 dark:to-mist-800"
        :class="layout === 'grid' ? 'aspect-[2/1] w-full' : layout === 'list' ? 'w-20 sm:w-24' : 'w-20 sm:aspect-[2/1] sm:w-full'"
      >
        <img
          v-if="asset.main_attachment_url"
          :src="asset.main_attachment_url"
          :alt="asset.name"
          loading="lazy"
          class="size-full object-cover transition duration-300 group-hover:scale-[1.03]"
        >
        <div
          v-else
          class="flex size-12 items-center justify-center rounded-2xl bg-white/80 text-attic-500 shadow-sm dark:bg-mist-700 dark:text-attic-300"
        >
          <UIcon
            name="i-lucide-package"
            class="size-6"
          />
        </div>
        <span
          v-if="features.categories"
          class="absolute left-3 top-3 max-w-[calc(100%-1.5rem)] truncate rounded-full bg-white/90 px-2.5 py-1 text-[10px] font-extrabold text-mist-600 shadow-sm backdrop-blur dark:bg-mist-900/85 dark:text-mist-300"
          :class="layout === 'list' ? 'hidden' : layout === 'responsive' ? 'hidden sm:inline' : ''"
        >{{ asset.category?.name || 'Uncategorized' }}</span>
      </div>
      <div class="min-w-0 flex-1 p-3.5">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="line-clamp-2 break-words font-extrabold text-mist-950 dark:text-white">{{ asset.name }}</h3>
              <span
                v-if="features.conditions && asset.condition"
                class="rounded-md bg-green-100 px-1.5 py-0.5 text-[10px] font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400"
              >{{ asset.condition.label }}</span>
            </div>
            <p
              v-if="features.categories && layout !== 'grid'"
              class="mt-1 truncate text-[10px] font-bold uppercase text-muted"
              :class="layout === 'responsive' ? 'sm:hidden' : ''"
            >{{ asset.category?.name || 'Uncategorized' }}</p>
            <p
              v-if="features.locations"
              class="mt-1 flex items-center gap-1.5 text-xs text-muted"
            >
              <UIcon
                name="i-lucide-map-pin"
                class="size-3.5 shrink-0"
              />
              <span class="truncate">{{ asset.location?.name || 'No location' }}</span>
            </p>
            <div
              v-if="features.collections"
              class="mt-2 flex min-w-0 items-center gap-1.5 text-[11px] text-muted"
              :title="asset.collections?.length ? asset.collections.map(collection => collection.name).join(', ') : 'No collections'"
            >
              <UIcon
                name="i-lucide-library"
                class="size-3.5 shrink-0"
              />
              <template v-if="asset.collections?.length">
                <span class="truncate rounded-md bg-attic-50 px-1.5 py-0.5 font-semibold text-attic-600 dark:bg-attic-500/10 dark:text-attic-300">{{ asset.collections[0]?.name }}</span>
                <span
                  v-if="asset.collections.length > 1"
                  class="shrink-0 font-bold text-attic-500 dark:text-attic-300"
                >+{{ asset.collections.length - 1 }}</span>
              </template>
              <span v-else>No collections</span>
            </div>
          </div>
          <UIcon
            name="i-lucide-arrow-up-right"
            class="mt-0.5 size-4 shrink-0 text-mist-300 transition group-hover:text-attic-500 dark:text-attic-300"
          />
        </div>
        <p
          v-if="showDescription && asset.description"
          class="mt-2 line-clamp-2 text-xs leading-relaxed text-muted"
        >{{ asset.description }}</p>
        <p
          v-if="priceLabel"
          class="mt-2 text-xs font-semibold tabular-nums text-default"
        >{{ priceLabel }}</p>
        <div
          v-if="$slots.metadata"
          class="mt-2 space-y-1 text-xs text-muted"
        ><slot name="metadata" /></div>
        <p
          v-if="updatedLabel"
          class="mt-2.5 text-[11px] font-semibold text-muted"
        >{{ updatedLabel }}</p>
      </div>
    </NuxtLink>
    <div
      v-if="$slots.footer"
      class="border-t border-mist-100 px-3.5 py-2 dark:border-mist-700"
    >
      <slot name="footer" />
    </div>
  </article>
</template>
