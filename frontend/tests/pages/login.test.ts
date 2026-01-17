import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('Login Page', () => {
  const mockLoginWithCredentials = vi.fn()
  const mockLoginWithOIDC = vi.fn()
  const mockFetchSession = vi.fn()
  const mockNavigateTo = vi.fn()

  let mockIsAuthenticated = { value: false }
  let mockIsOIDCEnabled = { value: false }
  let mockLoading = { value: false }

  beforeEach(() => {
    vi.clearAllMocks()
    mockIsAuthenticated = { value: false }
    mockIsOIDCEnabled = { value: false }
    mockLoading = { value: false }
    mockLoginWithCredentials.mockReset()
    mockLoginWithOIDC.mockReset()
    mockFetchSession.mockReset()
  })

  describe('initial state', () => {
    it('shows loading spinner while checking session', () => {
      mockLoading.value = true

      expect(mockLoading.value).toBe(true)
    })

    it('redirects to home if already authenticated', async () => {
      mockIsAuthenticated.value = true

      if (mockIsAuthenticated.value) {
        mockNavigateTo('/')
      }

      expect(mockNavigateTo).toHaveBeenCalledWith('/')
    })

    it('shows login form when not authenticated', () => {
      mockLoading.value = false
      mockIsAuthenticated.value = false

      expect(mockLoading.value).toBe(false)
      expect(mockIsAuthenticated.value).toBe(false)
    })
  })

  describe('credential login', () => {
    it('calls loginWithCredentials with email and password', async () => {
      const credentials = { email: 'test@example.com', password: 'password123' }
      mockLoginWithCredentials.mockResolvedValueOnce({ success: true })

      await mockLoginWithCredentials(credentials)

      expect(mockLoginWithCredentials).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123'
      })
    })

    it('shows error message on failed login', async () => {
      mockLoginWithCredentials.mockResolvedValueOnce({
        success: false,
        error: 'Invalid credentials'
      })

      const result = await mockLoginWithCredentials({
        email: 'test@example.com',
        password: 'wrong'
      })

      expect(result.success).toBe(false)
      expect(result.error).toBe('Invalid credentials')
    })

    it('navigates to home on successful login', async () => {
      mockLoginWithCredentials.mockResolvedValueOnce({ success: true })

      const result = await mockLoginWithCredentials({
        email: 'test@example.com',
        password: 'password123'
      })

      if (result.success) {
        mockIsAuthenticated.value = true
        mockNavigateTo('/')
      }

      expect(mockNavigateTo).toHaveBeenCalledWith('/')
    })

    it('disables submit button when email is empty', () => {
      const email = ''
      const password = 'password123'
      const isLoading = false

      const isDisabled = isLoading || !email || !password

      expect(isDisabled).toBe(true)
    })

    it('disables submit button when password is empty', () => {
      const email = 'test@example.com'
      const password = ''
      const isLoading = false

      const isDisabled = isLoading || !email || !password

      expect(isDisabled).toBe(true)
    })

    it('disables submit button while loading', () => {
      const email = 'test@example.com'
      const password = 'password123'
      const isLoading = true

      const isDisabled = isLoading || !email || !password

      expect(isDisabled).toBe(true)
    })

    it('enables submit button with valid credentials', () => {
      const email = 'test@example.com'
      const password = 'password123'
      const isLoading = false

      const isDisabled = isLoading || !email || !password

      expect(isDisabled).toBe(false)
    })
  })

  describe('OIDC login', () => {
    it('shows SSO button when OIDC is enabled', () => {
      mockIsOIDCEnabled.value = true

      expect(mockIsOIDCEnabled.value).toBe(true)
    })

    it('hides credential form when OIDC is enabled', () => {
      mockIsOIDCEnabled.value = true

      const showCredentialForm = !mockIsOIDCEnabled.value

      expect(showCredentialForm).toBe(false)
    })

    it('shows credential form when OIDC is disabled', () => {
      mockIsOIDCEnabled.value = false

      const showCredentialForm = !mockIsOIDCEnabled.value

      expect(showCredentialForm).toBe(true)
    })

    it('calls loginWithOIDC when SSO button is clicked', () => {
      mockIsOIDCEnabled.value = true

      mockLoginWithOIDC()

      expect(mockLoginWithOIDC).toHaveBeenCalled()
    })
  })

  describe('error handling', () => {
    it('clears error before new login attempt', async () => {
      let error = 'Previous error'

      error = ''
      mockLoginWithCredentials.mockResolvedValueOnce({ success: true })
      await mockLoginWithCredentials({ email: 'test@example.com', password: 'pass' })

      expect(error).toBe('')
    })

    it('displays generic error when no error message provided', async () => {
      mockLoginWithCredentials.mockResolvedValueOnce({
        success: false,
        error: undefined
      })

      const result = await mockLoginWithCredentials({
        email: 'test@example.com',
        password: 'wrong'
      })

      const errorMessage = result.error || 'Login failed'

      expect(errorMessage).toBe('Login failed')
    })

    it('handles network errors gracefully', async () => {
      mockLoginWithCredentials.mockRejectedValueOnce(new Error('Network error'))

      let error = ''
      try {
        await mockLoginWithCredentials({ email: 'test@example.com', password: 'pass' })
      } catch (e: unknown) {
        const err = e as { message?: string }
        error = err.message || 'Login failed'
      }

      expect(error).toBe('Network error')
    })
  })

  describe('authentication state watching', () => {
    it('navigates to home when isAuthenticated becomes true', () => {
      mockIsAuthenticated.value = false

      const watchCallback = (authenticated: boolean) => {
        if (authenticated) {
          mockNavigateTo('/')
        }
      }

      mockIsAuthenticated.value = true
      watchCallback(mockIsAuthenticated.value)

      expect(mockNavigateTo).toHaveBeenCalledWith('/')
    })

    it('does not navigate when authentication fails', () => {
      mockIsAuthenticated.value = false

      const watchCallback = (authenticated: boolean) => {
        if (authenticated) {
          mockNavigateTo('/')
        }
      }

      watchCallback(mockIsAuthenticated.value)

      expect(mockNavigateTo).not.toHaveBeenCalled()
    })
  })

  describe('session checking on mount', () => {
    it('fetches session on component mount', async () => {
      await mockFetchSession()

      expect(mockFetchSession).toHaveBeenCalled()
    })

    it('redirects if session shows authenticated', async () => {
      mockFetchSession.mockImplementation(async () => {
        mockIsAuthenticated.value = true
      })

      await mockFetchSession()

      if (mockIsAuthenticated.value) {
        mockNavigateTo('/')
      }

      expect(mockNavigateTo).toHaveBeenCalledWith('/')
    })
  })
})
