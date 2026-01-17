import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockFetch = vi.fn()

vi.stubGlobal('$fetch', mockFetch)

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetch.mockReset()
  })

  describe('fetchSession', () => {
    it('fetches session and updates state on success', async () => {
      const sessionData = {
        authenticated: true,
        user: { email: 'test@example.com', name: 'Test User', role: 'user' as const }
      }
      mockFetch.mockResolvedValueOnce(sessionData)

      const { useAuth } = await import('../../app/composables/useAuth')
      const { fetchSession, session, loading } = useAuth()

      await fetchSession()

      expect(mockFetch).toHaveBeenCalledWith('/auth/session', expect.objectContaining({
        credentials: 'include'
      }))
      expect(session.value).toEqual(sessionData)
      expect(loading.value).toBe(false)
    })

    it('sets authenticated to false on fetch error', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'))

      const { useAuth } = await import('../../app/composables/useAuth')
      const { fetchSession, session, loading } = useAuth()

      await fetchSession()

      expect(session.value).toEqual({ authenticated: false })
      expect(loading.value).toBe(false)
    })
  })

  describe('loginWithCredentials', () => {
    it('returns success and fetches session on successful login', async () => {
      const loginResponse = { success: true, user: { email: 'test@example.com', name: 'Test' } }
      const sessionData = { authenticated: true, user: loginResponse.user }

      mockFetch
        .mockResolvedValueOnce(loginResponse)
        .mockResolvedValueOnce(sessionData)

      const { useAuth } = await import('../../app/composables/useAuth')
      const { loginWithCredentials } = useAuth()

      const result = await loginWithCredentials({ email: 'test@example.com', password: 'password' })

      expect(result.success).toBe(true)
      expect(mockFetch).toHaveBeenCalledWith('/auth/login', expect.objectContaining({
        method: 'POST',
        body: { email: 'test@example.com', password: 'password' },
        credentials: 'include'
      }))
    })

    it('returns error on failed login', async () => {
      mockFetch.mockRejectedValueOnce({ data: { error: 'Invalid credentials' } })

      const { useAuth } = await import('../../app/composables/useAuth')
      const { loginWithCredentials } = useAuth()

      const result = await loginWithCredentials({ email: 'test@example.com', password: 'wrong' })

      expect(result.success).toBe(false)
      expect(result.error).toBe('Invalid credentials')
    })
  })

  describe('logout', () => {
    it('calls logout endpoint and resets session', async () => {
      mockFetch.mockResolvedValueOnce({})

      const { useAuth } = await import('../../app/composables/useAuth')
      const { logout, session } = useAuth()

      session.value = { authenticated: true, user: { email: 'test@example.com', name: 'Test' } }

      await logout()

      expect(mockFetch).toHaveBeenCalledWith('/auth/logout', expect.objectContaining({
        method: 'POST',
        credentials: 'include'
      }))
      expect(session.value).toEqual({ authenticated: false })
    })
  })

  describe('computed properties', () => {
    it('isAuthenticated returns correct value based on session', async () => {
      const { useAuth } = await import('../../app/composables/useAuth')
      const { isAuthenticated, session } = useAuth()

      session.value = { authenticated: true }
      expect(isAuthenticated.value).toBe(true)

      session.value = { authenticated: false }
      expect(isAuthenticated.value).toBe(false)

      session.value = null
      expect(isAuthenticated.value).toBe(false)
    })

    it('isAdmin returns true when user role is admin', async () => {
      const { useAuth } = await import('../../app/composables/useAuth')
      const { isAdmin, session } = useAuth()

      session.value = {
        authenticated: true,
        user: { email: 'admin@example.com', name: 'Admin', role: 'admin' }
      }
      expect(isAdmin.value).toBe(true)

      session.value = {
        authenticated: true,
        user: { email: 'user@example.com', name: 'User', role: 'user' }
      }
      expect(isAdmin.value).toBe(false)
    })

    it('isOIDCEnabled returns oidc_enabled value from session', async () => {
      const { useAuth } = await import('../../app/composables/useAuth')
      const { isOIDCEnabled, session } = useAuth()

      session.value = { authenticated: false, oidc_enabled: true }
      expect(isOIDCEnabled.value).toBe(true)

      session.value = { authenticated: false, oidc_enabled: false }
      expect(isOIDCEnabled.value).toBe(false)
    })

    it('user computed returns user from session', async () => {
      const { useAuth } = await import('../../app/composables/useAuth')
      const { user, session } = useAuth()

      const testUser = { email: 'test@example.com', name: 'Test User' }
      session.value = { authenticated: true, user: testUser }
      expect(user.value).toEqual(testUser)

      session.value = { authenticated: false }
      expect(user.value).toBeNull()
    })
  })

  describe('changePassword', () => {
    it('returns success on successful password change', async () => {
      mockFetch.mockResolvedValueOnce({})

      const { useAuth } = await import('../../app/composables/useAuth')
      const { changePassword } = useAuth()

      const result = await changePassword('oldPassword', 'newPassword')

      expect(result.success).toBe(true)
      expect(mockFetch).toHaveBeenCalledWith('/api/auth/password', expect.objectContaining({
        method: 'PUT',
        body: { current_password: 'oldPassword', new_password: 'newPassword' },
        credentials: 'include'
      }))
    })

    it('returns error on failed password change', async () => {
      mockFetch.mockRejectedValueOnce({ data: { error: 'Current password is incorrect' } })

      const { useAuth } = await import('../../app/composables/useAuth')
      const { changePassword } = useAuth()

      const result = await changePassword('wrongPassword', 'newPassword')

      expect(result.success).toBe(false)
      expect(result.error).toBe('Current password is incorrect')
    })
  })
})
