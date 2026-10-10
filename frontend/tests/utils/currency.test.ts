import { describe, expect, it } from 'vitest'
import { currencies } from '../../app/utils/currency'

describe('currency picker', () => {
  it('offers CZK, EUR and USD', () => {
    expect(currencies).toEqual(expect.arrayContaining(['CZK', 'EUR', 'USD']))
  })

  it('hides codes the backend rejects', () => {
    for (const code of ['MRU', 'SLE', 'VES', 'XCG', 'ZWG']) expect(currencies).not.toContain(code)
  })
})
