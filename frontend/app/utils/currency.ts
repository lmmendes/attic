import registry from '../data/currencies.json'

export const DEFAULT_CURRENCY = 'USD'

// The backend uses the same ISO 4217 snapshot; a parity test prevents drift.
// ISO reserves XXX for no currency and XTS for testing, not asset prices.
export const currencies = registry.codes.filter(code => code !== 'XXX' && code !== 'XTS')

export function formatPurchasePrice(value: number, currency: string): string {
  const { minimumFractionDigits = 2, maximumFractionDigits = 2 } = new Intl.NumberFormat('en-US', {
    style: 'currency', currency
  }).resolvedOptions()
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency,
    minimumFractionDigits: Math.max(2, minimumFractionDigits),
    maximumFractionDigits: Math.max(2, maximumFractionDigits)
  }).format(value)
}

// Whole totals stay concise; fractional totals retain the currency's precision.
export function formatPurchaseTotal(value: number, currency: string): string {
  const { maximumFractionDigits = 2 } = new Intl.NumberFormat('en-US', {
    style: 'currency', currency
  }).resolvedOptions()
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: Math.max(2, maximumFractionDigits)
  }).format(value)
}
