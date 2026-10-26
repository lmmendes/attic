import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import PasswordChangeDialog from '../../app/components/PasswordChangeDialog.vue'

const { changePassword } = vi.hoisted(() => ({ changePassword: vi.fn() }))
mockNuxtImport('useAuth', () => () => ({ changePassword }))

async function mountDialog() {
  return mountSuspended(PasswordChangeDialog, {
    props: { open: true },
    global: { stubs: { UModal: { template: '<section><slot name="body" /><slot name="footer" /></section>' } } }
  })
}

beforeEach(() => changePassword.mockReset())

describe('password change dialog', () => {
  it('rejects mismatched and short passwords before calling the API', async () => {
    const wrapper = await mountDialog()
    const inputs = wrapper.findAll('input')
    await inputs[0]!.setValue('current-password')
    await inputs[1]!.setValue('new-password')
    await inputs[2]!.setValue('different-password')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.text()).toContain('New passwords do not match')
    await inputs[1]!.setValue('short')
    await inputs[2]!.setValue('short')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.text()).toContain('Password must be at least 8 characters')
    expect(changePassword).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('shows server errors and clears sensitive fields after a successful retry', async () => {
    changePassword.mockResolvedValueOnce({ success: false, error: 'Current password is incorrect' }).mockResolvedValueOnce({ success: true })
    const wrapper = await mountDialog()
    const inputs = wrapper.findAll('input')
    await inputs[0]!.setValue('current-password')
    await inputs[1]!.setValue('new-password')
    await inputs[2]!.setValue('new-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('Current password is incorrect')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(changePassword).toHaveBeenLastCalledWith('current-password', 'new-password')
    expect(wrapper.text()).toContain('Password changed successfully')
    expect(wrapper.findAll('input')).toHaveLength(0)
    await wrapper.setProps({ open: false })
    await wrapper.setProps({ open: true })
    expect(wrapper.findAll('input').every(input => input.element.value === '')).toBe(true)
    expect(wrapper.text()).not.toContain('Password changed successfully')
    wrapper.unmount()
  })

  it('discards an unfinished form when the dialog is closed', async () => {
    const wrapper = await mountDialog()
    await wrapper.findAll('input')[0]!.setValue('do-not-retain')
    await wrapper.setProps({ open: false })
    await wrapper.setProps({ open: true })
    expect(wrapper.findAll('input').every(input => input.element.value === '')).toBe(true)
    wrapper.unmount()
  })
})
