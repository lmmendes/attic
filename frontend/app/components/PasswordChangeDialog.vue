<script setup lang="ts">
const open = defineModel<boolean>('open', { default: false })
const { changePassword } = useAuth()
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref('')
const success = ref(false)
const busy = ref(false)

watch(open, () => {
  currentPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  error.value = ''
  success.value = false
})

async function submit() {
  if (busy.value) return
  error.value = ''
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'New passwords do not match'
    return
  }
  if (newPassword.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }
  busy.value = true
  try {
    const result = await changePassword(currentPassword.value, newPassword.value)
    if (!result.success) {
      error.value = result.error || 'Could not change your password. Please try again.'
      return
    }
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    success.value = true
  } catch {
    error.value = 'Could not change your password. Please try again.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Change password"
    description="Enter your current password and choose a new password."
    :dismissible="!busy"
    :close="!busy"
  >
    <template #body>
      <form
        id="profile-password-form"
        class="space-y-5"
        @submit.prevent="submit"
      >
        <UAlert
          v-if="error"
          color="error"
          :title="error"
          icon="i-lucide-alert-circle"
        />
        <UAlert
          v-if="success"
          color="success"
          title="Password changed successfully"
          description="Use your new password the next time you sign in."
          icon="i-lucide-shield-check"
        />
        <template v-else>
          <UFormField
            label="Current password"
            name="currentPassword"
          >
            <UInput
              v-model="currentPassword"
              type="password"
              autocomplete="current-password"
              class="w-full"
              size="lg"
              required
              :disabled="busy"
            />
          </UFormField>
          <UFormField
            label="New password"
            name="newPassword"
            help="Use at least 8 characters."
          >
            <UInput
              v-model="newPassword"
              type="password"
              autocomplete="new-password"
              class="w-full"
              size="lg"
              required
              :disabled="busy"
            />
          </UFormField>
          <UFormField
            label="Confirm new password"
            name="confirmPassword"
          >
            <UInput
              v-model="confirmPassword"
              type="password"
              autocomplete="new-password"
              class="w-full"
              size="lg"
              required
              :disabled="busy"
            />
          </UFormField>
        </template>
      </form>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton
          variant="ghost"
          color="neutral"
          :disabled="busy"
          @click="open = false"
        >
          {{ success ? 'Done' : 'Cancel' }}
        </UButton>
        <UButton
          v-if="!success"
          type="submit"
          form="profile-password-form"
          :loading="busy"
          :disabled="busy || !currentPassword || !newPassword || !confirmPassword"
        >
          Update password
        </UButton>
      </div>
    </template>
  </UModal>
</template>
