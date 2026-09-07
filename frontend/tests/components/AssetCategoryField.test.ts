import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { Category } from '../../app/types/api'
import AssetCategoryField from '../../app/components/AssetCategoryField.vue'

function category(id: string, name: string, parentId?: string): Category {
  return {
    id,
    organization_id: 'org',
    name,
    parent_id: parentId,
    created_at: '',
    updated_at: '',
    attributes: []
  }
}

describe('AssetCategoryField', () => {
  it('explains a selected category path and its inherited fields', async () => {
    const electronics = category('electronics', 'Electronics')
    const computers = category('computers', 'Computers', 'electronics')
    const laptops = category('laptops', 'Laptops', 'computers')
    const selected = {
      ...laptops,
      attributes: [
        { id: 'own', category_id: 'laptops', attribute_id: 'screen', required: false, sort_order: 0, created_at: '' },
        { id: 'parent', category_id: 'electronics', attribute_id: 'brand', required: true, sort_order: 1, created_at: '', inherited: true }
      ]
    }

    const wrapper = await mountSuspended(AssetCategoryField, {
      props: {
        modelValue: 'laptops',
        categories: [electronics, computers, laptops],
        selectedCategory: selected
      }
    })

    expect(wrapper.text()).toContain('Electronics / Computers')
    expect(wrapper.text()).toContain('Laptops')
    expect(wrapper.text()).toContain('1 own')
    expect(wrapper.text()).toContain('1 inherited')
    expect(wrapper.text()).toContain('2 fields to complete')
  })
})
