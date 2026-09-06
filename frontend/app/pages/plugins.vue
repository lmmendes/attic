<script setup lang="ts">
import type { Plugin, PluginsResponse } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const { data: pluginsData, status } = useApi<PluginsResponse>('/api/plugins')

const plugins = computed(() => pluginsData.value?.plugins || [])
const activePlugins = computed(() => plugins.value.filter(plugin => getPluginStatus(plugin) === 'active').length)
const pluginsNeedingSetup = computed(() => plugins.value.filter(plugin => getPluginStatus(plugin) === 'disabled').length)
const importModalOpen = ref(false)
const selectedPluginId = ref<string>()

// Track which plugins have expanded attributes
const expandedAttributes = ref<Set<string>>(new Set())

function toggleAttributes(pluginId: string) {
  if (expandedAttributes.value.has(pluginId)) {
    expandedAttributes.value.delete(pluginId)
  } else {
    expandedAttributes.value.add(pluginId)
  }
}

function isExpanded(pluginId: string): boolean {
  return expandedAttributes.value.has(pluginId)
}

function startImport(plugin: Plugin) {
  selectedPluginId.value = plugin.id
  importModalOpen.value = true
}

function getPluginIcon(plugin: Plugin): string {
  const identity = `${plugin.id} ${plugin.name}`.toLowerCase()
  if (identity.includes('book')) return 'i-lucide-book-open'
  if (identity.includes('movie') || identity.includes('tmdb')) return 'i-lucide-film'
  if (identity.includes('game')) return 'i-lucide-gamepad-2'
  if (identity.includes('music')) return 'i-lucide-music'
  return 'i-lucide-database-zap'
}

function getPluginStatus(plugin: Plugin): 'active' | 'disabled' {
  return plugin.enabled ? 'active' : 'disabled'
}

// Get attribute color based on index for visual variety
function getAttributeStyle(index: number): { bg: string, text: string, border: string } {
  const styles = [
    { bg: 'bg-amber-50 dark:bg-amber-900/20', text: 'text-amber-700 dark:text-amber-300', border: 'border-amber-200 dark:border-amber-800/30' },
    { bg: 'bg-blue-50 dark:bg-blue-900/20', text: 'text-blue-700 dark:text-blue-300', border: 'border-blue-200 dark:border-blue-800/30' },
    { bg: 'bg-emerald-50 dark:bg-emerald-900/20', text: 'text-emerald-700 dark:text-emerald-300', border: 'border-emerald-200 dark:border-emerald-800/30' },
    { bg: 'bg-rose-50 dark:bg-rose-900/20', text: 'text-rose-700 dark:text-rose-300', border: 'border-rose-200 dark:border-rose-800/30' }
  ]
  return styles[index % styles.length]!
}
</script>

<template>
  <div class="space-y-5 pb-6">
    <!-- Page Header -->
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Connected sources
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Import plugins
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          Add assets faster using trusted metadata from external catalogs.
        </p>
      </div>
      <div class="attic-panel flex divide-x divide-mist-100 rounded-xl px-2 py-2 dark:divide-mist-700">
        <div class="px-3">
          <p class="text-[10px] font-bold uppercase text-muted">
            Active
          </p>
          <p class="font-black text-mist-950 dark:text-white">
            {{ activePlugins }}
          </p>
        </div>
        <div class="px-3">
          <p class="text-[10px] font-bold uppercase text-muted">
            Needs setup
          </p>
          <p class="font-black text-mist-950 dark:text-white">
            {{ pluginsNeedingSetup }}
          </p>
        </div>
      </div>
    </header>

    <!-- Loading State -->
    <div
      v-if="status === 'pending'"
      class="flex items-center justify-center py-20"
    >
      <UIcon
        name="i-lucide-loader-2"
        class="w-8 h-8 text-attic-500 animate-spin"
      />
    </div>

    <!-- Empty State -->
    <div
      v-else-if="plugins.length === 0"
      class="attic-panel overflow-hidden rounded-[20px]"
    >
      <div class="flex flex-col items-center justify-center py-20 px-4 text-center">
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-puzzle"
            class="w-8 h-8 text-muted"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No Plugins Available
        </h3>
        <p class="text-sm text-muted max-w-sm">
          Import plugins will appear here when they are configured.
        </p>
      </div>
    </div>

    <!-- Plugin Cards Grid -->
    <div
      v-else
      class="grid grid-cols-1 gap-4 lg:grid-cols-2"
    >
      <div
        v-for="plugin in plugins"
        :key="plugin.id"
        class="flex flex-col overflow-hidden rounded-[20px] border shadow-sm"
        :class="plugin.enabled
          ? 'bg-white dark:bg-mist-800 border-mist-100 dark:border-mist-700'
          : 'bg-mist-50 dark:bg-mist-900 border-mist-200 dark:border-mist-700 opacity-75'"
      >
        <!-- Card Header -->
        <div class="mb-4 flex items-start justify-between gap-3 p-5 pb-0">
          <div class="flex min-w-0 items-center gap-3">
            <div
              class="flex size-11 shrink-0 items-center justify-center rounded-2xl"
              :class="plugin.enabled ? 'bg-attic-50 text-attic-500 dark:bg-attic-500/10' : 'bg-mist-100 text-muted dark:bg-mist-700'"
            >
              <UIcon
                :name="getPluginIcon(plugin)"
                class="size-5"
              />
            </div>
            <div class="min-w-0">
              <h3
                class="truncate text-lg font-extrabold"
                :class="plugin.enabled ? 'text-mist-950 dark:text-white' : 'text-muted dark:text-mist-400'"
              >
                {{ plugin.name }}
              </h3>
              <p class="text-xs text-muted">
                {{ plugin.category_name }} catalog
              </p>
            </div>
          </div>
          <span
            class="px-2.5 py-0.5 rounded-full text-xs font-semibold border"
            :class="{
              'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 border-emerald-100 dark:border-emerald-800/30': getPluginStatus(plugin) === 'active',
              'bg-amber-50 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 border-amber-100 dark:border-amber-800/30': getPluginStatus(plugin) === 'disabled'
            }"
          >
            {{ getPluginStatus(plugin) === 'active' ? 'Active' : 'Needs setup' }}
          </span>
        </div>

        <!-- Card Content -->
        <div class="flex-grow space-y-4 px-5">
          <!-- Description -->
          <p class="min-h-10 text-sm leading-5 text-muted">
            {{ plugin.description }}
          </p>

          <!-- Category -->
          <div class="flex flex-wrap items-center gap-2 rounded-xl bg-mist-50 px-3 py-2.5 dark:bg-mist-700/40">
            <div class="flex min-w-0 flex-1 items-center gap-2">
              <UIcon
                name="i-lucide-folder"
                class="size-4 shrink-0 text-muted"
              />
              <span class="truncate text-xs font-bold text-mist-700 dark:text-mist-200">
                {{ plugin.category_name }}
              </span>
              <span
                v-if="plugin.category_id"
                class="px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 text-[10px] font-bold uppercase"
              >
                Created
              </span>
              <span
                v-else
                class="text-[10px] font-semibold text-muted"
              >
                {{ plugin.enabled ? 'Initialization failed' : 'Created when enabled' }}
              </span>
            </div>
            <span class="text-[11px] font-semibold text-muted">
              {{ plugin.search_fields.length }} search · {{ plugin.attributes.length }} imported fields
            </span>
          </div>

          <div
            v-if="isExpanded(plugin.id)"
            class="space-y-4 rounded-xl border border-mist-100 p-3 dark:border-mist-700"
          >
            <div class="space-y-2">
              <p class="text-[10px] font-bold uppercase tracking-wider text-muted">
                Search by
              </p>
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="field in plugin.search_fields"
                  :key="field.key"
                  class="rounded-lg bg-mist-100 px-2 py-1 text-[11px] font-semibold text-muted dark:bg-mist-700"
                >{{ field.label }}</span>
              </div>
            </div>
            <div class="space-y-2">
              <p class="text-[10px] font-bold uppercase tracking-wider text-muted">
                Imported fields
              </p>
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="(attr, index) in plugin.attributes"
                  :key="attr.key"
                  class="rounded-lg border px-2 py-1 text-[11px] font-medium"
                  :class="[getAttributeStyle(index).bg, getAttributeStyle(index).text, getAttributeStyle(index).border]"
                >
                  {{ attr.name }}
                </span>
              </div>
            </div>
          </div>

          <button
            type="button"
            class="inline-flex items-center gap-1 text-xs font-bold text-muted transition-colors hover:text-attic-500"
            :aria-expanded="isExpanded(plugin.id)"
            @click="toggleAttributes(plugin.id)"
          >
            <UIcon
              :name="isExpanded(plugin.id) ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
              class="size-3.5"
            />
            {{ isExpanded(plugin.id) ? 'Hide technical details' : 'View technical details' }}
          </button>
        </div>

        <!-- Card Footer -->
        <div class="mt-5 border-t border-mist-100 p-5 dark:border-mist-700">
          <!-- Disabled Reason -->
          <div
            v-if="!plugin.enabled && plugin.disabled_reason"
            class="mb-4 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800/30"
          >
            <div class="flex items-start gap-2">
              <UIcon
                name="i-lucide-alert-triangle"
                class="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0"
              />
              <p class="text-xs text-amber-700 dark:text-amber-300">
                {{ plugin.disabled_reason }}
              </p>
            </div>
          </div>

          <UButton
            v-if="plugin.enabled"
            block
            icon="i-lucide-download"
            class="rounded-xl font-bold shadow-primary"
            @click="startImport(plugin)"
          >
            Import with {{ plugin.name }}
          </UButton>
          <div
            v-else
            class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg border border-mist-200 dark:border-mist-600 text-muted dark:text-mist-500 font-bold text-sm cursor-not-allowed"
          >
            <UIcon
              name="i-lucide-download"
              class="w-4 h-4"
            />
            Unavailable
          </div>
        </div>
      </div>
    </div>

    <!-- Info Section -->
    <section class="attic-panel overflow-hidden rounded-[20px]">
      <div class="flex flex-col gap-3 border-b border-mist-100 p-5 dark:border-mist-700 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="font-extrabold text-mist-950 dark:text-white">
            From catalog to collection
          </h2>
          <p class="text-xs text-muted">
            Plugins fill the metadata; you keep control of the final record.
          </p>
        </div>
        <UButton
          to="/assets"
          variant="soft"
          icon="i-lucide-package-open"
          class="rounded-xl"
        >
          View assets
        </UButton>
      </div>

      <div class="grid gap-px bg-mist-100 dark:bg-mist-700 md:grid-cols-3">
        <div
          v-for="step in [
            { number: '01', icon: 'i-lucide-search', title: 'Find the item', copy: 'Choose a source and search using its supported identifiers.' },
            { number: '02', icon: 'i-lucide-list-checks', title: 'Review the match', copy: 'Compare results and select the correct edition or release.' },
            { number: '03', icon: 'i-lucide-package-plus', title: 'Import and refine', copy: 'Create the asset with metadata, then add your location and notes.' }
          ]"
          :key="step.number"
          class="bg-white p-5 dark:bg-mist-800"
        >
          <div class="mb-3 flex items-center justify-between">
            <div class="flex size-9 items-center justify-center rounded-xl bg-attic-50 text-attic-500 dark:bg-attic-500/10">
              <UIcon
                :name="step.icon"
                class="size-4.5"
              />
            </div>
            <span class="font-mono text-xs font-bold text-mist-300">{{ step.number }}</span>
          </div>
          <h3 class="text-sm font-extrabold text-mist-950 dark:text-white">
            {{ step.title }}
          </h3>
          <p class="mt-1 text-xs leading-5 text-muted">
            {{ step.copy }}
          </p>
        </div>
      </div>
    </section>

    <ImportModal
      v-model:open="importModalOpen"
      :initial-plugin-id="selectedPluginId"
    />
  </div>
</template>
