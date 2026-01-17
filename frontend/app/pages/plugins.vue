<script setup lang="ts">
import type { Plugin, PluginsResponse } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const { data: pluginsData, status } = useApi<PluginsResponse>('/api/plugins')

const plugins = computed(() => pluginsData.value?.plugins || [])

function getCategoryStatus(plugin: Plugin): 'active' | 'pending' {
  return plugin.category_id ? 'active' : 'pending'
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
  <div class="space-y-8">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          Import Plugins
        </h1>
        <p class="text-mist-500 max-w-2xl">
          Plugins allow you to import items from external sources like Google Books, TMDB, and more.
        </p>
      </div>
    </div>

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
      class="overflow-hidden rounded-xl border border-mist-100 dark:border-mist-700 bg-white dark:bg-mist-800 shadow-sm"
    >
      <div class="flex flex-col items-center justify-center py-20 px-4 text-center">
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-puzzle"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No Plugins Available
        </h3>
        <p class="text-sm text-mist-500 max-w-sm">
          Import plugins will appear here when they are configured.
        </p>
      </div>
    </div>

    <!-- Plugin Cards Grid -->
    <div
      v-else
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
    >
      <div
        v-for="plugin in plugins"
        :key="plugin.id"
        class="bg-white dark:bg-mist-800 rounded-xl border border-mist-100 dark:border-mist-700 shadow-sm flex flex-col overflow-hidden"
      >
        <!-- Card Header -->
        <div class="p-6 pb-0 flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-mist-950 dark:text-white">
            {{ plugin.name }}
          </h3>
          <span
            class="px-2.5 py-0.5 rounded-full text-xs font-semibold border"
            :class="getCategoryStatus(plugin) === 'active'
              ? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 border-emerald-100 dark:border-emerald-800/30'
              : 'bg-mist-100 dark:bg-mist-700 text-mist-600 dark:text-mist-300 border-mist-200 dark:border-mist-600'"
          >
            {{ getCategoryStatus(plugin) === 'active' ? 'Active' : 'Available' }}
          </span>
        </div>

        <!-- Card Content -->
        <div class="px-6 space-y-6 flex-grow">
          <!-- Description -->
          <p class="text-mist-500 text-sm leading-relaxed">
            {{ plugin.description }}
          </p>

          <!-- Category -->
          <div class="space-y-2">
            <p class="text-[10px] font-bold text-mist-400 uppercase tracking-wider">
              Category
            </p>
            <div class="flex items-center gap-2">
              <UIcon
                name="i-lucide-folder"
                class="w-4 h-4 text-mist-400"
              />
              <span class="text-sm font-medium text-mist-700 dark:text-mist-200">
                {{ plugin.category_name }}
              </span>
              <span
                v-if="plugin.category_id"
                class="px-1.5 py-0.5 rounded bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 text-[10px] font-bold uppercase"
              >
                Created
              </span>
            </div>
          </div>

          <!-- Search Fields -->
          <div class="space-y-2">
            <p class="text-[10px] font-bold text-mist-400 uppercase tracking-wider">
              Search Fields
            </p>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="field in plugin.search_fields"
                :key="field.key"
                class="px-2 py-0.5 rounded bg-mist-100 dark:bg-mist-700 text-mist-500 text-[11px]"
              >
                {{ field.label }}
              </span>
            </div>
          </div>

          <!-- Attributes -->
          <div class="space-y-2">
            <p class="text-[10px] font-bold text-mist-400 uppercase tracking-wider">
              Attributes ({{ plugin.attributes.length }})
            </p>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="(attr, index) in plugin.attributes.slice(0, 4)"
                :key="attr.key"
                class="px-2 py-0.5 rounded text-[11px] font-medium border"
                :class="[getAttributeStyle(index).bg, getAttributeStyle(index).text, getAttributeStyle(index).border]"
              >
                {{ attr.name }}
              </span>
              <span
                v-if="plugin.attributes.length > 4"
                class="px-2 py-0.5 rounded bg-mist-100 dark:bg-mist-700 text-mist-500 text-[11px] font-medium border border-mist-200 dark:border-mist-600"
              >
                +{{ plugin.attributes.length - 4 }} more
              </span>
            </div>
          </div>
        </div>

        <!-- Card Footer -->
        <div class="p-6 border-t border-mist-100 dark:border-mist-700 mt-6">
          <NuxtLink
            to="/assets"
            class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg border border-attic-500/20 text-attic-600 dark:text-attic-400 font-bold text-sm hover:bg-attic-500/5 transition-colors"
          >
            <UIcon
              name="i-lucide-download"
              class="w-4 h-4"
            />
            Use to Import
          </NuxtLink>
        </div>
      </div>
    </div>

    <!-- Info Section -->
    <div class="bg-white dark:bg-mist-800 rounded-xl border-l-4 border-l-blue-400 overflow-hidden border-y border-r border-mist-100 dark:border-mist-700 shadow-sm">
      <!-- Info Header -->
      <div class="p-6 border-b border-mist-50 dark:border-mist-700 flex items-center gap-3">
        <UIcon
          name="i-lucide-info"
          class="w-5 h-5 text-blue-500"
        />
        <h2 class="text-lg font-bold text-mist-950 dark:text-white">
          How Import Plugins Work
        </h2>
      </div>

      <!-- Steps Grid -->
      <div class="p-8 grid grid-cols-1 md:grid-cols-2 gap-x-12 gap-y-6">
        <!-- Left Column -->
        <div class="space-y-6">
          <div class="flex gap-4">
            <div class="flex-shrink-0 w-6 h-6 rounded-full bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-bold">
              1
            </div>
            <p class="text-sm leading-relaxed text-mist-500">
              <span class="font-bold text-mist-950 dark:text-white">Go to Assets</span> and click the <span class="font-bold">Import</span> button in the top right.
            </p>
          </div>
          <div class="flex gap-4">
            <div class="flex-shrink-0 w-6 h-6 rounded-full bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-bold">
              2
            </div>
            <p class="text-sm leading-relaxed text-mist-500">
              <span class="font-bold text-mist-950 dark:text-white">Select a plugin</span> that matches the item you're adding (e.g., Google Books).
            </p>
          </div>
          <div class="flex gap-4">
            <div class="flex-shrink-0 w-6 h-6 rounded-full bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-bold">
              3
            </div>
            <p class="text-sm leading-relaxed text-mist-500">
              <span class="font-bold text-mist-950 dark:text-white">Search</span> for the item by title, ISBN, or other identifiers defined in the plugin.
            </p>
          </div>
        </div>

        <!-- Right Column -->
        <div class="space-y-6">
          <div class="flex gap-4">
            <div class="flex-shrink-0 w-6 h-6 rounded-full bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-bold">
              4
            </div>
            <p class="text-sm leading-relaxed text-mist-500">
              <span class="font-bold text-mist-950 dark:text-white">Click Import</span> on the best result - the item will be added with all available metadata.
            </p>
          </div>
          <div class="flex gap-4">
            <div class="flex-shrink-0 w-6 h-6 rounded-full bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-bold">
              5
            </div>
            <p class="text-sm leading-relaxed text-mist-500">
              <span class="font-bold text-mist-950 dark:text-white">Review and edit</span> the imported item to add any specific personal details or notes.
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
