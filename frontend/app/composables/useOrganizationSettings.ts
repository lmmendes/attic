import type { OrganizationSettings } from '~/types/api'

// Organization-wide preferences (the currency of every asset price), loaded once after login (see app.vue).
export function useOrganizationSettings() {
  const settings = useState<OrganizationSettings>('organization-settings', () => ({ currency: DEFAULT_CURRENCY }))
  // Until loaded, settings hold only the fallback; never save that over the real value.
  const loaded = useState<boolean>('organization-settings-loaded', () => false)
  const error = useState<Error | null>('organization-settings-error', () => null)

  const load = async () => {
    loaded.value = false
    error.value = null
    try {
      settings.value = await useApiFetch()<OrganizationSettings>('/api/organization/settings')
      loaded.value = true
    } catch (cause) {
      // Keep the current value: prices still render with the fallback currency.
      error.value = cause instanceof Error ? cause : new Error('Organization settings could not be loaded')
    }
  }

  const update = async (next: OrganizationSettings) => {
    settings.value = await useApiFetch()<OrganizationSettings>('/api/organization/settings', {
      method: 'PUT',
      body: JSON.stringify(next)
    })
    loaded.value = true
    error.value = null
    return settings.value
  }

  // Called on logout, so a failed load after the next login (possibly against another
  // instance on the same address) never reuses the previous instance's currency.
  const reset = () => {
    settings.value = { currency: DEFAULT_CURRENCY }
    loaded.value = false
    error.value = null
  }

  return { settings, loaded, error, load, update, reset }
}
