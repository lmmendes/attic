<script setup lang="ts">
useHead({
  meta: [
    { name: 'viewport', content: 'width=device-width, initial-scale=1' }
  ],
  link: [
    { rel: 'icon', href: '/favicon.ico' }
  ],
  htmlAttrs: {
    lang: 'en'
  }
})

const title = 'Attic - Home Inventory'
const description = 'A self-hosted home inventory for your belongings, rooms, warranties, receipts, and collections.'

useSeoMeta({
  title,
  description,
  ogTitle: title,
  ogDescription: description
})

const { isAuthenticated: loggedIn, user, isAdmin, logout, fetchSession, isOIDCEnabled, changePassword } = useAuth()
const config = useRuntimeConfig()

type AppInfo = {
  status: string
  version: string
}

const { data: appInfo, execute: fetchAppInfo } = useApi<AppInfo>('/api/', {
  immediate: false
})
const softwareVersion = computed(() => appInfo.value?.version || 'unknown')
const apiDocsUrl = computed(() => `${config.public.apiBase || ''}/api/docs`)

watch(loggedIn, (isLoggedIn) => {
  if (isLoggedIn) {
    void fetchAppInfo()
  }
}, { immediate: true })

// Fetch session on app load
onMounted(() => {
  fetchSession()
})

// Password change modal state
const passwordModalOpen = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordError = ref('')
const passwordSuccess = ref(false)
const passwordLoading = ref(false)

const openPasswordModal = () => {
  currentPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  passwordError.value = ''
  passwordSuccess.value = false
  passwordModalOpen.value = true
}

const handleChangePassword = async () => {
  passwordError.value = ''
  passwordSuccess.value = false

  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = 'New passwords do not match'
    return
  }

  if (newPassword.value.length < 8) {
    passwordError.value = 'Password must be at least 8 characters'
    return
  }

  passwordLoading.value = true
  const result = await changePassword(currentPassword.value, newPassword.value)
  passwordLoading.value = false

  if (result.success) {
    passwordSuccess.value = true
    setTimeout(() => {
      passwordModalOpen.value = false
    }, 1500)
  } else {
    passwordError.value = result.error || 'Failed to change password'
  }
}

const route = useRoute()

const baseNavigation = [
  { label: 'Dashboard', to: '/', icon: 'i-lucide-layout-dashboard' },
  { label: 'All Assets', to: '/assets', icon: 'i-lucide-package' },
  { label: 'Locations', to: '/locations', icon: 'i-lucide-map-pin' },
  { label: 'Collections', to: '/collections', icon: 'i-lucide-library' },
  { label: 'Categories', to: '/categories', icon: 'i-lucide-tag' }
]

const secondaryNavigation = [
  { label: 'Attributes', to: '/attributes', icon: 'i-lucide-sliders-horizontal' },
  { label: 'Conditions', to: '/conditions', icon: 'i-lucide-activity' },
  { label: 'Warranties', to: '/warranties', icon: 'i-lucide-shield-check' },
  { label: 'Plugins', to: '/plugins', icon: 'i-lucide-puzzle' }
]

const navigation = computed(() => {
  const items = [...baseNavigation]
  return items
})

const secondaryNav = computed(() => {
  const items = [...secondaryNavigation]
  if (isAdmin.value) {
    items.push({ label: 'Users', to: '/users', icon: 'i-lucide-users' })
  }
  return items
})

// Check if a nav item is active
const isActive = (to: string) => {
  if (to === '/') return route.path === '/'
  return route.path.startsWith(to)
}

type DropdownMenuItem = {
  label: string
  slot?: string
  disabled?: boolean
  icon?: string
  onSelect?: () => void
}

const userMenuItems = computed(() => {
  const items: DropdownMenuItem[][] = [
    [{
      label: user.value?.email || 'User',
      slot: 'account',
      disabled: true
    }]
  ]

  // Add change password option if not using OIDC
  if (!isOIDCEnabled.value) {
    items.push([{
      label: 'Change Password',
      icon: 'i-lucide-key',
      onSelect: openPasswordModal
    }])
  }

  items.push([{
    label: 'Sign out',
    icon: 'i-lucide-log-out',
    onSelect: () => logout()
  }])

  return items
})

// Mobile sidebar state
const sidebarOpen = ref(false)
const isAssetForm = computed(() => /^\/assets\/(?:new|[^/]+\/edit)\/?$/.test(route.path))
</script>

<template>
  <UApp class="bg-mist-50 dark:bg-mist-900">
    <!-- Login page without sidebar -->
    <template v-if="!loggedIn">
      <NuxtPage />
    </template>

    <!-- Main app layout with sidebar -->
    <template v-else>
      <div class="h-dvh flex overflow-hidden app-canvas">
        <!-- Sidebar -->
        <aside class="hidden lg:flex w-68 bg-white/92 dark:bg-mist-900/92 backdrop-blur-xl border-r border-mist-200/80 dark:border-mist-800 flex-col flex-shrink-0">
          <!-- Logo -->
          <div class="p-5">
            <NuxtLink
              to="/"
              class="flex items-center gap-3 px-2 mb-9"
            >
              <div class="bg-gradient-to-br from-attic-500 to-attic-700 rounded-[14px] size-11 flex items-center justify-center text-white shadow-primary ring-1 ring-white/20">
                <UIcon
                  name="i-lucide-archive"
                  class="w-5.5 h-5.5"
                />
              </div>
              <div>
                <h1 class="text-mist-950 dark:text-white text-lg font-extrabold leading-none">
                  Attic
                </h1>
                <p class="text-muted dark:text-mist-400 text-[11px] font-semibold tracking-wide uppercase mt-1">
                  Home inventory
                </p>
              </div>
            </NuxtLink>

            <!-- Navigation -->
            <nav class="space-y-1">
              <p class="px-3 pb-2 text-[10px] font-extrabold uppercase tracking-[0.16em] text-muted">
                Your home
              </p>
              <NuxtLink
                v-for="item in navigation"
                :key="item.to"
                :to="item.to"
                class="flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all"
                :class="isActive(item.to)
                  ? 'bg-attic-50 text-attic-600 shadow-sm ring-1 ring-attic-100 dark:bg-attic-500/15 dark:text-attic-300 dark:ring-attic-500/20'
                  : 'text-mist-600 dark:text-mist-300 hover:bg-mist-50 dark:hover:bg-mist-800'"
              >
                <UIcon
                  :name="item.icon"
                  class="w-5 h-5"
                />
                <span
                  class="text-sm"
                  :class="isActive(item.to) ? 'font-bold' : 'font-semibold'"
                >
                  {{ item.label }}
                </span>
              </NuxtLink>

              <!-- Divider -->
              <div class="pt-5 mt-5 border-t border-mist-100 dark:border-mist-800">
                <p class="px-3 pb-2 text-[10px] font-extrabold uppercase tracking-[0.16em] text-muted">
                  Manage
                </p>
                <NuxtLink
                  v-for="item in secondaryNav"
                  :key="item.to"
                  :to="item.to"
                  class="flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all"
                  :class="isActive(item.to)
                    ? 'bg-attic-50 text-attic-600 shadow-sm ring-1 ring-attic-100 dark:bg-attic-500/15 dark:text-attic-300 dark:ring-attic-500/20'
                    : 'text-mist-600 dark:text-mist-300 hover:bg-mist-50 dark:hover:bg-mist-800'"
                >
                  <UIcon
                    :name="item.icon"
                    class="w-5 h-5"
                  />
                  <span
                    class="text-sm"
                    :class="isActive(item.to) ? 'font-bold' : 'font-semibold'"
                  >
                    {{ item.label }}
                  </span>
                </NuxtLink>
              </div>
            </nav>
          </div>

          <!-- User section at bottom -->
          <div class="mt-auto p-4 border-t border-mist-100 dark:border-mist-800">
            <div class="flex items-center gap-3 p-2 rounded-2xl bg-mist-50/80 dark:bg-mist-800/70">
              <div class="w-9 h-9 rounded-xl bg-attic-100 dark:bg-attic-500/20 flex items-center justify-center text-attic-600 dark:text-attic-300">
                <UIcon
                  name="i-lucide-user"
                  class="w-4 h-4"
                />
              </div>
              <div class="flex flex-col flex-1 min-w-0">
                <span class="text-sm font-bold text-mist-950 dark:text-white truncate">
                  {{ user?.name || user?.email?.split('@')[0] || 'User' }}
                </span>
                <span class="text-xs text-muted dark:text-gray-400 truncate">
                  {{ user?.email }}
                </span>
              </div>
              <UDropdownMenu :items="userMenuItems">
                <UButton
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-settings"
                  size="sm"
                  aria-label="Open account menu"
                  title="Open account menu"
                />
                <template #account>
                  <div class="text-left">
                    <p class="font-medium">
                      {{ user?.name || 'User' }}
                    </p>
                    <p class="text-xs text-muted truncate">
                      {{ user?.email }}
                    </p>
                  </div>
                </template>
              </UDropdownMenu>
              <UColorModeButton />
            </div>
          </div>
        </aside>

        <!-- Mobile sidebar overlay -->
        <div
          v-if="sidebarOpen"
          class="fixed inset-0 z-40 bg-black/50 lg:hidden"
          @click="sidebarOpen = false"
        />

        <!-- Mobile sidebar -->
        <aside
          class="fixed inset-y-0 left-0 z-50 w-64 bg-white dark:bg-mist-900 border-r border-gray-200 dark:border-gray-800 flex flex-col lg:hidden transform transition-transform duration-200"
          :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
        >
          <div class="p-6">
            <div class="flex items-center justify-between mb-8">
              <NuxtLink
                to="/"
                class="flex items-center gap-3"
                @click="sidebarOpen = false"
              >
                <div class="bg-attic-500 rounded-xl size-10 flex items-center justify-center text-white">
                  <UIcon
                    name="i-lucide-archive"
                    class="w-5 h-5"
                  />
                </div>
                <div>
                  <h1 class="text-mist-950 dark:text-white text-lg font-extrabold leading-none">
                    Attic
                  </h1>
                  <p class="text-muted dark:text-gray-400 text-xs font-medium">
                    Home inventory
                  </p>
                </div>
              </NuxtLink>
              <UButton
                color="neutral"
                variant="ghost"
                icon="i-lucide-x"
                aria-label="Close navigation"
                title="Close navigation"
                @click="sidebarOpen = false"
              />
            </div>

            <nav class="space-y-1">
              <NuxtLink
                v-for="item in navigation"
                :key="item.to"
                :to="item.to"
                class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors"
                :class="isActive(item.to)
                  ? 'bg-attic-500/10 text-attic-500'
                  : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'"
                @click="sidebarOpen = false"
              >
                <UIcon
                  :name="item.icon"
                  class="w-5 h-5"
                />
                <span
                  class="text-sm"
                  :class="isActive(item.to) ? 'font-bold' : 'font-semibold'"
                >
                  {{ item.label }}
                </span>
              </NuxtLink>

              <div class="pt-4 mt-4 border-t border-gray-100 dark:border-gray-800">
                <NuxtLink
                  v-for="item in secondaryNav"
                  :key="item.to"
                  :to="item.to"
                  class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors"
                  :class="isActive(item.to)
                    ? 'bg-attic-500/10 text-attic-500'
                    : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'"
                  @click="sidebarOpen = false"
                >
                  <UIcon
                    :name="item.icon"
                    class="w-5 h-5"
                  />
                  <span
                    class="text-sm"
                    :class="isActive(item.to) ? 'font-bold' : 'font-semibold'"
                  >
                    {{ item.label }}
                  </span>
                </NuxtLink>
              </div>
            </nav>
          </div>

          <div class="mt-auto p-6 border-t border-gray-100 dark:border-gray-800">
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-full bg-attic-500 flex items-center justify-center text-white">
                <UIcon
                  name="i-lucide-user"
                  class="w-4 h-4"
                />
              </div>
              <div class="flex flex-col flex-1 min-w-0">
                <span class="text-sm font-bold text-mist-950 dark:text-white truncate">
                  {{ user?.name || user?.email?.split('@')[0] || 'User' }}
                </span>
                <span class="text-xs text-muted dark:text-gray-400 truncate">
                  {{ user?.email }}
                </span>
              </div>
            </div>
          </div>
        </aside>

        <!-- Main content area -->
        <main class="flex-1 flex flex-col h-full overflow-hidden">
          <!-- Top header bar (mobile only) -->
          <header class="lg:hidden h-16 flex items-center justify-between px-4 bg-white dark:bg-mist-900 border-b border-gray-100 dark:border-gray-800 flex-shrink-0">
            <UButton
              color="neutral"
              variant="ghost"
              icon="i-lucide-menu"
              aria-label="Open navigation"
              title="Open navigation"
              @click="sidebarOpen = true"
            />

            <NuxtLink
              to="/"
              class="flex items-center gap-2"
            >
              <div class="bg-attic-500 rounded-lg size-8 flex items-center justify-center text-white">
                <UIcon
                  name="i-lucide-archive"
                  class="w-4 h-4"
                />
              </div>
              <span class="font-extrabold text-mist-950 dark:text-white">Attic</span>
            </NuxtLink>

            <UColorModeButton />
          </header>

          <!-- Scrollable content -->
          <div class="flex-1 overflow-y-auto custom-scrollbar p-4 md:p-6 xl:p-7">
            <div class="mx-auto flex min-h-full max-w-[1440px] flex-col">
              <div class="flex-1">
                <NuxtPage />
              </div>

              <footer class="shrink-0 pt-6 text-center text-xs font-medium text-muted dark:text-mist-400">
                <span>Attic version {{ softwareVersion }}</span>
                <span class="mx-2 text-mist-300 dark:text-mist-600">·</span>
                <a
                  :href="apiDocsUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-attic-600 underline decoration-attic-300 underline-offset-2 transition-colors hover:text-attic-700 dark:text-attic-300 dark:decoration-attic-500 dark:hover:text-attic-200"
                >
                  API
                </a>
              </footer>
            </div>
          </div>
        </main>

        <!-- Mobile FAB -->
        <NuxtLink
          v-if="!isAssetForm"
          to="/assets/new"
          aria-label="Add asset"
          title="Add asset"
          class="lg:hidden fixed bottom-6 right-6 size-14 rounded-2xl bg-attic-500 text-white shadow-primary flex items-center justify-center hover:scale-105 active:scale-95 transition-all z-50"
        >
          <UIcon
            name="i-lucide-plus"
            class="w-7 h-7"
          />
        </NuxtLink>
      </div>
    </template>

    <!-- Password Change Modal -->
    <UModal
      v-model:open="passwordModalOpen"
      title="Change Password"
      description="Enter your current password and choose a new password."
    >
      <template #content>
        <div class="w-full max-w-lg overflow-hidden rounded-[24px] bg-white shadow-xl ring-1 ring-mist-200/80 dark:bg-mist-800 dark:ring-mist-700">
          <div class="relative overflow-hidden bg-gradient-to-br from-attic-500 via-attic-600 to-[#174AE8] px-6 py-6 text-white sm:px-7">
            <div class="pointer-events-none absolute -right-12 -top-16 size-44 rounded-full border-[24px] border-white/10" />
            <div class="relative flex items-start gap-4">
              <div class="flex size-12 shrink-0 items-center justify-center rounded-2xl bg-white/15 ring-1 ring-white/20">
                <UIcon
                  name="i-lucide-shield-check"
                  class="size-6"
                />
              </div>
              <div>
                <p class="text-[11px] font-extrabold uppercase tracking-[0.16em] text-white/70">
                  Account security
                </p>
                <h2 class="mt-1 text-xl font-extrabold tracking-[-0.03em]">
                  Change your password
                </h2>
                <p class="mt-1 text-sm text-white/75">
                  Keep your account protected with a password you don’t reuse elsewhere.
                </p>
              </div>
            </div>
          </div>

          <form
            class="space-y-5 p-6 sm:p-7"
            @submit.prevent="handleChangePassword"
          >
            <UAlert
              v-if="passwordError"
              color="error"
              :title="passwordError"
              icon="i-lucide-alert-circle"
            />

            <UAlert
              v-if="passwordSuccess"
              color="success"
              title="Password changed successfully"
              icon="i-lucide-check-circle"
            />

            <div class="space-y-4">
              <UFormField
                label="Current password"
                name="currentPassword"
              >
                <UInput
                  v-model="currentPassword"
                  type="password"
                  placeholder="Enter current password"
                  autocomplete="current-password"
                  icon="i-lucide-lock-keyhole"
                  size="lg"
                  required
                />
              </UFormField>

              <div class="border-t border-mist-100 pt-4 dark:border-mist-700">
                <p class="mb-3 text-xs font-bold uppercase tracking-[0.12em] text-muted">
                  New password
                </p>
                <div class="space-y-4">
                  <UFormField
                    label="New password"
                    name="newPassword"
                    help="Use at least 8 characters."
                  >
                    <UInput
                      v-model="newPassword"
                      type="password"
                      placeholder="Create a new password"
                      autocomplete="new-password"
                      icon="i-lucide-key-round"
                      size="lg"
                      required
                    />
                  </UFormField>

                  <UFormField
                    label="Confirm new password"
                    name="confirmPassword"
                  >
                    <UInput
                      v-model="confirmPassword"
                      type="password"
                      placeholder="Repeat your new password"
                      autocomplete="new-password"
                      icon="i-lucide-check"
                      size="lg"
                      required
                    />
                  </UFormField>
                </div>
              </div>
            </div>

            <div class="flex flex-col-reverse gap-2 border-t border-mist-100 pt-5 sm:flex-row sm:justify-end dark:border-mist-700">
              <UButton
                color="neutral"
                variant="ghost"
                class="rounded-xl font-bold"
                @click="passwordModalOpen = false"
              >
                Cancel
              </UButton>
              <UButton
                type="submit"
                class="rounded-xl font-bold shadow-primary"
                :loading="passwordLoading"
                :disabled="passwordLoading || !currentPassword || !newPassword || !confirmPassword"
              >
                <UIcon
                  v-if="!passwordLoading"
                  name="i-lucide-check"
                  class="size-4"
                />
                Update password
              </UButton>
            </div>
          </form>
        </div>
      </template>
    </UModal>
  </UApp>
</template>
