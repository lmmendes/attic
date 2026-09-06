export const QUANTITY_VALIDATION_MESSAGE = 'Quantity must be a whole number of at least 1'

export function isValidAssetQuantity(quantity: unknown): quantity is number {
  return typeof quantity === 'number'
    && Number.isFinite(quantity)
    && Number.isInteger(quantity)
    && quantity >= 1
}
