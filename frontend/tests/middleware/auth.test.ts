import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('Auth Middleware', () => {
  const mockNavigateTo = vi.fn()
  const mockFetchSession = vi.fn()
  let mockIsAuthenticated = { value: false }
  let mockLoading = { value: true }

  beforeEach(() => {
    vi.clearAllMocks()
    mockIsAuthenticated = { value: false }
    mockLoading = { value: true }
  })

  const createMiddleware = () => {
    return async (to: { path: string }) => {
      if (to.path === '/login') return

      if (mockLoading.value) {
        await mockFetchSession()
      }

      if (!mockIsAuthenticated.value) {
        return mockNavigateTo('/login')
      }
    }
  }

  describe('route protection', () => {
    it('allows access to login page without authentication', async () => {
      const middleware = createMiddleware()

      await middleware({ path: '/login' })

      expect(mockNavigateTo).not.toHaveBeenCalled()
      expect(mockFetchSession).not.toHaveBeenCalled()
    })

    it('redirects to login when user is not authenticated', async () => {
      mockLoading.value = false
      mockIsAuthenticated.value = false
      const middleware = createMiddleware()

      await middleware({ path: '/dashboard' })

      expect(mockNavigateTo).toHaveBeenCalledWith('/login')
    })

    it('allows access when user is authenticated', async () => {
      mockLoading.value = false
      mockIsAuthenticated.value = true
      const middleware = createMiddleware()

      await middleware({ path: '/dashboard' })

      expect(mockNavigateTo).not.toHaveBeenCalled()
    })

    it('fetches session when loading is true', async () => {
      mockLoading.value = true
      mockIsAuthenticated.value = false
      const middleware = createMiddleware()

      await middleware({ path: '/dashboard' })

      expect(mockFetchSession).toHaveBeenCalled()
    })

    it('does not fetch session when already loaded', async () => {
      mockLoading.value = false
      mockIsAuthenticated.value = true
      const middleware = createMiddleware()

      await middleware({ path: '/dashboard' })

      expect(mockFetchSession).not.toHaveBeenCalled()
    })
  })

  describe('protected routes', () => {
    const protectedRoutes = [
      '/',
      '/assets',
      '/assets/new',
      '/assets/123',
      '/assets/123/edit',
      '/categories',
      '/categories/new',
      '/locations',
      '/attributes',
      '/conditions',
      '/warranties',
      '/users',
      '/plugins'
    ]

    it.each(protectedRoutes)('protects route: %s', async (route) => {
      mockLoading.value = false
      mockIsAuthenticated.value = false
      const middleware = createMiddleware()

      await middleware({ path: route })

      expect(mockNavigateTo).toHaveBeenCalledWith('/login')
    })

    it.each(protectedRoutes)('allows authenticated access to: %s', async (route) => {
      mockLoading.value = false
      mockIsAuthenticated.value = true
      const middleware = createMiddleware()

      await middleware({ path: route })

      expect(mockNavigateTo).not.toHaveBeenCalled()
    })
  })

  describe('session loading behavior', () => {
    it('waits for session fetch before checking auth', async () => {
      const callOrder: string[] = []

      mockLoading.value = true
      mockFetchSession.mockImplementation(async () => {
        callOrder.push('fetchSession')
        mockLoading.value = false
        mockIsAuthenticated.value = true
      })

      const middleware = createMiddleware()
      await middleware({ path: '/dashboard' })

      expect(callOrder).toContain('fetchSession')
      expect(mockNavigateTo).not.toHaveBeenCalled()
    })

    it('redirects after session fetch if not authenticated', async () => {
      mockLoading.value = true
      mockFetchSession.mockImplementation(async () => {
        mockLoading.value = false
        mockIsAuthenticated.value = false
      })

      const middleware = createMiddleware()
      await middleware({ path: '/dashboard' })

      expect(mockFetchSession).toHaveBeenCalled()
      expect(mockNavigateTo).toHaveBeenCalledWith('/login')
    })
  })
})
