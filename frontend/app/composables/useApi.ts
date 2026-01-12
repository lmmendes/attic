import type { UseFetchOptions } from 'nuxt/app'

export function useApi<T>(
  url: string | (() => string),
  options: UseFetchOptions<T> = {}
) {
  const config = useRuntimeConfig()

  const defaults: UseFetchOptions<T> = {
    baseURL: config.public.apiBase as string,
    key: typeof url === 'string' ? url : undefined,
    credentials: 'include', // Include cookies for auth
    onResponseError({ response }) {
      if (response.status === 401) {
        // Token expired or invalid - redirect to login
        window.location.href = `${config.public.apiBase}/auth/login`
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

type HttpMethod = 'GET' | 'HEAD' | 'PATCH' | 'POST' | 'PUT' | 'DELETE' | 'CONNECT' | 'OPTIONS' | 'TRACE'

interface ApiFetchOptions extends Omit<RequestInit, 'method'> {
  method?: HttpMethod
}

// Composable for making authenticated API mutations (POST, PUT, DELETE)
export function useApiFetch() {
  const config = useRuntimeConfig()

  return async <T>(url: string, options: ApiFetchOptions = {}): Promise<T> => {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers
    }

    const response = await $fetch<T>(url, {
      baseURL: config.public.apiBase as string,
      credentials: 'include', // Include cookies for auth
      ...options,
      headers
    })

    return response
  }
}
