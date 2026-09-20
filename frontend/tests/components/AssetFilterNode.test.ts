import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { defineComponent, h, ref } from 'vue'
import AssetFilterNode from '../../app/components/AssetFilterNode.vue'
import type { FilterNode } from '../../app/types/api'

const select = {
  props: ['modelValue', 'items', 'multiple'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" :multiple="multiple" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="item in items" :key="item.value" :value="item.value">{{ item.label }}</option></select>'
}

async function setup(initial: FilterNode) {
  const node = ref(initial)
  const wrapper = await mountSuspended(defineComponent({
    setup() {
      return () => h(AssetFilterNode, {
        'modelValue': node.value,
        'onUpdate:modelValue': (value: FilterNode) => { node.value = value },
        'attributes': [{ id: 'vendor', name: 'Vendor', key: 'vendor', data_type: 'string', organization_id: '', created_at: '', updated_at: '' }],
        'options': { collections: [{ label: 'Computers', value: 'one' }, { label: 'Favorites', value: 'two' }] },
        'features': { attributes: true, categories: true, collections: true, locations: true, conditions: true, plugins: true, warranties: true },
        'ruleCount': 1
      })
    }
  }), { global: { stubs: { USelect: select, USelectMenu: select } } })
  return { node, wrapper }
}

describe('filter rule editing', () => {
  it('preserves selected collections when switching between any and all', async () => {
    const { node, wrapper } = await setup({ kind: 'rule', field: 'collections', operator: 'any', values: ['one', 'two'] })
    await wrapper.get('select[aria-label="Rule comparison"]').setValue('all')
    expect(node.value).toMatchObject({ operator: 'all', values: ['one', 'two'] })
    wrapper.unmount()
  })

  it('removes scalar values from boolean empty checks', async () => {
    const { node, wrapper } = await setup({ kind: 'rule', field: 'attribute', attribute_id: 'boolean', data_type: 'boolean', operator: 'eq', value: false })
    await wrapper.get('select[aria-label="Rule comparison"]').setValue('empty')
    expect(node.value).toMatchObject({ operator: 'empty', value: undefined })
    expect(wrapper.find('select[aria-label="Rule value"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('does not reset a value when the same field is selected again', async () => {
    const { node, wrapper } = await setup({ kind: 'rule', field: 'attribute', attribute_id: 'vendor', data_type: 'string', operator: 'eq', value: 'Commodore' })
    await wrapper.get('select[aria-label="Rule field"]').setValue('attribute:vendor')
    expect(node.value).toMatchObject({ value: 'Commodore' })
    await wrapper.get('input[aria-label="Rule value"]').setValue('Amiga')
    expect(node.value).toMatchObject({ value: 'Amiga' })
    wrapper.unmount()
  })

  it('does not replace a stale rule with an unavailable field value', async () => {
    const stale: FilterNode = { kind: 'rule', field: 'attribute', attribute_id: 'missing', data_type: 'string', operator: 'eq', value: 'Commodore' }
    const { node, wrapper } = await setup(stale)
    await wrapper.get('select[aria-label="Rule field"]').setValue('attribute:missing')
    expect(node.value).toEqual(stale)
    wrapper.unmount()
  })
})
