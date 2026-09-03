export function getLocationNameError(name: string): string | null {
  return name.trim() ? null : 'Location name is required'
}
