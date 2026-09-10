<script setup lang="ts">
import type { FeatureConfiguration, FeatureName } from '~/types/api'

definePageMeta({ middleware: 'auth' })

const { isAdmin } = useAuth()
const { configuration, fetchConfiguration, updateConfiguration } = useConfiguration()
const toast = useToast()
const saving = ref(false)

const features: { key: FeatureName, label: string, description: string, icon: string }[] = [
  { key: 'collections', label: 'Collections', description: 'Group assets and filter inventory by shared collections.', icon: 'i-lucide-library' },
  { key: 'plugins', label: 'Plugins', description: 'Import assets and expose plugin-managed categories and fields.', icon: 'i-lucide-puzzle' },
  { key: 'conditions', label: 'Conditions', description: 'Assign condition values to assets and filter by condition.', icon: 'i-lucide-activity' },
  { key: 'locations', label: 'Locations', description: 'Organize assets in rooms, shelves, boxes, and other spaces.', icon: 'i-lucide-map-pin' },
  { key: 'warranties', label: 'Warranties', description: 'Track warranty coverage, providers, and expiration dates.', icon: 'i-lucide-shield-check' }
]

onMounted(async () => {
  if (!isAdmin.value) {
    await navigateTo('/')
    return
  }
  try {
    await fetchConfiguration(true)
  } catch {
    toast.add({ title: 'Could not load configuration', color: 'error' })
  }
})

async function toggle(feature: FeatureName, enabled: boolean) {
  const field = `${feature}_enabled` as keyof FeatureConfiguration
  const previous = configuration.value[field]
  saving.value = true
  try {
    await updateConfiguration({ [field]: enabled })
    const label = features.find(item => item.key === feature)?.label || feature
    toast.add({ title: `${label} ${enabled ? 'enabled' : 'disabled'}`, color: 'success' })
  } catch (error: unknown) {
    configuration.value = { ...configuration.value, [field]: previous }
    const failure = error as { data?: { error?: string }, message?: string }
    toast.add({ title: failure.data?.error || failure.message || 'Could not update configuration', color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-4xl space-y-6 pb-8">
    <header>
      <p class="text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
        Administration
      </p>
      <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
        Server configuration
      </h1>
      <p class="mt-1 text-sm text-muted">
        Choose which optional features are available in this workspace.
      </p>
    </header>

    <UAlert
      color="neutral"
      icon="i-lucide-database"
      title="Disabling a feature never deletes its data"
      description="Existing values are preserved and will return when the feature is enabled again."
    />

    <section class="attic-panel divide-y divide-mist-100 overflow-hidden rounded-[20px] dark:divide-mist-700">
      <div
        v-for="feature in features"
        :key="feature.key"
        class="flex items-center gap-4 p-5 sm:p-6"
      >
        <div class="flex size-11 shrink-0 items-center justify-center rounded-xl bg-attic-50 text-attic-600 dark:bg-attic-950/30 dark:text-attic-300">
          <UIcon
            :name="feature.icon"
            class="size-5"
          />
        </div>
        <div class="min-w-0 flex-1">
          <h2 class="font-extrabold text-mist-950 dark:text-white">
            {{ feature.label }}
          </h2>
          <p class="text-sm text-muted">
            {{ feature.description }}
          </p>
        </div>
        <USwitch
          :model-value="configuration[`${feature.key}_enabled`]"
          :disabled="saving"
          :aria-label="`${configuration[`${feature.key}_enabled`] ? 'Disable' : 'Enable'} ${feature.label}`"
          @update:model-value="toggle(feature.key, $event)"
        />
      </div>
    </section>
  </div>
</template>
