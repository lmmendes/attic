<script setup lang="ts">
defineProps<{
  name: string
  description?: string
  icon: string
  assetCount: number
  assetsTo: string
  editTo?: string
  iconClass?: string
}>()
const emit = defineEmits<{ edit: [], delete: [] }>()
</script>

<template>
  <article
    role="listitem"
    class="grid min-w-0 gap-4 px-4 py-4 transition-colors hover:bg-mist-50/70 dark:hover:bg-mist-800/60 sm:grid-cols-[minmax(0,1.05fr)_minmax(0,1.35fr)_auto] sm:items-center sm:px-5"
  >
    <div class="flex min-w-0 items-center gap-3">
      <div
        class="flex size-11 shrink-0 items-center justify-center rounded-xl"
        :class="iconClass || 'bg-attic-500/10 text-attic-500 dark:text-attic-300'"
      >
        <UIcon
          :name="icon"
          class="size-5"
        />
      </div>
      <div class="min-w-0">
        <h3
          class="truncate text-sm font-extrabold text-mist-950 dark:text-white sm:text-base"
          :title="name"
        >
          {{ name }}
        </h3>
        <p class="mt-0.5 text-xs text-muted">
          {{ assetCount }} {{ assetCount === 1 ? 'asset' : 'assets' }}
        </p>
      </div>
    </div>

    <div class="min-w-0 pl-14 sm:pl-0">
      <p class="line-clamp-2 text-sm leading-5 text-muted">
        {{ description || 'No description yet.' }}
      </p>
      <div
        v-if="$slots.metadata"
        class="mt-2 flex flex-wrap items-center gap-2"
      >
        <slot name="metadata" />
      </div>
    </div>

    <div class="flex items-center justify-end gap-1.5 pl-14 sm:pl-0">
      <NuxtLink
        :to="assetsTo"
        :aria-label="`View assets in ${name}`"
        class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-sm font-bold text-attic-600 hover:bg-attic-50 focus-visible:outline-2 focus-visible:outline-attic-500 dark:text-attic-300 dark:hover:bg-attic-500/10"
      >
        View assets
        <UIcon
          name="i-lucide-arrow-right"
          class="size-4"
        />
      </NuxtLink>
      <UButton
        :to="editTo"
        icon="i-lucide-pencil"
        color="neutral"
        variant="outline"
        size="sm"
        class="rounded-lg"
        :aria-label="`Edit ${name}`"
        @click="emit('edit')"
      >
        Edit
      </UButton>
      <UDropdownMenu :items="[{ label: 'Delete', icon: 'i-lucide-trash-2', color: 'error', onSelect: () => emit('delete') }]">
        <UButton
          icon="i-lucide-ellipsis"
          color="neutral"
          variant="ghost"
          class="shrink-0 rounded-lg"
          :aria-label="`Actions for ${name}`"
        />
      </UDropdownMenu>
    </div>
  </article>
</template>
