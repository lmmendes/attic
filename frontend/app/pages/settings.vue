<script setup lang="ts">
import type { OrganizationFeatures } from '~/types/api'

definePageMeta({ middleware: 'auth' })
const { isAdmin, loading: authLoading, fetchSession } = useAuth()
const { features, load, update } = useFeatures()
const toast = useToast()
const saving = ref(false)
const labels: Array<{ key: keyof OrganizationFeatures, label: string, description: string }> = [
  { key: 'locations', label: 'Locations', description: 'Track where assets are stored.' },
  { key: 'collections', label: 'Collections', description: 'Group assets into custom collections.' },
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
  <div class="max-w-3xl mx-auto p-6 lg:p-10 space-y-8">
    <div>
      <h1 class="text-3xl font-black">
        Settings
      </h1>
      <p class="text-muted mt-2">
        Choose which inventory features are available to your organization.
      </p>
    </div>
    <UCard>
      <div class="divide-y divide-mist-100 dark:divide-mist-800">
        <div
          v-for="item in labels"
          :key="item.key"
          class="py-5 flex items-center justify-between gap-6"
        >
          <div>
            <p class="font-bold">
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
  </div>
</template>
