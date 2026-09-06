export function getDashboardUrls(locationId: string) {
  const locationQuery = locationId === 'all'
    ? ''
    : `location_id=${encodeURIComponent(locationId)}`

  return {
    assets: `/api/assets?limit=4${locationQuery ? `&${locationQuery}` : ''}`,
    stats: `/api/assets/stats${locationQuery ? `?${locationQuery}` : ''}`,
    inventory: `/assets${locationQuery ? `?${locationQuery}` : ''}`
  }
}
