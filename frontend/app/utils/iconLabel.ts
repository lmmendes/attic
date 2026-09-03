export function getIconLabel(icon: string): string {
  return icon
    .replace(/^i-lucide-/, '')
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}
