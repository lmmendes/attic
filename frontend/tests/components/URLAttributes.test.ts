import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import URLAttributeInput from '../../app/components/URLAttributeInput.vue'
import URLAttributeDisplay from '../../app/components/URLAttributeDisplay.vue'

describe('URL attribute components', () => {
  it('renders escaped link text and safe new-tab attributes', async () => {
    const wrapper = await mountSuspended(URLAttributeDisplay, { props: { value: { url: 'https://paperless.local/docs/1', label: '<b>Invoice</b>' } } })
    const link = wrapper.get('a')
    expect(link.text()).toBe('<b>Invoice</b>')
    expect(link.find('b').exists()).toBe(false)
    expect(link.attributes()).toMatchObject({ href: 'https://paperless.local/docs/1', target: '_blank', rel: 'noopener noreferrer' })
    wrapper.unmount()
  })
  it('falls back to the URL and leaves malformed destinations as text', async () => {
    const wrapper = await mountSuspended(URLAttributeDisplay, { props: { value: { url: 'http://localhost:8000' } } })
    expect(wrapper.get('a').text()).toBe('http://localhost:8000')
    await wrapper.setProps({ value: { url: 'javascript:alert(1)', label: 'Invoice' } })
    expect(wrapper.find('a').exists()).toBe(false)
    expect(wrapper.text()).toBe('Invoice')
    await wrapper.setProps({ value: undefined })
    expect(wrapper.text()).toBe('-')
    wrapper.unmount()
  })
  it('edits URL and text without losing either value, then clears both', async () => {
    const wrapper = await mountSuspended(URLAttributeInput, { props: { name: 'Invoice', modelValue: { url: 'https://example.com', label: 'Invoice' } } })
    const inputs = wrapper.findAll('input')
    expect(inputs[0]!.attributes('type')).toBe('url')
    await inputs[0]!.setValue('http://localhost:8000/docs/1')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({ url: 'http://localhost:8000/docs/1', label: 'Invoice' })
    await wrapper.setProps({ modelValue: { url: '', label: 'Invoice' } })
    expect(inputs[0]!.attributes('required')).toBeDefined()
    await inputs[1]!.setValue('')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBeUndefined()
    wrapper.unmount()
  })
})
