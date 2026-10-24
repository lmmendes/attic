<script setup lang="ts">
import type { TabsItem } from '@nuxt/ui'
import type { OrganizationFeatures } from '~/types/api'

definePageMeta({ middleware: 'auth' })
const activeTab = ref('inventory')
const tabs = [
  { label: 'Inventory', value: 'inventory', icon: 'i-lucide-sliders-horizontal', slot: 'inventory' as const },
  { label: 'Integrations', value: 'integrations', icon: 'i-lucide-plug', slot: 'integrations' as const }
] satisfies TabsItem[]
const { isAdmin, loading: authLoading, fetchSession } = useAuth()
const { features, load, update } = useFeatures()
const toast = useToast()
const saving = ref(false)
const api = useApiFetch()
const { data: mcp, refresh: refreshMCP } = useApi<{ enabled: boolean }>('/api/organization/mcp')
const mcpSaving = ref(false)
const endpoint = computed(() => `${useRuntimeConfig().public.apiBase || (import.meta.client ? window.location.origin : '')}/mcp`)
const authorizationTemplate = 'Authorization: Bearer <your-api-key>'
const copiedMCP = ref('')
const clientConfig = computed(() => JSON.stringify({
  mcpServers: {
    attic: {
      url: endpoint.value,
      headers: { Authorization: 'Bearer <your-api-key>' }
    }
  }
}, null, 2))
async function copyMCP(value: string, field: string) {
  try {
    await navigator.clipboard.writeText(value)
    copiedMCP.value = field
    toast.add({ title: 'Copied to clipboard', color: 'success' })
  } catch {
    toast.add({ title: 'Could not copy. Select and copy the text instead.', color: 'error' })
  }
}
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
})

async function save() {
  saving.value = true
  try {
    await update({
      ...features.value,
      attributes: features.value.categories
    })
    toast.add({ title: 'Settings saved', color: 'success' })
  } catch {
    await load()
    toast.add({ title: 'Could not save settings', color: 'error' })
  } finally {
    saving.value = false
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
        <p class="mb-4 text-sm text-muted">
          Choose which inventory features are available to your organization.
        </p>

        <UCard>
          <div class="divide-y divide-mist-100 dark:divide-mist-800">
            <div
              v-for="item in labels"
              :key="item.key"
              class="py-5 flex items-center justify-between gap-6"
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
              :loading="saving"
              @click="save"
            >
              Save changes
            </UButton>
          </div>
        </UCard>
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
              :disabled="mcpSaving || !mcp"
              aria-label="Enable MCP server"
              @update:model-value="setMCP"
            />
          </div>
          <div
            v-if="mcp && !mcp.enabled"
            class="mt-5 flex items-start gap-2 rounded-xl bg-mist-50 px-3 py-2.5 text-xs leading-5 text-muted dark:bg-mist-900/50"
          >
            <UIcon
              name="i-lucide-info"
              class="mt-0.5 size-4 shrink-0"
            />
            Enable the MCP server above before connecting your assistant.
          </div>

          <ol class="mt-6 space-y-6 border-t border-mist-100 pt-6 dark:border-mist-700">
            <li class="flex gap-3">
              <span
                aria-hidden="true"
                class="flex size-6 shrink-0 items-center justify-center rounded-full bg-attic-50 text-xs font-bold text-attic-600 dark:bg-attic-500/15 dark:text-attic-300"
              >1</span>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-bold text-mist-950 dark:text-white">
                  Create an API key
                </h3>
                <p class="mt-1 text-sm leading-6 text-muted">
                  Choose the permissions your assistant needs, then save the key when it appears.
                </p>
                <UButton
                  class="mt-3"
                  to="/api-keys"
                  variant="outline"
                  size="sm"
                  icon="i-lucide-key-round"
                >
                  Manage API keys
                </UButton>
              </div>
            </li>
            <li class="flex gap-3">
              <span
                aria-hidden="true"
                class="flex size-6 shrink-0 items-center justify-center rounded-full bg-attic-50 text-xs font-bold text-attic-600 dark:bg-attic-500/15 dark:text-attic-300"
              >2</span>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-bold text-mist-950 dark:text-white">
                  Add Attic to your assistant
                </h3>
                <p class="mt-1 text-sm leading-6 text-muted">
                  In your client's MCP settings, add a server using these details.
                </p>
                <dl class="mt-4 space-y-4">
                  <div class="flex items-center gap-3 text-sm">
                    <dt class="text-muted">
                      Transport
                    </dt>
                    <dd class="font-medium text-mist-950 dark:text-white">
                      Streamable HTTP
                    </dd>
                  </div>
                  <div>
                    <dt class="mb-1.5 text-xs font-semibold text-mist-700 dark:text-mist-200">
                      Server URL
                    </dt>
                    <dd class="flex items-start gap-2 rounded-xl border border-mist-200 bg-mist-50 p-3 dark:border-mist-700 dark:bg-mist-900/50">
                      <code class="min-w-0 flex-1 break-all py-1 text-xs leading-5 text-mist-800 dark:text-mist-100">{{ endpoint }}</code>
                      <UButton
                        :icon="copiedMCP === 'endpoint' ? 'i-lucide-check' : 'i-lucide-copy'"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        :aria-label="copiedMCP === 'endpoint' ? 'Server URL copied' : 'Copy server URL'"
                        title="Copy server URL"
                        @click="copyMCP(endpoint, 'endpoint')"
                      />
                    </dd>
                  </div>
                  <div>
                    <dt class="mb-1.5 text-xs font-semibold text-mist-700 dark:text-mist-200">
                      Authorization header
                    </dt>
                    <dd class="flex items-start gap-2 rounded-xl border border-mist-200 bg-mist-50 p-3 dark:border-mist-700 dark:bg-mist-900/50">
                      <code class="min-w-0 flex-1 break-all py-1 text-xs leading-5 text-mist-800 dark:text-mist-100">{{ authorizationTemplate }}</code>
                      <UButton
                        :icon="copiedMCP === 'header' ? 'i-lucide-check' : 'i-lucide-copy'"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        :aria-label="copiedMCP === 'header' ? 'Header template copied' : 'Copy authorization header template'"
                        title="Copy header template"
                        @click="copyMCP(authorizationTemplate, 'header')"
                      />
                    </dd>
                    <p class="mt-1.5 text-xs leading-5 text-muted">
                      Replace <code>&lt;your-api-key&gt;</code> with the key you saved.
                    </p>
                  </div>
                </dl>
              </div>
            </li>
            <li class="flex gap-3">
              <span
                aria-hidden="true"
                class="flex size-6 shrink-0 items-center justify-center rounded-full bg-attic-50 text-xs font-bold text-attic-600 dark:bg-attic-500/15 dark:text-attic-300"
              >3</span>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-bold text-mist-950 dark:text-white">
                  Save and connect
                </h3>
                <p class="mt-1 text-sm leading-6 text-muted">
                  Save the connection in your client. Your assistant can now discover the tools allowed by your key.
                </p>
              </div>
            </li>
          </ol>

          <details class="mt-6 rounded-xl border border-mist-200 dark:border-mist-700">
            <summary class="cursor-pointer px-4 py-3 text-sm font-semibold text-mist-950 dark:text-white">
              Client configuration example
            </summary>
            <div class="border-t border-mist-100 p-4 dark:border-mist-700">
              <p class="mb-3 text-xs leading-5 text-muted">
                For clients that accept an <code>mcpServers</code> JSON configuration. Replace the API key placeholder before saving.
              </p>
              <div class="overflow-hidden rounded-xl border border-mist-200 bg-mist-50 dark:border-mist-700 dark:bg-mist-900/50">
                <div class="flex items-center justify-between border-b border-mist-200 px-3 py-2 dark:border-mist-700">
                  <span class="text-xs font-medium text-muted">JSON</span>
                  <UButton
                    :icon="copiedMCP === 'config' ? 'i-lucide-check' : 'i-lucide-copy'"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    :aria-label="copiedMCP === 'config' ? 'Configuration copied' : 'Copy client configuration'"
                    @click="copyMCP(clientConfig, 'config')"
                  />
                </div>
                <pre class="overflow-x-auto p-4 text-xs leading-6 text-mist-800 dark:text-mist-100"><code>{{ clientConfig }}</code></pre>
              </div>
            </div>
          </details>

          <p class="mt-5 text-xs leading-5 text-muted">
            Your key's permissions and enabled inventory features determine the available tools. Turning off MCP blocks assistant connections; other API key access stays available.
          </p>
        </UCard>
      </template>
    </UTabs>
  </div>
</template>
