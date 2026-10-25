import { describe, expect, it } from 'vitest'
import { formatPurchasePrice, formatPurchaseTotal } from '../../app/utils/currency'

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
