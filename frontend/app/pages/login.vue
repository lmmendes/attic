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
  <div class="min-h-screen flex items-center justify-center bg-primary-600 dark:bg-primary-800 p-6">
    <div class="w-full max-w-md">
      <!-- Branding -->
      <div class="text-center mb-8">
        <UIcon
          name="i-lucide-archive"
          class="w-16 h-16 text-white mx-auto mb-4"
        />
        <h1 class="text-3xl lg:text-4xl font-bold text-white mb-2">
          Attic
        </h1>
        <p class="text-primary-100">
          Asset Management System
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
                class="w-full"
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
                class="w-full"
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

      <p class="text-center text-sm text-primary-200 mt-8">
        Powered by Attic
      </p>
    </div>
  </div>
</template>
