import { describe, expect, it } from 'vitest'
import { safeAttributeURL, urlAttributeValue } from '../../app/utils/urlAttributes'

describe('URL attributes', () => {
  it('preserves the destination and optional plain-text label', () => {
    expect(urlAttributeValue({ url: 'https://paperless.local:8000/docs/1', label: '<b>Invoice</b>' }))
      .toEqual({ url: 'https://paperless.local:8000/docs/1', label: '<b>Invoice</b>' })
    expect(urlAttributeValue({ url: 'http://localhost/docs/1' })).toEqual({ url: 'http://localhost/docs/1' })
  })
  it.each(['javascript:alert(1)', 'data:text/html,test', '/docs/1', '//evil.example', 'https:///evil.example', 'https://example.com\\evil', 'https://exa mple.com'])('rejects unsafe destinations: %s', (url) => {
    expect(safeAttributeURL({ url })).toBeUndefined()
  })
  it.each(['http://localhost:8000/docs/1', 'https://paperless.local/docs/1?q=a#preview'])('accepts web destinations: %s', (url) => {
    expect(safeAttributeURL({ url })).toBe(url)
  })
  it.each([null, [], 'https://example.com', { url: 42 }])('handles malformed stored values: %s', (value) => {
    expect(safeAttributeURL(value)).toBeUndefined()
  })
})
