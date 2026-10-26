export function getLocationNameError(name: string): string | null {
  return name.trim() ? null : 'Location name is required'
}

export const maxLocationPhotoSize = 10 * 1024 * 1024

export function getLocationPhotoError(file: { size: number, type: string }): string | null {
  if (file.size > maxLocationPhotoSize) return 'Choose a photo smaller than 10 MB.'
  if (file.type && !['image/jpeg', 'image/png', 'image/gif'].includes(file.type)) return 'Choose a JPEG, PNG or GIF photo.'
  return null
}
