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

const baseNavigation = [
  { label: 'Dashboard', to: '/', icon: 'i-lucide-layout-dashboard' },
  { label: 'Assets', to: '/assets', icon: 'i-lucide-box' },
  { label: 'Categories', to: '/categories', icon: 'i-lucide-folder-tree' },
  { label: 'Attributes', to: '/attributes', icon: 'i-lucide-tags' },
  { label: 'Locations', to: '/locations', icon: 'i-lucide-map-pin' },
  { label: 'Conditions', to: '/conditions', icon: 'i-lucide-activity' },
  { label: 'Warranties', to: '/warranties', icon: 'i-lucide-shield-check' },
  { label: 'Plugins', to: '/plugins', icon: 'i-lucide-puzzle' }
]

const navigation = computed(() => {
  const items = [...baseNavigation]
  if (isAdmin.value) {
    items.push({ label: 'Users', to: '/users', icon: 'i-lucide-users' })
  }
  return items
})

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
</script>

<template>
  <UApp>
    <UHeader>
      <template #left>
        <NuxtLink
          to="/"
          class="flex items-center gap-2"
        >
          <UIcon
            name="i-lucide-archive"
            class="w-6 h-6 text-primary"
          />
          <span class="font-semibold text-lg">Attic</span>
        </NuxtLink>

        <UNavigationMenu
          v-if="loggedIn"
          :items="navigation"
          class="ml-6 hidden md:flex"
        />
      </template>

      <template #right>
        <UColorModeButton />

        <template v-if="loggedIn">
          <UDropdownMenu :items="userMenuItems">
            <UButton
              color="neutral"
              variant="ghost"
              icon="i-lucide-user"
              :label="user?.name || user?.email || 'User'"
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
        </template>
        <template v-else>
          <UButton
            color="primary"
            @click="login()"
          >
            Sign in
          </UButton>
        </template>
      </template>
    </UHeader>

    <UMain>
      <NuxtPage />
    </UMain>

    <UFooter>
      <template #left>
        <p class="text-sm text-muted">
          Attic Asset Management
        </p>
      </template>
    </UFooter>
  </UApp>
</template>
