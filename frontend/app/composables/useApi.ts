import type { UseFetchOptions } from 'nuxt/app'

export function useApi<T>(
  url: string | (() => string),
  options: UseFetchOptions<T> = {}
) {
  const config = useRuntimeConfig()
  const { user } = useOidcAuth()

  const defaults: UseFetchOptions<T> = {
    baseURL: config.public.apiBase as string,
    key: typeof url === 'string' ? url : undefined,
    onRequest({ options }) {
      // Add Bearer token from OIDC session
      const accessToken = user.value?.accessToken
      if (accessToken) {
        options.headers = {
          ...options.headers,
          Authorization: `Bearer ${accessToken}`
        }
      }
    },
    onResponseError({ response }) {
      if (response.status === 401) {
        // Token expired or invalid - redirect to login
        navigateTo('/auth/keycloak/login')
      }
    }
  }

  // Merge options
  const mergedOptions = { ...defaults, ...options }

  return useFetch(url, mergedOptions)
}

export function useApiLazy<T>(
  url: string | (() => string),
  options: UseFetchOptions<T> = {}
) {
  return useApi<T>(url, { ...options, lazy: true })
}

// Composable for making authenticated API mutations (POST, PUT, DELETE)
export function useApiFetch() {
  const config = useRuntimeConfig()
  const { user } = useOidcAuth()

  return async <T>(url: string, options: RequestInit = {}): Promise<T> => {
    const accessToken = user.value?.accessToken
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers
    }

    if (accessToken) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${accessToken}`
    }

    const response = await $fetch<T>(url, {
      baseURL: config.public.apiBase as string,
      ...options,
      headers
    })

    return response
  }
}
