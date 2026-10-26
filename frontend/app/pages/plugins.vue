<script setup lang="ts">
import type { Plugin, PluginsResponse } from '~/types/api'
import { compareImportSources, importProvider, importSetupGroups, importSourceIcon, type ImportSetupGroup } from '~/utils/importSources'

definePageMeta({ middleware: 'auth' })

const { data, status, error, refresh } = useApi<PluginsResponse>('/api/plugins')
const { features } = useFeatures()
const plugins = computed(() => data.value?.plugins || [])
const configured = computed(() => plugins.value.filter(plugin => plugin.enabled).sort(compareImportSources))
const setupGroups = computed(() => importSetupGroups(plugins.value))
const setupCount = computed(() => plugins.value.filter(plugin => !plugin.enabled).length)
const importModalOpen = ref(false)
const selectedPluginId = ref<string>()
const panelOpen = ref(false)
const detailSource = ref<Plugin | null>(null)
const setupGroup = ref<ImportSetupGroup | null>(null)
const panelTitle = computed(() => setupGroup.value ? `Connect ${setupGroup.value.provider.name}` : detailSource.value?.category_name || 'Source details')
const panelDescription = computed(() => setupGroup.value
  ? `Enable ${setupGroup.value.label} imports on your Attic server.`
  : `${detailSource.value ? importProvider(detailSource.value).name : 'Catalog'} source details`)
let panelTrigger: HTMLElement | null = null
let pendingImport: Plugin | null = null

function showDetails(plugin: Plugin, event: MouseEvent) {
  panelTrigger = event.currentTarget as HTMLElement
  setupGroup.value = null
  detailSource.value = plugin
  panelOpen.value = true
}

function showSetup(group: ImportSetupGroup, event: MouseEvent) {
  panelTrigger = event.currentTarget as HTMLElement
  detailSource.value = null
  setupGroup.value = group
  panelOpen.value = true
}

function startImport(plugin: Plugin) {
  if (panelOpen.value) {
    pendingImport = plugin
    panelOpen.value = false
    return
  }
  selectedPluginId.value = plugin.id
  importModalOpen.value = true
}

function afterPanelClose() {
  if (pendingImport) {
    const plugin = pendingImport
    pendingImport = null
    startImport(plugin)
  }
}

function restorePanelFocus(event: Event) {
  event.preventDefault()
  panelTrigger?.focus()
}
</script>

<template>
  <div class="mx-auto max-w-5xl space-y-6 pb-20 sm:pb-6">
    <header>
      <p class="mb-1.5 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
        Catalog import
      </p>
      <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
        Import sources
      </h1>
      <p class="mt-2 text-sm leading-6 text-muted">
        Find an item and add its details automatically.
      </p>
    </header>

    <div
      v-if="status === 'pending'"
      role="status"
      class="attic-panel flex items-center gap-3 rounded-2xl p-6 text-sm text-muted"
    >
      <UIcon
        name="i-lucide-loader-2"
        class="size-5 animate-spin text-attic-500"
      />
      Loading import sources…
    </div>
    <div
      v-else-if="error || status === 'error'"
      role="alert"
      class="attic-panel space-y-3 rounded-2xl p-6"
    >
      <h2 class="font-bold text-mist-950 dark:text-white">
        Couldn’t load import sources
      </h2>
      <p class="text-sm text-muted">
        Please try again to see your available catalogs.
      </p>
      <UButton
        variant="outline"
        icon="i-lucide-refresh-cw"
        class="min-h-11"
        @click="refresh()"
      >
        Try again
      </UButton>
    </div>
    <div
      v-else-if="plugins.length === 0"
      class="attic-panel rounded-2xl px-6 py-12 text-center"
    >
      <UIcon
        name="i-lucide-library"
        class="mb-3 size-8 text-attic-500"
      />
      <h2 class="font-bold text-mist-950 dark:text-white">
        No import sources available
      </h2>
      <p class="mt-2 text-sm text-muted">
        Catalogs will appear here when they are available on your Attic server.
      </p>
    </div>
    <template v-else>
      <section
        v-if="configured.length"
        aria-labelledby="configured-heading"
      >
        <div class="mb-3 flex items-center gap-2.5">
          <h2
            id="configured-heading"
            class="text-sm font-bold text-mist-800 dark:text-mist-100"
          >
            Configured
          </h2>
          <span class="rounded-md bg-attic-50 px-2 py-0.5 text-xs font-bold text-attic-600 dark:bg-attic-500/15 dark:text-attic-300">{{ configured.length }}</span>
        </div>
        <ul class="attic-panel divide-y divide-mist-100 overflow-hidden rounded-2xl dark:divide-mist-700/60">
          <li
            v-for="plugin in configured"
            :key="plugin.id"
            class="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 transition-colors hover:bg-mist-50/70 dark:hover:bg-mist-700/20 sm:flex-nowrap sm:gap-4 sm:px-5"
          >
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-attic-50 text-attic-500 dark:bg-attic-500/10"
              aria-hidden="true"
            >
              <UIcon
                :name="importSourceIcon(plugin)"
                class="size-5"
              />
            </div>
            <div class="min-w-0 flex-1 basis-[calc(100%-3.5rem)] sm:basis-auto">
              <h3 class="break-words text-sm font-bold leading-5 text-mist-950 dark:text-white sm:text-base">
                {{ plugin.category_name }}
              </h3>
              <p class="mt-0.5 text-xs leading-5 text-muted">
                {{ importProvider(plugin).name }}
              </p>
            </div>
            <div class="ml-[3.25rem] flex w-full items-center justify-between gap-2 sm:ml-0 sm:w-auto sm:shrink-0">
              <UButton
                color="neutral"
                variant="ghost"
                class="min-h-11 rounded-lg px-3 text-xs font-semibold text-muted"
                :aria-label="`Details for ${plugin.category_name}`"
                @click="showDetails(plugin, $event)"
              >
                Details
              </UButton>
              <UButton
                class="min-h-11 rounded-lg px-4 font-bold"
                icon="i-lucide-download"
                :aria-label="`Import ${plugin.category_name}`"
                @click="startImport(plugin)"
              >
                Import
              </UButton>
            </div>
          </li>
        </ul>
      </section>
      <section
        v-if="setupGroups.length"
        aria-labelledby="setup-heading"
      >
        <div class="mb-3 flex items-center gap-2.5">
          <h2
            id="setup-heading"
            class="text-sm font-bold text-mist-800 dark:text-mist-100"
          >
            Needs setup
          </h2>
          <span class="rounded-md bg-mist-100 px-2 py-0.5 text-xs font-bold text-muted dark:bg-mist-700">{{ setupCount }}</span>
        </div>
        <p
          v-if="!configured.length"
          class="mb-3 text-sm leading-6 text-muted"
        >
          Connect a catalog to start importing items.
        </p>
        <ul class="attic-panel divide-y divide-mist-100 overflow-hidden rounded-2xl dark:divide-mist-700/60">
          <li
            v-for="group in setupGroups"
            :key="group.provider.id"
            class="flex flex-wrap items-center gap-3 px-4 py-3 sm:flex-nowrap sm:gap-4 sm:px-5"
          >
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-mist-100 text-mist-500 dark:bg-mist-700 dark:text-mist-300"
              aria-hidden="true"
            >
              <UIcon
                :name="importSourceIcon(group.sources[0]!)"
                class="size-5"
              />
            </div>
            <div class="min-w-0 flex-1 basis-[calc(100%-3.5rem)] sm:basis-auto">
              <h3 class="break-words text-sm font-bold leading-5 text-mist-800 dark:text-mist-100 sm:text-base">
                {{ group.label }}
              </h3>
              <p class="mt-0.5 text-xs leading-5 text-muted">
                {{ group.provider.name }} · Server setup required
              </p>
            </div>
            <UButton
              color="neutral"
              variant="outline"
              class="ml-[3.25rem] min-h-11 rounded-lg text-xs font-semibold sm:ml-0 sm:shrink-0"
              :aria-label="`Setup instructions for ${group.provider.name}`"
              @click="showSetup(group, $event)"
            >
              Setup instructions
            </UButton>
          </li>
        </ul>
      </section>
    </template>

    <details class="group border-t border-mist-200 pt-4 dark:border-mist-700">
      <summary class="flex min-h-11 cursor-pointer list-none items-center justify-between gap-2 text-sm font-semibold text-muted [&::-webkit-details-marker]:hidden">
        <span class="flex items-center gap-2"><UIcon
          name="i-lucide-circle-help"
          class="size-4"
        />How importing works</span>
        <UIcon
          name="i-lucide-chevron-down"
          class="size-4 transition-transform group-open:rotate-180"
        />
      </summary>
      <ol class="mt-3 grid gap-4 pb-3 sm:grid-cols-3">
        <li
          v-for="(step, index) in [
            { title: 'Find the item', copy: 'Choose a source and search using its supported identifiers.' },
            { title: 'Review the match', copy: 'Compare results and select the correct edition or release.' },
            { title: 'Import and refine', copy: 'Create the asset with metadata, then add your location and notes.' }
          ]"
          :key="step.title"
          class="text-sm leading-6"
        >
          <h3 class="font-bold text-mist-800 dark:text-mist-100">
            {{ index + 1 }}. {{ step.title }}
          </h3>
          <p class="mt-1 text-muted">
            {{ step.copy }}
          </p>
        </li>
      </ol>
    </details>

    <USlideover
      v-model:open="panelOpen"
      :title="panelTitle"
      :description="panelDescription"
      :ui="{ content: 'w-full sm:max-w-lg', header: 'pr-14', title: 'break-words', footer: 'justify-between', close: 'min-h-11 min-w-11' }"
      :content="{ onCloseAutoFocus: restorePanelFocus }"
      @after:leave="afterPanelClose"
    >
      <template #body>
        <div
          v-if="detailSource"
          class="space-y-7 text-mist-800 dark:text-mist-100"
        >
          <p class="text-sm leading-6 text-muted">
            {{ detailSource.description }}
          </p>
          <section aria-labelledby="search-methods-heading">
            <h3
              id="search-methods-heading"
              class="mb-3 text-sm font-bold"
            >
              Search by
            </h3>
            <ul class="flex flex-wrap gap-2">
              <li
                v-for="field in detailSource.search_fields"
                :key="field.key"
                class="rounded-lg bg-mist-100 px-3 py-1.5 text-sm text-mist-700 dark:bg-mist-700 dark:text-mist-100"
              >
                {{ field.label }}
              </li>
            </ul>
          </section>
          <section
            v-if="features.attributes && detailSource.attributes.length"
            aria-labelledby="imported-fields-heading"
          >
            <h3
              id="imported-fields-heading"
              class="mb-3 text-sm font-bold"
            >
              Imported fields
            </h3>
            <dl class="divide-y divide-mist-100 dark:divide-mist-700">
              <div
                v-for="attr in detailSource.attributes"
                :key="attr.key"
                class="flex justify-between gap-4 py-2.5 text-sm"
              >
                <dt class="min-w-0 break-words">
                  {{ attr.name }}
                </dt>
                <dd class="shrink-0 capitalize text-muted">
                  {{ attr.data_type }}
                </dd>
              </div>
            </dl>
          </section>
          <section
            v-if="features.categories"
            aria-labelledby="import-category-heading"
            class="rounded-xl bg-mist-50 p-4 dark:bg-mist-800"
          >
            <h3
              id="import-category-heading"
              class="text-sm font-bold"
            >
              Category
            </h3>
            <p class="mt-2 text-sm text-muted">
              {{ detailSource.category_name }}
            </p>
            <p class="mt-1 text-xs leading-5 text-muted">
              {{ detailSource.category_id ? 'Imported items are added to this category.' : 'This category will be initialized when you import an item.' }}
            </p>
          </section>
          <p class="text-xs leading-5 text-muted">
            Configured on your server. Availability and credentials are checked when you search or import.
          </p>
        </div>
        <div
          v-else-if="setupGroup"
          class="space-y-6 text-mist-800 dark:text-mist-100"
        >
          <div class="rounded-xl bg-mist-50 p-4 dark:bg-mist-800">
            <p class="text-xs font-bold uppercase tracking-wide text-muted">
              Enables imports for
            </p>
            <ul class="mt-2 space-y-1 text-sm font-semibold">
              <li
                v-for="source in setupGroup.sources"
                :key="source.id"
              >
                {{ source.category_name }}
              </li>
            </ul>
          </div>
          <template v-if="setupGroup.provider.credentials.length">
            <ol class="list-decimal space-y-5 pl-5 text-sm leading-6 text-muted marker:font-bold marker:text-attic-500">
              <li>
                <p>{{ setupGroup.provider.instruction }}</p>
                <a
                  :href="setupGroup.provider.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="mt-1 inline-flex min-h-11 items-center gap-1.5 font-bold text-attic-600 underline underline-offset-4 dark:text-attic-300"
                >
                  {{ setupGroup.provider.linkLabel }}<UIcon
                    name="i-lucide-arrow-up-right"
                    class="size-4"
                  />
                </a>
              </li>
              <li>
                Set {{ setupGroup.provider.credentials.length === 1 ? 'this environment variable' : 'these environment variables' }} on your Attic server:
                <ul class="mt-3 space-y-2">
                  <li
                    v-for="credential in setupGroup.provider.credentials"
                    :key="credential"
                    class="rounded-lg border border-mist-200 bg-mist-50 px-3 py-2 dark:border-mist-700 dark:bg-mist-800"
                  >
                    <code class="break-all text-xs font-semibold text-mist-800 dark:text-mist-100">{{ credential }}</code>
                  </li>
                </ul>
              </li>
              <li>Restart Attic, then reload this page.</li>
            </ol>
            <p
              v-if="setupGroup.provider.note"
              class="rounded-xl border border-attic-200 bg-attic-50/60 p-4 text-sm leading-6 text-mist-700 dark:border-attic-500/30 dark:bg-attic-500/10 dark:text-mist-200"
            >
              {{ setupGroup.provider.note }}
            </p>
            <p class="text-xs leading-5 text-muted">
              Credentials stay on your server. Ask your server administrator to complete setup if you don’t manage the deployment.
            </p>
          </template>
          <div
            v-else
            class="space-y-3 text-sm leading-6 text-muted"
          >
            <p
              v-for="source in setupGroup.sources"
              :key="source.id"
              class="break-words"
            >
              {{ source.disabled_reason || 'Contact your server administrator to configure this source.' }}
            </p>
          </div>
        </div>
      </template>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          class="min-h-11 rounded-lg"
          @click="panelOpen = false"
        >
          Close
        </UButton>
        <UButton
          v-if="detailSource?.enabled"
          icon="i-lucide-download"
          class="min-h-11 rounded-lg font-bold"
          :aria-label="`Import ${detailSource.category_name}`"
          @click="startImport(detailSource)"
        >
          Import
        </UButton>
      </template>
    </USlideover>

    <ImportModal
      v-model:open="importModalOpen"
      :initial-plugin-id="selectedPluginId"
    />
  </div>
</template>
