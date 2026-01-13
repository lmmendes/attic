interface AuthSession {
  authenticated: boolean
  oidc_enabled?: boolean
  user?: {
    id?: string
    sub?: string
    email: string
    name: string
    role?: 'user' | 'admin'
  }
  expires_at?: string
}

interface LoginCredentials {
  email: string
  password: string
}

interface LoginResponse {
  success: boolean
  user?: AuthSession['user']
  error?: string
}

export function useAuth() {
  const session = useState<AuthSession | null>('auth-session', () => null)
  const loading = useState<boolean>('auth-loading', () => true)
  const config = useRuntimeConfig()

  const fetchSession = async () => {
    loading.value = true
    try {
      const data = await $fetch<AuthSession>('/auth/session', {
        baseURL: config.public.apiBase as string,
        credentials: 'include'
      })
      session.value = data
    } catch {
      session.value = { authenticated: false }
    } finally {
      loading.value = false
    }
  }

  const loginWithCredentials = async (credentials: LoginCredentials): Promise<LoginResponse> => {
    try {
      const data = await $fetch<LoginResponse>('/auth/login', {
        baseURL: config.public.apiBase as string,
        method: 'POST',
        body: credentials,
        credentials: 'include'
      })
      if (data.success) {
        await fetchSession()
      }
      return data
    } catch (error: any) {
      const message = error?.data?.error || error?.message || 'Login failed'
      return { success: false, error: message }
    }
  }

  const loginWithOIDC = () => {
    window.location.href = `${config.public.apiBase}/auth/oidc/login`
  }

  const login = () => {
    // For backwards compatibility - redirect to login page
    navigateTo('/login')
  }

  const logout = async () => {
    try {
      await $fetch('/auth/logout', {
        baseURL: config.public.apiBase as string,
        method: 'POST',
        credentials: 'include'
      })
    } catch {
      // Ignore errors
    }
    session.value = { authenticated: false }
    navigateTo('/login')
  }

  const changePassword = async (currentPassword: string, newPassword: string): Promise<{ success: boolean; error?: string }> => {
    try {
      await $fetch('/api/auth/password', {
        baseURL: config.public.apiBase as string,
        method: 'PUT',
        body: { current_password: currentPassword, new_password: newPassword },
        credentials: 'include'
      })
      return { success: true }
    } catch (error: any) {
      const message = error?.data?.error || error?.message || 'Password change failed'
      return { success: false, error: message }
    }
  }

  const isAuthenticated = computed(() => session.value?.authenticated ?? false)
  const isOIDCEnabled = computed(() => session.value?.oidc_enabled ?? false)
  const user = computed(() => session.value?.user ?? null)
  const isAdmin = computed(() => session.value?.user?.role === 'admin')

  return {
    session,
    loading,
    isAuthenticated,
    isOIDCEnabled,
    user,
    isAdmin,
    fetchSession,
    loginWithCredentials,
    loginWithOIDC,
    login,
    logout,
    changePassword
  }
}
