import { describe, expect, it } from 'vitest'

import {
  applyOfficialPriceMultiplier,
  findUnsyncedAllowlistModels,
  inferOfficialPriceMultiplier,
  type ModelPricingTokenValues
} from '../groupModelPricing'

const current: ModelPricingTokenValues = {
  input_price: 9,
  output_price: 8,
  cache_write_price: 7,
  cache_write_1h_price: 6,
  cache_read_price: 5,
  image_input_price: 4,
  image_output_price: 3
}

describe('group model pricing multiplier', () => {
  it('updates every available official token price, including zero', () => {
    expect(applyOfficialPriceMultiplier(current, {
      input_price: 0.1,
      output_price: 0,
      cache_write_price: 0.07,
      image_output_price: 0.1 + 0.2
    }, 1.5)).toEqual({
      ...current,
      input_price: 0.15,
      output_price: 0,
      cache_write_price: 0.105,
      image_output_price: 0.45
    })
  })

  it('preserves user values when the official price is absent', () => {
    expect(applyOfficialPriceMultiplier(current, { input_price: 2 }, 2)).toEqual({
      ...current,
      input_price: 4
    })
  })

  it('infers one consistent multiplier from legacy saved prices', () => {
    expect(inferOfficialPriceMultiplier({
      ...current,
      input_price: 3,
      output_price: 15,
      cache_write_price: 3.75,
      cache_write_1h_price: 6,
      cache_read_price: 0.075,
      image_input_price: 0,
      image_output_price: 0
    }, {
      input_price: 10,
      output_price: 50,
      cache_write_price: 12.5,
      cache_write_1h_price: 20,
      cache_read_price: 0.25,
      image_input_price: 0,
      image_output_price: 0
    })).toBe(0.3)
  })

  it('does not infer a multiplier from inconsistent custom prices', () => {
    expect(inferOfficialPriceMultiplier({
      ...current,
      input_price: 3,
      output_price: 20
    }, {
      input_price: 10,
      output_price: 50
    })).toBeNull()
  })
})

describe('group model pricing whitelist sync', () => {
  it('returns only missing allowlist models and preserves their order', () => {
    expect(findUnsyncedAllowlistModels(
      [' claude-opus-4-1 ', 'CLAUDE-SONNET-4-5', 'claude-haiku-*', 'claude-haiku-*'],
      ['claude-opus-4-1', 'claude-sonnet-4-5']
    )).toEqual(['claude-haiku-*'])
  })

  it('ignores blank allowlist entries', () => {
    expect(findUnsyncedAllowlistModels(['', '   ', 'gpt-5'], [])).toEqual(['gpt-5'])
  })
})
