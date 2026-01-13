<script setup lang="ts">
definePageMeta({
  layout: false
})

const { isAuthenticated, isOIDCEnabled, loginWithCredentials, loginWithOIDC, fetchSession, loading } = useAuth()

const email = ref('')
const password = ref('')
const error = ref('')
const isLoading = ref(false)

// Check if already authenticated
onMounted(async () => {
  await fetchSession()
  if (isAuthenticated.value) {
    navigateTo('/')
  }
})

// Watch for authentication changes
watch(isAuthenticated, (authenticated) => {
  if (authenticated) {
    navigateTo('/')
  }
})

const handleLogin = async () => {
  error.value = ''
  isLoading.value = true

  const result = await loginWithCredentials({ email: email.value, password: password.value })

  if (!result.success) {
    error.value = result.error || 'Login failed'
  }

  isLoading.value = false
}

const handleOIDCLogin = () => {
  loginWithOIDC()
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div class="text-center">
        <div class="flex justify-center">
          <UIcon
            name="i-lucide-archive"
            class="w-16 h-16 text-primary"
          />
        </div>
        <h2 class="mt-6 text-3xl font-bold text-gray-900 dark:text-white">
          Sign in to Attic
        </h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Asset Management System
        </p>
      </div>

      <UCard class="mt-8">
        <div v-if="loading" class="flex justify-center py-8">
          <UIcon name="i-lucide-loader-2" class="w-8 h-8 animate-spin text-primary" />
        </div>

        <template v-else>
          <!-- OIDC Login Button -->
          <div v-if="isOIDCEnabled" class="space-y-4">
            <UButton
              block
              size="lg"
              color="primary"
              icon="i-lucide-log-in"
              @click="handleOIDCLogin"
            >
              Sign in with SSO
            </UButton>
          </div>

          <!-- Email/Password Login Form -->
          <form v-else @submit.prevent="handleLogin" class="space-y-6">
            <UAlert
              v-if="error"
              color="error"
              :title="error"
              icon="i-lucide-alert-circle"
            />

            <UFormField label="Email" name="email">
              <UInput
                v-model="email"
                type="text"
                placeholder="Enter your email"
                icon="i-lucide-mail"
                size="lg"
                autocomplete="username"
                required
              />
            </UFormField>

            <UFormField label="Password" name="password">
              <UInput
                v-model="password"
                type="password"
                placeholder="Enter your password"
                icon="i-lucide-lock"
                size="lg"
                autocomplete="current-password"
                required
              />
            </UFormField>

            <UButton
              type="submit"
              block
              size="lg"
              color="primary"
              :loading="isLoading"
              :disabled="isLoading || !email || !password"
            >
              Sign in
            </UButton>
          </form>
        </template>
      </UCard>

      <p class="text-center text-sm text-gray-500 dark:text-gray-400">
        Powered by Attic
      </p>
    </div>
  </div>
</template>
