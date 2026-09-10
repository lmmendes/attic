import type { FeatureName } from '~/types/api'

const featureRoutes: Partial<Record<string, FeatureName>> = {
  collections: 'collections',
  plugins: 'plugins',
  conditions: 'conditions',
  locations: 'locations',
  warranties: 'warranties'
}

export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server || to.path === '/login' || to.path === '/configuration') return

  const { isAuthenticated, loading, fetchSession } = useAuth()
  if (loading.value) await fetchSession()
  if (!isAuthenticated.value) return

  const { fetchConfiguration, isFeatureEnabled } = useConfiguration()
  try {
    await fetchConfiguration()
  } catch {
    return
  }

  const section = to.path.split('/')[1]
  if (!section) return
  const feature = featureRoutes[section]
  if (feature && !isFeatureEnabled(feature)) return navigateTo('/')
})
