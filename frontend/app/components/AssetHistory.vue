<script setup lang="ts">
import type { AssetEvent } from '~/types/api'
import { getIconLabel } from '~/utils/iconLabel'

const props = defineProps<{
  assetId: string
  createdAt: string
  updatedAt: string
}>()

type TimelineEntry = {
  id: string
  title: string
  description: string
  icon: string
  timestamp: string
  custom?: AssetEvent
}

const apiFetch = useApiFetch()
const toast = useToast()
const eventsUrl = computed(() => `/api/assets/${props.assetId}/events`)
const { data: events, refresh } = useApi<AssetEvent[]>(() => eventsUrl.value, { key: eventsUrl })

const icons = [
  'i-lucide-calendar', 'i-lucide-wrench', 'i-lucide-hammer', 'i-lucide-settings',
  'i-lucide-notebook-pen', 'i-lucide-circle-alert', 'i-lucide-circle-check',
  'i-lucide-package-check', 'i-lucide-truck', 'i-lucide-rotate-ccw',
  'i-lucide-sparkles', 'i-lucide-star'
]

const modalOpen = ref(false)
const deleting = ref<AssetEvent | null>(null)
const deletingOpen = computed({
  get: () => deleting.value !== null,
  set: (open: boolean) => {
    if (!open) deleting.value = null
  }
})
const editing = ref<AssetEvent | null>(null)
const busy = ref(false)
const formError = ref('')
const form = reactive({
  title: '',
  description: '',
  icon: 'i-lucide-calendar',
  occurred_at: localDateTime(new Date())
})

const timeline = computed<TimelineEntry[]>(() => {
  const entries: TimelineEntry[] = (events.value || []).map(event => ({
    id: event.id,
    title: event.title,
    description: event.description,
    icon: event.icon,
    timestamp: event.occurred_at,
    custom: event
  }))
  entries.push({
    id: 'system-updated', title: 'Last Updated', description: 'Asset details were modified.',
    icon: 'i-lucide-pencil', timestamp: props.updatedAt
  })
  entries.push({
    id: 'system-created', title: 'Asset Created', description: 'Initial entry created.',
    icon: 'i-lucide-package-plus', timestamp: props.createdAt
  })
  return entries.sort((a, b) => Date.parse(b.timestamp) - Date.parse(a.timestamp)
    || Number(Boolean(b.custom)) - Number(Boolean(a.custom))
    || b.id.localeCompare(a.id))
})

/** Opens an empty event form with local current date and time. */
function openCreate() {
  editing.value = null
  Object.assign(form, {
    title: '', description: '', icon: 'i-lucide-calendar', occurred_at: localDateTime(new Date())
  })
  formError.value = ''
  modalOpen.value = true
}

/** Opens the event form populated from an existing event. */
function openEdit(event: AssetEvent) {
  editing.value = event
  Object.assign(form, {
    title: event.title,
    description: event.description,
    icon: event.icon,
    occurred_at: localDateTime(event.occurred_at)
  })
  formError.value = ''
  modalOpen.value = true
}

/** Builds the edit and delete actions for a custom event. */
function eventActions(event: AssetEvent) {
  return [
    {
      label: 'Edit',
      icon: 'i-lucide-pencil',
      onSelect: () => openEdit(event)
    },
    {
      label: 'Delete',
      icon: 'i-lucide-trash-2',
      color: 'error' as const,
      onSelect: () => { deleting.value = event }
    }
  ]
}

/** Validates and persists the current event form. */
async function save() {
  formError.value = ''
  if (!form.title.trim()) {
    formError.value = 'Title is required.'
    return
  }
  const occurredAt = new Date(form.occurred_at)
  if (!form.occurred_at || Number.isNaN(occurredAt.getTime())) {
    formError.value = 'Date and time are required.'
    return
  }
  busy.value = true
  try {
    const url = editing.value ? `${eventsUrl.value}/${editing.value.id}` : eventsUrl.value
    await apiFetch(url, {
      method: editing.value ? 'PUT' : 'POST',
      body: JSON.stringify({
        title: form.title.trim(),
        description: form.description.trim(),
        icon: form.icon,
        occurred_at: occurredAt.toISOString()
      })
    })
    toast.add({ title: editing.value ? 'Event updated' : 'Event added', color: 'success' })
    modalOpen.value = false
    await refresh()
  } catch (error) {
    formError.value = apiErrorMessage(error, 'Failed to save event')
  } finally {
    busy.value = false
  }
}

/** Permanently deletes the selected custom event. */
async function deleteEvent() {
  if (!deleting.value) return
  busy.value = true
  try {
    await apiFetch(`${eventsUrl.value}/${deleting.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Event deleted', color: 'success' })
    deleting.value = null
    await refresh()
  } catch (error) {
    toast.add({ title: apiErrorMessage(error, 'Failed to delete event'), color: 'error' })
  } finally {
    busy.value = false
  }
}

/** Formats an event instant in the viewer's local timezone. */
function formatTimestamp(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  }).format(new Date(value))
}

/** Converts an instant to a value accepted by a datetime-local input. */
function localDateTime(value: Date | string): string {
  const date = value instanceof Date ? value : new Date(value)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

/** Extracts a useful API error without discarding the fallback message. */
function apiErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object') {
    const data = 'data' in error ? error.data : undefined
    if (data && typeof data === 'object' && 'error' in data && typeof data.error === 'string') return data.error
    if ('message' in error && typeof error.message === 'string') return error.message
  }
  return fallback
}
</script>

<template>
  <section class="space-y-3 pb-6">
    <div class="flex items-center justify-between px-1">
      <h3 class="text-base font-extrabold text-mist-950 dark:text-white">
        Asset History
      </h3>
      <UButton
        icon="i-lucide-plus"
        variant="soft"
        size="sm"
        @click="openCreate"
      >
        Add event
      </UButton>
    </div>
    <div class="attic-panel overflow-hidden rounded-[20px]">
      <ul class="relative my-6 ml-8 space-y-8 border-l-2 border-gray-200 dark:border-gray-700">
        <li
          v-for="entry in timeline"
          :key="entry.id"
          class="relative pl-8"
        >
          <span class="absolute -left-4 top-0 flex size-8 items-center justify-center rounded-full bg-white text-attic-500 ring-2 ring-gray-200 shadow-sm dark:bg-mist-800 dark:ring-gray-700">
            <UIcon
              :name="entry.icon"
              class="size-4"
            />
          </span>
          <div class="grid grid-cols-[minmax(0,1fr)_7.5rem_2rem] items-start gap-3 pr-5 sm:grid-cols-[minmax(0,1fr)_10rem_2rem]">
            <div class="min-w-0">
              <p class="text-sm font-bold text-mist-950 dark:text-white">
                {{ entry.title }}
              </p>
              <p
                v-if="entry.description"
                class="mt-1 whitespace-pre-wrap text-xs text-gray-500 dark:text-gray-400"
              >
                {{ entry.description }}
              </p>
            </div>
            <span class="pt-1 text-right text-xs font-medium tabular-nums text-gray-500 dark:text-gray-400">
              {{ formatTimestamp(entry.timestamp) }}
            </span>
            <UDropdownMenu
              v-if="entry.custom"
              :items="eventActions(entry.custom)"
            >
              <UButton
                :aria-label="`Actions for ${entry.title}`"
                icon="i-lucide-ellipsis"
                variant="ghost"
                color="neutral"
                size="sm"
              />
            </UDropdownMenu>
            <UButton
              v-else
              :aria-label="`No actions available for ${entry.title}`"
              icon="i-lucide-ellipsis"
              variant="ghost"
              color="neutral"
              size="sm"
              disabled
            />
          </div>
        </li>
      </ul>
    </div>

    <UModal
      v-model:open="modalOpen"
      :title="editing ? 'Edit event' : 'Add event'"
      description="Record a milestone in this asset's history."
    >
      <template #content>
        <form
          class="w-full max-w-lg space-y-4 rounded-[20px] bg-white p-6 shadow-xl dark:bg-mist-800"
          @submit.prevent="save"
        >
          <UFormField
            label="Title"
            required
          >
            <UInput
              v-model="form.title"
              maxlength="255"
              class="w-full"
              autofocus
            />
          </UFormField>
          <UFormField
            label="Description"
          >
            <UTextarea
              v-model="form.description"
              maxlength="2000"
              :rows="4"
              class="w-full"
            />
          </UFormField>
          <UFormField
            label="Date and time"
            required
          >
            <UInput
              v-model="form.occurred_at"
              type="datetime-local"
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
              type="button"
              color="neutral"
              variant="ghost"
              :disabled="busy"
              @click="modalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              type="submit"
              :loading="busy"
            >
              {{ editing ? 'Save changes' : 'Add event' }}
            </UButton>
          </div>
        </form>
      </template>
    </UModal>

    <UModal
      v-model:open="deletingOpen"
      title="Delete event"
      description="This action cannot be undone."
    >
      <template #content>
        <div class="w-full max-w-md space-y-5 rounded-[20px] bg-white p-6 shadow-xl dark:bg-mist-800">
          <p>Delete <strong>{{ deleting?.title }}</strong> from this asset's history?</p>
          <div class="flex justify-end gap-2">
            <UButton
              color="neutral"
              variant="ghost"
              :disabled="busy"
              @click="deleting = null"
            >
              Cancel
            </UButton>
            <UButton
              color="error"
              :loading="busy"
              @click="deleteEvent"
            >
              Delete event
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </section>
</template>
