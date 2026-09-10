import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockFetch = vi.fn()
vi.stubGlobal('$fetch', mockFetch)

describe('useConfiguration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetch.mockReset()
  })

  it('defaults every optional feature to enabled', async () => {
    const { useConfiguration } = await import('../../app/composables/useConfiguration')
    const { configuration, isFeatureEnabled } = useConfiguration()

    configuration.value = {
      collections_enabled: true,
      plugins_enabled: true,
      conditions_enabled: true,
      locations_enabled: true,
      warranties_enabled: true
    }

    expect(isFeatureEnabled('collections')).toBe(true)
    expect(isFeatureEnabled('plugins')).toBe(true)
    expect(isFeatureEnabled('conditions')).toBe(true)
    expect(isFeatureEnabled('locations')).toBe(true)
    expect(isFeatureEnabled('warranties')).toBe(true)
  })

  it('loads and updates organization configuration', async () => {
    const loadedConfiguration = {
      collections_enabled: true,
      plugins_enabled: false,
      conditions_enabled: true,
      locations_enabled: false,
      warranties_enabled: true
    }
    const updatedConfiguration = { ...loadedConfiguration, locations_enabled: true }
    mockFetch.mockResolvedValueOnce(loadedConfiguration).mockResolvedValueOnce(updatedConfiguration)

    const { useConfiguration } = await import('../../app/composables/useConfiguration')
    const { configuration, loaded, fetchConfiguration, updateConfiguration } = useConfiguration()
    loaded.value = false

    await fetchConfiguration()
    expect(configuration.value).toEqual(loadedConfiguration)
    expect(mockFetch).toHaveBeenCalledWith('/api/configuration', expect.objectContaining({ credentials: 'include' }))

    await updateConfiguration({ locations_enabled: true })
    expect(configuration.value).toEqual(updatedConfiguration)
    expect(mockFetch).toHaveBeenLastCalledWith('/api/configuration', expect.objectContaining({
      method: 'PATCH',
      body: { locations_enabled: true }
    }))
  })
})
