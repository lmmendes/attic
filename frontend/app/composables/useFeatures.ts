import type { OrganizationFeatures } from '~/types/api'

const disabledFeatures: OrganizationFeatures = {
  locations: false,
  collections: false,
  categories: false,
  attributes: false,
  conditions: false,
  warranties: false,
  plugins: false
}

const featureKeys: Array<keyof OrganizationFeatures> = [
  'locations',
  'collections',
  'categories',
  'attributes',
  'conditions',
  'warranties',
  'plugins'
]

function isFeatureMap(value: unknown): value is OrganizationFeatures {
  if (!value || typeof value !== 'object') return false
  return featureKeys.every(key => typeof (value as Record<string, unknown>)[key] === 'boolean')
}

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
  const features = useState<OrganizationFeatures>('organization-features', () => ({ ...disabledFeatures }))
  const loaded = useState<boolean>('organization-features-loaded', () => false)
  const { data, error, status, refresh } = useApi<OrganizationFeatures>('/api/organization/features', { immediate: false })
  const invalidResponseError = useState<Error | null>('organization-features-invalid-response', () => null)
  const loadError = computed(() => error.value || invalidResponseError.value)

  const load = async () => {
    loaded.value = false
    features.value = { ...disabledFeatures }
    invalidResponseError.value = null
    try {
      await refresh()
    } catch (cause) {
      invalidResponseError.value = cause instanceof Error ? cause : new Error('Feature settings could not be loaded')
      return false
    }

    if (status.value !== 'success') {
      if (!error.value) invalidResponseError.value = new Error('Feature settings could not be loaded')
      return false
    }
    if (!isFeatureMap(data.value)) {
      invalidResponseError.value = new Error('Feature settings response is invalid')
      return false
    }
    features.value = {
      ...defaults,
      ...data.value,
      attributes: data.value.categories
    }
    loaded.value = true
    return true
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

  return { features, loaded, error: loadError, status, load, retry: load, update }
}
