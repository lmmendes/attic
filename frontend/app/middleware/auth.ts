export default defineNuxtRouteMiddleware(async (to) => {
  // Skip on server-side
  if (import.meta.server) return

  // Allow login page without auth
  if (to.path === '/login') return

  const { isAuthenticated, loading, fetchSession } = useAuth()

  // If session hasn't been fetched yet, fetch it
  if (loading.value) {
    await fetchSession()
  }

  if (!isAuthenticated.value) {
    // Redirect to login page
    return navigateTo('/login')
  }
})
