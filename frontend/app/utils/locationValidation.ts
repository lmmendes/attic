/**
 * Validates a location name.
 *
 * @param name - The location name to validate
 * @returns `null` when the trimmed name is non-empty; otherwise, an error message
 */
export function getLocationNameError(name: string): string | null {
  return name.trim() ? null : 'Location name is required'
}
