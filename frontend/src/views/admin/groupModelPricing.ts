export const modelPricingTokenFields = [
  'input_price',
  'output_price',
  'cache_write_price',
  'cache_write_1h_price',
  'cache_read_price',
  'image_input_price',
  'image_output_price'
] as const

export type ModelPricingTokenField = (typeof modelPricingTokenFields)[number]
export type ModelPricingTokenValues = Record<ModelPricingTokenField, number | string | null>
export type OfficialModelPricing = Partial<Record<ModelPricingTokenField, number | null>>

export function findUnsyncedAllowlistModels(
  allowlistModels: string[],
  pricedModels: string[]
): string[] {
  const seen = new Set(
    pricedModels
      .map(model => model.trim().toLowerCase())
      .filter(Boolean)
  )
  const missing: string[] = []

  for (const rawModel of allowlistModels) {
    const model = rawModel.trim()
    const key = model.toLowerCase()
    if (!model || seen.has(key)) continue

    seen.add(key)
    missing.push(model)
  }

  return missing
}

function cleanProduct(value: number): number {
  if (value === 0) return 0
  return Number.parseFloat(value.toPrecision(12))
}

export function inferOfficialPriceMultiplier(
  current: ModelPricingTokenValues,
  official: OfficialModelPricing
): number | null {
  const ratios: number[] = []
  for (const field of modelPricingTokenFields) {
    const officialValue = official[field]
    const currentValue = current[field]
    if (
      officialValue === undefined ||
      officialValue === null ||
      officialValue <= 0 ||
      currentValue === null ||
      currentValue === ''
    ) continue

    const numericCurrent = Number(currentValue)
    if (!Number.isFinite(numericCurrent) || numericCurrent <= 0) return null
    ratios.push(numericCurrent / officialValue)
  }

  if (ratios.length === 0) return null
  const first = ratios[0]
  const tolerance = Math.max(1e-9, Math.abs(first) * 1e-6)
  if (ratios.some(ratio => Math.abs(ratio - first) > tolerance)) return null
  return cleanProduct(first)
}

export function applyOfficialPriceMultiplier(
  current: ModelPricingTokenValues,
  official: OfficialModelPricing,
  multiplier: number
): ModelPricingTokenValues {
  if (!Number.isFinite(multiplier) || multiplier <= 0) return { ...current }

  const next = { ...current }
  for (const field of modelPricingTokenFields) {
    const officialValue = official[field]
    if (officialValue !== undefined && officialValue !== null) {
      next[field] = cleanProduct(officialValue * multiplier)
    }
  }
  return next
}
