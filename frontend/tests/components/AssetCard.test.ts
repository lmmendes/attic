import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { Asset } from '../../app/types/api'
import AssetCard from '../../app/components/AssetCard.vue'

const now = Date.parse('2026-10-24T12:00:00Z')
function asset(overrides: Partial<Asset> = {}): Asset {
  return {
    id: 'asset-123',
    organization_id: 'org',
    name: 'Camera',
    quantity: 1,
    created_at: '2026-10-24T10:00:00Z',
    updated_at: '2026-10-24T10:00:00Z',
    ...overrides
  }
}

afterEach(() => vi.restoreAllMocks())

describe('AssetCard', () => {
  it('links to the asset and shows fallback metadata and image placeholder', async () => {
    vi.spyOn(Date, 'now').mockReturnValue(now)
    const wrapper = await mountSuspended(AssetCard, { props: { asset: asset() } })
    expect(wrapper.get('a').attributes('href')).toBe('/assets/asset-123')
    expect(wrapper.text()).toContain('Camera')
    expect(wrapper.text()).toContain('Uncategorized')
    expect(wrapper.text()).toContain('No location')
    expect(wrapper.text()).toContain('No collections')
    expect(wrapper.text()).toContain('Today')
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'UIcon' }).exists()).toBe(true)
  })

  it('shows the image, category, location, first collection and remaining count', async () => {
    const timestamp = '2026-10-24T10:00:00Z'
    const wrapper = await mountSuspended(AssetCard, {
      props: {
        asset: asset({
          main_attachment_url: 'https://example.test/camera.png',
          category: { id: 'category', organization_id: 'org', name: 'Electronics', created_at: timestamp, updated_at: timestamp },
          location: { id: 'location', organization_id: 'org', name: 'Office', created_at: timestamp, updated_at: timestamp },
          collections: ['Photography', 'Travel', 'Equipment'].map((name, index) => ({ id: `collection-${index}`, organization_id: 'org', name, icon: 'i-lucide-library', asset_count: 1, created_at: timestamp, updated_at: timestamp }))
        })
      }
    })
    expect(wrapper.get('img').attributes('src')).toBe('https://example.test/camera.png')
    expect(wrapper.get('img').attributes('alt')).toBe('Camera')
    expect(wrapper.text()).toContain('Electronics')
    expect(wrapper.text()).toContain('Office')
    expect(wrapper.text()).toContain('Photography')
    expect(wrapper.text()).toContain('+2')
    expect(wrapper.get('[title]').attributes('title')).toBe('Photography, Travel, Equipment')
    expect(wrapper.text()).not.toContain('No collections')
  })

  it('does not show an overflow count for a single collection', async () => {
    const wrapper = await mountSuspended(AssetCard, {
      props: { asset: asset({ collections: [{ id: 'collection', organization_id: 'org', name: 'Photography', icon: 'i-lucide-library', asset_count: 1, created_at: '', updated_at: '' }] }) }
    })
    expect(wrapper.text()).toContain('Photography')
    expect(wrapper.text()).not.toContain('+')
  })

  it.each([
    [0, 'Today'],
    [1, 'Yesterday'],
    [3, '3 days ago'],
    [7, '1 week ago'],
    [14, '2 weeks ago']
  ])('shows the last update from %i days ago as %s', async (days, label) => {
    vi.spyOn(Date, 'now').mockReturnValue(now)
    const wrapper = await mountSuspended(AssetCard, {
      props: { asset: asset({ updated_at: new Date(now - days * 86400000).toISOString() }) }
    })
    expect(wrapper.text()).toContain(label)
  })
})
