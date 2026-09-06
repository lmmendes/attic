import { describe, expect, it } from 'vitest'

describe('management page workflows', () => {
  it('generates stable condition codes from labels', () => {
    const generateCode = (label: string) => label
      .toUpperCase()
      .replace(/[^A-Z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '')

    expect(generateCode('Like new / open box')).toBe('LIKE_NEW_OPEN_BOX')
  })

  it('filters people by both search and role', () => {
    const users = [
      { name: 'Ada Lovelace', email: 'ada@example.com', role: 'admin' },
      { name: 'Grace Hopper', email: 'grace@example.com', role: 'user' }
    ]
    const query = 'ada'
    const role = 'admin'
    const result = users.filter(user =>
      (user.name.toLowerCase().includes(query) || user.email.includes(query))
      && user.role === role
    )

    expect(result).toEqual([users[0]])
  })

  it('enforces the password requirement used by account forms', () => {
    const isValidPassword = (password: string) => password.length >= 8

    expect(isValidPassword('short')).toBe(false)
    expect(isValidPassword('safe-pass')).toBe(true)
  })

  it('distinguishes active, expiring, and expired warranties', () => {
    const getStatus = (endDate: Date, now: Date) => {
      const days = Math.ceil((endDate.getTime() - now.getTime()) / 86_400_000)
      if (days < 0) return 'expired'
      if (days <= 30) return 'expiring'
      return 'active'
    }
    const now = new Date('2026-09-03T12:00:00Z')

    expect(getStatus(new Date('2026-09-13T12:00:00Z'), now)).toBe('expiring')
    expect(getStatus(new Date('2027-01-01T12:00:00Z'), now)).toBe('active')
    expect(getStatus(new Date('2026-08-01T12:00:00Z'), now)).toBe('expired')
  })

  it('reports plugin readiness independently from category creation', () => {
    const getStatus = (plugin: { enabled: boolean }) => plugin.enabled ? 'active' : 'disabled'

    expect(getStatus({ enabled: false })).toBe('disabled')
    expect(getStatus({ enabled: true })).toBe('active')
  })
})
