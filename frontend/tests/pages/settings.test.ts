import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import SettingsPage from '../../app/pages/settings.vue'

const { features, featureRef, authLoading, fetchSession, load, update, toast } = vi.hoisted(() => {
  const features = {
    locations: true, collections: true, categories: true, attributes: true,
    conditions: true, warranties: true, plugins: true
  }
  return {
    features,
    featureRef: { __v_isRef: true, value: features },
    authLoading: { __v_isRef: true, value: false },
    fetchSession: vi.fn(),
    load: vi.fn(),
    update: vi.fn(),
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
mockNuxtImport('useToast', () => () => ({ add: toast }))

describe('organization feature settings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authLoading.value = false
    Object.keys(features).forEach(key => features[key as keyof typeof features] = true)
    update.mockResolvedValue(features)
  })

  it('renders loaded features without restarting the app feature gate', async () => {
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()

    expect(load).not.toHaveBeenCalled()
    for (const label of ['Locations', 'Collections', 'Categories & attributes', 'Conditions', 'Warranties', 'Plugins']) {
      expect(wrapper.text()).toContain(label)
    }
    expect(wrapper.text()).not.toContain('Use custom category properties')
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
    const saveButton = wrapper.findAll('button').find(button => button.text().includes('Save changes'))!
    await saveButton.trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledWith({ ...features })
    expect(toast).toHaveBeenCalledWith({ title: 'Settings saved', color: 'success' })
    wrapper.unmount()
  })

  it('saves attributes with the categories setting', async () => {
    features.categories = false
    const wrapper = await mountSuspended(SettingsPage)
    const saveButton = wrapper.findAll('button').find(button => button.text().includes('Save changes'))!
    await saveButton.trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledWith(expect.objectContaining({
      categories: false,
      attributes: false
    }))
    wrapper.unmount()
  })

  it('reloads server settings after a failed save', async () => {
    features.locations = false
    update.mockRejectedValueOnce(new Error('save failed'))
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()
    const saveButton = wrapper.findAll('button').find(button => button.text().includes('Save changes'))!
    await saveButton.trigger('click')
    await flushPromises()

    expect(load).toHaveBeenCalledOnce()
    expect(toast).toHaveBeenCalledWith({ title: 'Could not save settings', color: 'error' })
    wrapper.unmount()
  })
})
