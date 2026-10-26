<script setup lang="ts">
import type { TabsItem } from '@nuxt/ui'

definePageMeta({ middleware: 'auth' })
useHead({ title: 'Profile · Attic' })

const route = useRoute()
const router = useRouter()
const { user, isAdmin } = useAuth()
const tabs = [
  { label: 'Account', value: 'account', icon: 'i-lucide-user-round', slot: 'account' as const },
  { label: 'API keys', value: 'api-keys', icon: 'i-lucide-key-round', slot: 'api-keys' as const },
  { label: 'MCP connection', value: 'mcp', icon: 'i-lucide-plug', slot: 'mcp' as const }
] satisfies TabsItem[]
const activeTab = computed({
  get: () => tabs.some(tab => tab.value === route.query.tab) ? route.query.tab as string : 'account',
  set: (tab: string | number) => {
    void router.push({ path: '/profile', query: { ...route.query, tab: String(tab) } })
  }
})
const displayName = computed(() => user.value?.name || user.value?.email || 'Your account')
const initials = computed(() => displayName.value.split(/\s+/).slice(0, 2).map(part => part[0]).join('').toUpperCase())
</script>

<template>
  <div class="mx-auto w-full max-w-5xl space-y-7 pb-8">
    <header>
      <p class="mb-1.5 text-[11px] font-extrabold uppercase tracking-[0.16em] text-attic-600 dark:text-attic-300">
        Your space
      </p>
      <h1 class="text-3xl font-extrabold tracking-[-0.04em] text-mist-950 dark:text-white">
        Profile
      </h1>
      <p class="mt-2 text-sm leading-6 text-muted">
        Your account, your keys, and the assistants you connect.
      </p>
    </header>

    <section
      aria-label="Your identity"
      class="attic-panel flex min-w-0 items-center gap-4 rounded-2xl p-5 sm:gap-5 sm:p-6"
    >
      <div
        aria-hidden="true"
        class="flex size-14 shrink-0 items-center justify-center rounded-2xl bg-attic-50 text-lg font-extrabold text-attic-600 ring-1 ring-attic-100 dark:bg-attic-500/10 dark:text-attic-300 dark:ring-attic-500/20 sm:size-16"
      >
        {{ initials }}
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="break-words text-lg font-extrabold text-mist-950 dark:text-white">
            {{ displayName }}
          </h2>
          <span
            v-if="user?.role"
            class="rounded-full bg-mist-100 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wide text-mist-600 dark:bg-mist-700 dark:text-mist-200"
          >
            {{ isAdmin ? 'Administrator' : 'Member' }}
          </span>
        </div>
        <p
          v-if="user?.email && user.email !== displayName"
          class="mt-1 break-all text-sm text-muted"
        >
          {{ user.email }}
        </p>
        <p class="mt-1 text-xs text-muted">
          Personal settings for your Attic account.
        </p>
      </div>
    </section>

    <UTabs
      v-model="activeTab"
      :items="tabs"
      :unmount-on-hide="true"
      variant="link"
      class="w-full gap-6"
      :ui="{ trigger: 'flex-1 px-2 sm:flex-none sm:px-4', leadingIcon: 'hidden sm:block', label: 'text-xs sm:text-sm' }"
    >
      <template #account>
        <ProfileAccount />
      </template>
      <template #api-keys>
        <ProfileApiKeys />
      </template>
      <template #mcp>
        <ProfileMcpConnection />
      </template>
    </UTabs>
  </div>
</template>
