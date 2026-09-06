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

const { isAdmin, fetchSession } = useAuth()
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
const selectedRole = ref<'all' | 'user' | 'admin'>('all')

// Pagination
const currentPage = ref(1)
const itemsPerPage = ref(10)

// Filtered users
const filteredUsers = computed(() => {
  if (!users.value) return []
  const query = searchQuery.value.trim().toLowerCase()
  return users.value.filter((user) => {
    const matchesSearch = !query
      || user.email.toLowerCase().includes(query)
      || user.name?.toLowerCase().includes(query)
    const matchesRole = selectedRole.value === 'all' || user.role === selectedRole.value
    return matchesSearch && matchesRole
  })
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
watch([searchQuery, selectedRole], () => {
  currentPage.value = 1
})

watch(totalPages, (pages) => {
  if (currentPage.value > Math.max(1, pages)) currentPage.value = Math.max(1, pages)
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
  if (!users.value) return { total: 0, admins: 0, members: 0 }
  const admins = users.value.filter(u => u.role === 'admin').length
  return { total: users.value.length, admins, members: users.value.length - admins }
})

// Modals
const isCreateModalOpen = ref(false)
const isEditModalOpen = ref(false)
const isResetPasswordModalOpen = ref(false)
const isDeleteModalOpen = ref(false)
const selectedUser = ref<User | null>(null)
const isLoading = ref(false)
const showCreatePassword = ref(false)

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
] as const

const openCreateModal = () => {
  createForm.value = { email: '', name: '', password: '', role: 'user' }
  showCreatePassword.value = false
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
  if (!createForm.value.email.trim()) {
    toast.add({ title: 'Please enter an email address', color: 'error' })
    return
  }
  if (createForm.value.password.length < 8) {
    toast.add({ title: 'Password must be at least 8 characters', color: 'error' })
    return
  }

  isLoading.value = true
  try {
    await apiFetch('/api/users', {
      method: 'POST',
      body: JSON.stringify({ ...createForm.value, email: createForm.value.email.trim() })
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
  if (!editForm.value.email.trim()) {
    toast.add({ title: 'Please enter an email address', color: 'error' })
    return
  }

  isLoading.value = true
  try {
    await apiFetch(`/api/users/${selectedUser.value.id}`, {
      method: 'PUT',
      body: JSON.stringify({ ...editForm.value, email: editForm.value.email.trim() })
    })
    toast.add({ title: 'User updated successfully', color: 'success' })
    isEditModalOpen.value = false
    await Promise.all([refresh(), fetchSession()])
  } catch (error: unknown) {
    const err = error as { data?: { error?: string } }
    toast.add({ title: err?.data?.error || 'Failed to update user', color: 'error' })
  } finally {
    isLoading.value = false
  }
}

const resetPassword = async () => {
  if (!selectedUser.value) return
  if (resetPasswordForm.value.password.length < 8) {
    toast.add({ title: 'Password must be at least 8 characters', color: 'error' })
    return
  }

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
  <div class="space-y-5 pb-6">
    <!-- Page Header -->
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="mb-1 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-500">
          Organization access
        </p>
        <h1 class="text-2xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white md:text-3xl">
          People
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          Invite members, assign administrator access, and manage sign-in methods.
        </p>
      </div>
      <!-- Quick Stats -->
      <div class="attic-panel flex divide-x divide-mist-100 rounded-xl px-2 py-2 dark:divide-mist-700">
        <div class="px-3">
          <div>
            <p class="text-[10px] font-bold uppercase tracking-wider text-muted">
              Total
            </p>
            <p class="text-lg font-bold text-mist-950 dark:text-white leading-none">
              {{ stats.total }}
            </p>
          </div>
        </div>
        <div class="px-3">
          <div>
            <p class="text-[10px] font-bold uppercase tracking-wider text-muted">
              Admins
            </p>
            <p class="text-lg font-bold text-mist-950 dark:text-white leading-none">
              {{ stats.admins }}
            </p>
          </div>
        </div>
        <div class="px-3">
          <div>
            <p class="text-[10px] font-bold uppercase tracking-wider text-muted">
              Members
            </p>
            <p class="text-lg font-bold leading-none text-mist-950 dark:text-white">
              {{ stats.members }}
            </p>
          </div>
        </div>
      </div>
    </header>

    <!-- Toolbar -->
    <div class="attic-panel flex flex-col gap-3 rounded-[18px] p-3 md:flex-row md:items-center md:justify-between md:p-4">
      <!-- Search -->
      <div class="flex w-full flex-col gap-3 sm:flex-row">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Search name or email"
          class="w-full md:max-w-sm"
          size="lg"
        />
        <div class="flex gap-1 rounded-xl bg-mist-50 p-1 dark:bg-mist-700/60">
          <button
            v-for="role in ['all', 'user', 'admin'] as const"
            :key="role"
            type="button"
            class="rounded-lg px-3 py-1.5 text-xs font-bold capitalize transition-colors"
            :class="selectedRole === role ? 'bg-white text-attic-600 shadow-sm dark:bg-mist-800 dark:text-attic-300' : 'text-muted hover:text-mist-600'"
            @click="selectedRole = role"
          >
            {{ role === 'user' ? 'Members' : role }}
          </button>
        </div>
      </div>
      <!-- Actions -->
      <UButton
        icon="i-lucide-plus"
        class="w-full shrink-0 whitespace-nowrap rounded-xl font-bold shadow-primary md:w-auto"
        @click="openCreateModal"
      >
        Add person
      </UButton>
    </div>

    <!-- Data Table -->
    <div class="attic-panel overflow-hidden rounded-[20px]">
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
        v-else-if="!users?.length"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <div class="size-16 rounded-full bg-mist-100 dark:bg-mist-700 flex items-center justify-center mb-4">
          <UIcon
            name="i-lucide-users"
            class="w-8 h-8 text-muted"
          />
        </div>
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No users yet
        </h3>
        <p class="text-sm text-muted mb-4 max-w-sm">
          Create your first user to start managing access to your organization.
        </p>
        <UButton @click="openCreateModal">
          Add User
        </UButton>
      </div>

      <!-- No Results -->
      <div
        v-else-if="!filteredUsers.length"
        class="flex flex-col items-center justify-center py-20 px-4 text-center"
      >
        <UIcon
          name="i-lucide-search-x"
          class="w-12 h-12 text-mist-300 mb-4"
        />
        <h3 class="text-lg font-bold text-mist-950 dark:text-white mb-2">
          No results found
        </h3>
        <p class="text-sm text-muted">
          No people match these filters.
        </p>
        <button
          type="button"
          class="mt-2 text-sm font-semibold text-attic-500"
          @click="searchQuery = ''; selectedRole = 'all'"
        >
          Clear filters
        </button>
      </div>

      <!-- Table -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[800px] border-collapse">
            <thead class="bg-mist-50/50 dark:bg-mist-700/30 border-b border-mist-100 dark:border-mist-700">
              <tr>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-muted">
                  User Details
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-muted">
                  Email Address
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-muted">
                  Role
                </th>
                <th class="px-6 py-4 text-left text-xs font-bold uppercase tracking-wider text-muted">
                  Auth
                </th>
                <th class="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-muted">
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
                      <p class="text-xs text-muted">
                        {{ formatRelativeDate(user.created_at) }}
                      </p>
                    </div>
                  </div>
                </td>

                <!-- Email Address -->
                <td class="px-6 py-4">
                  <div class="flex items-center gap-2 text-sm text-muted">
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
                  <div class="flex items-center justify-end gap-1">
                    <button
                      class="size-8 rounded flex items-center justify-center text-muted hover:text-attic-500 hover:bg-attic-500/10 transition-colors"
                      title="Reset Password"
                      @click="openResetPasswordModal(user)"
                    >
                      <UIcon
                        name="i-lucide-key"
                        class="w-4 h-4"
                      />
                    </button>
                    <button
                      class="size-8 rounded flex items-center justify-center text-muted hover:text-attic-500 hover:bg-attic-500/10 transition-colors"
                      title="Edit User"
                      @click="openEditModal(user)"
                    >
                      <UIcon
                        name="i-lucide-edit"
                        class="w-4 h-4"
                      />
                    </button>
                    <button
                      class="size-8 rounded flex items-center justify-center text-muted hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
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
          <p class="text-xs text-muted">
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
            <span class="text-xs text-muted px-2">
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
    <UModal
      v-model:open="isCreateModalOpen"
      title="Add a person"
      description="Create their sign-in and choose organization access."
    >
      <template #content>
        <div class="w-full max-w-lg overflow-hidden rounded-[22px] bg-white shadow-xl dark:bg-mist-800">
          <div class="flex items-center gap-4 border-b border-mist-100 bg-mist-50/70 px-6 py-5 dark:border-mist-700 dark:bg-mist-900/30">
            <div class="flex size-11 shrink-0 items-center justify-center rounded-2xl bg-attic-500 text-white shadow-primary">
              <UIcon
                name="i-lucide-user-plus"
                class="size-5"
              />
            </div>
            <div>
              <p class="text-[10px] font-extrabold uppercase tracking-[0.14em] text-attic-500">
                New account
              </p>
              <h3 class="text-lg font-extrabold text-mist-950 dark:text-white">
                Add a person
              </h3>
              <p class="text-xs text-muted">
                Create their sign-in and choose organization access.
              </p>
            </div>
          </div>

          <form
            id="create-person-form"
            class="space-y-5 p-6"
            @submit.prevent="createUser"
          >
            <div class="space-y-2">
              <label class="block text-xs font-bold uppercase tracking-wider text-muted">
                Email address <span class="text-terracotta-500">*</span>
              </label>
              <input
                v-model="createForm.email"
                type="email"
                placeholder="user@example.com"
                required
                autocomplete="email"
                class="w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm outline-none transition-all placeholder:text-dimmed focus:border-attic-500 focus:ring-2 focus:ring-attic-500/15 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
              >
            </div>

            <div class="space-y-2">
              <label class="block text-xs font-bold uppercase tracking-wider text-muted">
                Display name
              </label>
              <input
                v-model="createForm.name"
                type="text"
                placeholder="e.g. John Doe"
                autocomplete="name"
                class="w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm outline-none transition-all placeholder:text-dimmed focus:border-attic-500 focus:ring-2 focus:ring-attic-500/15 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
              >
            </div>

            <div class="space-y-2">
              <label class="block text-xs font-bold uppercase tracking-wider text-muted">
                Temporary password <span class="text-terracotta-500">*</span>
              </label>
              <div class="relative">
                <input
                  v-model="createForm.password"
                  :type="showCreatePassword ? 'text' : 'password'"
                  placeholder="At least 8 characters"
                  minlength="8"
                  required
                  autocomplete="new-password"
                  class="w-full rounded-xl border border-mist-200 bg-white py-3 pl-4 pr-11 text-sm text-mist-950 shadow-sm outline-none transition-all placeholder:text-dimmed focus:border-attic-500 focus:ring-2 focus:ring-attic-500/15 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
                >
                <button
                  type="button"
                  class="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-muted hover:text-mist-600"
                  :aria-label="showCreatePassword ? 'Hide password' : 'Show password'"
                  @click="showCreatePassword = !showCreatePassword"
                >
                  <UIcon
                    :name="showCreatePassword ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                    class="size-4"
                  />
                </button>
              </div>
              <p class="text-xs text-muted">
                Share it securely—the person can change it after signing in.
              </p>
            </div>

            <fieldset class="space-y-2">
              <legend class="text-xs font-bold uppercase tracking-wider text-muted">
                Organization role
              </legend>
              <div class="grid grid-cols-2 gap-3">
                <button
                  v-for="option in roleOptions"
                  :key="option.value"
                  type="button"
                  class="flex items-start gap-3 rounded-xl border p-3 text-left transition-all"
                  :class="createForm.role === option.value ? 'border-attic-500 bg-attic-50 ring-2 ring-attic-500/10 dark:bg-attic-500/10' : 'border-mist-200 hover:border-attic-300 dark:border-mist-600'"
                  :aria-pressed="createForm.role === option.value"
                  @click="createForm.role = option.value"
                >
                  <div
                    class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg"
                    :class="createForm.role === option.value ? 'bg-attic-500 text-white' : 'bg-mist-100 text-muted dark:bg-mist-700'"
                  >
                    <UIcon
                      :name="option.value === 'admin' ? 'i-lucide-shield' : 'i-lucide-user'"
                      class="size-3.5"
                    />
                  </div>
                  <div>
                    <p class="text-sm font-extrabold text-mist-900 dark:text-white">
                      {{ option.value === 'admin' ? 'Administrator' : 'Member' }}
                    </p><p class="mt-0.5 text-[11px] leading-4 text-muted">
                      {{ option.value === 'admin' ? 'Full organization access' : 'Standard inventory access' }}
                    </p>
                  </div>
                </button>
              </div>
            </fieldset>
          </form>

          <div class="flex justify-end gap-2 border-t border-mist-100 bg-mist-50/60 px-6 py-4 dark:border-mist-700 dark:bg-mist-900/30">
            <UButton
              variant="ghost"
              color="neutral"
              class="rounded-xl"
              @click="isCreateModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              type="submit"
              form="create-person-form"
              icon="i-lucide-user-plus"
              :loading="isLoading"
              class="rounded-xl font-bold shadow-primary"
            >
              Add person
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Edit User Modal -->
    <UModal
      v-model:open="isEditModalOpen"
      :title="`Edit ${selectedUser?.name || selectedUser?.email.split('@')[0] || 'person'}`"
      :description="selectedUser?.email || 'Update account details and organization access.'"
    >
      <template #content>
        <div class="w-full max-w-lg overflow-hidden rounded-[22px] bg-white shadow-xl dark:bg-mist-800">
          <div class="flex items-center gap-4 border-b border-mist-100 bg-mist-50/70 px-6 py-5 dark:border-mist-700 dark:bg-mist-900/30">
            <div
              v-if="selectedUser"
              class="flex size-11 shrink-0 items-center justify-center rounded-2xl text-sm font-black"
              :class="[getAvatarColor(selectedUser).bg, getAvatarColor(selectedUser).text]"
            >
              {{ getInitials(selectedUser) }}
            </div>
            <div class="min-w-0">
              <p class="text-[10px] font-extrabold uppercase tracking-[0.14em] text-attic-500">
                Account settings
              </p>
              <h3 class="truncate text-lg font-extrabold text-mist-950 dark:text-white">
                Edit {{ selectedUser?.name || selectedUser?.email.split('@')[0] }}
              </h3>
              <p class="truncate text-xs text-muted">
                {{ selectedUser?.email }}
              </p>
            </div>
          </div>

          <form
            id="edit-person-form"
            class="space-y-5 p-6"
            @submit.prevent="updateUser"
          >
            <div class="space-y-2">
              <label class="block text-xs font-bold uppercase tracking-wider text-muted">
                Email address
              </label>
              <input
                v-model="editForm.email"
                type="email"
                placeholder="user@example.com"
                required
                class="w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm outline-none transition-all placeholder:text-dimmed focus:border-attic-500 focus:ring-2 focus:ring-attic-500/15 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
              >
            </div>

            <div class="space-y-2">
              <label class="block text-xs font-bold uppercase tracking-wider text-muted">
                Display name
              </label>
              <input
                v-model="editForm.name"
                type="text"
                placeholder="e.g. John Doe"
                class="w-full rounded-xl border border-mist-200 bg-white px-4 py-3 text-sm text-mist-950 shadow-sm outline-none transition-all placeholder:text-dimmed focus:border-attic-500 focus:ring-2 focus:ring-attic-500/15 dark:border-mist-600 dark:bg-mist-800 dark:text-white"
              >
            </div>

            <fieldset class="space-y-2">
              <legend class="text-xs font-bold uppercase tracking-wider text-muted">
                Organization role
              </legend>
              <div class="grid grid-cols-2 gap-3">
                <button
                  v-for="option in roleOptions"
                  :key="option.value"
                  type="button"
                  class="flex items-start gap-3 rounded-xl border p-3 text-left transition-all"
                  :class="editForm.role === option.value
                    ? 'border-attic-500 bg-attic-50 ring-2 ring-attic-500/10 dark:bg-attic-500/10'
                    : 'border-mist-200 hover:border-attic-300 dark:border-mist-600'"
                  :aria-pressed="editForm.role === option.value"
                  @click="editForm.role = option.value"
                >
                  <div
                    class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg"
                    :class="editForm.role === option.value ? 'bg-attic-500 text-white' : 'bg-mist-100 text-muted dark:bg-mist-700'"
                  >
                    <UIcon
                      :name="option.value === 'admin' ? 'i-lucide-shield' : 'i-lucide-user'"
                      class="size-3.5"
                    />
                  </div>
                  <div>
                    <p class="text-sm font-extrabold text-mist-900 dark:text-white">
                      {{ option.value === 'admin' ? 'Administrator' : 'Member' }}
                    </p>
                    <p class="mt-0.5 text-[11px] leading-4 text-muted">
                      {{ option.value === 'admin' ? 'Full organization access' : 'Standard inventory access' }}
                    </p>
                  </div>
                </button>
              </div>
            </fieldset>

            <div class="flex items-start gap-2 rounded-xl bg-amber-50 px-3 py-2.5 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
              <UIcon
                name="i-lucide-info"
                class="mt-0.5 size-3.5 shrink-0"
              />
              Role changes take effect the next time this person accesses the application.
            </div>
          </form>

          <div class="flex justify-end gap-2 border-t border-mist-100 bg-mist-50/60 px-6 py-4 dark:border-mist-700 dark:bg-mist-900/30">
            <UButton
              variant="ghost"
              color="neutral"
              class="rounded-xl"
              @click="isEditModalOpen = false"
            >
              Cancel
            </UButton>
            <UButton
              type="submit"
              form="edit-person-form"
              icon="i-lucide-save"
              :loading="isLoading"
              class="rounded-xl font-bold shadow-primary"
            >
              Save changes
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Reset Password Modal -->
    <UModal
      v-model:open="isResetPasswordModalOpen"
      title="Reset Password"
      :description="`Set a new password for ${selectedUser?.email || 'this user'}.`"
    >
      <template #content>
        <div class="w-full max-w-lg overflow-hidden rounded-[24px] bg-white shadow-xl ring-1 ring-mist-200/80 dark:bg-mist-800 dark:ring-mist-700">
          <div class="relative overflow-hidden bg-gradient-to-br from-attic-500 via-attic-600 to-[#174AE8] px-6 py-6 text-white sm:px-7">
            <div class="pointer-events-none absolute -right-12 -top-16 size-44 rounded-full border-[24px] border-white/10" />
            <div class="relative flex items-start gap-4">
              <div class="flex size-12 shrink-0 items-center justify-center rounded-2xl bg-white/15 ring-1 ring-white/20">
                <UIcon
                  name="i-lucide-key-round"
                  class="size-6"
                />
              </div>
              <div>
                <p class="text-[11px] font-extrabold uppercase tracking-[0.16em] text-white/70">
                  Account administration
                </p>
                <h2 class="mt-1 text-xl font-extrabold tracking-[-0.03em]">
                  Reset password
                </h2>
                <p class="mt-1 text-sm text-white/75">
                  Set a fresh sign-in password for this team member.
                </p>
              </div>
            </div>
          </div>

          <form
            class="space-y-5 p-6 sm:p-7"
            @submit.prevent="resetPassword"
          >
            <div class="flex items-center gap-3 rounded-2xl bg-attic-50 px-4 py-3 dark:bg-attic-500/10">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-white text-attic-500 shadow-sm dark:bg-mist-800">
                <UIcon
                  name="i-lucide-user-round"
                  class="size-4"
                />
              </div>
              <div class="min-w-0">
                <p class="text-[11px] font-extrabold uppercase tracking-[0.12em] text-attic-600 dark:text-attic-300">
                  Resetting for
                </p>
                <p class="truncate text-sm font-bold text-mist-800 dark:text-white">
                  {{ selectedUser?.email || 'Selected user' }}
                </p>
              </div>
            </div>

            <UFormField
              label="New password"
              name="resetPassword"
              help="Use at least 8 characters."
            >
              <UInput
                v-model="resetPasswordForm.password"
                type="password"
                placeholder="Create a temporary password"
                autocomplete="new-password"
                icon="i-lucide-lock-keyhole"
                size="lg"
                required
              />
            </UFormField>

            <div class="flex flex-col-reverse gap-2 border-t border-mist-100 pt-5 sm:flex-row sm:justify-end dark:border-mist-700">
              <UButton
                type="button"
                variant="ghost"
                color="neutral"
                class="rounded-xl font-bold"
                @click="isResetPasswordModalOpen = false"
              >
                Cancel
              </UButton>
              <UButton
                type="submit"
                class="rounded-xl font-bold shadow-primary"
                :loading="isLoading"
                :disabled="isLoading || !resetPasswordForm.password"
              >
                <UIcon
                  v-if="!isLoading"
                  name="i-lucide-key-round"
                  class="size-4"
                />
                Reset password
              </UButton>
            </div>
          </form>
        </div>
      </template>
    </UModal>

    <!-- Delete User Modal -->
    <UModal
      v-model:open="isDeleteModalOpen"
      title="Delete User"
      description="Confirm permanent deletion of this user account."
    >
      <template #content>
        <div class="w-full max-w-md rounded-[20px] bg-white p-6 shadow-xl dark:bg-mist-800">
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
              <p class="text-sm text-muted mt-2">
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
