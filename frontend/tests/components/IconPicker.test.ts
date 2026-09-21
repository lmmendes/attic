import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, ref } from 'vue'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { waitFor, within } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import IconPicker from '../../app/components/IconPicker.vue'

const cleanup: Array<() => void> = []
afterEach(() => {
  for (const fn of cleanup.splice(0)) fn()
})

async function mountPicker(initial = 'i-lucide-tag') {
  const harness = defineComponent({
    components: { IconPicker },
    setup() {
      const icon = ref<string | undefined>(initial)
      return { icon }
    },
    template: '<div><IconPicker v-model="icon" /><output aria-label="Selected icon">{{ icon }}</output></div>'
  })
  const wrapper = await mountSuspended(harness, { attachTo: document.body })
  cleanup.push(() => wrapper.unmount())
}

describe('IconPicker', () => {
  it('browses the catalog in pages and selects an icon found by search', async () => {
    await mountPicker()
    const page = within(document.body)
    const user = userEvent.setup()

    expect(document.querySelectorAll('button[aria-pressed]')).toHaveLength(60)
    expect(page.getByText(/Showing 60 of \d+ icons/)).toBeTruthy()

    await user.click(page.getByRole('button', { name: 'Show more' }))
    expect(document.querySelectorAll('button[aria-pressed]')).toHaveLength(120)

    await user.type(page.getByRole('textbox', { name: 'Search icons' }), 'telescope')
    const telescope = page.getByRole('button', { name: 'Telescope icon' })
    await user.click(telescope)

    await waitFor(() => expect(page.getByLabelText('Selected icon').textContent).toBe('i-lucide-telescope'))
    expect(telescope.getAttribute('aria-pressed')).toBe('true')
    expect(page.getByText('Selected').nextElementSibling?.textContent).toBe('Telescope')
  })

  it('matches legacy aliases without displaying duplicate alias entries', async () => {
    await mountPicker()
    const page = within(document.body)

    await userEvent.setup().type(page.getByRole('textbox', { name: 'Search icons' }), 'alert circle')

    expect(page.getByRole('button', { name: 'Circle Alert icon' })).toBeTruthy()
    expect(page.queryByRole('button', { name: 'Alert Circle icon' })).toBeNull()
  })

  it('reports an empty search result', async () => {
    await mountPicker()
    const page = within(document.body)

    await userEvent.setup().type(page.getByRole('textbox', { name: 'Search icons' }), 'no-such-lucide-icon')

    expect(page.getByText(/No icons match/)).toBeTruthy()
    expect(page.getByText('Showing 0 of 0 icons')).toBeTruthy()
  })
})
