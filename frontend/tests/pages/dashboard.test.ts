import { describe, expect, it } from 'vitest'
import { getDashboardUrls } from '../../app/utils/dashboardUrls'

describe('Dashboard location filter', () => {
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
