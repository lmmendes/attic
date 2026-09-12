import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, ref } from 'vue'
import { mountSuspended, registerEndpoint } from '@nuxt/test-utils/runtime'
import { within, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import AttributeImpactModal from '../../app/components/AttributeImpactModal.vue'

const cleanup: Array<() => void> = []
afterEach(() => {
  for (const fn of cleanup.splice(0).reverse()) fn()
})

async function mountFlow(action: 'delete_option' | 'update_option' = 'delete_option') {
  const harness = defineComponent({
    components: { AttributeImpactModal },
    setup() {
      const modal = ref<InstanceType<typeof AttributeImpactModal>>()
      const completed = ref(false)
      function start() {
        modal.value?.run({
          attributeId: 'field', optionId: 'option', action,
          ...(action === 'update_option' ? { changes: { label: 'New label' } } : {})
        }, () => { completed.value = true })
      }
      return { modal, start, completed }
    },
    template: '<div><button @click="start">Start change</button><span v-if="completed">Saved successfully</span><AttributeImpactModal ref="modal" /></div>'
  })
  const wrapper = await mountSuspended(harness, { attachTo: document.body })
  cleanup.push(() => wrapper.unmount())
}
async function mountBatchDeletion() {
  const harness = defineComponent({
    components: { AttributeImpactModal },
    setup() {
      const modal = ref<InstanceType<typeof AttributeImpactModal>>()
      function start() {
        modal.value?.run({ attributeId: 'field', action: 'update_attribute', deletedOptionCount: 1, changes: { options: [] } }, () => {})
      }
      return { modal, start }
    },
    template: '<div><button @click="start">Save attribute</button><AttributeImpactModal ref="modal" /></div>'
  })
  const wrapper = await mountSuspended(harness, { attachTo: document.body })
  cleanup.push(() => wrapper.unmount())
}
function preview(count: number, token = 'preview-token', requiredFieldsLeftEmpty = 0) {
  return { affected_assets: count, deleted_assets: 0, required_fields_left_empty: requiredFieldsLeftEmpty, current: 'Linux', proposed: 'New label', confirmation_token: token }
}

describe('attribute impact confirmation', () => {
  it('shows the affected count and never deletes when canceled', async () => {
    let writes = 0
    cleanup.push(registerEndpoint('/api/attributes/field/impact-preview', { method: 'POST', handler: () => preview(37) }))
    cleanup.push(registerEndpoint('/api/attributes/field/options/option', {
      method: 'DELETE',
      handler: () => {
        writes++
        return {}
      }
    }))
    await mountFlow()
    const user = userEvent.setup()
    const page = within(document.body)
    await user.click(page.getByRole('button', { name: 'Start change' }))
    await waitFor(() => expect(page.getByText('This option is used by 37 assets and will be removed from all of them.')).toBeTruthy())
    expect(writes).toBe(0)
    await user.click(page.getByRole('button', { name: 'Cancel' }))
    expect(writes).toBe(0)
  })

  it('refreshes stale counts and requires a second explicit confirmation', async () => {
    let previews = 0
    let writes = 0
    cleanup.push(registerEndpoint('/api/attributes/field/impact-preview', {
      method: 'POST',
      handler: () => {
        previews++
        return preview(previews === 1 ? 37 : 38, 'token-' + previews)
      }
    }))
    cleanup.push(registerEndpoint('/api/attributes/field/options/option', {
      method: 'DELETE',
      handler: () => {
        writes++
        if (writes === 1) return new Response(JSON.stringify({ code: 'impact_preview_required' }), { status: 409, headers: { 'Content-Type': 'application/json' } })
        return {}
      }
    }))
    await mountFlow()
    const user = userEvent.setup()
    const page = within(document.body)
    await user.click(page.getByRole('button', { name: 'Start change' }))
    await waitFor(() => expect(page.getByText(/used by 37 assets/)).toBeTruthy())
    await user.click(page.getByRole('button', { name: 'Delete option', exact: true }))
    await waitFor(() => expect(page.getByText(/used by 38 assets/)).toBeTruthy())
    expect(writes).toBe(1)
    await user.click(page.getByRole('button', { name: 'Delete option', exact: true }))
    await waitFor(() => expect(page.getByText('Saved successfully')).toBeTruthy())
    expect(writes).toBe(2)
  })

  it('shows rename wording and waits for confirmation', async () => {
    let writes = 0
    cleanup.push(registerEndpoint('/api/attributes/field/impact-preview', { method: 'POST', handler: () => preview(1) }))
    cleanup.push(registerEndpoint('/api/attributes/field/options/option', {
      method: 'PATCH',
      handler: () => {
        writes++
        return {}
      }
    }))
    await mountFlow('update_option')
    const page = within(document.body)
    const user = userEvent.setup()
    await user.click(page.getByRole('button', { name: 'Start change' }))
    await waitFor(() => expect(page.getByText('This option is used by 1 asset and will be displayed as “New label” in it.')).toBeTruthy())
    expect(writes).toBe(0)
    await user.click(page.getByRole('button', { name: 'Update option', exact: true }))
    await waitFor(() => expect(page.getByText('Saved successfully')).toBeTruthy())
    expect(writes).toBe(1)
  })

  it('confirms an unused option deletion as part of the attribute save', async () => {
    let writes = 0
    cleanup.push(registerEndpoint('/api/attributes/field/impact-preview', { method: 'POST', handler: () => preview(0) }))
    cleanup.push(registerEndpoint('/api/attributes/field', {
      method: 'PUT',
      handler: () => {
        writes++
        return {}
      }
    }))
    await mountBatchDeletion()
    const page = within(document.body)
    await userEvent.setup().click(page.getByRole('button', { name: 'Save attribute' }))
    await waitFor(() => expect(page.getByText('No assets use the options you deleted. Your other option changes will also be saved.')).toBeTruthy())
    expect(writes).toBe(0)
    await userEvent.setup().click(page.getByRole('button', { name: 'Update field', exact: true }))
    await waitFor(() => expect(writes).toBe(1))
  })

  it('explains combined option changes in plain language', async () => {
    cleanup.push(registerEndpoint('/api/attributes/field/impact-preview', { method: 'POST', handler: () => preview(1, 'preview-token', 1) }))
    await mountBatchDeletion()
    const page = within(document.body)
    await userEvent.setup().click(page.getByRole('button', { name: 'Save attribute' }))
    await waitFor(() => expect(page.getByText('1 asset uses an option you changed. Deleted selections will be removed, and renamed values will be updated automatically.')).toBeTruthy())
    expect(page.getByText('1 asset will have no selection. Because this attribute is required, it must be updated the next time it is saved.')).toBeTruthy()
    expect(page.queryByText('Any changed stored values or keys will also be updated on the affected assets.')).toBeNull()
  })
})
