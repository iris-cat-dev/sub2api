import { describe, expect, it } from 'vitest'
import { formatCompactNumber, formatCurrency } from '../format'

describe('formatCompactNumber', () => {
  it('formats boundary values with K/M/B', () => {
    expect(formatCompactNumber(0)).toBe('0')
    expect(formatCompactNumber(999)).toBe('999')
    expect(formatCompactNumber(1000)).toBe('1.0K')
    expect(formatCompactNumber(999999)).toBe('1000.0K')
    expect(formatCompactNumber(1000000)).toBe('1.0M')
    expect(formatCompactNumber(1000000000)).toBe('1.0B')
  })

  it('supports disabling billion unit (requests style)', () => {
    expect(formatCompactNumber(1000000000, { allowBillions: false })).toBe('1000.0M')
  })

  it('returns 0 for nullish input', () => {
    expect(formatCompactNumber(null)).toBe('0')
    expect(formatCompactNumber(undefined)).toBe('0')
  })
})

describe('formatCurrency', () => {
  it('defaults platform amounts to the narrow CNY symbol', () => {
    expect(formatCurrency(3)).toBe('¥3.00')
    expect(formatCurrency(null)).toBe('¥0.00')
    expect(formatCurrency(undefined)).toBe('¥0.00')
  })

  it('keeps explicit USD formatting available', () => {
    expect(formatCurrency(3, 'USD')).toBe('$3.00')
    expect(formatCurrency(null, 'USD')).toBe('$0.00')
  })
})
