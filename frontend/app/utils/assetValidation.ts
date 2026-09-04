export const QUANTITY_VALIDATION_MESSAGE = 'Quantity must be a whole number of at least 1'

/**
 * Determines whether a value is a valid asset quantity.
 *
 * @param quantity - The value to validate
 * @returns `true` if the value is a finite integer greater than or equal to 1, `false` otherwise.
 */
export function isValidAssetQuantity(quantity: unknown): quantity is number {
  return typeof quantity === 'number'
    && Number.isFinite(quantity)
    && Number.isInteger(quantity)
    && quantity >= 1
}
