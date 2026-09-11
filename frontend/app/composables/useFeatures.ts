import type { OrganizationFeatures } from '~/types/api'

const defaults: OrganizationFeatures = {
  locations: true,
  collections: true,
  categories: true,
  attributes: true,
  conditions: true,
  warranties: true,
  plugins: true
}

export function useFeatures() {
  const features = useState<OrganizationFeatures>('organization-features', () => ({ ...defaults }))
  const loaded = useState<boolean>('organization-features-loaded', () => false)
  const { data, refresh } = useApi<OrganizationFeatures>('/api/organization/features', { immediate: false })

  const load = async () => {
    await refresh()
    if (data.value) {
      features.value = {
        ...defaults,
        ...data.value,
        attributes: data.value.categories
      }
    }
    loaded.value = true
  }

  const update = async (next: OrganizationFeatures) => {
    const apiFetch = useApiFetch()
    const saved = await apiFetch<OrganizationFeatures>('/api/organization/features', {
      method: 'PUT',
      body: JSON.stringify({ ...next, attributes: next.categories })
    })
    features.value = { ...saved, attributes: saved.categories }
    return saved
  }

  return { features, loaded, load, update }
}
