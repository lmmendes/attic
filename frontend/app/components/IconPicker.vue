<script setup lang="ts">
import type { LucideIconCatalogEntry } from '~/utils/iconCatalog'
import { filterIconCatalog, iconPickerPageSize } from '~/utils/iconCatalog'
import { getIconLabel } from '~/utils/iconLabel'

interface Props {
  modelValue?: string
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: undefined,
  label: 'Icon'
})
const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const config = useRuntimeConfig()
const catalog = config.public.lucideIconCatalog as LucideIconCatalogEntry[]
const search = ref('')
const visibleCount = ref(iconPickerPageSize)

watch(search, () => {
  visibleCount.value = iconPickerPageSize
})

const filteredIcons = computed(() => filterIconCatalog(catalog, search.value))
const visibleIcons = computed(() => filteredIcons.value.slice(0, visibleCount.value))
const hasMore = computed(() => visibleCount.value < filteredIcons.value.length)

function selectIcon(icon: string) {
  emit('update:modelValue', icon)
}

function showMore() {
  visibleCount.value += iconPickerPageSize
}
</script>

<template>
  <fieldset class="space-y-3">
    <legend class="text-sm font-medium text-default">
      {{ props.label }}
    </legend>

    <div
      v-if="props.modelValue"
      class="flex items-center gap-3 rounded-xl border border-subtle bg-muted/40 px-3 py-2"
    >
      <span class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-default text-attic-500 shadow-sm">
        <UIcon
          :name="props.modelValue"
          class="size-5"
        />
      </span>
      <span class="min-w-0">
        <span class="block text-[11px] font-bold uppercase tracking-wider text-muted">Selected</span>
        <span class="block truncate text-sm font-semibold text-default">{{ getIconLabel(props.modelValue) }}</span>
      </span>
    </div>

    <UInput
      v-model="search"
      :aria-label="`Search ${props.label.toLowerCase()}s`"
      icon="i-lucide-search"
      placeholder="Search Lucide icons"
      autocomplete="off"
      class="w-full"
    />

    <div
      v-if="filteredIcons.length"
      class="max-h-72 overflow-y-auto pr-1 custom-scrollbar"
    >
      <div class="grid grid-cols-6 gap-2 sm:grid-cols-8">
        <button
          v-for="icon in visibleIcons"
          :key="icon.name"
          type="button"
          :aria-label="`${getIconLabel(icon.name)} icon`"
          :title="`${getIconLabel(icon.name)} icon`"
          :aria-pressed="props.modelValue === icon.name"
          class="flex aspect-square items-center justify-center rounded-xl border transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-attic-500"
          :class="props.modelValue === icon.name
            ? 'border-attic-500 bg-attic-500 text-white'
            : 'border-subtle bg-default text-muted hover:border-attic-400 hover:text-attic-500'"
          @click="selectIcon(icon.name)"
        >
          <UIcon
            :name="icon.name"
            class="size-5"
          />
        </button>
      </div>
      <UButton
        v-if="hasMore"
        type="button"
        color="neutral"
        variant="soft"
        block
        class="mt-3"
        @click="showMore"
      >
        Show more
      </UButton>
    </div>
    <p
      v-else
      role="status"
      class="rounded-xl border border-dashed border-subtle px-4 py-6 text-center text-sm text-muted"
    >
      No icons match “{{ search.trim() }}”.
    </p>

    <p
      class="text-xs text-muted"
      aria-live="polite"
    >
      Showing {{ visibleIcons.length }} of {{ filteredIcons.length }} icons
    </p>
  </fieldset>
</template>
