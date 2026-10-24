<script setup lang="ts">
import { formatKeyLastUsed } from '~/utils/apiKeys'

definePageMeta({ middleware: 'auth' })
interface KeyMetadata {
  id: string
  description: string
  prefix: string
  permissions: Record<string, string>
  expires_at: string | null
  revoked_at: string | null
  last_used_at: string | null
}
interface Permission { resource: string, label: string, levels: string[] }
const api = useApiFetch()
const toast = useToast()
const { data: permissions, error: permissionError } = useApi<Permission[]>('/api/me/api-keys/permissions')
const { data: keys, refresh, status, error: loadError } = useApi<KeyMetadata[]>('/api/me/api-keys')
const activeKeys = computed(() => (keys.value || []).filter(key => !key.revoked_at && (!key.expires_at || new Date(key.expires_at).getTime() > Date.now())))
const selectablePermissions = computed(() => (permissions.value || []).filter(permission => permission.levels.includes('read')))
const createOpen = ref(false)
const editOpen = ref(false)
const deleteOpen = ref(false)
const selectedKey = ref<KeyMetadata | null>(null)
const name = ref('')
const expiry = ref('never')
const customExpiry = ref('')
const preset = ref('read')
const customize = ref(false)
const grants = ref<Record<string, string>>({})
const secret = ref('')
const secretOpen = computed({
  get: () => !!secret.value,
  set: (open: boolean) => {
    if (!open) secret.value = ''
  }
})
const busy = ref(false)
const formError = ref('')
const copied = ref(false)
const inputClass = 'w-full rounded-xl border border-mist-200 bg-white px-3 py-2.5 text-sm text-mist-950 outline-none transition focus:border-attic-500 focus:ring-2 focus:ring-attic-500/15 dark:border-mist-600 dark:bg-mist-800 dark:text-white'
const fullDate = (value: string | null) => value ? new Date(value).toLocaleString() : undefined
const expiryDate = (value: string | null) => value ? new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) : 'No expiry'

function applyPreset(level: string) {
  preset.value = level
  grants.value = Object.fromEntries((permissions.value || [])
    .filter(permission => permission.resource !== 'users' && permission.resource !== 'organization')
    .map(permission => [permission.resource, permission.levels.includes(level) ? level : 'read']))
}
function openCreate() {
  name.value = ''
  expiry.value = 'never'
  customExpiry.value = ''
  customize.value = false
  formError.value = ''
  applyPreset('read')
  createOpen.value = true
}
function openEdit(key: KeyMetadata) {
  selectedKey.value = key
  name.value = key.description
  formError.value = ''
  editOpen.value = true
}
function openDelete(key: KeyMetadata) {
  selectedKey.value = key
  formError.value = ''
  deleteOpen.value = true
}
async function run(action: () => Promise<void>) {
  busy.value = true
  formError.value = ''
  try {
    await action()
  } catch (err) {
    formError.value = (err as { data?: { error?: string } }).data?.error || 'Something went wrong. Please try again.'
  } finally { busy.value = false }
}
async function create() {
  await run(async () => {
    const permissions = Object.fromEntries(Object.entries(grants.value).filter(([, value]) => value !== 'none'))
    let expiresAt: string | null = null
    if (expiry.value === 'custom') expiresAt = new Date(customExpiry.value).toISOString()
    else if (expiry.value !== 'never') expiresAt = new Date(Date.now() + Number(expiry.value) * 86_400_000).toISOString()
    const result = await api<KeyMetadata & { secret: string }>('/api/me/api-keys', {
      method: 'POST', body: JSON.stringify({ description: name.value, permissions, expires_at: expiresAt })
    })
    createOpen.value = false
    copied.value = false
    secret.value = result.secret
    await refresh()
  })
}
async function rename() {
  if (!selectedKey.value) return
  await run(async () => {
    await api(`/api/me/api-keys/${selectedKey.value!.id}`, { method: 'PATCH', body: JSON.stringify({ description: name.value }) })
    editOpen.value = false
    await refresh()
    toast.add({ title: 'Key name updated', color: 'success' })
  })
}
async function remove() {
  if (!selectedKey.value) return
  await run(async () => {
    await api(`/api/me/api-keys/${selectedKey.value!.id}`, { method: 'DELETE' })
    deleteOpen.value = false
    await refresh()
    toast.add({ title: 'API key deleted', color: 'success' })
  })
}
async function copy() {
  try {
    await navigator.clipboard.writeText(secret.value)
    copied.value = true
  } catch { toast.add({ title: 'Select and copy the key below', color: 'error' }) }
}
</script>

<template>
  <div class="space-y-5 pb-6">
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500 dark:text-attic-400">
          Account settings
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          API keys
        </h1>
        <p class="mt-1 text-sm text-muted">
          Manage the apps and assistants connected to your account.
        </p>
      </div>
      <UButton
        icon="i-lucide-plus"
        class="self-start rounded-xl font-bold sm:self-auto"
        :disabled="!permissions || !!permissionError || !!secret"
        @click="openCreate"
      >
        Create key
      </UButton>
    </header>

    <UAlert
      v-if="loadError || permissionError"
      color="error"
      title="Could not load API keys. Please refresh the page and try again."
    />

    <section
      class="attic-panel overflow-hidden rounded-2xl"
      aria-labelledby="active-keys-heading"
    >
      <div class="flex items-center justify-between border-b border-mist-100 px-5 py-4 dark:border-mist-700 sm:px-6">
        <div class="flex items-center gap-2.5">
          <h2
            id="active-keys-heading"
            class="text-sm font-bold text-mist-950 dark:text-white"
          >
            Active keys
          </h2>
          <span class="rounded-md bg-mist-100 px-2 py-0.5 text-xs font-semibold tabular-nums text-muted dark:bg-mist-700">{{ activeKeys.length }}</span>
        </div>
        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="ghost"
          size="sm"
          aria-label="Refresh API keys"
          :loading="status === 'pending'"
          @click="refresh()"
        />
      </div>
      <div
        v-if="status === 'pending' && !keys"
        class="space-y-5 p-6"
        aria-label="Loading API keys"
      >
        <USkeleton
          v-for="row in 3"
          :key="row"
          class="h-10 w-full"
        />
      </div>
      <div
        v-else-if="!activeKeys.length && !loadError"
        class="px-6 py-16 text-center"
      >
        <div class="mx-auto mb-4 flex size-12 items-center justify-center rounded-2xl bg-mist-100 text-mist-400 dark:bg-mist-700">
          <UIcon
            name="i-lucide-key-round"
            class="size-6"
          />
        </div>
        <h3 class="font-bold text-mist-950 dark:text-white">
          No active API keys
        </h3>
        <p class="mx-auto mt-1.5 max-w-sm text-sm leading-6 text-muted">
          Create a key to connect an app or assistant to your inventory.
        </p>
        <UButton
          class="mt-5 rounded-xl"
          variant="outline"
          icon="i-lucide-plus"
          :disabled="!permissions || !!permissionError"
          @click="openCreate"
        >
          Create your first key
        </UButton>
      </div>
      <div
        v-else-if="activeKeys.length"
        class="overflow-x-auto"
      >
        <table class="w-full text-left text-sm">
          <thead class="border-b border-mist-100 bg-mist-50/50 text-xs font-semibold text-muted dark:border-mist-700 dark:bg-mist-700/20">
            <tr>
              <th
                scope="col"
                class="px-5 py-3 sm:px-6"
              >
                Name
              </th>
              <th
                scope="col"
                class="hidden px-5 py-3 sm:table-cell"
              >
                Last used
              </th>
              <th
                scope="col"
                class="hidden px-5 py-3 lg:table-cell"
              >
                Expires
              </th>
              <th
                scope="col"
                class="px-5 py-3 text-right sm:px-6"
              >
                <span class="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-mist-100 dark:divide-mist-700">
            <tr
              v-for="key in activeKeys"
              :key="key.id"
              class="transition-colors hover:bg-mist-50/50 dark:hover:bg-mist-700/20"
            >
              <td class="px-5 py-5 sm:px-6">
                <p class="break-words font-semibold text-mist-950 dark:text-white">
                  {{ key.description }}
                </p>
                <p class="mt-1 font-mono text-xs text-muted">
                  {{ key.prefix }}…
                </p>
                <p class="mt-2 text-xs text-muted sm:hidden">
                  Last used: {{ formatKeyLastUsed(key.last_used_at) }}
                </p>
                <p
                  v-if="key.expires_at"
                  class="mt-1 text-xs text-muted lg:hidden"
                >
                  Expires {{ expiryDate(key.expires_at) }}
                </p>
              </td>
              <td class="hidden whitespace-nowrap px-5 py-5 text-muted sm:table-cell">
                <time
                  :datetime="key.last_used_at || undefined"
                  :title="fullDate(key.last_used_at)"
                >{{ formatKeyLastUsed(key.last_used_at) }}</time>
              </td>
              <td class="hidden whitespace-nowrap px-5 py-5 text-muted lg:table-cell">
                {{ expiryDate(key.expires_at) }}
              </td>
              <td class="px-5 py-5 sm:px-6">
                <div class="flex justify-end gap-1">
                  <UButton
                    icon="i-lucide-pencil"
                    variant="ghost"
                    color="neutral"
                    :aria-label="`Edit ${key.description}`"
                    title="Edit key"
                    @click="openEdit(key)"
                  />
                  <UButton
                    icon="i-lucide-trash-2"
                    variant="ghost"
                    color="error"
                    :aria-label="`Delete ${key.description}`"
                    title="Delete key"
                    @click="openDelete(key)"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <p class="flex items-start gap-2 px-1 text-xs leading-5 text-muted">
      <UIcon
        name="i-lucide-shield-check"
        class="mt-0.5 size-4 shrink-0"
      />Keys act on your behalf. Only give them to apps you trust.
    </p>

    <UModal
      v-model:open="createOpen"
      title="Create API key"
      description="Choose a name and the access your app needs."
      :dismissible="!busy"
    >
      <template #body>
        <form
          id="create-api-key"
          class="space-y-5"
          @submit.prevent="create"
        >
          <UAlert
            v-if="formError"
            color="error"
            :title="formError"
          />
          <label class="block space-y-2"><span class="text-sm font-semibold">Name</span><input
            v-model="name"
            :class="inputClass"
            maxlength="255"
            required
            placeholder="e.g. Home assistant"
            aria-label="Name"
            autofocus
          ></label>
          <fieldset class="space-y-2">
            <legend class="mb-2 text-sm font-semibold">
              Access
            </legend>
            <div class="grid grid-cols-2 gap-3">
              <button
                v-for="option in [{ value: 'read', label: 'Read only', help: 'View your inventory.' }, { value: 'write', label: 'Read & write', help: 'View and make changes.' }]"
                :key="option.value"
                type="button"
                class="rounded-xl border p-3 text-left transition-colors"
                :class="preset === option.value ? 'border-attic-500 bg-attic-50 dark:bg-attic-500/10' : 'border-mist-200 dark:border-mist-600'"
                :aria-pressed="preset === option.value"
                @click="applyPreset(option.value)"
              >
                <span class="block text-sm font-bold">{{ option.label }}</span><span class="mt-1 block text-xs text-muted">{{ option.help }}</span>
              </button>
            </div>
            <button
              type="button"
              class="flex items-center gap-1 py-1 text-xs font-semibold text-attic-600 dark:text-attic-400"
              :aria-expanded="customize"
              @click="customize = !customize"
            >
              Customize permissions<UIcon
                :name="customize ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                class="size-3.5"
              />
            </button>
            <div
              v-if="customize"
              class="max-h-60 divide-y divide-mist-100 overflow-y-auto rounded-xl border border-mist-200 px-3 dark:divide-mist-700 dark:border-mist-600"
            >
              <label
                v-for="permission in selectablePermissions"
                :key="permission.resource"
                class="flex items-center justify-between gap-3 py-2.5"
              ><span class="text-sm">{{ permission.label }}</span><select
                :value="grants[permission.resource] || 'none'"
                :aria-label="`${permission.label} access`"
                class="rounded-lg border border-mist-200 bg-white px-2 py-1.5 text-xs dark:border-mist-600 dark:bg-mist-800"
                @change="grants[permission.resource] = ($event.target as HTMLSelectElement).value; preset = 'custom'"
              ><option
                v-for="level in permission.levels"
                :key="level"
                :value="level"
              >{{ level === 'none' ? 'No access' : level === 'read' ? 'Read only' : 'Read & write' }}</option></select></label>
            </div>
          </fieldset>
          <label class="block space-y-2"><span class="text-sm font-semibold">Expiration</span><select
            v-model="expiry"
            :class="inputClass"
            aria-label="Expiration"
          ><option value="never">No expiration</option><option value="30">In 30 days</option><option value="90">In 90 days</option><option value="365">In 1 year</option><option value="custom">Choose a date</option></select></label>
          <input
            v-if="expiry === 'custom'"
            v-model="customExpiry"
            type="datetime-local"
            :class="inputClass"
            aria-label="Expiration date"
            required
          >
        </form>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton
            color="neutral"
            variant="ghost"
            :disabled="busy"
            @click="createOpen = false"
          >
            Cancel
          </UButton><UButton
            type="submit"
            form="create-api-key"
            :loading="busy"
            :disabled="!name.trim() || !Object.values(grants).some(value => value !== 'none')"
          >
            Create key
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="secretOpen"
      title="Your API key is ready"
      description="Copy it now. You won't be able to see this key again."
      :dismissible="false"
      :close="false"
    >
      <template #body>
        <code
          class="block break-all rounded-xl border border-attic-200 bg-attic-50 p-4 text-sm text-attic-700 dark:border-attic-800 dark:bg-attic-500/10 dark:text-attic-300"
          data-testid="new-api-key"
        >{{ secret }}</code>
        <UButton
          class="mt-4"
          :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
          variant="outline"
          @click="copy"
        >
          {{ copied ? 'Copied' : 'Copy key' }}
        </UButton>
      </template>
      <template #footer>
        <div class="flex w-full justify-end">
          <UButton @click="secret = ''">
            I saved my key
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="editOpen"
      title="Edit API key"
      description="Update the name to make this key easier to recognize."
      :dismissible="!busy"
    >
      <template #body>
        <form
          id="edit-api-key"
          class="space-y-4"
          @submit.prevent="rename"
        >
          <UAlert
            v-if="formError"
            color="error"
            :title="formError"
          />
          <label class="block space-y-2"><span class="text-sm font-semibold">Name</span><input
            v-model="name"
            :class="inputClass"
            maxlength="255"
            aria-label="Name"
            required
          ></label>
          <details class="rounded-xl border border-mist-200 p-3 text-sm dark:border-mist-600">
            <summary class="cursor-pointer font-semibold">
              Permissions
            </summary><ul class="mt-3 space-y-2 text-xs text-muted">
              <li
                v-for="(level, resource) in selectedKey?.permissions"
                :key="resource"
                class="flex justify-between gap-4"
              >
                <span>{{ permissions?.find(item => item.resource === resource)?.label || resource }}</span><span>{{ level === 'write' ? 'Read & write' : 'Read only' }}</span>
              </li>
            </ul>
          </details>
          <p class="text-xs leading-5 text-muted">
            To change permissions or expiration, create a new key and delete this one.
          </p>
        </form>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton
            color="neutral"
            variant="ghost"
            :disabled="busy"
            @click="editOpen = false"
          >
            Cancel
          </UButton><UButton
            type="submit"
            form="edit-api-key"
            :loading="busy"
            :disabled="!name.trim()"
          >
            Save changes
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="deleteOpen"
      title="Delete API key?"
      :description="`Any app using “${selectedKey?.description || 'this key'}” will lose access immediately.`"
      :dismissible="!busy"
    >
      <template #body>
        <UAlert
          v-if="formError"
          color="error"
          :title="formError"
        /><p class="text-sm text-muted">
          You can create a new key whenever you need one.
        </p>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton
            color="neutral"
            variant="ghost"
            :disabled="busy"
            @click="deleteOpen = false"
          >
            Cancel
          </UButton><UButton
            color="error"
            :loading="busy"
            @click="remove"
          >
            Delete key
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
