import { describe, expect, it } from 'vitest'
import { attributeOptionPayload } from '../../app/utils/attributeOptions'

describe('attribute option drafts', () => {
  it('omits local draft IDs and lets the backend identify new options', () => {
    expect(attributeOptionPayload([
      { draftId: 'draft-new', label: 'New', value: 'new', sort_order: 0 },
      { id: 'persisted-id', draftId: 'draft-existing', label: 'Existing', value: 'existing', sort_order: 1 }
    ])).toEqual([
      { label: 'New', value: 'new', sort_order: 0 },
      { id: 'persisted-id', label: 'Existing', value: 'existing', sort_order: 1 }
    ])
  })
})
