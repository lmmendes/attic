export const DEFAULT_CURRENCY = 'USD'

// Recent ISO 4217 codes missing from the backend's table (golang.org/x/text/currency), which rejects them.
const BACKEND_UNSUPPORTED = new Set(['MRU', 'SLE', 'VES', 'XCG', 'ZWG'])

// Every ISO 4217 code the runtime knows that the backend accepts; curate a shortlist if the long menu gets in the way.
export const currencies = Intl.supportedValuesOf('currency').filter(code => !BACKEND_UNSUPPORTED.has(code))
