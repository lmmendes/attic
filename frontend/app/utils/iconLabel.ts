/**
 * Converts a Lucide icon identifier into a human-readable label.
 *
 * @param icon - The icon identifier with an optional `i-lucide-` prefix
 * @returns The icon name with hyphen-separated words capitalized and joined by spaces
 */
export function getIconLabel(icon: string): string {
  return icon
    .replace(/^i-lucide-/, '')
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}
