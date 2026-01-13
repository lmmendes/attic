<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

interface User {
  id: string
  email: string
  name: string | null
  role: 'user' | 'admin'
  has_password: boolean
  has_oidc: boolean
  created_at: string
}

const { isAdmin } = useAuth()
const apiFetch = useApiFetch()
const toast = useToast()

// Redirect non-admins
onMounted(() => {
  if (!isAdmin.value) {
    navigateTo('/')
  }
})

const { data: users, refresh, status } = useApi<User[]>('/api/users')

const isCreateModalOpen = ref(false)
const isEditModalOpen = ref(false)
const isResetPasswordModalOpen = ref(false)
const isDeleteModalOpen = ref(false)
const selectedUser = ref<User | null>(null)
const isLoading = ref(false)

// Create user form
const createForm = ref({
  email: '',
  name: '',
  password: '',
  role: 'user' as 'user' | 'admin'
})

// Edit user form
const editForm = ref({
  email: '',
  name: '',
  role: 'user' as 'user' | 'admin'
})

// Reset password form
const resetPasswordForm = ref({
  password: ''
})

const roleOptions = [
  { label: 'User', value: 'user' },
  { label: 'Admin', value: 'admin' }
]

const columns = [
  { accessorKey: 'email', id: 'email', header: 'Email' },
  { accessorKey: 'name', id: 'name', header: 'Name' },
  { accessorKey: 'role', id: 'role', header: 'Role' },
  { id: 'auth', header: 'Auth' },
  { accessorKey: 'created_at', id: 'created_at', header: 'Created' },
  { id: 'actions', header: '' }
]

const openCreateModal = () => {
  createForm.value = { email: '', name: '', password: '', role: 'user' }
  isCreateModalOpen.value = true
}

const openEditModal = (user: User) => {
  selectedUser.value = user
  editForm.value = {
    email: user.email,
    name: user.name || '',
    role: user.role
  }
  isEditModalOpen.value = true
}

const openResetPasswordModal = (user: User) => {
  selectedUser.value = user
  resetPasswordForm.value = { password: '' }
  isResetPasswordModalOpen.value = true
}

const openDeleteModal = (user: User) => {
  selectedUser.value = user
  isDeleteModalOpen.value = true
}

const createUser = async () => {
  isLoading.value = true
  try {
    await apiFetch('/api/users', {
      method: 'POST',
      body: JSON.stringify(createForm.value)
    })
    toast.add({ title: 'User created successfully', color: 'success' })
    isCreateModalOpen.value = false
    refresh()
  } catch (error: any) {
    toast.add({ title: error?.data?.error || 'Failed to create user', color: 'error' })
  } finally {
    isLoading.value = false
  }
}

const updateUser = async () => {
  if (!selectedUser.value) return
  isLoading.value = true
  try {
    await apiFetch(`/api/users/${selectedUser.value.id}`, {
      method: 'PUT',
      body: JSON.stringify(editForm.value)
    })
    toast.add({ title: 'User updated successfully', color: 'success' })
    isEditModalOpen.value = false
    refresh()
  } catch (error: any) {
    toast.add({ title: error?.data?.error || 'Failed to update user', color: 'error' })
  } finally {
    isLoading.value = false
  }
}

const resetPassword = async () => {
  if (!selectedUser.value) return
  isLoading.value = true
  try {
    await apiFetch(`/api/users/${selectedUser.value.id}/reset-password`, {
      method: 'POST',
      body: JSON.stringify(resetPasswordForm.value)
    })
    toast.add({ title: 'Password reset successfully', color: 'success' })
    isResetPasswordModalOpen.value = false
  } catch (error: any) {
    toast.add({ title: error?.data?.error || 'Failed to reset password', color: 'error' })
  } finally {
    isLoading.value = false
  }
}

const deleteUser = async () => {
  if (!selectedUser.value) return
  isLoading.value = true
  try {
    await apiFetch(`/api/users/${selectedUser.value.id}`, {
      method: 'DELETE'
    })
    toast.add({ title: 'User deleted successfully', color: 'success' })
    isDeleteModalOpen.value = false
    refresh()
  } catch (error: any) {
    toast.add({ title: error?.data?.error || 'Failed to delete user', color: 'error' })
  } finally {
    isLoading.value = false
  }
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleDateString()
}
</script>

<template>
  <UContainer class="py-8">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold">Users</h1>
        <p class="text-gray-500 dark:text-gray-400">Manage user accounts and permissions</p>
      </div>
      <UButton
        icon="i-lucide-plus"
        @click="openCreateModal"
      >
        Add User
      </UButton>
    </div>

    <UCard>
      <UTable
        :data="users || []"
        :columns="columns"
        :loading="status === 'pending'"
      >
        <template #email-cell="{ row }">
          <span class="font-medium">{{ row.original.email }}</span>
        </template>

        <template #name-cell="{ row }">
          {{ row.original.name || '-' }}
        </template>

        <template #role-cell="{ row }">
          <UBadge
            :color="row.original.role === 'admin' ? 'primary' : 'neutral'"
            variant="subtle"
          >
            {{ row.original.role }}
          </UBadge>
        </template>

        <template #auth-cell="{ row }">
          <div class="flex gap-1">
            <UBadge v-if="row.original.has_password" color="success" variant="subtle" size="xs">
              Password
            </UBadge>
            <UBadge v-if="row.original.has_oidc" color="info" variant="subtle" size="xs">
              OIDC
            </UBadge>
          </div>
        </template>

        <template #created_at-cell="{ row }">
          {{ formatDate(row.original.created_at) }}
        </template>

        <template #actions-cell="{ row }">
          <div class="flex gap-1 justify-end">
            <UButton
              icon="i-lucide-key"
              color="neutral"
              variant="ghost"
              size="xs"
              @click="openResetPasswordModal(row.original)"
            />
            <UButton
              icon="i-lucide-pencil"
              color="neutral"
              variant="ghost"
              size="xs"
              @click="openEditModal(row.original)"
            />
            <UButton
              icon="i-lucide-trash-2"
              color="error"
              variant="ghost"
              size="xs"
              @click="openDeleteModal(row.original)"
            />
          </div>
        </template>
      </UTable>
    </UCard>

    <!-- Create User Modal -->
    <UModal v-model:open="isCreateModalOpen">
      <template #body>
        <h3 class="text-lg font-semibold mb-4">Create User</h3>
        <form @submit.prevent="createUser" class="space-y-4">
          <UFormField label="Email" name="email" required>
            <UInput v-model="createForm.email" placeholder="user@example.com" />
          </UFormField>

          <UFormField label="Name" name="name">
            <UInput v-model="createForm.name" placeholder="John Doe" />
          </UFormField>

          <UFormField label="Password" name="password" required>
            <UInput v-model="createForm.password" type="password" placeholder="Minimum 8 characters" />
          </UFormField>

          <UFormField label="Role" name="role">
            <USelectMenu v-model="createForm.role" :items="roleOptions" value-key="value" />
          </UFormField>
        </form>
      </template>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="isCreateModalOpen = false">
            Cancel
          </UButton>
          <UButton :loading="isLoading" @click="createUser">
            Create User
          </UButton>
        </div>
      </template>
    </UModal>

    <!-- Edit User Modal -->
    <UModal v-model:open="isEditModalOpen">
      <template #body>
        <h3 class="text-lg font-semibold mb-4">Edit User</h3>
        <form @submit.prevent="updateUser" class="space-y-4">
          <UFormField label="Email" name="email">
            <UInput v-model="editForm.email" placeholder="user@example.com" />
          </UFormField>

          <UFormField label="Name" name="name">
            <UInput v-model="editForm.name" placeholder="John Doe" />
          </UFormField>

          <UFormField label="Role" name="role">
            <USelectMenu v-model="editForm.role" :items="roleOptions" value-key="value" />
          </UFormField>
        </form>
      </template>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="isEditModalOpen = false">
            Cancel
          </UButton>
          <UButton :loading="isLoading" @click="updateUser">
            Save Changes
          </UButton>
        </div>
      </template>
    </UModal>

    <!-- Reset Password Modal -->
    <UModal v-model:open="isResetPasswordModalOpen">
      <template #body>
        <h3 class="text-lg font-semibold mb-4">Reset Password</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
          Set a new password for <strong>{{ selectedUser?.email }}</strong>
        </p>
        <form @submit.prevent="resetPassword" class="space-y-4">
          <UFormField label="New Password" name="password" required>
            <UInput v-model="resetPasswordForm.password" type="password" placeholder="Minimum 8 characters" />
          </UFormField>
        </form>
      </template>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="isResetPasswordModalOpen = false">
            Cancel
          </UButton>
          <UButton :loading="isLoading" @click="resetPassword">
            Reset Password
          </UButton>
        </div>
      </template>
    </UModal>

    <!-- Delete User Modal -->
    <UModal v-model:open="isDeleteModalOpen">
      <template #body>
        <h3 class="text-lg font-semibold mb-4">Delete User</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Are you sure you want to delete <strong>{{ selectedUser?.email }}</strong>? This action cannot be undone.
        </p>
      </template>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="isDeleteModalOpen = false">
            Cancel
          </UButton>
          <UButton color="error" :loading="isLoading" @click="deleteUser">
            Delete User
          </UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>
