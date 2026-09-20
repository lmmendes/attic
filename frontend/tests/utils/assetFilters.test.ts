import { describe, expect, it } from 'vitest'
import type { Attribute, FilterCriteria, FilterNode, FilterRule } from '../../app/types/api'
import { copyCriteria, criteriaEqual, issueLabel, newAttributeRule, readCriteria, ruleOperators, validateExpression } from '../../app/utils/assetFilters'

describe('asset filter definitions', () => {
  it('roundtrips full criteria without pagination and isolates nested drafts', () => {
    const criteria: FilterCriteria & { offset: number } = {
      version: 1, q: 'computer', attribute_q: 'blue', category_id: 'category', collection_id: 'collection', offset: 48,
      expression: { kind: 'group', match: 'any', children: [{ kind: 'rule', field: 'collections', operator: 'all', values: ['one', 'two'] }] }
    }
    const draft = copyCriteria(criteria)
    expect(draft).not.toHaveProperty('offset')
    expect(readCriteria({ criteria: JSON.stringify(draft) }).criteria).toEqual(draft)
    expect(criteriaEqual(draft, criteria)).toBe(true)
    if (draft.expression?.kind === 'group') draft.expression.children.pop()
    expect(criteriaEqual(draft, criteria)).toBe(false)
  })

  it('preserves criteria versions while copying', () => {
    expect(copyCriteria({ version: 2, q: 'future' })).toEqual({ version: 2, q: 'future' })
  })

  it('allows linked collection overrides while preserving a structured expression', () => {
    const criteria = { version: 1, collection_id: 'old', expression: { kind: 'rule', field: 'q', operator: 'search', value: 'a' } }
    expect(readCriteria({ criteria: JSON.stringify(criteria), collection_id: 'new' }).criteria).toEqual({ ...criteria, collection_id: 'new' })
  })

  it.each(['{', '{"version":2}', '{"version":1,"q":42}', '{"version":1,"expression":{"kind":"group","children":null}}'])('rejects malformed URL %s without throwing', (criteria) => {
    expect(readCriteria({ criteria }).error).toBeTruthy()
  })

  it('snapshots attribute type and select mode without converting option IDs to stored values', () => {
    const attribute = { id: 'attribute', data_type: 'select', selection_mode: 'multiple', options: [{ id: 'option-id', value: 'stored-value', label: 'Label' }] } as Attribute
    expect(newAttributeRule(attribute)).toEqual({ kind: 'rule', field: 'attribute', attribute_id: 'attribute', data_type: 'select', selection_mode: 'multiple', operator: 'any' })
    expect(ruleOperators(newAttributeRule(attribute))).toContain('all')
    expect(ruleOperators(newAttributeRule({ ...attribute, selection_mode: 'single' }))).not.toContain('all')
  })

  it('accepts zero and false and rejects missing values and reversed bounds', () => {
    const rule: FilterRule = { kind: 'rule', field: 'attribute', attribute_id: 'a', data_type: 'number', operator: 'eq', value: 0 }
    expect(validateExpression(rule)).toEqual([])
    expect(validateExpression({ ...rule, data_type: 'boolean', value: false })).toEqual([])
    expect(validateExpression({ ...rule, value: undefined })).not.toEqual([])
    expect(validateExpression({ ...rule, operator: 'between', value: 4, upper: 2 })).not.toEqual([])
    expect(validateExpression({ ...rule, operator: 'empty', value: undefined })).toEqual([])
  })

  it('enforces nesting, rule and membership limits', () => {
    const rule: FilterNode = { kind: 'rule', field: 'q', operator: 'search', value: 'test' }
    let nested = rule
    for (let i = 0; i < 5; i++) nested = { kind: 'group', match: 'all', children: [nested] }
    expect(validateExpression(nested)).toEqual([])
    expect(readCriteria({ criteria: JSON.stringify({ version: 1, expression: nested }) }).error).toBeUndefined()
    expect(validateExpression({ kind: 'group', match: 'all', children: [nested] })).not.toEqual([])
    expect(validateExpression({ kind: 'group', match: 'all', children: Array.from({ length: 51 }, () => rule) })).not.toEqual([])
    expect(validateExpression({ kind: 'rule', field: 'collections', operator: 'any', values: Array.from({ length: 101 }, (_, i) => String(i)) })).not.toEqual([])
  })

  it('labels backend paths with fields and rule numbers', () => {
    const criteria: FilterCriteria = { version: 1, expression: { kind: 'group', match: 'all', children: [{ kind: 'rule', field: 'attribute', attribute_id: 'a', operator: 'eq' }] } }
    expect(issueLabel({ path: 'criteria.category_id', message: 'gone' }, criteria)).toBe('Category')
    expect(issueLabel({ path: 'expression.children[0].value', message: 'invalid' }, criteria, [{ id: 'a', name: 'Release year' } as Attribute])).toBe('Release year (rule 1)')
  })
})
