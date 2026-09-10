import type { FeatureConfiguration, FeatureName } from '~/types/api'

const defaultConfiguration: FeatureConfiguration = {
  collections_enabled: true,
  plugins_enabled: true,
  conditions_enabled: true,
  locations_enabled: true,
  warranties_enabled: true
}

let pendingConfigurationRequest: Promise<void> | null = null

export function useConfiguration() {
  const configuration = useState<FeatureConfiguration>('feature-configuration', () => ({ ...defaultConfiguration }))
  if (!configuration.value) configuration.value = { ...defaultConfiguration }
  const loaded = useState<boolean>('feature-configuration-loaded', () => false)
  const loading = useState<boolean>('feature-configuration-loading', () => false)
  const runtimeConfig = useRuntimeConfig()

  async function fetchConfiguration(force = false) {
    if (loaded.value && !force) return
    if (pendingConfigurationRequest) return pendingConfigurationRequest

    pendingConfigurationRequest = (async () => {
      loading.value = true
      try {
        configuration.value = await $fetch<FeatureConfiguration>('/api/configuration', {
          baseURL: runtimeConfig.public.apiBase as string,
          credentials: 'include'
        })
        loaded.value = true
      } finally {
        loading.value = false
        pendingConfigurationRequest = null
      }
    })()
    return pendingConfigurationRequest
  }

  async function updateConfiguration(update: Partial<FeatureConfiguration>) {
    configuration.value = await $fetch<FeatureConfiguration>('/api/configuration', {
      baseURL: runtimeConfig.public.apiBase as string,
      method: 'PATCH',
      body: update,
      credentials: 'include'
    })
    loaded.value = true
  }

  function isFeatureEnabled(feature: FeatureName) {
    return configuration.value[`${feature}_enabled`]
  }

  return { configuration, loaded, loading, fetchConfiguration, updateConfiguration, isFeatureEnabled }
}
