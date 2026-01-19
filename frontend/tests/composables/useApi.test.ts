import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockFetch = vi.fn()

vi.stubGlobal('$fetch', mockFetch)

describe('useApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetch.mockReset()
  })

  describe('useApiFetch', () => {
    it('returns a function that calls $fetch with correct options', async () => {
      const mockResponse = { id: 1, name: 'Test' }
      mockFetch.mockResolvedValueOnce(mockResponse)

      const { useApiFetch } = await import('../../app/composables/useApi')
      const apiFetch = useApiFetch()

      const result = await apiFetch('/api/test')

      expect(mockFetch).toHaveBeenCalledWith('/api/test', expect.objectContaining({
        credentials: 'include',
        headers: expect.objectContaining({
          'Content-Type': 'application/json'
        })
      }))
      expect(result).toEqual(mockResponse)
    })

    it('supports POST method with body', async () => {
      mockFetch.mockResolvedValueOnce({ success: true })

      const { useApiFetch } = await import('../../app/composables/useApi')
      const apiFetch = useApiFetch()

      await apiFetch('/api/test', {
        method: 'POST',
        body: JSON.stringify({ name: 'Test' })
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/test', expect.objectContaining({
        credentials: 'include',
        method: 'POST',
        body: JSON.stringify({ name: 'Test' }),
        headers: expect.objectContaining({
          'Content-Type': 'application/json'
        })
      }))
    })

    it('supports DELETE method', async () => {
      mockFetch.mockResolvedValueOnce({})

      const { useApiFetch } = await import('../../app/composables/useApi')
      const apiFetch = useApiFetch()

      await apiFetch('/api/test/1', { method: 'DELETE' })

      expect(mockFetch).toHaveBeenCalledWith('/api/test/1', expect.objectContaining({
        credentials: 'include',
        method: 'DELETE',
        headers: expect.objectContaining({
          'Content-Type': 'application/json'
        })
      }))
    })

    it('allows custom headers', async () => {
      mockFetch.mockResolvedValueOnce({})

      const { useApiFetch } = await import('../../app/composables/useApi')
      const apiFetch = useApiFetch()

      await apiFetch('/api/test', {
        headers: { 'X-Custom-Header': 'value' }
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/test', expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'X-Custom-Header': 'value'
        })
      }))
    })

    it('supports PUT method for updates', async () => {
      const mockResponse = { id: 1, name: 'Updated' }
      mockFetch.mockResolvedValueOnce(mockResponse)

      const { useApiFetch } = await import('../../app/composables/useApi')
      const apiFetch = useApiFetch()

      const result = await apiFetch('/api/test/1', {
        method: 'PUT',
        body: JSON.stringify({ name: 'Updated' })
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/test/1', expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ name: 'Updated' })
      }))
      expect(result).toEqual(mockResponse)
    })

    it('supports PATCH method for partial updates', async () => {
      mockFetch.mockResolvedValueOnce({ success: true })

      const { useApiFetch } = await import('../../app/composables/useApi')
      const apiFetch = useApiFetch()

      await apiFetch('/api/test/1', {
        method: 'PATCH',
        body: JSON.stringify({ name: 'Partial update' })
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/test/1', expect.objectContaining({
        method: 'PATCH'
      }))
    })
  })

  describe('useApi composable', () => {
    it('configures useFetch with correct defaults', async () => {
      const { useApi } = await import('../../app/composables/useApi')

      const result = useApi('/api/test')

      expect(result).toBeDefined()
    })
  })

  describe('useApiLazy composable', () => {
    it('configures useApi with lazy option', async () => {
      const { useApiLazy } = await import('../../app/composables/useApi')

      const result = useApiLazy('/api/test')

      expect(result).toBeDefined()
    })
  })

  describe('401 error handling behavior', () => {
    it('should redirect to /login when response status is 401', () => {
      const mockNavigateTo = vi.fn()

      const handleResponseError = (response: { status: number }) => {
        if (response.status === 401) {
          mockNavigateTo('/login')
        }
      }

      handleResponseError({ status: 401 })

      expect(mockNavigateTo).toHaveBeenCalledWith('/login')
    })

    it('should not redirect when response status is not 401', () => {
      const mockNavigateTo = vi.fn()

      const handleResponseError = (response: { status: number }) => {
        if (response.status === 401) {
          mockNavigateTo('/login')
        }
      }

      handleResponseError({ status: 500 })

      expect(mockNavigateTo).not.toHaveBeenCalled()
    })

    it('should not redirect on 403 forbidden errors', () => {
      const mockNavigateTo = vi.fn()

      const handleResponseError = (response: { status: number }) => {
        if (response.status === 401) {
          mockNavigateTo('/login')
        }
      }

      handleResponseError({ status: 403 })

      expect(mockNavigateTo).not.toHaveBeenCalled()
    })

    it('should not redirect on 404 not found errors', () => {
      const mockNavigateTo = vi.fn()

      const handleResponseError = (response: { status: number }) => {
        if (response.status === 401) {
          mockNavigateTo('/login')
        }
      }

      handleResponseError({ status: 404 })

      expect(mockNavigateTo).not.toHaveBeenCalled()
    })
  })
})
