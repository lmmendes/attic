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

const title = 'Attic - Asset Management'
const description = 'A simple, powerful asset management system for organizations.'

useSeoMeta({
  title,
  description,
  ogTitle: title,
  ogDescription: description
})

const { isAuthenticated: loggedIn, user, isAdmin, login, logout, fetchSession, isOIDCEnabled } = useAuth()

// Fetch session on app load
onMounted(() => {
  fetchSession()
})

const route = useRoute()

const baseNavigation = [
  { label: 'Dashboard', to: '/', icon: 'i-lucide-layout-dashboard' },
  { label: 'All Assets', to: '/assets', icon: 'i-lucide-package' },
  { label: 'Locations', to: '/locations', icon: 'i-lucide-map-pin' },
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
  click?: () => void
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
      click: () => navigateTo('/settings')
    }])
  }

  items.push([{
    label: 'Sign out',
    icon: 'i-lucide-log-out',
    click: () => logout()
  }])

  return items
})

// Mobile sidebar state
const sidebarOpen = ref(false)
</script>

<template>
  <UApp class="bg-mist-50 dark:bg-mist-900">
    <!-- Login page without sidebar -->
    <template v-if="!loggedIn">
      <div class="min-h-screen flex items-center justify-center p-4">
        <NuxtPage />
      </div>
    </template>

    <!-- Main app layout with sidebar -->
    <template v-else>
      <div class="h-screen flex overflow-hidden">
        <!-- Sidebar -->
        <aside class="hidden lg:flex w-64 bg-white dark:bg-mist-900 border-r border-gray-200 dark:border-gray-800 flex-col flex-shrink-0">
          <!-- Logo -->
          <div class="p-6">
            <NuxtLink
              to="/"
              class="flex items-center gap-3 mb-8"
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
                <p class="text-mist-500 dark:text-gray-400 text-xs font-medium">
                  Home Management
                </p>
              </div>
            </NuxtLink>

            <!-- Navigation -->
            <nav class="space-y-1">
              <NuxtLink
                v-for="item in navigation"
                :key="item.to"
                :to="item.to"
                class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors"
                :class="isActive(item.to)
                  ? 'bg-attic-500/10 text-attic-500'
                  : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'"
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
              <div class="pt-4 mt-4 border-t border-gray-100 dark:border-gray-800">
                <NuxtLink
                  v-for="item in secondaryNav"
                  :key="item.to"
                  :to="item.to"
                  class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors"
                  :class="isActive(item.to)
                    ? 'bg-attic-500/10 text-attic-500'
                    : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'"
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
          <div class="mt-auto p-6">
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
                <span class="text-xs text-mist-500 dark:text-gray-400 truncate">
                  {{ user?.email }}
                </span>
              </div>
              <UDropdownMenu :items="userMenuItems">
                <UButton
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-settings"
                  size="sm"
                />
                <template #account>
                  <div class="text-left">
                    <p class="font-medium">
                      {{ user?.name || 'User' }}
                    </p>
                    <p class="text-xs text-mist-500 truncate">
                      {{ user?.email }}
                    </p>
                  </div>
                </template>
              </UDropdownMenu>
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
                  <p class="text-mist-500 dark:text-gray-400 text-xs font-medium">
                    Home Management
                  </p>
                </div>
              </NuxtLink>
              <UButton
                color="neutral"
                variant="ghost"
                icon="i-lucide-x"
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
                <span class="text-xs text-mist-500 dark:text-gray-400 truncate">
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
          <div class="flex-1 overflow-y-auto custom-scrollbar p-4 md:p-8">
            <div class="max-w-7xl mx-auto">
              <NuxtPage />
            </div>
          </div>
        </main>

        <!-- Mobile FAB -->
        <NuxtLink
          to="/assets/new"
          class="lg:hidden fixed bottom-6 right-6 size-14 rounded-full bg-attic-500 text-white shadow-2xl flex items-center justify-center hover:scale-110 active:scale-95 transition-all z-50"
        >
          <UIcon
            name="i-lucide-plus"
            class="w-7 h-7"
          />
        </NuxtLink>
      </div>
    </template>
  </UApp>
</template>
