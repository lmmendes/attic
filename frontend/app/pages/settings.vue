<script setup lang="ts">
import type { TabsItem } from '@nuxt/ui'
import type { OrganizationFeatures, CurrencyOptionsResponse } from '~/types/api'

definePageMeta({ middleware: 'auth' })
const route = useRoute()
const router = useRouter()
const activeTab = computed({
  get: () => route.query.tab === 'integrations' ? 'integrations' : 'inventory',
  set: (tab: string | number) => {
    void router.push({ path: '/settings', query: { ...route.query, tab: String(tab) } })
  }
})
const tabs = [
  { label: 'Inventory', value: 'inventory', icon: 'i-lucide-sliders-horizontal', slot: 'inventory' as const },
  { label: 'Integrations', value: 'integrations', icon: 'i-lucide-plug', slot: 'integrations' as const }
] satisfies TabsItem[]
const { isAdmin, loading: authLoading, fetchSession } = useAuth()
const { features, load, update } = useFeatures()
const { settings, loaded: settingsLoaded, error: settingsError, load: loadSettings, update: updateSettings } = useOrganizationSettings()
const { data: currencyCatalog, status: currencyStatus, error: currencyError, refresh: loadCurrencies } = useApi<CurrencyOptionsResponse>(
  '/api/organization/settings/currencies', { immediate: false }
)
const currencies = computed(() => currencyCatalog.value?.currencies || [])
const currenciesReady = computed(() => currencyStatus.value === 'success' && !currencyError.value && currencies.value.length > 0)
const currencyLoadFailed = computed(() => !!currencyError.value || currencyStatus.value === 'error' || (currencyStatus.value === 'success' && currencies.value.length === 0))
// Edit a copy, so unsaved changes (e.g. the currency) don't affect other pages; it follows the saved values.
const draft = ref({ ...settings.value })
watch(settings, (saved) => {
  draft.value = { ...saved }
})
const toast = useToast()
const api = useApiFetch()
const { data: mcp, refresh: refreshMCP, status: mcpStatus, error: mcpError } = useApi<{ enabled: boolean }>('/api/organization/mcp')
const mcpSaving = ref(false)
async function setMCP(enabled: boolean) {
  mcpSaving.value = true
  try {
    await api('/api/organization/mcp', { method: 'PUT', body: JSON.stringify({ enabled }) })
    await refreshMCP()
    toast.add({ title: enabled ? 'MCP enabled' : 'MCP disabled', color: 'success' })
  } catch {
    toast.add({ title: 'Could not update MCP', color: 'error' })
  } finally { mcpSaving.value = false }
}
const savingFeatures = ref(false)
const savingGeneral = ref(false)
const labels: Array<{ key: keyof OrganizationFeatures, label: string, description: string }> = [
  { key: 'locations', label: 'Locations', description: 'Track where assets are stored.' },
  { key: 'collections', label: 'Collections', description: 'Group assets into custom collections.' },
  { key: 'tags', label: 'Tags', description: 'Add flexible tags to assets.' },
  { key: 'categories', label: 'Categories & attributes', description: 'Organize assets and define their custom fields.' },
  { key: 'conditions', label: 'Conditions', description: 'Track asset condition.' },
  { key: 'warranties', label: 'Warranties', description: 'Track warranty coverage and expiry.' },
  { key: 'plugins', label: 'Plugins', description: 'Enable external import plugins and their metadata.' }
]

onMounted(async () => {
  if (authLoading.value) await fetchSession()
  if (!isAdmin.value) return navigateTo('/')
  // Recovers from a failed load at login, so the real currency is shown before anyone can save.
  await Promise.all([
    ...(!settingsLoaded.value ? [loadSettings()] : []),
    ...(!currenciesReady.value ? [loadCurrencies()] : [])
  ])
})

// Each section saves with one request, so a save is never half-done.
async function saveFeatures() {
  savingFeatures.value = true
  try {
    await update({
      ...features.value,
      attributes: features.value.categories
    })
    toast.add({ title: 'Features saved', color: 'success' })
  } catch {
    await load()
    toast.add({ title: 'Could not save features', color: 'error' })
  } finally {
    savingFeatures.value = false
  }
}

async function saveGeneralSettings() {
  if (savingGeneral.value || !settingsLoaded.value || !currenciesReady.value || !currencies.value.includes(draft.value.currency)) return
  savingGeneral.value = true
  try {
    await updateSettings({ ...draft.value })
    toast.add({ title: 'General settings saved', color: 'success' })
  } catch {
    await loadSettings()
    draft.value = { ...settings.value }
    toast.add({ title: 'Could not save general settings', color: 'error' })
  } finally {
    savingGeneral.value = false
  }
}
</script>

<template>
  <div class="max-w-3xl mx-auto p-6 lg:p-10 space-y-6">
    <div>
      <h1 class="text-3xl font-black text-mist-950 dark:text-white">
        Settings
      </h1>
      <p class="text-muted mt-2">
        Manage your inventory preferences and connected apps.
      </p>
    </div>
    <UTabs
      v-model="activeTab"
      :items="tabs"
      variant="link"
      class="w-full gap-6"
    >
      <template #inventory>
        <div class="space-y-6">
          <UCard>
            <template #header>
              <h2 class="text-lg font-extrabold text-mist-950 dark:text-white">
                Features
              </h2>
            </template>
            <div class="divide-y divide-mist-100 dark:divide-mist-800">
              <div
                v-for="item in labels"
                :key="item.key"
                class="py-5 first:pt-0 flex items-center justify-between gap-6"
              >
                <div>
                  <p class="font-bold text-mist-950 dark:text-white">
                    {{ item.label }}
                  </p>
                  <p class="text-sm text-muted mt-1">
                    {{ item.description }}
                  </p>
                </div>
                <USwitch
                  v-model="features[item.key]"
                  :aria-label="`Enable ${item.label}`"
                />
              </div>
            </div>
            <div class="mt-6 flex justify-end">
              <UButton
                :loading="savingFeatures"
                @click="saveFeatures"
              >
                Save features
              </UButton>
            </div>
          </UCard>
          <UCard>
            <template #header>
              <h2 class="text-lg font-extrabold text-mist-950 dark:text-white">
                General settings
              </h2>
            </template>
            <div class="pb-5 flex items-center justify-between gap-6">
              <div>
                <p class="font-bold text-mist-950 dark:text-white">
                  Currency
                </p>
                <p class="text-sm text-muted mt-1">
                  Used for all asset prices. Changing it relabels existing prices. Amounts are not converted.
                </p>
              </div>
              <USelectMenu
                v-model="draft.currency"
                aria-label="Currency"
                :items="currencies"
                :disabled="!settingsLoaded || !currenciesReady"
                class="w-28 shrink-0"
              />
            </div>
            <p
              v-if="currencyStatus === 'pending'"
              role="status"
              class="mb-4 text-sm text-muted"
            >
              Loading currency choices…
            </p>
            <div
              v-else-if="currencyLoadFailed"
              role="alert"
              class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg bg-error/10 px-4 py-3 text-sm text-error"
            >
              <span>Currency choices could not be loaded. Your selected currency is unchanged.</span>
              <UButton
                size="sm"
                color="error"
                variant="soft"
                @click="loadCurrencies()"
              >
                Retry currencies
              </UButton>
            </div>
            <div
              v-if="settingsError"
              role="alert"
              class="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-error/10 px-4 py-3 text-sm text-error"
            >
              <span>General settings could not be loaded, so they can't be changed right now.</span>
              <UButton
                size="sm"
                color="error"
                variant="soft"
                @click="loadSettings"
              >
                Retry
              </UButton>
            </div>
            <div class="mt-6 flex justify-end">
              <UButton
                :loading="savingGeneral"
                :disabled="!settingsLoaded || !currenciesReady || !currencies.includes(draft.currency)"
                @click="saveGeneralSettings"
              >
                Save settings
              </UButton>
            </div>
          </UCard>
        </div>
      </template>
      <template #integrations>
        <p class="mb-4 text-sm text-muted">
          Manage connections to external apps and assistants.
        </p>
        <UCard>
          <div class="flex items-center justify-between gap-6">
            <div>
              <h2 class="text-lg font-bold">
                MCP server
              </h2>
              <p class="mt-1 text-sm text-muted">
                Connect an assistant to your inventory using scoped API keys.
              </p>
            </div>
            <USwitch
              :model-value="mcp?.enabled ?? false"
              :disabled="mcpSaving || mcpStatus !== 'success' || !!mcpError || !mcp"
              aria-label="Enable MCP server"
              @update:model-value="setMCP"
            />
          </div>
          <p
            v-if="mcpStatus === 'pending'"
            role="status"
            class="mt-4 text-sm text-muted"
          >
            Loading MCP status…
          </p>
          <UAlert
            v-else-if="mcpError"
            class="mt-4"
            color="error"
            title="Could not load MCP status"
          >
            <template #actions>
              <UButton
                label="Retry"
                color="error"
                variant="outline"
                @click="refreshMCP()"
              />
            </template>
          </UAlert>
          <p
            v-else-if="mcp"
            class="mt-4 text-sm leading-6 text-muted"
          >
            {{ mcp.enabled ? 'Members can connect assistants using their own API keys.' : 'MCP is off for this workspace. API keys still work with the REST API.' }}
          </p>
          <div class="mt-5 border-t border-mist-100 pt-5 dark:border-mist-700">
            <UButton
              to="/profile?tab=mcp"
              variant="outline"
              icon="i-lucide-arrow-up-right"
            >
              View connection instructions in your profile
            </UButton>
          </div>
        </UCard>
      </template>
    </UTabs>
  </div>
</template>
