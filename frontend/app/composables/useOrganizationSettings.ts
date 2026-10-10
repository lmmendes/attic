import type { OrganizationSettings } from '~/types/api'

// Organization-wide preferences (the currency of every asset price), loaded once after login (see app.vue).
export function useOrganizationSettings() {
  const settings = useState<OrganizationSettings>('organization-settings', () => ({ currency: DEFAULT_CURRENCY }))

  const load = async () => {
    try {
      settings.value = await useApiFetch()<OrganizationSettings>('/api/organization/settings')
    } catch {
      // Keep the current value: prices still render with the fallback currency.
    }
  }

  const update = async (next: OrganizationSettings) => {
    settings.value = await useApiFetch()<OrganizationSettings>('/api/organization/settings', {
      method: 'PUT',
      body: JSON.stringify(next)
    })
    return settings.value
  }

  // Called on logout, so a failed load after the next login (possibly against another
  // instance on the same address) never reuses the previous instance's currency.
  const reset = () => {
    settings.value = { currency: DEFAULT_CURRENCY }
  }

  return { settings, load, update, reset }
}
