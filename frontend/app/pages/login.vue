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
  <div class="min-h-screen flex bg-gray-50 dark:bg-gray-900">
    <!-- Left side - Branding (hidden on mobile) -->
    <div class="hidden lg:flex lg:w-1/2 xl:w-3/5 bg-primary-600 dark:bg-primary-800 items-center justify-center p-12">
      <div class="max-w-lg text-center">
        <UIcon
          name="i-lucide-archive"
          class="w-24 h-24 text-white mx-auto mb-8"
        />
        <h1 class="text-4xl xl:text-5xl font-bold text-white mb-4">
          Attic
        </h1>
        <p class="text-xl text-primary-100">
          Asset Management System
        </p>
        <p class="mt-6 text-primary-200 text-lg">
          Organize, track, and manage your assets efficiently
        </p>
      </div>
    </div>

    <!-- Right side - Login Form -->
    <div class="w-full lg:w-1/2 xl:w-2/5 flex items-center justify-center p-6 sm:p-12">
      <div class="w-full max-w-md">
        <!-- Mobile header (shown only on mobile) -->
        <div class="text-center lg:hidden mb-8">
          <div class="flex justify-center">
            <UIcon
              name="i-lucide-archive"
              class="w-16 h-16 text-primary"
            />
          </div>
          <h2 class="mt-4 text-3xl font-bold text-gray-900 dark:text-white">
            Attic
          </h2>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Asset Management System
          </p>
        </div>

        <!-- Desktop header (shown only on desktop) -->
        <div class="hidden lg:block mb-8">
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
            Welcome back
          </h2>
          <p class="mt-2 text-gray-600 dark:text-gray-400">
            Sign in to your account to continue
          </p>
        </div>

        <UCard>
          <div
            v-if="loading"
            class="flex justify-center py-8"
          >
            <UIcon
              name="i-lucide-loader-2"
              class="w-8 h-8 animate-spin text-primary"
            />
          </div>

          <template v-else>
            <!-- OIDC Login Button -->
            <div
              v-if="isOIDCEnabled"
              class="space-y-4"
            >
              <UButton
                block
                size="xl"
                color="primary"
                icon="i-lucide-log-in"
                @click="handleOIDCLogin"
              >
                Sign in with SSO
              </UButton>
            </div>

            <!-- Email/Password Login Form -->
            <form
              v-else
              class="space-y-6"
              @submit.prevent="handleLogin"
            >
              <UAlert
                v-if="error"
                color="error"
                :title="error"
                icon="i-lucide-alert-circle"
              />

              <UFormField
                label="Email"
                name="email"
              >
                <UInput
                  v-model="email"
                  type="text"
                  placeholder="Enter your email"
                  icon="i-lucide-mail"
                  size="xl"
                  autocomplete="username"
                  required
                />
              </UFormField>

              <UFormField
                label="Password"
                name="password"
              >
                <UInput
                  v-model="password"
                  type="password"
                  placeholder="Enter your password"
                  icon="i-lucide-lock"
                  size="xl"
                  autocomplete="current-password"
                  required
                />
              </UFormField>

              <UButton
                type="submit"
                block
                size="xl"
                color="primary"
                :loading="isLoading"
                :disabled="isLoading || !email || !password"
              >
                Sign in
              </UButton>
            </form>
          </template>
        </UCard>

        <p class="text-center text-sm text-gray-500 dark:text-gray-400 mt-8">
          Powered by Attic
        </p>
      </div>
    </div>
  </div>
</template>
