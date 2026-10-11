<script setup lang="ts">
import type { Asset } from '~/types/api'

const props = defineProps<{ asset: Asset, editing?: boolean, removed?: boolean }>()
const emit = defineEmits<{ remove: [], undo: [] }>()
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
      <span
        class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-mist-100 text-mist-400 dark:bg-mist-800"
        aria-hidden="true"
      ><UIcon
        name="i-lucide-package"
        class="size-4"
      /></span>
      <div class="min-w-0 flex-1">
        <NuxtLink
          :to="`/assets/${asset.id}`"
          class="break-words text-sm font-bold text-default transition-colors hover:text-attic-500"
        >{{ asset.name }}</NuxtLink>
        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
          <span
            v-if="asset.child_count"
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
        :aria-label="`${removed ? 'Undo removal of' : 'Detach'} ${asset.name}`"
        @click="removed ? emit('undo') : emit('remove')"
      >
        {{ removed ? 'Undo' : 'Detach' }}
      </UButton>
    </div>
  </li>
</template>
