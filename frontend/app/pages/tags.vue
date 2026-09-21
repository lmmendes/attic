<script setup lang="ts">
import type { Tag } from '~/types/api'

definePageMeta({ middleware: 'auth' })
const { data: tags, status, error, refresh } = useApi<Tag[]>('/api/tags')
const apiFetch = useApiFetch()
const toast = useToast()
const search = ref('')
const editorOpen = ref(false)
const editing = ref<Tag>()
const deleting = ref<Tag | null>(null)
const deleteOpen = computed({
  get: () => deleting.value !== null,
  set: (value) => { if (!value) deleting.value = null }
})
const busy = ref(false)
const formError = ref('')
const form = reactive({ name: '', description: '' })
const filtered = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return (tags.value || []).filter(tag => !query || tag.name.toLocaleLowerCase().includes(query) || tag.description?.toLocaleLowerCase().includes(query))
})
const summary = computed(() => {
  const items = tags.value || []
  return [
    { label: 'Tags', value: status.value === 'success' ? items.length : '—' },
    { label: 'In use', value: status.value === 'success' ? items.filter(tag => tag.asset_count > 0).length : '—' },
    { label: 'Assignments', value: status.value === 'success' ? items.reduce((total, tag) => total + tag.asset_count, 0) : '—' }
  ]
})

function edit(tag?: Tag) {
  editing.value = tag
  form.name = tag?.name || ''
  form.description = tag?.description || ''
  formError.value = ''
  editorOpen.value = true
}

async function save() {
  if (busy.value || !form.name.trim()) return
  busy.value = true
  formError.value = ''
  try {
    await apiFetch(editing.value ? `/api/tags/${editing.value.id}` : '/api/tags', {
      method: editing.value ? 'PUT' : 'POST',
      body: JSON.stringify({ name: form.name.trim(), description: form.description.trim() || undefined })
    })
    editorOpen.value = false
    toast.add({ title: editing.value ? 'Tag updated' : 'Tag created', color: 'success' })
    await refresh()
  } catch (err) {
    formError.value = (err as { data?: { error?: string } })?.data?.error || 'Could not save tag. Please try again.'
  } finally { busy.value = false }
}

async function remove() {
  if (!deleting.value || busy.value) return
  busy.value = true
  try {
    await apiFetch(`/api/tags/${deleting.value.id}`, { method: 'DELETE' })
    deleting.value = null
    toast.add({ title: 'Tag deleted. Your assets are safe.', color: 'success' })
    await refresh()
  } catch (err) {
    const message = (err as { data?: { error?: string } })?.data?.error || 'Could not delete tag. Please try again.'
    toast.add({ title: message, color: 'error' })
  } finally { busy.value = false }
}
</script>

<template>
  <div class="space-y-6 pb-6">
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Flexible organization
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          Tags
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          Add lightweight shared tags to assets, then rename and describe them here.
        </p>
      </div>
      <UButton
        icon="i-lucide-plus"
        class="shrink-0 rounded-xl font-bold shadow-primary"
        @click="edit()"
      >
        New tag
      </UButton>
    </header>
    <LibrarySummary :items="summary" />
    <LibraryToolbar
      v-model="search"
      title="Tag library"
      placeholder="Search tags"
      :count="filtered.length"
      :total="tags?.length || 0"
    />
    <div
      v-if="status === 'pending'"
      role="status"
      class="py-12 text-center text-muted"
    >
      Loading tags…
    </div>
    <div
      v-else-if="error"
      role="alert"
      class="attic-panel rounded-2xl p-8 text-center"
    >
      <p>Tags could not be loaded.</p>
      <UButton
        class="mt-3"
        @click="refresh()"
      >
        Try again
      </UButton>
    </div>
    <div
      v-else-if="!tags?.length"
      class="attic-panel rounded-2xl px-6 py-16 text-center"
    >
      <UIcon
        name="i-lucide-tags"
        class="size-12 text-attic-500"
      />
      <h2 class="mt-4 text-xl font-bold text-default">
        Create your first tag
      </h2>
      <p class="mx-auto mt-2 max-w-md text-sm text-muted">
        Tags can also be created directly while adding or editing an asset.
      </p>
      <UButton
        class="mt-5"
        icon="i-lucide-plus"
        @click="edit()"
      >
        New tag
      </UButton>
    </div>
    <div
      v-else-if="!filtered.length"
      class="attic-panel rounded-2xl px-6 py-14 text-center text-muted"
    >
      No matching tags
    </div>
    <div
      v-else
      class="attic-panel overflow-hidden rounded-[20px]"
    >
      <div
        role="list"
        class="divide-y divide-mist-100 dark:divide-mist-700"
      >
        <div
          v-for="tag in filtered"
          :key="tag.id"
          role="listitem"
          class="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center"
        >
          <div class="min-w-0 flex-1">
            <NuxtLink
              :to="{ path: '/assets', query: { tag_id: tag.id } }"
              class="font-bold text-mist-950 hover:text-attic-500 dark:text-white"
            >{{ tag.name }}</NuxtLink>
            <p class="mt-1 text-sm text-muted">
              {{ tag.description || 'No description' }}
            </p>
          </div>
          <span class="text-xs font-semibold text-muted">{{ tag.asset_count }} {{ tag.asset_count === 1 ? 'asset' : 'assets' }}</span>
          <div class="flex gap-1">
            <UButton
              icon="i-lucide-pencil"
              variant="ghost"
              color="neutral"
              :aria-label="`Edit ${tag.name}`"
              @click="edit(tag)"
            />
            <UButton
              icon="i-lucide-trash-2"
              variant="ghost"
              color="error"
              :aria-label="`Delete ${tag.name}`"
              @click="deleting = tag"
            />
          </div>
        </div>
      </div>
    </div>
    <UModal
      v-model:open="editorOpen"
      :title="editing ? 'Edit tag' : 'New tag'"
      :dismissible="!busy"
    >
      <template #body>
        <form
          class="space-y-4"
          @submit.prevent="save"
        >
          <UFormField
            label="Name"
            required
          >
            <UInput
              v-model="form.name"
              maxlength="100"
              class="w-full"
              autofocus
            />
          </UFormField>
          <UFormField label="Description">
            <UTextarea
              v-model="form.description"
              maxlength="2000"
              class="w-full"
            />
          </UFormField>
          <p
            v-if="formError"
            role="alert"
            class="text-sm text-error"
          >
            {{ formError }}
          </p>
        </form>
      </template>
      <template #footer>
        <UButton
          variant="ghost"
          color="neutral"
          :disabled="busy"
          @click="editorOpen = false"
        >
          Cancel
        </UButton>
        <UButton
          :loading="busy"
          :disabled="!form.name.trim()"
          @click="save"
        >
          Save
        </UButton>
      </template>
    </UModal>
    <UModal
      v-model:open="deleteOpen"
      title="Delete tag"
      :description="`Remove “${deleting?.name || ''}” from ${deleting?.asset_count || 0} assets? The assets will not be deleted.`"
      :dismissible="!busy"
    >
      <template #footer>
        <UButton
          variant="ghost"
          color="neutral"
          :disabled="busy"
          @click="deleting = null"
        >
          Cancel
        </UButton>
        <UButton
          color="error"
          :loading="busy"
          @click="remove"
        >
          Delete tag
        </UButton>
      </template>
    </UModal>
  </div>
</template>
