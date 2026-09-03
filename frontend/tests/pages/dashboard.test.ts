import { describe, expect, it } from 'vitest'

describe('Dashboard location filter', () => {
  const getDashboardUrls = (locationId: string) => {
    const locationQuery = locationId === 'all'
      ? ''
      : `location_id=${encodeURIComponent(locationId)}`

    return {
      assets: `/api/assets?limit=4${locationQuery ? `&${locationQuery}` : ''}`,
      stats: `/api/assets/stats${locationQuery ? `?${locationQuery}` : ''}`,
      inventory: `/assets${locationQuery ? `?${locationQuery}` : ''}`
    }
  }

  it('uses the unfiltered dashboard endpoints for all locations', () => {
    expect(getDashboardUrls('all')).toEqual({
      assets: '/api/assets?limit=4',
      stats: '/api/assets/stats',
      inventory: '/assets'
    })
  })

  it('applies the selected location to dashboard data and inventory links', () => {
    expect(getDashboardUrls('location/with spaces')).toEqual({
      assets: '/api/assets?limit=4&location_id=location%2Fwith%20spaces',
      stats: '/api/assets/stats?location_id=location%2Fwith%20spaces',
      inventory: '/assets?location_id=location%2Fwith%20spaces'
    })
  })
})
