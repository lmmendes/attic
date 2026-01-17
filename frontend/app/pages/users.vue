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

// Search
const searchQuery = ref('')

// Pagination
const currentPage = ref(1)
const itemsPerPage = ref(10)

// Filtered users
const filteredUsers = computed(() => {
  if (!users.value) return []
  if (!searchQuery.value.trim()) return users.value
  const query = searchQuery.value.toLowerCase()
  return users.value.filter(
    u => u.email.toLowerCase().includes(query)
      || (u.name && u.name.toLowerCase().includes(query))
      || u.role.toLowerCase().includes(query)
  )
})

// Paginated users
const paginatedUsers = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value
  const end = start + itemsPerPage.value
  return filteredUsers.value.slice(start, end)
})

// Total pages
const totalPages = computed(() => Math.ceil(filteredUsers.value.length / itemsPerPage.value))

// Reset to page 1 when search changes
watch(searchQuery, () => {
  currentPage.value = 1
})

// Pagination helpers
function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
  }
}

function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
  }
}

// Stats
const stats = computed(() => {
  if (!users.value) return { total: 0, admins: 0 }
  const admins = users.value.filter(u => u.role === 'admin').length
  return { total: users.value.length, admins }
})

// Modals
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
  } catch (error: unknown) {
    const err = error as { data?: { error?: string } }
    toast.add({ title: err?.data?.error || 'Failed to create user', color: 'error' })
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
  } catch (error: unknown) {
    const err = error as { data?: { error?: string } }
    toast.add({ title: err?.data?.error || 'Failed to update user', color: 'error' })
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
  } catch (error: unknown) {
    const err = error as { data?: { error?: string } }
    toast.add({ title: err?.data?.error || 'Failed to reset password', color: 'error' })
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
  } catch (error: unknown) {
    const err = error as { data?: { error?: string } }
    toast.add({ title: err?.data?.error || 'Failed to delete user', color: 'error' })
  } finally {
    isLoading.value = false
  }
}

// Get user initials
function getInitials(user: User): string {
  if (user.name) {
    const parts = user.name.split(' ')
    if (parts.length >= 2) {
      const first = parts[0]?.[0] ?? ''
      const second = parts[1]?.[0] ?? ''
      if (first && second) {
        return (first + second).toUpperCase()
      }
    }
    return user.name.substring(0, 2).toUpperCase()
  }
  return user.email.substring(0, 2).toUpperCase()
}

// Get avatar color based on user
function getAvatarColor(user: User): { bg: string, text: string } {
  const colors: Array<{ bg: string, text: string }> = [
    { bg: 'bg-attic-100 dark:bg-attic-900/30', text: 'text-attic-700 dark:text-attic-300' },
    { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-300' },
    { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-700 dark:text-blue-300' },
    { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-300' },
    { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-700 dark:text-emerald-300' },
    { bg: 'bg-pink-100 dark:bg-pink-900/30', text: 'text-pink-700 dark:text-pink-300' }
  ]
  const index = user.email.charCodeAt(0) % colors.length
  return colors[index]!
}

// Get role style
function getRoleStyle(role: string): { bgColor: string, textColor: string, borderColor: string } {
  if (role === 'admin') {
    return {
      bgColor: 'bg-purple-50 dark:bg-purple-900/30',
      textColor: 'text-purple-700 dark:text-purple-300',
      borderColor: 'border-purple-100 dark:border-purple-900/50'
    }
  }
  return {
    bgColor: 'bg-mist-100 dark:bg-mist-700',
    textColor: 'text-mist-700 dark:text-mist-300',
    borderColor: 'border-mist-200 dark:border-mist-600'
  }
}

// Format relative date
function formatRelativeDate(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return 'Added today'
  if (diffDays === 1) return 'Added yesterday'
  if (diffDays < 7) return `Added ${diffDays} days ago`
  if (diffDays < 30) return `Added ${Math.floor(diffDays / 7)} week${Math.floor(diffDays / 7) > 1 ? 's' : ''} ago`
  if (diffDays < 365) return `Added ${Math.floor(diffDays / 30)} month${Math.floor(diffDays / 30) > 1 ? 's' : ''} ago`
  return `Added ${Math.floor(diffDays / 365)} year${Math.floor(diffDays / 365) > 1 ? 's' : ''} ago`
}
</script>

<template>
  <div class="space-y-8">
    <!-- Page Header -->
    <div class="flex flex-col md:flex-row md:items-end justify-between gap-6">
      <div>
        <h1 class="text-3xl md:text-4xl font-black tracking-tight text-mist-950 dark:text-white mb-1">
          User Management
        </h1>
        <p class="text-mist-500">
          Oversee user access, manage roles, and maintain security protocols for your organization.
        </p>
      </div>
      <!-- Quick Stats -->
      <div class="flex gap-4">
        <div class="bg-white dark:bg-mist-800 px-4 py-2 rounded-lg shadow-sm border border-mist-100 dark:border-mist-700 flex items-center gap-3">
          <div class="bg-attic-50 dark:bg-attic-900/20 p-1.5 rounded text-attic-500">
            <UIcon
              name="i-lucide-users"
              class="w-5 h-5"
            />
          </div>
          <div>
            <p class="text-xs text-mist-500 font-medium uppercase tracking-wider">
              Total
            </p>
            <p class="text-lg font-bold text-mist-950 dark:text-white leading-none">
              {{ stats.total }}
            </p>
          </div>
        </div>
        <div class="bg-white dark:bg-mist-800 px-4 py-2 rounded-lg shadow-sm border border-mist-100 dark:border-mist-700 flex items-center gap-3">
          <div class="bg-purple-100 dark:bg-purple-900/30 p-1.5 rounded text-purple-600 dark:text-purple-300">
            <UIcon
              name="i-lucide-shield"
              class="w-5 h-5"
            />
          </div>
          <div>
            <p class="text-xs text-mist-500 font-medium uppercase tracking-wider">
              Admins
            </p>
            <p class="text-lg font-bold text-mist-950 dark:text-white leading-none">
              {{ stats.admins }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-col md:flex-row items-center justify-between gap-4">
      <!-- Search -->
      <div class="relative w-full md:max-w-md">
        <UIcon
          name="i-lucide-search"
          class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-mist-400"
        />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search users by name, email or role..."
          class="w-full pl-10 pr-4 py-2.5 bg-mist-50 dark:bg-mist-800 border border-mist-200 dark:border-mist-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-attic-500/20 focus:border-attic-500 text-mist-950 dark:text-white placeholder-mist-400"
        >
      </div>
      <!-- Actions -->
      <UButton
        icon="i-lucide-plus"
        class="h-11 px-6 font-bold shadow-lg shadow-attic-500/20"
        @click="openCreateModal"
      >
        Add User
      </UButton>
    </div>

    <!-- Data Table -->
    <div class="overflow-hidden rounded-xl border border-mist-100 dark:border-mist-700 bg-white dark:bg-mist-800 shadow-sm">
      <!-- Loading State -->
      <div
        v-if="status === 'pending'"
        class="flex items-center justify-center py-20"
      >
        <UIcon
          name="i-lucide-loader-2"
          class="w-8 h-8 text-attic-500 animate-spin"
        />
      </div>

      <!-- Empty State -->
      <div
        v-else-if="!filteredUsers.length && !searchQuery"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-users"
            class="w-8 h-8 text-mist-400"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No users yet
        </h3>
        <p class="text-sm text-mist-500 mb-4 max-w-sm">
          Create your first user to start managing access to your organization.
        </p>
        <UButton @click="openCreateModal">
          Add User
        </UButton>
      </div>

      <!-- No Results -->
      <div
        v-else-if="!filteredUsers.length && searchQuery"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <UIcon
          name="i-lucide-search-x"
          class="w-12 h-12 text-mist-300 mb-4"
        />
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No results found
        </h3>
        <p class="text-sm text-mist-500">
          No users match "{{ searchQuery }}"
        </p>
      </div>

      <!-- Table -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[800px] border-collapse">
            <thead class="bg-mist-50/50 dark:bg-mist-700/30 border-b border-mist-100 dark:border-mist-700">
              <tr>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  User Details
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Email Address
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Role
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-mist-500">
                  Auth
                </th>
                <th class="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-mist-500">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-mist-100 dark:divide-mist-700">
              <tr
                v-for="user in paginatedUsers"
                :key="user.id"
                class="group hover:bg-mist-50/50 dark:hover:bg-mist-700/30 transition-colors"
              >
                <!-- User Details with Avatar -->
                <td class="px-6 py-4">
                  <div class="flex items-center gap-3">
                    <div
                      class="size-10 rounded-full flex items-center justify-center font-bold text-sm"
                      :class="[getAvatarColor(user).bg, getAvatarColor(user).text]"
                    >
                      {{ getInitials(user) }}
                    </div>
                    <div>
                      <p class="text-sm font-semibold text-mist-950 dark:text-white">
                        {{ user.name || user.email.split('@')[0] }}
                      </p>
                      <p class="text-xs text-mist-500">
                        {{ formatRelativeDate(user.created_at) }}
                      </p>
                    </div>
                  </div>
                </td>

                <!-- Email Address -->
                <td class="px-6 py-4">
                  <div class="flex items-center gap-2 text-sm text-mist-500">
                    <UIcon
                      name="i-lucide-mail"
                      class="w-4 h-4 opacity-70"
                    />
                    {{ user.email }}
                  </div>
                </td>

                <!-- Role Badge -->
                <td class="px-6 py-4">
                  <span
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-semibold border cursor-pointer"
                    :class="[getRoleStyle(user.role).bgColor, getRoleStyle(user.role).textColor, getRoleStyle(user.role).borderColor]"
                    @click="openEditModal(user)"
                  >
                    <UIcon
                      :name="user.role === 'admin' ? 'i-lucide-shield' : 'i-lucide-user'"
                      class="w-3.5 h-3.5"
                    />
                    {{ user.role === 'admin' ? 'Admin' : 'User' }}
                  </span>
                </td>

                <!-- Auth Methods -->
                <td class="px-6 py-4">
                  <div class="flex gap-1.5">
                    <span
                      v-if="user.has_password"
                      class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300"
                    >
                      Password
                    </span>
                    <span
                      v-if="user.has_oidc"
                      class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
                    >
                      OIDC
                    </span>
                  </div>
                </td>

                <!-- Actions -->
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      class="size-8 rounded flex items-center justify-center text-mist-400 hover:text-attic-500 hover:bg-attic-500/10 transition-colors"
                      title="Reset Password"
                      @click="openResetPasswordModal(user)"
                    >
                      <UIcon
                        name="i-lucide-key"
                        class="w-4 h-4"
                      />
                    </button>
                    <button
                      class="size-8 rounded flex items-center justify-center text-mist-400 hover:text-attic-500 hover:bg-attic-500/10 transition-colors"
                      title="Edit User"
                      @click="openEditModal(user)"
                    >
                      <UIcon
                        name="i-lucide-edit"
                        class="w-4 h-4"
                      />
                    </button>
                    <button
                      class="size-8 rounded flex items-center justify-center text-mist-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                      title="Delete User"
                      @click="openDeleteModal(user)"
                    >
                      <UIcon
                        name="i-lucide-trash-2"
                        class="w-4 h-4"
                      />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Footer with Pagination -->
        <div class="px-6 py-3 border-t border-mist-100 dark:border-mist-700 bg-mist-50/50 dark:bg-mist-700/20 flex items-center justify-between">
          <p class="text-xs text-mist-500">
            Showing {{ (currentPage - 1) * itemsPerPage + 1 }}-{{ Math.min(currentPage * itemsPerPage, filteredUsers.length) }} of {{ filteredUsers.length }} users
            <span v-if="searchQuery && users?.length !== filteredUsers.length">
              (filtered from {{ users?.length || 0 }})
            </span>
          </p>
          <div
            v-if="totalPages > 1"
            class="flex items-center gap-2"
          >
            <button
              class="px-3 py-1.5 text-xs font-medium border border-mist-200 dark:border-mist-600 rounded-lg hover:bg-mist-100 dark:hover:bg-mist-700 text-mist-600 dark:text-mist-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              :disabled="currentPage === 1"
              @click="prevPage"
            >
              Prev
            </button>
            <span class="text-xs text-mist-500 px-2">
              Page {{ currentPage }} of {{ totalPages }}
            </span>
            <button
              class="px-3 py-1.5 text-xs font-medium border border-mist-200 dark:border-mist-600 rounded-lg hover:bg-mist-100 dark:hover:bg-mist-700 text-mist-600 dark:text-mist-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              :disabled="currentPage === totalPages"
              @click="nextPage"
            >
              Next
            </button>
          </div>
        </div>
      </template>
    </div>

    <!-- Create User Modal -->
    <UModal v-model:open="isCreateModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl p-6 max-w-md">
          <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-4">
            Create User
          </h3>
          <form
            class="space-y-4"
            @submit.prevent="createUser"
          >
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Email
              </label>
              <input
                v-model="createForm.email"
                type="email"
                placeholder="user@example.com"
                required
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm text-mist-950 dark:text-white"
              >
            </div>

            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Name
              </label>
              <input
                v-model="createForm.name"
                type="text"
                placeholder="John Doe"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm text-mist-950 dark:text-white"
              >
            </div>

            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Password
              </label>
              <input
                v-model="createForm.password"
                type="password"
                placeholder="Minimum 8 characters"
                required
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm text-mist-950 dark:text-white"
              >
            </div>

            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Role
              </label>
              <select
                v-model="createForm.role"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all text-sm text-mist-950 dark:text-white"
              >
                <option
                  v-for="option in roleOptions"
                  :key="option.value"
                  :value="option.value"
                >
                  {{ option.label }}
                </option>
              </select>
            </div>
          </form>
          <div class="flex justify-end gap-3 mt-6">
            <UButton
              variant="ghost"
              color="neutral"
              @click="isCreateModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              :loading="isLoading"
              @click="createUser"
            >
              Create User
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Edit User Modal -->
    <UModal v-model:open="isEditModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl p-6 max-w-md">
          <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-4">
            Edit User
          </h3>
          <form
            class="space-y-4"
            @submit.prevent="updateUser"
          >
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Email
              </label>
              <input
                v-model="editForm.email"
                type="email"
                placeholder="user@example.com"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm text-mist-950 dark:text-white"
              >
            </div>

            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Name
              </label>
              <input
                v-model="editForm.name"
                type="text"
                placeholder="John Doe"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm text-mist-950 dark:text-white"
              >
            </div>

            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                Role
              </label>
              <select
                v-model="editForm.role"
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all text-sm text-mist-950 dark:text-white"
              >
                <option
                  v-for="option in roleOptions"
                  :key="option.value"
                  :value="option.value"
                >
                  {{ option.label }}
                </option>
              </select>
            </div>
          </form>
          <div class="flex justify-end gap-3 mt-6">
            <UButton
              variant="ghost"
              color="neutral"
              @click="isEditModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              :loading="isLoading"
              @click="updateUser"
            >
              Save Changes
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Reset Password Modal -->
    <UModal v-model:open="isResetPasswordModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl p-6 max-w-md">
          <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
            Reset Password
          </h3>
          <p class="text-sm text-mist-500 mb-4">
            Set a new password for <strong class="text-mist-700 dark:text-mist-300">{{ selectedUser?.email }}</strong>
          </p>
          <form
            class="space-y-4"
            @submit.prevent="resetPassword"
          >
            <div>
              <label class="block text-sm font-semibold text-mist-700 dark:text-mist-300 mb-2">
                New Password
              </label>
              <input
                v-model="resetPasswordForm.password"
                type="password"
                placeholder="Minimum 8 characters"
                required
                class="w-full px-4 py-3 rounded-lg bg-mist-50 dark:bg-mist-900 border border-mist-200 dark:border-mist-600 focus:border-attic-500 focus:ring-1 focus:ring-attic-500 outline-none transition-all placeholder:text-mist-400 text-sm text-mist-950 dark:text-white"
              >
            </div>
          </form>
          <div class="flex justify-end gap-3 mt-6">
            <UButton
              variant="ghost"
              color="neutral"
              @click="isResetPasswordModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              :loading="isLoading"
              @click="resetPassword"
            >
              Reset Password
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Delete User Modal -->
    <UModal v-model:open="isDeleteModalOpen">
      <template #content>
        <div class="bg-white dark:bg-mist-800 rounded-xl shadow-xl p-6 max-w-md">
          <div class="flex items-start gap-4">
            <div class="p-3 bg-red-100 dark:bg-red-900/30 rounded-full">
              <UIcon
                name="i-lucide-alert-triangle"
                class="w-6 h-6 text-red-600 dark:text-red-400"
              />
            </div>
            <div class="flex-1">
              <h3 class="text-lg font-bold text-mist-950 dark:text-white">
                Delete User
              </h3>
              <p class="text-sm text-mist-500 mt-2">
                Are you sure you want to delete <strong class="text-mist-700 dark:text-mist-300">{{ selectedUser?.email }}</strong>? This action cannot be undone.
              </p>
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <UButton
              variant="ghost"
              color="neutral"
              @click="isDeleteModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              color="error"
              :loading="isLoading"
              @click="deleteUser"
            >
              Delete User
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
