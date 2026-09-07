<script setup lang="ts">
import type { Category } from '~/types/api'
import { buildCategoryTreeRows } from '~/utils/categoryHierarchy'

const props = defineProps<{
  modelValue?: string
  categories: Category[]
  selectedCategory?: Category | null
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | undefined]
}>()

const selection = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value)
})

const rows = computed(() => buildCategoryTreeRows(props.categories))
const options = computed(() => [
  { label: 'No category', value: undefined, icon: 'i-lucide-inbox' },
  ...rows.value.map(row => ({
    label: [...row.ancestors, row.category].map(category => category.name).join(' / '),
    value: row.category.id,
    icon: row.category.icon || 'i-lucide-tag'
  }))
])

const selectedRow = computed(() => rows.value.find(row => row.category.id === props.modelValue))
const selectedFromList = computed(() => selectedRow.value?.category)
const selectedPath = computed(() => selectedRow.value
  ? selectedRow.value.ancestors.map(category => category.name).join(' / ')
  : '')
const directFieldCount = computed(() =>
  props.selectedCategory?.attributes?.filter(attribute => !attribute.inherited).length || 0
)
const inheritedFieldCount = computed(() =>
  props.selectedCategory?.attributes?.filter(attribute => attribute.inherited).length || 0
)
const availableFieldCount = computed(() => directFieldCount.value + inheritedFieldCount.value)
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-end justify-between gap-2">
      <div>
        <label
          for="asset-category"
          class="block text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400"
        >
          Category <span class="normal-case font-medium text-muted">(optional)</span>
        </label>
        <p class="mt-1 text-xs text-muted">
          Choose the most specific category that fits this asset.
        </p>
      </div>
      <NuxtLink
        to="/categories"
        class="text-xs font-bold text-attic-500 hover:text-attic-600 hover:underline"
      >
        Manage categories
      </NuxtLink>
    </div>

    <USelectMenu
      id="asset-category"
      v-model="selection"
      aria-label="Category"
      :items="options"
      value-key="value"
      placeholder="Search or choose a category"
      icon="i-lucide-tags"
      size="lg"
      class="w-full"
    />

    <div
      v-if="selectedFromList"
      class="flex items-start gap-3 rounded-xl border border-attic-200 bg-attic-50/60 p-3 dark:border-attic-800 dark:bg-attic-950/20"
    >
      <div class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-white text-attic-500 shadow-sm dark:bg-mist-800">
        <UIcon
          :name="selectedFromList.icon || 'i-lucide-tag'"
          class="size-4.5"
        />
      </div>
      <div class="min-w-0 flex-1">
        <p
          v-if="selectedPath"
          class="truncate text-[10px] font-extrabold uppercase tracking-[0.12em] text-attic-500"
        >
          {{ selectedPath }}
        </p>
        <p class="font-bold text-mist-950 dark:text-white">
          {{ selectedFromList.name }}
        </p>
        <div class="mt-1 flex flex-wrap items-center gap-1.5 text-xs">
          <span
            v-if="loading"
            class="inline-flex items-center gap-1.5 text-muted"
          >
            <UIcon
              name="i-lucide-loader-2"
              class="size-3 animate-spin"
            />
            Loading category fields…
          </span>
          <template v-else-if="availableFieldCount">
            <span class="font-semibold text-mist-700 dark:text-mist-200">
              {{ directFieldCount }} own
            </span>
            <template v-if="inheritedFieldCount">
              <span class="text-muted">+</span>
              <span class="font-semibold text-attic-600 dark:text-attic-300">
                {{ inheritedFieldCount }} inherited
              </span>
            </template>
            <span class="text-muted">·</span>
            <a
              href="#category-fields"
              class="font-bold text-attic-600 hover:underline dark:text-attic-300"
            >
              {{ availableFieldCount }} {{ availableFieldCount === 1 ? 'field' : 'fields' }} to complete
            </a>
          </template>
          <span
            v-else
            class="text-muted"
          >
            This category adds no extra fields.
          </span>
        </div>
      </div>
      <UIcon
        name="i-lucide-check-circle-2"
        class="mt-1 size-4 shrink-0 text-attic-500"
      />
    </div>

    <p
      v-else-if="!categories.length"
      class="text-sm text-gray-400"
    >
      No categories available. You can keep this asset uncategorized or
      <NuxtLink
        to="/categories/new"
        class="text-attic-500 hover:underline"
      >create one</NuxtLink>.
    </p>
    <p
      v-else
      class="text-xs text-muted"
    >
      Without a category, this asset will not have category-specific fields.
    </p>
  </div>
</template>
