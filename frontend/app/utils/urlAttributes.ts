import type { URLAttributeValue } from '~/types/api'

export function urlAttributeValue(value: unknown): URLAttributeValue | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
  const candidate = value as Record<string, unknown>
  if (typeof candidate.url !== 'string') return undefined
  return { url: candidate.url, ...(typeof candidate.label === 'string' ? { label: candidate.label } : {}) }
}

export function safeAttributeURL(value: unknown): string | undefined {
  const destination = urlAttributeValue(value)?.url
  if (!destination || /[\s\\]/.test(destination) || !/^https?:\/\/[^/?#]/i.test(destination)) return undefined
  try {
    const parsed = new URL(destination)
    if (parsed.hostname && ['http:', 'https:'].includes(parsed.protocol)) return destination
  } catch { /* Invalid stored values are displayed as text. */ }
  return undefined
}
