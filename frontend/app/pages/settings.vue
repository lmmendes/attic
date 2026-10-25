<script setup lang="ts">
import type { OrganizationFeatures } from '~/types/api'

definePageMeta({ middleware: 'auth' })
const { isAdmin, loading: authLoading, fetchSession } = useAuth()
const { features, load, update } = useFeatures()
const { settings, loaded: settingsLoaded, error: settingsError, load: loadSettings, update: updateSettings } = useOrganizationSettings()
// Edit a copy, so unsaved changes (e.g. the currency) don't affect other pages; it follows the saved values.
const draft = ref({ ...settings.value })
watch(settings, (saved) => {
  draft.value = { ...saved }
})
const toast = useToast()
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
  if (!settingsLoaded.value) await loadSettings()
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
  if (savingGeneral.value || !settingsLoaded.value) return
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
  <div class="max-w-3xl mx-auto p-6 lg:p-10 space-y-8">
    <div>
      <h1 class="text-3xl font-black text-mist-950 dark:text-white">
        Settings
      </h1>
      <p class="text-muted mt-2">
        Choose which inventory features are available to your organization and set its currency.
      </p>
    </div>
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
          :disabled="!settingsLoaded"
          class="w-28 shrink-0"
        />
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
          :disabled="!settingsLoaded"
          @click="saveGeneralSettings"
        >
          Save settings
        </UButton>
      </div>
    </UCard>
  </div>
</template>
