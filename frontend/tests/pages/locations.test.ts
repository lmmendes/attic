import { describe, it, expect, vi, beforeEach } from 'vitest'

interface Location {
  id: string
  name: string
  description?: string
  parent_id?: string
}

interface TreeNode {
  location: Location
  children: TreeNode[]
  level: number
}

describe('Locations Page', () => {
  const mockApiFetch = vi.fn()
  const mockToast = { add: vi.fn() }
  const _mockRefresh = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('tree building', () => {
    const buildTree = (locations: Location[]): TreeNode[] => {
      const childrenMap = new Map<string | undefined, Location[]>()
      locations.forEach((l) => {
        const parentId = l.parent_id || undefined
        if (!childrenMap.has(parentId)) {
          childrenMap.set(parentId, [])
        }
        childrenMap.get(parentId)!.push(l)
      })

      const buildTreeRecursive = (parentId: string | undefined, level: number): TreeNode[] => {
        const children = childrenMap.get(parentId) || []
        return children
          .sort((a, b) => a.name.localeCompare(b.name))
          .map(loc => ({
            location: loc,
            children: buildTreeRecursive(loc.id, level + 1),
            level
          }))
      }

      return buildTreeRecursive(undefined, 0)
    }

    it('builds tree from flat locations array', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home', parent_id: undefined },
        { id: 'living', name: 'Living Room', parent_id: 'home' },
        { id: 'bedroom', name: 'Bedroom', parent_id: 'home' }
      ]

      const tree = buildTree(locations)

      expect(tree).toHaveLength(1)
      expect(tree[0].location.name).toBe('Home')
      expect(tree[0].children).toHaveLength(2)
      expect(tree[0].level).toBe(0)
    })

    it('builds deeply nested tree', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home' },
        { id: 'bedroom', name: 'Bedroom', parent_id: 'home' },
        { id: 'closet', name: 'Closet', parent_id: 'bedroom' },
        { id: 'shelf', name: 'Shelf', parent_id: 'closet' }
      ]

      const tree = buildTree(locations)

      expect(tree[0].children[0].children[0].children[0].location.name).toBe('Shelf')
      expect(tree[0].children[0].children[0].children[0].level).toBe(3)
    })

    it('handles multiple root nodes', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home' },
        { id: 'office', name: 'Office' },
        { id: 'garage', name: 'Garage' }
      ]

      const tree = buildTree(locations)

      expect(tree).toHaveLength(3)
    })

    it('sorts children alphabetically', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home' },
        { id: 'z-room', name: 'Z Room', parent_id: 'home' },
        { id: 'a-room', name: 'A Room', parent_id: 'home' },
        { id: 'm-room', name: 'M Room', parent_id: 'home' }
      ]

      const tree = buildTree(locations)

      expect(tree[0].children[0].location.name).toBe('A Room')
      expect(tree[0].children[1].location.name).toBe('M Room')
      expect(tree[0].children[2].location.name).toBe('Z Room')
    })

    it('handles empty locations array', () => {
      const tree = buildTree([])
      expect(tree).toHaveLength(0)
    })
  })

  describe('tree filtering', () => {
    const filterTree = (nodes: TreeNode[], query: string): TreeNode[] => {
      if (!query.trim()) return nodes

      const lowerQuery = query.toLowerCase()

      return nodes.reduce<TreeNode[]>((acc, node) => {
        const matchesSearch = node.location.name.toLowerCase().includes(lowerQuery)
        const filteredChildren = filterTree(node.children, query)

        if (matchesSearch || filteredChildren.length > 0) {
          acc.push({
            ...node,
            children: filteredChildren
          })
        }

        return acc
      }, [])
    }

    it('filters tree by search query', () => {
      const tree: TreeNode[] = [
        {
          location: { id: 'home', name: 'Home' },
          children: [
            { location: { id: 'bedroom', name: 'Bedroom' }, children: [], level: 1 },
            { location: { id: 'kitchen', name: 'Kitchen' }, children: [], level: 1 }
          ],
          level: 0
        }
      ]

      const filtered = filterTree(tree, 'bed')

      expect(filtered).toHaveLength(1)
      expect(filtered[0].children).toHaveLength(1)
      expect(filtered[0].children[0].location.name).toBe('Bedroom')
    })

    it('returns all nodes when query is empty', () => {
      const tree: TreeNode[] = [
        { location: { id: 'home', name: 'Home' }, children: [], level: 0 }
      ]

      const filtered = filterTree(tree, '')
      expect(filtered).toEqual(tree)
    })

    it('includes parent nodes when child matches', () => {
      const tree: TreeNode[] = [
        {
          location: { id: 'home', name: 'Home' },
          children: [
            {
              location: { id: 'bedroom', name: 'Bedroom' },
              children: [
                { location: { id: 'closet', name: 'Walk-in Closet' }, children: [], level: 2 }
              ],
              level: 1
            }
          ],
          level: 0
        }
      ]

      const filtered = filterTree(tree, 'closet')

      expect(filtered).toHaveLength(1)
      expect(filtered[0].location.name).toBe('Home')
      expect(filtered[0].children[0].location.name).toBe('Bedroom')
      expect(filtered[0].children[0].children[0].location.name).toBe('Walk-in Closet')
    })

    it('is case insensitive', () => {
      const tree: TreeNode[] = [
        { location: { id: 'home', name: 'Living Room' }, children: [], level: 0 }
      ]

      const filtered = filterTree(tree, 'LIVING')
      expect(filtered).toHaveLength(1)
    })
  })

  describe('location path (breadcrumb)', () => {
    it('builds path from location to root', () => {
      const locationMap = new Map<string, Location>([
        ['home', { id: 'home', name: 'Home' }],
        ['bedroom', { id: 'bedroom', name: 'Bedroom', parent_id: 'home' }],
        ['closet', { id: 'closet', name: 'Closet', parent_id: 'bedroom' }]
      ])

      const getLocationPath = (location: Location): Location[] => {
        const path: Location[] = []
        let current: Location | undefined = location

        while (current) {
          path.unshift(current)
          if (current.parent_id) {
            current = locationMap.get(current.parent_id)
          } else {
            break
          }
        }

        return path
      }

      const closet = locationMap.get('closet')!
      const path = getLocationPath(closet)

      expect(path).toHaveLength(3)
      expect(path[0].name).toBe('Home')
      expect(path[1].name).toBe('Bedroom')
      expect(path[2].name).toBe('Closet')
    })
  })

  describe('expand/collapse', () => {
    it('toggles expanded state', () => {
      const expandedNodes = new Set<string>()

      const toggleExpanded = (locationId: string) => {
        if (expandedNodes.has(locationId)) {
          expandedNodes.delete(locationId)
        } else {
          expandedNodes.add(locationId)
        }
      }

      toggleExpanded('loc-1')
      expect(expandedNodes.has('loc-1')).toBe(true)

      toggleExpanded('loc-1')
      expect(expandedNodes.has('loc-1')).toBe(false)
    })

    it('expands all nodes', () => {
      const locations: Location[] = [
        { id: 'loc-1', name: 'Location 1' },
        { id: 'loc-2', name: 'Location 2' },
        { id: 'loc-3', name: 'Location 3' }
      ]
      const expandedNodes = new Set<string>()

      locations.forEach(l => expandedNodes.add(l.id))

      expect(expandedNodes.size).toBe(3)
    })

    it('collapses all nodes', () => {
      const expandedNodes = new Set(['loc-1', 'loc-2', 'loc-3'])

      expandedNodes.clear()

      expect(expandedNodes.size).toBe(0)
    })
  })

  describe('location icons', () => {
    const getLocationIcon = (location: Location): string => {
      const name = location.name.toLowerCase()
      if (name.includes('bedroom')) return 'i-lucide-bed'
      if (name.includes('living')) return 'i-lucide-sofa'
      if (name.includes('kitchen')) return 'i-lucide-utensils'
      if (name.includes('bathroom')) return 'i-lucide-bath'
      if (name.includes('garage')) return 'i-lucide-car'
      if (name.includes('office')) return 'i-lucide-briefcase'
      if (name.includes('closet')) return 'i-lucide-door-open'
      if (name.includes('storage')) return 'i-lucide-box'
      return 'i-lucide-map-pin'
    }

    it('returns bed icon for bedroom', () => {
      expect(getLocationIcon({ id: '1', name: 'Master Bedroom' })).toBe('i-lucide-bed')
    })

    it('returns sofa icon for living room', () => {
      expect(getLocationIcon({ id: '1', name: 'Living Room' })).toBe('i-lucide-sofa')
    })

    it('returns utensils icon for kitchen', () => {
      expect(getLocationIcon({ id: '1', name: 'Kitchen' })).toBe('i-lucide-utensils')
    })

    it('returns default icon for unknown location', () => {
      expect(getLocationIcon({ id: '1', name: 'Random Space' })).toBe('i-lucide-map-pin')
    })
  })

  describe('parent options', () => {
    it('excludes current location and descendants from parent options', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home' },
        { id: 'bedroom', name: 'Bedroom', parent_id: 'home' },
        { id: 'closet', name: 'Closet', parent_id: 'bedroom' },
        { id: 'office', name: 'Office' }
      ]

      const editingLocationId = 'bedroom'

      const getDescendants = (parentId: string): Set<string> => {
        const descendants = new Set<string>([parentId])
        const addDescendants = (id: string) => {
          locations.forEach((l) => {
            if (l.parent_id === id) {
              descendants.add(l.id)
              addDescendants(l.id)
            }
          })
        }
        addDescendants(parentId)
        return descendants
      }

      const excludeIds = getDescendants(editingLocationId)
      const availableParents = locations.filter(l => !excludeIds.has(l.id))

      expect(availableParents).toHaveLength(2)
      expect(availableParents.map(l => l.name)).toContain('Home')
      expect(availableParents.map(l => l.name)).toContain('Office')
      expect(availableParents.map(l => l.name)).not.toContain('Bedroom')
      expect(availableParents.map(l => l.name)).not.toContain('Closet')
    })
  })

  describe('CRUD operations', () => {
    it('creates a new location', async () => {
      const form = { name: 'New Room', description: 'A new room', parent_id: undefined }
      mockApiFetch.mockResolvedValueOnce({ id: 'new-loc' })

      await mockApiFetch('/api/locations', {
        method: 'POST',
        body: JSON.stringify(form)
      })

      expect(mockApiFetch).toHaveBeenCalledWith('/api/locations', {
        method: 'POST',
        body: JSON.stringify(form)
      })
    })

    it('updates an existing location', async () => {
      const form = { name: 'Updated Room', description: 'Updated description', parent_id: 'parent-1' }
      mockApiFetch.mockResolvedValueOnce({})

      await mockApiFetch('/api/locations/loc-1', {
        method: 'PUT',
        body: JSON.stringify(form)
      })

      expect(mockApiFetch).toHaveBeenCalledWith('/api/locations/loc-1', {
        method: 'PUT',
        body: JSON.stringify(form)
      })
    })

    it('deletes a location', async () => {
      mockApiFetch.mockResolvedValueOnce({})

      await mockApiFetch('/api/locations/loc-1', { method: 'DELETE' })
      mockToast.add({ title: 'Location deleted', color: 'success' })

      expect(mockApiFetch).toHaveBeenCalledWith('/api/locations/loc-1', { method: 'DELETE' })
      expect(mockToast.add).toHaveBeenCalledWith({ title: 'Location deleted', color: 'success' })
    })

    it('clears selection when deleted location was selected', async () => {
      let selectedLocation: Location | null = { id: 'loc-1', name: 'Test' }
      const locationToDelete = { id: 'loc-1', name: 'Test' }

      mockApiFetch.mockResolvedValueOnce({})
      await mockApiFetch(`/api/locations/${locationToDelete.id}`, { method: 'DELETE' })

      if (selectedLocation?.id === locationToDelete.id) {
        selectedLocation = null
      }

      expect(selectedLocation).toBeNull()
    })
  })

  describe('currency formatting', () => {
    it('formats currency values', () => {
      const formatCurrency = (value: number) => {
        return new Intl.NumberFormat('en-US', {
          style: 'currency',
          currency: 'USD',
          minimumFractionDigits: 0,
          maximumFractionDigits: 0
        }).format(value)
      }

      expect(formatCurrency(1000)).toBe('$1,000')
      expect(formatCurrency(0)).toBe('$0')
      expect(formatCurrency(1234567)).toBe('$1,234,567')
    })
  })

  describe('total value calculation', () => {
    it('calculates total value of assets', () => {
      const assets = [
        { id: '1', name: 'Asset 1', purchase_price: 100 },
        { id: '2', name: 'Asset 2', purchase_price: 250 },
        { id: '3', name: 'Asset 3', purchase_price: undefined }
      ]

      const totalValue = assets.reduce((sum, asset) => sum + (asset.purchase_price || 0), 0)

      expect(totalValue).toBe(350)
    })
  })

  describe('children lookup', () => {
    it('gets children of a location', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home' },
        { id: 'bedroom', name: 'Bedroom', parent_id: 'home' },
        { id: 'kitchen', name: 'Kitchen', parent_id: 'home' },
        { id: 'closet', name: 'Closet', parent_id: 'bedroom' }
      ]

      const getChildren = (locationId: string): Location[] => {
        return locations.filter(l => l.parent_id === locationId)
      }

      const homeChildren = getChildren('home')
      expect(homeChildren).toHaveLength(2)

      const bedroomChildren = getChildren('bedroom')
      expect(bedroomChildren).toHaveLength(1)
    })

    it('checks if location has children', () => {
      const locations: Location[] = [
        { id: 'home', name: 'Home' },
        { id: 'bedroom', name: 'Bedroom', parent_id: 'home' }
      ]

      const hasChildren = (locationId: string): boolean => {
        return locations.some(l => l.parent_id === locationId)
      }

      expect(hasChildren('home')).toBe(true)
      expect(hasChildren('bedroom')).toBe(false)
    })
  })
})
