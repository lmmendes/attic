import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, ref } from 'vue'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { within, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import SelectAttributeInput from '../../app/components/SelectAttributeInput.vue'
import AttributeOptionsEditor from '../../app/components/AttributeOptionsEditor.vue'
import type { Attribute, AttributeOption } from '../../app/types/api'

const cleanup: Array<() => void> = []
afterEach(() => {
  for (const fn of cleanup.splice(0)) fn()
})
const field: Attribute = {
  id: 'field', organization_id: 'org', name: 'Platform', key: 'platform', data_type: 'select',
  selection_mode: 'single', created_at: '', updated_at: '',
  options: [
    { id: 'linux', label: 'Linux', value: 'linux', sort_order: 0 },
    { id: 'mac', label: 'macOS', value: 'macos', sort_order: 1 }
  ]
}
async function mountInput(mode: 'single' | 'multiple', initial?: string | string[]) {
  const harness = defineComponent({
    components: { SelectAttributeInput },
    setup() {
      const value = ref(initial)
      return { value, field: { ...field, selection_mode: mode } }
    },
    template: '<div><SelectAttributeInput v-model="value" :attribute="field" /><output aria-label="Saved selection">{{ JSON.stringify(value) }}</output></div>'
  })
  const wrapper = await mountSuspended(harness, { attachTo: document.body })
  cleanup.push(() => wrapper.unmount())
}

describe('select field inputs', () => {
  it('selects a label and stores its canonical string', async () => {
    await mountInput('single')
    const page = within(document.body)
    const user = userEvent.setup()
    await user.click(page.getByRole('button', { name: 'Platform' }))
    await user.click(await page.findByRole('option', { name: 'Linux' }))
    await waitFor(() => expect(page.getByLabelText('Saved selection').textContent).toBe('"linux"'))
    await user.click(page.getByRole('button', { name: 'Clear selection' }))
    await waitFor(() => expect(page.getByLabelText('Saved selection').textContent).toBe(''))
  })

  it('removes a chip without losing other selected values', async () => {
    await mountInput('multiple', ['linux', 'macos'])
    const page = within(document.body)
    await userEvent.setup().click(page.getByRole('button', { name: 'Remove Linux' }))
    await waitFor(() => expect(page.getByLabelText('Saved selection').textContent).toBe('["macos"]'))
    expect(page.getByRole('button', { name: 'Remove macOS' })).toBeTruthy()
  })

  it('adds options with generated canonical values before field creation', async () => {
    const harness = defineComponent({
      components: { AttributeOptionsEditor },
      setup() {
        const options = ref<AttributeOption[]>([])
        return { options }
      },
      template: '<div><AttributeOptionsEditor v-model="options" /><output aria-label="Configured values">{{ options.map(o => o.value).join(",") }}</output></div>'
    })
    const wrapper = await mountSuspended(harness, { attachTo: document.body })
    cleanup.push(() => wrapper.unmount())
    const page = within(document.body)
    const user = userEvent.setup()
    await user.type(page.getByRole('textbox', { name: 'New option label' }), 'Commodore International')
    await user.click(page.getByRole('button', { name: 'Add option' }))
    await waitFor(() => expect(page.getByLabelText('Configured values').textContent).toBe('commodore_international'))
    expect(page.queryByRole('button', { name: 'Save option' })).toBeNull()

    const storedValue = page.getByRole('textbox', { name: 'Stored value for Commodore International' })
    await user.clear(storedValue)
    await user.type(storedValue, 'commodore')
    await waitFor(() => expect(page.getByLabelText('Configured values').textContent).toBe('commodore'))

    const label = page.getByRole('textbox', { name: 'Label for Commodore International' })
    await user.clear(label)
    await user.type(label, 'Commodore')
    await user.click(page.getByRole('button', { name: 'Delete Commodore', exact: true }))
    await waitFor(() => expect(page.getByLabelText('Configured values').textContent).toBe(''))
    expect(page.queryByRole('dialog')).toBeNull()
  })

  it('keeps edits to existing options in the attribute draft', async () => {
    const harness = defineComponent({
      components: { AttributeOptionsEditor },
      setup() {
        const options = ref<AttributeOption[]>([
          { id: 'existing', label: 'Original', value: 'original', sort_order: 0 }
        ])
        return { options }
      },
      template: '<div><AttributeOptionsEditor v-model="options" /><output aria-label="Option draft">{{ JSON.stringify(options) }}</output></div>'
    })
    const wrapper = await mountSuspended(harness, { attachTo: document.body })
    cleanup.push(() => wrapper.unmount())
    const page = within(document.body)
    const user = userEvent.setup()
    const label = page.getByRole('textbox', { name: 'Label for Original' })
    await user.clear(label)
    await user.type(label, 'Renamed')
    await waitFor(() => expect(page.getByLabelText('Option draft').textContent).toContain('Renamed'))
    expect(page.queryByRole('button', { name: 'Save option' })).toBeNull()
    await user.click(page.getByRole('button', { name: 'Delete Renamed' }))
    await waitFor(() => expect(page.getByLabelText('Option draft').textContent).toBe('[]'))
    expect(page.queryByRole('dialog')).toBeNull()
  })
})
