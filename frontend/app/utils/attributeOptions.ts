import type { AttributeOption, AttributeOptionDraft, AttributeOptionInput } from '~/types/api'

let nextDraftID = 0

export function createAttributeOptionDraft(option: AttributeOptionInput): AttributeOptionDraft {
  nextDraftID++
  return { ...option, draftId: `attribute-option-draft-${nextDraftID}` }
}

export function createAttributeOptionDrafts(options: AttributeOption[]): AttributeOptionDraft[] {
  return options.map(createAttributeOptionDraft)
}

export function attributeOptionPayload(options: AttributeOptionDraft[]): AttributeOptionInput[] {
  // draftId exists only to key unsaved UI rows and must never reach the API.
  return options.map(({ draftId: _, ...option }) => option)
}
