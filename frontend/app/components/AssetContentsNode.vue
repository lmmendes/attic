<script setup lang="ts">
import type { Asset, AssetsResponse } from '~/types/api'

const props = defineProps<{ asset: Asset, editing?: boolean, removed?: boolean }>()
const emit = defineEmits<{ remove: [], undo: [] }>()
const expanded = ref(true)
const offset = ref(0)
const hasChildren = computed(() => (props.asset.child_count || 0) > 0)
const childrenUrl = computed(() => `/api/assets?${new URLSearchParams({ parent_id: props.asset.id, limit: '10', offset: String(offset.value) })}`)
const { data, status, error, refresh } = useApi<AssetsResponse>(() => childrenUrl.value, { immediate: hasChildren.value })
const price = computed(() => props.asset.purchase_price == null
  ? 'Price unknown'
  : new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(props.asset.purchase_price * props.asset.quantity))
</script>

<template>
  <li class="contents-node min-w-0">
    <div
      class="contents-row flex items-center gap-3 rounded-xl p-3"
      :class="{ 'opacity-50': removed }"
    >
      <button
        v-if="hasChildren"
        type="button"
        :aria-label="`${expanded ? 'Collapse' : 'Expand'} ${asset.name}`"
        :aria-expanded="expanded"
        :aria-controls="`contents-${asset.id}`"
        class="flex size-8 shrink-0 items-center justify-center rounded-lg text-muted hover:bg-mist-200/60 focus-visible:outline-2 focus-visible:outline-attic-500 dark:hover:bg-mist-700"
        @click="expanded = !expanded"
      >
        <UIcon
          :name="expanded ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
          class="size-4"
        />
      </button>
      <span
        v-else
        class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-mist-100 text-mist-400 dark:bg-mist-800"
        aria-hidden="true"
      >
        <UIcon
          name="i-lucide-package"
          class="size-4"
        />
      </span>
      <div class="min-w-0 flex-1">
        <NuxtLink
          :to="`/assets/${asset.id}`"
          class="break-words text-sm font-bold text-default transition-colors hover:text-attic-500"
        >{{ asset.name }}</NuxtLink>
        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
          <span
            v-if="hasChildren"
            class="inline-flex items-center gap-1 text-attic-500 dark:text-attic-300"
          >
            <UIcon
              name="i-lucide-layers"
              class="size-3"
              aria-hidden="true"
            />
            {{ asset.child_count }} {{ asset.child_count === 1 ? 'component' : 'components' }}
          </span>
          <span>Qty {{ asset.quantity }}</span>
          <span>{{ price }}</span>
        </div>
        <p
          v-if="removed"
          class="text-xs text-muted"
        >
          Will detach on save, with its contents
        </p>
      </div>
      <UButton
        v-if="editing"
        type="button"
        variant="link"
        :aria-label="`${removed ? 'Undo removal of' : 'Remove'} ${asset.name}`"
        @click="removed ? emit('undo') : emit('remove')"
      >
        {{ removed ? 'Undo' : 'Remove' }}
      </UButton>
    </div>
    <div
      v-if="hasChildren && expanded"
      :id="`contents-${asset.id}`"
      class="contents-children ml-7 pl-5"
    >
      <p
        v-if="status === 'pending'"
        role="status"
        class="px-2 py-2 text-xs text-muted"
      >
        Loading contents of {{ asset.name }}…
      </p>
      <div
        v-else-if="error"
        role="alert"
        class="px-2 py-2 text-sm text-error"
      >
        Contents of {{ asset.name }} could not be loaded.
        <UButton
          type="button"
          variant="link"
          @click="refresh()"
        >
          Try again
        </UButton>
      </div>
      <ul
        v-else
        :aria-label="`Contents of ${asset.name}`"
      >
        <AssetContentsNode
          v-for="child in data?.assets"
          :key="child.id"
          :asset="child"
        />
      </ul>
      <div
        v-if="!error && (data?.total || 0) > 10"
        class="flex flex-wrap items-center gap-2 px-2 py-2"
      >
        <UButton
          type="button"
          variant="ghost"
          size="xs"
          :disabled="offset === 0 || status === 'pending'"
          :aria-label="`Previous contents of ${asset.name}`"
          @click="offset -= 10"
        >
          Previous
        </UButton>
        <span class="text-xs text-muted">{{ offset + 1 }}–{{ Math.min(offset + 10, data?.total || 0) }} of {{ data?.total }}</span>
        <UButton
          type="button"
          variant="ghost"
          size="xs"
          :disabled="offset + 10 >= (data?.total || 0) || status === 'pending'"
          :aria-label="`Next contents of ${asset.name}`"
          @click="offset += 10"
        >
          Next
        </UButton>
      </div>
    </div>
  </li>
</template>

<style scoped>
.contents-row { transition: background-color 150ms; }
.contents-row:hover { background: var(--ui-bg-muted); }
.contents-children { position: relative; padding-top: 6px; }
.contents-children > ul > .contents-node { position: relative; }
.contents-children > ul > .contents-node::before { content: ''; position: absolute; left: -20px; top: -6px; width: 14px; height: 40px; border-left: 1px solid var(--ui-border-accented); border-bottom: 1px solid var(--ui-border-accented); border-bottom-left-radius: 8px; }
.contents-children > ul > .contents-node:not(:last-child)::after { content: ''; position: absolute; left: -20px; top: 0; bottom: -6px; border-left: 1px solid var(--ui-border-accented); }
@media (max-width: 480px) {
  .contents-row { gap: 8px; padding: 10px 8px; }
  .contents-children { margin-left: 23px; padding-left: 14px; }
  .contents-children > ul > .contents-node::before, .contents-children > ul > .contents-node::after { left: -14px; }
  .contents-children > ul > .contents-node::before { width: 10px; }
}
</style>
