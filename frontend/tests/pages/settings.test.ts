import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import SettingsPage from '../../app/pages/settings.vue'

const { features, featureRef, load, update, toast } = vi.hoisted(() => {
  const features = {
    locations: true, collections: true, categories: true, attributes: true,
    conditions: true, warranties: true, plugins: true
  }
  return {
    features,
    featureRef: { __v_isRef: true, value: features },
    load: vi.fn(),
    update: vi.fn(),
    toast: vi.fn()
  }
})

mockNuxtImport('useAuth', () => () => ({
  isAdmin: { __v_isRef: true, value: true },
  isAuthenticated: { __v_isRef: true, value: true },
  loading: { __v_isRef: true, value: false },
  fetchSession: vi.fn()
}))
mockNuxtImport('useFeatures', () => () => ({ features: featureRef, load, update }))
mockNuxtImport('useToast', () => () => ({ add: toast }))

describe('organization feature settings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(features).forEach(key => features[key as keyof typeof features] = true)
    update.mockResolvedValue(features)
  })

  it('loads and renders every organization feature', async () => {
    const wrapper = await mountSuspended(SettingsPage)
    await flushPromises()

    expect(load).toHaveBeenCalledOnce()
    for (const label of ['Locations', 'Collections', 'Categories & attributes', 'Conditions', 'Warranties', 'Plugins']) {
      expect(wrapper.text()).toContain(label)
    }
    expect(wrapper.text()).not.toContain('Use custom category properties')
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

    expect(load).toHaveBeenCalledTimes(2)
    expect(toast).toHaveBeenCalledWith({ title: 'Could not save settings', color: 'error' })
    wrapper.unmount()
  })
})
