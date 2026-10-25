import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import SettingsPage from '../../app/pages/settings.vue'

const { currencyApi, loadCurrencies } = vi.hoisted(() => ({ currencyApi: vi.fn(), loadCurrencies: vi.fn() }))
mockNuxtImport('useApi', () => currencyApi)
const currencyCatalog = ref<{ currencies: string[] } | null>({ currencies: ['CZK', 'EUR', 'GBP', 'USD'] })
const currencyStatus = ref('success')
const currencyError = ref<Error | null>(null)

const { features, featureRef, settingsRef, settingsLoaded, settingsError, authLoading, fetchSession, load, update, loadSettings, updateSettings, toast } = vi.hoisted(() => {
  const features = {
    locations: true, collections: true, tags: true, categories: true, attributes: true,
    conditions: true, warranties: true, plugins: true
  }
  return {
    features,
    featureRef: { __v_isRef: true, value: features },
    settingsRef: { __v_isRef: true, value: { currency: 'USD' } },
    settingsLoaded: { __v_isRef: true, value: true },
    settingsError: { __v_isRef: true, value: null as Error | null },
    authLoading: { __v_isRef: true, value: false },
    fetchSession: vi.fn(),
    load: vi.fn(),
    update: vi.fn(),
    loadSettings: vi.fn(),
    updateSettings: vi.fn(),
    toast: vi.fn()
  }
})

mockNuxtImport('useAuth', () => () => ({
  isAdmin: { __v_isRef: true, value: true },
  isAuthenticated: { __v_isRef: true, value: true },
  loading: authLoading,
  fetchSession
}))
mockNuxtImport('useFeatures', () => () => ({ features: featureRef, load, update }))
mockNuxtImport('useOrganizationSettings', () => () => ({
  settings: settingsRef, loaded: settingsLoaded, error: settingsError, load: loadSettings, update: updateSettings
}))
mockNuxtImport('useToast', () => () => ({ add: toast }))

const currencyMenu = {
  props: ['modelValue', 'items', 'disabled'],
  emits: ['update:modelValue'],
  template: '<button aria-label="Currency" :disabled="disabled" :data-options="items.join(\',\')" @click="$emit(\'update:modelValue\', \'CZK\')">{{ modelValue }}</button>'
}

const button = (wrapper: Awaited<ReturnType<typeof mountSuspended>>, label: string) =>
  wrapper.findAll('button').find(element => element.text().includes(label))!

beforeEach(() => {
  vi.clearAllMocks()
  currencyCatalog.value = { currencies: ['CZK', 'EUR', 'GBP', 'USD'] }
  currencyStatus.value = 'success'
  currencyError.value = null
  loadCurrencies.mockReset().mockResolvedValue(undefined)
  currencyApi.mockReturnValue({ data: currencyCatalog, status: currencyStatus, error: currencyError, refresh: loadCurrencies })
  authLoading.value = false
  Object.keys(features).forEach(key => features[key as keyof typeof features] = true)
  update.mockResolvedValue(features)
  settingsRef.value = { currency: 'USD' }
  settingsLoaded.value = true
  settingsError.value = null
  updateSettings.mockResolvedValue(settingsRef.value)
})

describe('organization feature settings', () => {
  it('renders loaded features without restarting the app feature gate', async () => {
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()

    expect(load).not.toHaveBeenCalled()
    for (const label of ['Locations', 'Collections', 'Tags', 'Categories & attributes', 'Conditions', 'Warranties', 'Plugins']) {
      expect(wrapper.text()).toContain(label)
    }
    expect(wrapper.text()).not.toContain('Use custom category properties')
    expect(wrapper.findAll('h2').map(heading => heading.text())).toEqual(['Features', 'General settings'])
    wrapper.unmount()
  })

  it('uses readable text colors in light and dark mode', async () => {
    const wrapper = await mountSuspended(SettingsPage)
    const heading = wrapper.get('h1')
    const featureLabel = wrapper.findAll('p').find(element => element.text() === 'Locations')!

    expect(heading.classes()).toEqual(expect.arrayContaining(['text-mist-950', 'dark:text-white']))
    expect(featureLabel.classes()).toEqual(expect.arrayContaining(['text-mist-950', 'dark:text-white']))
    wrapper.unmount()
  })

  it('resolves authentication before loading settings', async () => {
    authLoading.value = true
    fetchSession.mockImplementationOnce(() => {
      authLoading.value = false
    })

    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()

    expect(fetchSession).toHaveBeenCalledOnce()
    expect(load).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('persists the complete feature map', async () => {
    features.plugins = false
    const wrapper = await mountSuspended(SettingsPage)
    const saveButton = button(wrapper, 'Save features')
    await saveButton.trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledWith({ ...features })
    expect(updateSettings).not.toHaveBeenCalled()
    expect(toast).toHaveBeenCalledWith({ title: 'Features saved', color: 'success' })
    wrapper.unmount()
  })

  it('saves attributes with the categories setting', async () => {
    features.categories = false
    const wrapper = await mountSuspended(SettingsPage)
    const saveButton = button(wrapper, 'Save features')
    await saveButton.trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledWith(expect.objectContaining({
      categories: false,
      attributes: false
    }))
    wrapper.unmount()
  })

  it('reloads the features after a failed features save', async () => {
    features.locations = false
    update.mockRejectedValueOnce(new Error('save failed'))
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()
    const saveButton = button(wrapper, 'Save features')
    await saveButton.trigger('click')
    await flushPromises()

    expect(load).toHaveBeenCalledOnce()
    expect(loadSettings).not.toHaveBeenCalled()
    expect(toast).toHaveBeenCalledWith({ title: 'Could not save features', color: 'error' })
    wrapper.unmount()
  })

  it('saves the currency on its own', async () => {
    settingsRef.value = { currency: 'GBP' }
    const wrapper = await mountSuspended(SettingsPage)
    expect(wrapper.text()).toContain('Used for all asset prices')
    await button(wrapper, 'Save settings').trigger('click')
    await flushPromises()

    expect(updateSettings).toHaveBeenCalledWith({ currency: 'GBP' })
    expect(update).not.toHaveBeenCalled()
    expect(toast).toHaveBeenCalledWith({ title: 'General settings saved', color: 'success' })
    wrapper.unmount()
  })

  it('does not send overlapping saves when duplicate clicks arrive before a render', async () => {
    let finish!: (value: { currency: string }) => void
    updateSettings.mockReturnValueOnce(new Promise((resolve) => {
      finish = resolve
    }))
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    const save = button(wrapper, 'Save settings')
    const first = save.trigger('click')
    const second = save.trigger('click')
    await Promise.all([first, second])
    expect(updateSettings).toHaveBeenCalledOnce()
    finish({ currency: 'USD' })
    await flushPromises()
    expect(save.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('keeps an unsaved currency choice on this page until it is saved', async () => {
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    await wrapper.get('button[aria-label="Currency"]').trigger('click')

    expect(wrapper.get('button[aria-label="Currency"]').text()).toBe('CZK')
    expect(settingsRef.value.currency).toBe('USD')

    const saveButton = button(wrapper, 'Save settings')
    await saveButton.trigger('click')
    await flushPromises()
    expect(updateSettings).toHaveBeenCalledWith({ currency: 'CZK' })
    expect(update).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('puts the saved currency back after a failed save', async () => {
    updateSettings.mockRejectedValueOnce(new Error('save failed'))
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    await wrapper.get('button[aria-label="Currency"]').trigger('click')
    const saveButton = button(wrapper, 'Save settings')
    await saveButton.trigger('click')
    await flushPromises()

    expect(loadSettings).toHaveBeenCalledOnce()
    expect(wrapper.get('button[aria-label="Currency"]').text()).toBe('USD')
    expect(update).not.toHaveBeenCalled()
    expect(toast).toHaveBeenCalledWith({ title: 'Could not save general settings', color: 'error' })
    wrapper.unmount()
  })

  it('loads the settings when the page opens before they have loaded', async () => {
    settingsLoaded.value = false
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()

    expect(loadSettings).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('blocks saving the currency until the settings have loaded', async () => {
    settingsLoaded.value = false
    loadSettings.mockResolvedValue(undefined)
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    await flushPromises()

    expect(wrapper.get('button[aria-label="Currency"]').attributes('disabled')).toBeDefined()
    expect(button(wrapper, 'Save settings').attributes('disabled')).toBeDefined()
    await button(wrapper, 'Save settings').trigger('click')
    await flushPromises()
    expect(updateSettings).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('shows a load error with a retry, and still saves features', async () => {
    settingsLoaded.value = false
    settingsError.value = new Error('offline')
    loadSettings.mockResolvedValue(undefined)
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('General settings could not be loaded')
    loadSettings.mockClear()
    await button(wrapper, 'Retry').trigger('click')
    expect(loadSettings).toHaveBeenCalledOnce()

    await button(wrapper, 'Save features').trigger('click')
    await flushPromises()
    expect(update).toHaveBeenCalledOnce()
    expect(updateSettings).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

describe('server-provided currency choices', () => {
  it('loads choices when Settings opens and uses the returned code list', async () => {
    currencyCatalog.value = null
    currencyStatus.value = 'idle'
    loadCurrencies.mockImplementationOnce(async () => {
      currencyCatalog.value = { currencies: ['EUR', 'USD'] }
      currencyStatus.value = 'success'
    })
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    await flushPromises()
    expect(currencyApi).toHaveBeenCalledWith('/api/organization/settings/currencies', { immediate: false })
    expect(loadCurrencies).toHaveBeenCalledOnce()
    expect(wrapper.get('button[aria-label="Currency"]').attributes('data-options')).toBe('EUR,USD')
    wrapper.unmount()
  })

  it('blocks currency changes while choices load, without blocking feature saves', async () => {
    currencyStatus.value = 'pending'
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    expect(wrapper.get('[role="status"]').text()).toContain('Loading currency choices')
    expect(wrapper.get('button[aria-label="Currency"]').attributes('disabled')).toBeDefined()
    expect(button(wrapper, 'Save settings').attributes('disabled')).toBeDefined()
    await button(wrapper, 'Save settings').trigger('click')
    expect(updateSettings).not.toHaveBeenCalled()
    await button(wrapper, 'Save features').trigger('click')
    await flushPromises()
    expect(update).toHaveBeenCalledOnce()
    currencyStatus.value = 'success'
    await flushPromises()
    expect(button(wrapper, 'Save settings').attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('preserves the selected draft and retries failed currency choices', async () => {
    const wrapper = await mountSuspended(SettingsPage, { global: { stubs: { USelectMenu: currencyMenu } } })
    await wrapper.get('button[aria-label="Currency"]').trigger('click')
    currencyStatus.value = 'error'
    currencyError.value = new Error('offline')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('Currency choices could not be loaded')
    expect(wrapper.get('button[aria-label="Currency"]').text()).toBe('CZK')
    expect(button(wrapper, 'Save settings').attributes('disabled')).toBeDefined()
    await button(wrapper, 'Save features').trigger('click')
    await flushPromises()
    expect(update).toHaveBeenCalledOnce()
    loadCurrencies.mockImplementationOnce(async () => {
      currencyError.value = null
      currencyStatus.value = 'success'
    })
    await button(wrapper, 'Retry currencies').trigger('click')
    await flushPromises()
    expect(loadCurrencies).toHaveBeenCalledOnce()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.get('button[aria-label="Currency"]').text()).toBe('CZK')
    await button(wrapper, 'Save settings').trigger('click')
    expect(updateSettings).toHaveBeenCalledWith({ currency: 'CZK' })
    wrapper.unmount()
  })

  it('treats an empty successful catalog as unavailable', async () => {
    currencyCatalog.value = { currencies: [] }
    const wrapper = await mountSuspended(SettingsPage)
    expect(wrapper.get('[role="alert"]').text()).toContain('Currency choices could not be loaded')
    expect(button(wrapper, 'Save settings').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
