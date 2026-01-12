interface AuthSession {
  authenticated: boolean
  user?: {
    sub: string
    email: string
    name: string
  }
  expires_at?: string
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

  const login = () => {
    window.location.href = `${config.public.apiBase}/auth/login`
  }

  const logout = () => {
    window.location.href = `${config.public.apiBase}/auth/logout`
  }

  const isAuthenticated = computed(() => session.value?.authenticated ?? false)
  const user = computed(() => session.value?.user ?? null)

  return {
    session,
    loading,
    isAuthenticated,
    user,
    fetchSession,
    login,
    logout
  }
}
