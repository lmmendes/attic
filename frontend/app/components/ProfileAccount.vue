<script setup lang="ts">
const { user, isOIDCEnabled, isAuthDisabled } = useAuth()
const passwordOpen = ref(false)
const signInMethod = computed(() => isAuthDisabled.value ? 'Authentication disabled' : isOIDCEnabled.value ? 'Single sign-on' : 'Email and password')
</script>

<template>
  <section
    class="attic-panel overflow-hidden rounded-2xl"
    aria-labelledby="account-security-heading"
  >
    <div class="border-b border-mist-100 px-5 py-5 dark:border-mist-700 sm:px-6">
      <h2
        id="account-security-heading"
        class="text-lg font-extrabold text-mist-950 dark:text-white"
      >
        Account security
      </h2>
      <p class="mt-1 text-sm leading-6 text-muted">
        Manage how you sign in to Attic.
      </p>
    </div>
    <dl class="divide-y divide-mist-100 px-5 dark:divide-mist-700 sm:px-6">
      <div
        v-if="user?.email"
        class="flex flex-col gap-1 py-5 sm:flex-row sm:items-center sm:justify-between sm:gap-6"
      >
        <dt class="text-sm font-semibold text-mist-950 dark:text-white">
          Email address
        </dt>
        <dd class="break-all text-sm text-muted">
          {{ user.email }}
        </dd>
      </div>
      <div class="flex flex-col gap-1 py-5 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
        <dt class="text-sm font-semibold text-mist-950 dark:text-white">
          Sign-in method
        </dt>
        <dd class="text-sm text-muted">
          {{ signInMethod }}
        </dd>
      </div>
    </dl>
    <div class="flex flex-col items-start justify-between gap-4 border-t border-mist-100 bg-mist-50/50 px-5 py-5 dark:border-mist-700 dark:bg-mist-900/30 sm:flex-row sm:items-center sm:px-6">
      <div class="max-w-md">
        <h3 class="text-sm font-bold text-mist-950 dark:text-white">
          Password
        </h3>
        <p class="mt-1 text-sm leading-6 text-muted">
          {{ isAuthDisabled ? 'Password changes are unavailable while authentication is disabled.' : isOIDCEnabled ? 'Your password is managed by your sign-in provider. Change it through that provider.' : 'Use a unique password to keep your account protected.' }}
        </p>
      </div>
      <UButton
        v-if="!isOIDCEnabled && !isAuthDisabled"
        variant="outline"
        icon="i-lucide-lock-keyhole"
        class="shrink-0"
        @click="passwordOpen = true"
      >
        Change password
      </UButton>
    </div>
    <PasswordChangeDialog
      v-if="!isOIDCEnabled && !isAuthDisabled"
      v-model:open="passwordOpen"
    />
  </section>
</template>
