import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { defineComponent, h, ref } from 'vue'
import AssetTagsField from '../../app/components/AssetTagsField.vue'

const inputMenu = {
  props: ['modelValue', 'items'],
  emits: ['update:modelValue', 'create'],
  template: '<button aria-label="Create tag" @click="$emit(\'create\', \'retro\')">Create</button>'
}

describe('asset tag field', () => {
  it('reuses an existing tag case-insensitively when creating a chip', async () => {
    const value = ref<string[]>([])
    const wrapper = await mountSuspended(defineComponent({
      setup: () => () => h(AssetTagsField, {
        'modelValue': value.value,
        'tags': [{ id: 'one', organization_id: 'org', name: 'Retro', asset_count: 0, created_at: '', updated_at: '' }],
        'onUpdate:modelValue': (next) => { value.value = next }
      })
    }), { global: { stubs: { UInputMenu: inputMenu } } })
    await wrapper.get('button[aria-label="Tags"]').trigger('click')
    expect(value.value).toEqual(['Retro'])
    wrapper.unmount()
  })
})
