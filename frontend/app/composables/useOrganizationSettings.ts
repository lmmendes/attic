import type { OrganizationSettings } from '~/types/api'

// In-flight promises belong to the app, never to serialized Nuxt state.
const settingsLoads = new WeakMap<object, { generation: number, promise: Promise<void> }>()

// Organization-wide preferences (the currency of every asset price), loaded once after login (see app.vue).
export function useOrganizationSettings() {
  const settings = useState<OrganizationSettings>('organization-settings', () => ({ currency: DEFAULT_CURRENCY }))
  // Until loaded, settings hold only the fallback; never save that over the real value.
  const loaded = useState<boolean>('organization-settings-loaded', () => false)
  const error = useState<Error | null>('organization-settings-error', () => null)

  // Shared across consumers: logout/reset and newer requests invalidate old responses.
  const generation = useState<number>('organization-settings-generation', () => 0)

  const app = useNuxtApp()
  const load = (): Promise<void> => {
    const inFlight = settingsLoads.get(app)
    if (inFlight?.generation === generation.value) return inFlight.promise
    const requestGeneration = ++generation.value
    loaded.value = false
    error.value = null
    const promise = (async () => {
      try {
        const saved = await useApiFetch()<OrganizationSettings>('/api/organization/settings')
        if (requestGeneration !== generation.value) return
        settings.value = saved
        loaded.value = true
      } catch (cause) {
        if (requestGeneration !== generation.value) return
        // Keep the current value: prices still render with the fallback currency.
        error.value = cause instanceof Error ? cause : new Error('Organization settings could not be loaded')
      }
    })().finally(() => {
      if (settingsLoads.get(app)?.generation === requestGeneration) settingsLoads.delete(app)
    })
    settingsLoads.set(app, { generation: requestGeneration, promise })
    return promise
  }

  const update = async (next: OrganizationSettings) => {
    const requestGeneration = ++generation.value
    settingsLoads.delete(app)
    const saved = await useApiFetch()<OrganizationSettings>('/api/organization/settings', {
      method: 'PUT',
      body: JSON.stringify(next)
    })
    if (requestGeneration !== generation.value) return settings.value
    settings.value = saved
    loaded.value = true
    error.value = null
    return settings.value
  }

  // Called on logout, so a failed load after the next login (possibly against another
  // instance on the same address) never reuses the previous instance's currency.
  const reset = () => {
    generation.value++
    settingsLoads.delete(app)
    settings.value = { currency: DEFAULT_CURRENCY }
    loaded.value = false
    error.value = null
  }

  return { settings, loaded, error, load, update, reset }
}
