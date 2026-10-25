import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { currencies, formatPurchasePrice, formatPurchaseTotal } from '../../app/utils/currency'
import registry from '../../app/data/currencies.json'

describe('currency picker', () => {
  it('uses the published registry shared with the backend', () => {
    const backendRegistry = JSON.parse(readFileSync(resolve(process.cwd(), '../backend/internal/domain/currencies.json'), 'utf8'))
    expect(registry).toEqual(backendRegistry)
    expect(currencies).toEqual(registry.codes.filter(code => code !== 'XXX' && code !== 'XTS'))
    expect(currencies).not.toContain('XXX')
    expect(currencies).not.toContain('XTS')
    expect(currencies).toEqual(expect.arrayContaining(['CZK', 'EUR', 'USD', 'MRU', 'SLE', 'VES', 'XCG', 'ZWG']))
    expect(new Set(currencies).size).toBe(currencies.length)
    for (const currency of currencies) {
      expect(currency).toMatch(/^[A-Z]{3}$/)
      expect(() => formatPurchasePrice(12, currency)).not.toThrow()
    }
  })
})

describe('purchase price formatting', () => {
  it('always shows two decimals for zero-decimal currencies', () => {
    expect(formatPurchasePrice(12, 'JPY')).toBe('¥12.00')
    expect(formatPurchasePrice(12.5, 'JPY')).toBe('¥12.50')
    expect(formatPurchasePrice(0, 'JPY')).toBe('¥0.00')
  })
  it('preserves standard and higher currency precision', () => {
    expect(formatPurchasePrice(12, 'USD')).toBe('$12.00')
    expect(formatPurchasePrice(12, 'EUR')).toBe('€12.00')
    expect(formatPurchasePrice(12, 'BHD')).toBe(new Intl.NumberFormat('en-US', { style: 'currency', currency: 'BHD' }).format(12))
  })
})

describe('purchase total formatting', () => {
  it('keeps fractional amounts rather than rounding to a different whole amount', () => {
    expect(formatPurchaseTotal(0.5, 'USD')).toBe('$0.5')
    expect(formatPurchaseTotal(0.5, 'EUR')).toBe('€0.5')
    expect(formatPurchaseTotal(0.25, 'KWD')).toBe('KWD\u00a00.25')
    expect(formatPurchaseTotal(0.251, 'BHD')).toBe('BHD\u00a00.251')
    expect(formatPurchaseTotal(12, 'USD')).toBe('$12')
    expect(formatPurchaseTotal(0, 'JPY')).toBe('¥0')
  })
})
