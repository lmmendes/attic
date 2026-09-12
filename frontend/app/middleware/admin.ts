export default defineNuxtRouteMiddleware(async () => {
  const { isAdmin, fetchSession, loading } = useAuth()
  if (loading.value) await fetchSession()
  if (!isAdmin.value) return navigateTo('/attributes')
})
