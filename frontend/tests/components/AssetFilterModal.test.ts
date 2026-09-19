import { describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { defineComponent, h, ref } from 'vue'
import AssetFilterModal from '../../app/components/AssetFilterModal.vue'
import type { FilterCriteria, OrganizationFeatures } from '../../app/types/api'

const features: OrganizationFeatures = { attributes: true, categories: true, collections: true, locations: true, conditions: true, plugins: true, warranties: true }
const modal = { props: ['open', 'title'], template: '<div v-if="open"><h1>{{ title }}</h1><slot name="body" /><slot name="footer" /></div>' }
const input = { props: ['modelValue'], emits: ['update:modelValue'], template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)">' }

async function setup(criteria: FilterCriteria, enabled = features, mode: 'advanced' | 'edit' = 'advanced') {
  const save = vi.fn().mockResolvedValue(true)
  const apply = vi.fn()
  const wrapper = await mountSuspended(defineComponent({
    setup() {
      const editor = ref<InstanceType<typeof AssetFilterModal>>()
      return () => h('div', [
        h('button', { 'aria-label': 'Open filter', 'onClick': () => editor.value?.show(mode) }, 'Open filter'),
        h(AssetFilterModal, { ref: editor, criteria, attributes: [], options: {}, features: enabled, issues: [], message: '', busy: false, save, onApply: apply })
      ])
    }
  }), {
    global: { stubs: { UModal: modal, UInput: input } }
  })
  await wrapper.get('button[aria-label="Open filter"]').trigger('click')
  return { wrapper, save, apply }
}

describe('advanced filter drafts', () => {
  it('identifies the builder as editing when opened from a saved search action', async () => {
    const { wrapper } = await setup({ version: 1, q: 'retro' }, features, 'edit')
    expect(wrapper.get('h1').text()).toBe('Edit saved search')
    wrapper.unmount()
  })

  it('does not change applied criteria when editing or cancelling a draft', async () => {
    const criteria: FilterCriteria = { version: 1, q: 'original' }
    const { wrapper, apply } = await setup(criteria)
    await wrapper.get('button[aria-label="Remove Name / description criterion"]').trigger('click')
    expect(criteria.q).toBe('original')
    await wrapper.findAll('button').find(b => b.text() === 'Cancel')!.trigger('click')
    expect(apply).not.toHaveBeenCalled()
    await wrapper.get('button[aria-label="Open filter"]').trigger('click')
    expect(wrapper.text()).toContain('original')
    wrapper.unmount()
  })

  it('blocks unavailable top criteria and lets the user remove them before applying', async () => {
    const { wrapper, apply } = await setup({ version: 1, category_id: 'deleted', attribute_q: 'blue' }, { ...features, attributes: false })
    const clickApply = () => wrapper.findAll('button').find(b => b.text() === 'Apply')!.trigger('click')
    await clickApply()
    expect(apply).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('unavailable')
    await wrapper.get('button[aria-label="Remove Category criterion"]').trigger('click')
    await wrapper.get('button[aria-label="Remove Attribute values criterion"]').trigger('click')
    await clickApply()
    expect(apply).toHaveBeenCalledWith({ version: 1 })
    wrapper.unmount()
  })

  it('saves the complete isolated draft only on an explicit save action', async () => {
    const { wrapper, save } = await setup({ version: 1, q: 'retro', attribute_q: 'blue' })
    await wrapper.get('input[aria-label="Filter name"]').setValue('My filter')
    expect(save).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(b => b.text() === 'Save as new')!.trigger('click')
    expect(save).toHaveBeenCalledWith('My filter', { version: 1, q: 'retro', attribute_q: 'blue' }, false)
    wrapper.unmount()
  })
})
