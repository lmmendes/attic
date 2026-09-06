<script setup lang="ts">
import type { Collection } from '~/types/api'

definePageMeta({ middleware: 'auth' })
const { data: collections, status, error, refresh } = useApi<Collection[]>('/api/collections')
const apiFetch = useApiFetch()
const toast = useToast()
const search = ref('')
const filteredCollections = computed(() => {
  const query = search.value.trim().toLowerCase()
  return (collections.value || []).filter(collection =>
    collection.name.toLowerCase().includes(query) || collection.description?.toLowerCase().includes(query)
  )
})
const summary = computed(() => {
  const items = collections.value || []
  const withAssets = items.filter(collection => collection.asset_count > 0).length
  const ready = status.value === 'success' && !error.value
  return [
    { label: 'Collections', value: ready ? items.length : '—' },
    { label: 'With assets', value: ready ? withAssets : '—' },
    { label: 'Empty', value: ready ? items.length - withAssets : '—' }
  ]
})
const editorOpen = ref(false)
const deleting = ref<Collection | null>(null)
const deleteOpen = computed({
  get: () => deleting.value !== null,
  set: (value) => {
    if (!value) deleting.value = null
  }
})
const editing = ref<string | null>(null)
const busy = ref(false)
const formError = ref('')
const form = reactive({ name: '', description: '', icon: 'i-lucide-library' })
const icons = ['library', 'gamepad-2', 'disc-3', 'armchair', 'sofa', 'book-open', 'music', 'camera', 'headphones', 'monitor', 'shirt', 'watch', 'gem', 'wrench', 'cooking-pot', 'bike', 'flower-2', 'palette', 'puzzle', 'box'].map(name => `i-lucide-${name}`)

function errorMessage(err: unknown, fallback: string) {
  const response = err as { data?: { error?: string } }
  return response?.data?.error || fallback
}

function edit(collection?: Collection) {
  editing.value = collection?.id || null
  Object.assign(form, { name: collection?.name || '', description: collection?.description || '', icon: collection?.icon || 'i-lucide-library' })
  formError.value = ''
  editorOpen.value = true
}

async function save() {
  if (busy.value || !form.name.trim()) return
  busy.value = true
  formError.value = ''
  try {
    await apiFetch(editing.value ? `/api/collections/${editing.value}` : '/api/collections', {
      method: editing.value ? 'PUT' : 'POST',
      body: JSON.stringify({ ...form, name: form.name.trim(), description: form.description.trim() })
    })
    editorOpen.value = false
    toast.add({ title: editing.value ? 'Collection updated' : 'Collection created', color: 'success' })
    await refresh()
  } catch (err) {
    formError.value = errorMessage(err, 'Could not save collection. Please try again.')
  } finally { busy.value = false }
}

async function remove() {
  if (!deleting.value || busy.value) return
  busy.value = true
  formError.value = ''
  try {
    await apiFetch(`/api/collections/${deleting.value.id}`, { method: 'DELETE' })
    deleting.value = null
    toast.add({ title: 'Collection deleted. Your assets are safe.', color: 'success' })
    await refresh()
  } catch (err) {
    formError.value = errorMessage(err, 'Could not delete collection. Please try again.')
  } finally { busy.value = false }
}
</script>

<template>
  <div class="space-y-6 pb-6">
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Your things, together
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Collections
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          Gather games, furniture, and favorite finds into collections shared with everyone in your home.
        </p>
      </div>
      <UButton
        icon="i-lucide-plus"
        class="shrink-0 rounded-xl font-bold shadow-primary"
        @click="edit()"
      >
        New collection
      </UButton>
    </header>
    <LibrarySummary :items="summary" />
    <LibraryToolbar
      v-model="search"
      title="Collection library"
      placeholder="Search collections"
      :count="filteredCollections.length"
      :total="collections?.length || 0"
    />
    <div
      v-if="status === 'pending'"
      role="status"
      class="py-12 text-center text-muted"
    >
      Loading collections…
    </div>
    <div
      v-else-if="error"
      role="alert"
      class="attic-panel rounded-2xl p-8 text-center"
    >
      <p>Collections could not be loaded.</p>
      <UButton
        class="mt-3"
        @click="refresh()"
      >
        Try again
      </UButton>
    </div>
    <div
      v-else-if="!collections?.length"
      class="attic-panel rounded-2xl px-6 py-16 text-center"
    >
      <UIcon
        name="i-lucide-library"
        class="size-12 text-attic-500"
      />
      <h2 class="mt-4 text-xl font-bold">
        Make room for your collections
      </h2>
      <p class="mx-auto mt-2 max-w-md text-sm text-muted">
        Start with PS5 games, books, or furniture. Add assets to one or more collections from their asset form.
      </p>
      <UButton
        class="mt-5"
        icon="i-lucide-plus"
        @click="edit()"
      >
        Create your first collection
      </UButton>
    </div>
    <div
      v-else-if="!filteredCollections.length"
      class="attic-panel rounded-2xl px-6 py-14 text-center"
    >
      <UIcon
        name="i-lucide-search-x"
        class="mx-auto mb-3 size-7 text-mist-300"
      />
      <p class="font-bold text-mist-700 dark:text-mist-200">
        No matching collections
      </p>
      <UButton
        variant="ghost"
        class="mt-2"
        @click="search = ''"
      >
        Clear search
      </UButton>
    </div>
    <div
      v-else
      class="attic-panel overflow-hidden rounded-[20px]"
    >
      <div class="hidden grid-cols-[minmax(0,1.05fr)_minmax(0,1.35fr)_auto] gap-4 border-b border-mist-100 bg-mist-50/70 px-5 py-2.5 text-[10px] font-extrabold uppercase tracking-[0.14em] text-muted dark:border-mist-700 dark:bg-mist-800/70 sm:grid">
        <span>Collection</span>
        <span>Description</span>
        <span class="pr-2 text-right">Actions</span>
      </div>
      <div
        role="list"
        class="divide-y divide-mist-100 dark:divide-mist-700"
      >
        <LibraryCard
          v-for="collection in filteredCollections"
          :key="collection.id"
          :name="collection.name"
          :description="collection.description"
          :icon="collection.icon || 'i-lucide-library'"
          :asset-count="collection.asset_count"
          :assets-to="'/assets?collection_id=' + encodeURIComponent(collection.id)"
          @edit="edit(collection)"
          @delete="formError = ''; deleting = collection"
        />
      </div>
    </div>
    <UModal
      v-model:open="editorOpen"
      :title="editing ? 'Edit collection' : 'New collection'"
      description="Assets are assigned to collections from the asset form."
      :dismissible="!busy"
    >
      <template #body>
        <form
          class="space-y-5"
          @submit.prevent="save"
        >
          <UFormField
            label="Name"
            required
          >
            <UInput
              v-model="form.name"
              required
              maxlength="255"
              placeholder="e.g. PS5 games"
              class="w-full"
              autofocus
            />
          </UFormField>
          <UFormField label="Description">
            <UTextarea
              v-model="form.description"
              maxlength="2000"
              placeholder="What belongs in this collection?"
              class="w-full"
            />
          </UFormField>
          <fieldset>
            <legend class="mb-2 text-sm font-semibold">
              Icon
            </legend>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="icon in icons"
                :key="icon"
                type="button"
                :aria-label="getIconLabel(icon)"
                :aria-pressed="form.icon === icon"
                class="flex size-11 items-center justify-center rounded-xl border focus-visible:outline-2 focus-visible:outline-attic-500"
                :class="form.icon === icon ? 'border-attic-500 bg-attic-500/10 text-attic-500' : 'border-subtle text-muted'"
                @click="form.icon = icon"
              >
                <UIcon
                  :name="icon"
                  class="size-5"
                />
              </button>
            </div>
          </fieldset>
          <p
            v-if="formError"
            role="alert"
            class="text-sm text-error"
          >
            {{ formError }}
          </p>
          <div class="flex justify-end gap-2">
            <UButton
              color="neutral"
              variant="ghost"
              :disabled="busy"
              @click="editorOpen = false"
            >
              Cancel
            </UButton><UButton
              type="submit"
              :loading="busy"
              :disabled="!form.name.trim()"
            >
              {{ editing ? 'Save changes' : 'Create collection' }}
            </UButton>
          </div>
        </form>
      </template>
    </UModal>
    <UModal
      v-model:open="deleteOpen"
      title="Delete collection?"
      :description="`Delete ${deleting?.name || 'this collection'}? Its assets will stay in your inventory and in their other collections.`"
      :dismissible="!busy"
    >
      <template #body>
        <p
          v-if="formError"
          role="alert"
          class="mb-4 text-sm text-error"
        >
          {{ formError }}
        </p>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="ghost"
            :disabled="busy"
            @click="deleting = null"
          >
            Cancel
          </UButton><UButton
            color="error"
            :loading="busy"
            @click="remove"
          >
            Delete collection
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
