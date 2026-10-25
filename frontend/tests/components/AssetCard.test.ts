import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import { h, ref } from 'vue'
import AssetCard from '../../app/components/AssetCard.vue'
import type { Asset } from '../../app/types/api'

const flags = ref({ categories: true, locations: true, collections: true, conditions: true })
mockNuxtImport('useFeatures', () => () => ({ features: flags }))
const asset: Asset = {
  id: 'gpu', organization_id: 'org', name: 'Graphics card', quantity: 2, created_at: '', updated_at: '',
  main_attachment_url: '/files/gpu.jpg',
  category: { id: 'category', name: 'Components', organization_id: 'org', created_at: '', updated_at: '' },
  location: { id: 'location', name: 'Office', organization_id: 'org', created_at: '', updated_at: '' },
  collections: [
    { id: 'one', name: 'Computer', organization_id: 'org', created_at: '', updated_at: '' },
    { id: 'two', name: 'Spare parts', organization_id: 'org', created_at: '', updated_at: '' }
  ]
}
beforeEach(() => {
  flags.value = { categories: true, locations: true, collections: true, conditions: true }
})

describe('shared asset card', () => {
  it('shows consistent asset identity and optional price and update metadata', async () => {
    const wrapper = await mountSuspended(AssetCard, { props: { asset, priceLabel: '$0.00', updatedLabel: 'Updated today' } })
    const link = wrapper.get('a')
    expect(link.attributes()).toMatchObject({ 'href': '/assets/gpu', 'aria-label': 'Graphics card' })
    expect(wrapper.get('img').attributes()).toMatchObject({ src: '/files/gpu.jpg', alt: 'Graphics card', loading: 'lazy' })
    for (const text of ['Components', 'Office', 'Computer', '+1', '$0.00', 'Updated today']) expect(wrapper.text()).toContain(text)
    wrapper.unmount()
  })

  it('keeps contents actions outside the asset navigation link', async () => {
    const browse = vi.fn()
    const wrapper = await mountSuspended(AssetCard, {
      props: { asset, layout: 'list' },
      slots: {
        metadata: () => h('p', 'Qty 2 · Inside Gaming PC'),
        footer: () => h('button', { onClick: browse }, 'Browse contents')
      }
    })
    expect(wrapper.get('a').text()).toContain('Qty 2 · Inside Gaming PC')
    expect(wrapper.get('a').find('button').exists()).toBe(false)
    await wrapper.get('button').trigger('click')
    expect(browse).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('handles missing images and respects disabled metadata features', async () => {
    flags.value = { categories: false, locations: false, collections: false, conditions: false }
    const wrapper = await mountSuspended(AssetCard, { props: { asset: { ...asset, main_attachment_url: undefined }, layout: 'grid' } })
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.get('a').attributes('href')).toBe('/assets/gpu')
    for (const text of ['Components', 'Office', 'Computer', 'No collections']) expect(wrapper.text()).not.toContain(text)
    wrapper.unmount()
  })
})
