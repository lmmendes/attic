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
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-2xl font-bold">Import Plugins</h1>
          <p class="text-gray-500 mt-1">
            Plugins allow you to import items from external sources like Google Books, TMDB, and more.
          </p>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <UCard
          v-for="plugin in plugins"
          :key="plugin.id"
          class="relative"
        >
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold text-lg">{{ plugin.name }}</h3>
              <UBadge
                :color="getCategoryStatus(plugin) === 'active' ? 'success' : 'neutral'"
                variant="subtle"
              >
                {{ getCategoryStatus(plugin) === 'active' ? 'Active' : 'Available' }}
              </UBadge>
            </div>
          </template>

          <div class="space-y-4">
            <p class="text-sm text-gray-500">
              {{ plugin.description }}
            </p>

            <div>
              <h4 class="text-xs font-medium text-gray-400 uppercase tracking-wide mb-2">
                Category
              </h4>
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-folder" class="w-4 h-4 text-gray-400" />
                <span>{{ plugin.category_name }}</span>
                <UBadge v-if="plugin.category_id" color="success" variant="subtle" size="xs">
                  Created
                </UBadge>
              </div>
            </div>

            <div>
              <h4 class="text-xs font-medium text-gray-400 uppercase tracking-wide mb-2">
                Search Fields
              </h4>
              <div class="flex flex-wrap gap-1">
                <UBadge
                  v-for="field in plugin.search_fields"
                  :key="field.key"
                  color="neutral"
                  variant="subtle"
                  size="xs"
                >
                  {{ field.label }}
                </UBadge>
              </div>
            </div>

            <div>
              <h4 class="text-xs font-medium text-gray-400 uppercase tracking-wide mb-2">
                Attributes ({{ plugin.attributes.length }})
              </h4>
              <div class="flex flex-wrap gap-1">
                <UBadge
                  v-for="attr in plugin.attributes.slice(0, 5)"
                  :key="attr.key"
                  color="neutral"
                  variant="outline"
                  size="xs"
                >
                  {{ attr.name }}
                </UBadge>
                <UBadge
                  v-if="plugin.attributes.length > 5"
                  color="neutral"
                  variant="outline"
                  size="xs"
                >
                  +{{ plugin.attributes.length - 5 }} more
                </UBadge>
              </div>
            </div>
          </div>

          <template #footer>
            <div class="flex justify-end">
              <UButton
                to="/assets"
                variant="outline"
                size="sm"
                icon="i-lucide-download"
              >
                Use to Import
              </UButton>
            </div>
          </template>
        </UCard>
      </div>

      <UCard v-if="plugins.length === 0 && status !== 'pending'" class="mt-4">
        <div class="text-center py-8">
          <UIcon name="i-lucide-puzzle" class="w-12 h-12 mx-auto mb-4 text-gray-400" />
          <h3 class="font-medium text-gray-900 dark:text-white">No Plugins Available</h3>
          <p class="text-gray-500 mt-1">
            Import plugins will appear here when they are configured.
          </p>
        </div>
      </UCard>

      <UCard v-if="status === 'pending'" class="mt-4">
        <div class="flex items-center justify-center py-8">
          <UIcon name="i-lucide-loader-2" class="w-6 h-6 animate-spin text-gray-400" />
          <span class="ml-2 text-gray-500">Loading plugins...</span>
        </div>
      </UCard>

      <!-- Info section -->
      <UCard class="mt-6">
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-info" class="w-5 h-5" />
            <h3 class="font-medium">How Import Plugins Work</h3>
          </div>
        </template>
        <div class="prose prose-sm dark:prose-invert max-w-none">
          <ol class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
            <li>
              <strong>Go to Assets</strong> and click the <strong>Import</strong> button
            </li>
            <li>
              <strong>Select a plugin</strong> (e.g., Google Books)
            </li>
            <li>
              <strong>Search</strong> for the item you want to import by title, ISBN, or other identifiers
            </li>
            <li>
              <strong>Click Import</strong> on the result you want - the item will be added to your inventory with all available metadata
            </li>
            <li>
              <strong>Review and edit</strong> the imported item to add any additional details
            </li>
          </ol>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>
