import { describe, expect, it } from 'vitest'
import { tagAssignment } from '../../app/utils/tags'

describe('tag assignments', () => {
  it('separates existing IDs from case-insensitive new names', () => {
    const available = [{ id: 'retro-id', name: 'Retro' }]
    expect(tagAssignment(['retro', ' Portable ', 'PORTABLE'], available as never)).toEqual({
      tag_ids: ['retro-id'],
      new_tag_names: ['Portable']
    })
  })
})
