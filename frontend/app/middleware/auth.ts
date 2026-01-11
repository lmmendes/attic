export default defineNuxtRouteMiddleware((to) => {
  const { loggedIn, login } = useOidcAuth()

  if (!loggedIn.value) {
    // Redirect to Keycloak login
    return login()
  }
})
