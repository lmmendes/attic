export default defineNuxtRouteMiddleware(async () => {
  // Skip on server-side (SSR disabled anyway, but just in case)
  if (import.meta.server) return

  const { isAuthenticated, loading, fetchSession, login } = useAuth()

  // If session hasn't been fetched yet, fetch it
  if (loading.value) {
    await fetchSession()
  }

  if (!isAuthenticated.value) {
    // Redirect to login
    login()
    // Return a promise that never resolves to prevent navigation
    return new Promise(() => {})
  }
})
