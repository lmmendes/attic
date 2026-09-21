<script setup lang="ts">
import type { Attribute, FilterIssue, FilterNode, FilterRule, OrganizationFeatures } from '~/types/api'
import { newAttributeRule, normalizeIssuePath, operatorLabels, ruleOperators } from '~/utils/assetFilters'

interface Option { label: string, value: string }
const props = withDefaults(defineProps<{
  modelValue: FilterNode
  attributes: Attribute[]
  options: Record<string, Option[]>
  features: OrganizationFeatures
  issues?: FilterIssue[]
  path?: string
  depth?: number
  ruleCount: number
}>(), { path: 'expression', depth: 1, issues: () => [] })
const emit = defineEmits<{ 'update:modelValue': [node: FilterNode], 'remove': [] }>()
const fields = computed(() => [
  { label: 'Name / description', value: 'q' },
  ...(props.features.tags ? [{ label: 'Tags', value: 'tags' }] : []),
  ...(props.features.attributes ? [{ label: 'All attribute values', value: 'attribute_q' }] : []),
  ...(['collections', 'category', 'location', 'condition'] as const)
    .filter(field => props.features[({ collections: 'collections', category: 'categories', location: 'locations', condition: 'conditions' } as const)[field]])
    .map(field => ({ label: ({ collections: 'Collections', category: 'Category', location: 'Location', condition: 'Condition' })[field], value: field })),
  ...props.attributes.map(a => ({ label: `Attribute: ${a.name}`, value: `attribute:${a.id}` }))
])
const selectedField = computed(() => props.modelValue.kind === 'rule' ? props.modelValue.field === 'attribute' ? `attribute:${props.modelValue.attribute_id}` : props.modelValue.field : '')
const fieldOptions = computed(() => fields.value.some(f => f.value === selectedField.value) ? fields.value : [{ label: 'Unavailable field — choose a replacement', value: selectedField.value }, ...fields.value])
const attribute = computed(() => {
  const node = props.modelValue
  return node.kind === 'rule' ? props.attributes.find(a => a.id === node.attribute_id) : undefined
})
const stale = computed(() => {
  const rule = props.modelValue
  return rule.kind === 'rule' && rule.field === 'attribute' && (!attribute.value || attribute.value.data_type !== rule.data_type || (rule.data_type === 'select' && (attribute.value.selection_mode || 'single') !== rule.selection_mode))
})
const values = computed(() => {
  if (props.modelValue.kind !== 'rule') return []
  const available = props.modelValue.field === 'attribute'
    ? attribute.value?.options?.map(o => ({ label: o.label, value: o.id })) || []
    : props.options[props.modelValue.field] || []
  return [...available, ...(props.modelValue.values || []).filter(id => !available.some(o => o.value === id)).map(id => ({ label: `Unavailable value (${id})`, value: id }))]
})
const ownIssues = computed(() => props.issues.filter((issue) => {
  const path = normalizeIssuePath(issue.path)
  const own = normalizeIssuePath(props.path)
  return path === own || (path.startsWith(`${own}.`) && !path.slice(own.length).startsWith('.children'))
}))
function chooseField(value: string) {
  if (!fields.value.some(field => field.value === value)) return
  if (value === selectedField.value && !stale.value) return
  const attr = props.attributes.find(a => `attribute:${a.id}` === value)
  emit('update:modelValue', attr ? newAttributeRule(attr) : { kind: 'rule', field: value as FilterRule['field'], operator: value === 'q' ? 'search' : value === 'attribute_q' ? 'contains' : 'any' })
}
function patchRule(patch: Partial<FilterRule>) {
  if (props.modelValue.kind === 'rule') emit('update:modelValue', { ...props.modelValue, ...patch })
}
function operator(value: string) {
  if (props.modelValue.kind !== 'rule') return
  const rule = props.modelValue
  const empty = ['empty', 'not_empty'].includes(value)
  const membership = ['any', 'all'].includes(value)
  patchRule({
    operator: value,
    value: empty || membership ? undefined : rule.value ?? (rule.data_type === 'boolean' ? true : undefined),
    values: membership ? rule.values : undefined,
    upper: value === 'between' ? rule.upper : undefined
  })
}
function setScalar(event: Event, upper = false) {
  const raw = (event.target as HTMLInputElement).value
  const value = raw === '' ? undefined : props.modelValue.kind === 'rule' && props.modelValue.data_type === 'number' ? Number(raw) : raw
  patchRule(upper ? { upper: value } : { value })
}
function child(index: number, node?: FilterNode) {
  if (props.modelValue.kind !== 'group') return
  const children = [...props.modelValue.children]
  if (node) children[index] = node
  else children.splice(index, 1)
  emit('update:modelValue', { ...props.modelValue, children })
}
function add(group = false) {
  if (props.modelValue.kind !== 'group' || props.ruleCount >= 50) return
  const rule: FilterRule = { kind: 'rule', field: 'q', operator: 'search', value: '' }
  emit('update:modelValue', { ...props.modelValue, children: [...props.modelValue.children, group ? { kind: 'group', match: 'all', children: [rule] } : rule] })
}
</script>

<template>
  <fieldset
    class="min-w-0 space-y-3 rounded-xl border border-default p-3"
    :aria-label="modelValue.kind === 'group' ? 'Filter group' : 'Filter rule'"
  >
    <div
      v-if="modelValue.kind === 'group'"
      class="flex flex-wrap items-center gap-2"
    >
      <USelect
        :model-value="modelValue.match"
        :items="[{ label: 'Match all (AND)', value: 'all' }, { label: 'Match any (OR)', value: 'any' }]"
        aria-label="Group match"
        @update:model-value="emit('update:modelValue', { ...modelValue, match: $event as 'all' | 'any' })"
      />
      <UButton
        size="xs"
        variant="soft"
        :disabled="ruleCount >= 50"
        @click="add()"
      >
        Add rule
      </UButton>
      <UButton
        size="xs"
        variant="soft"
        :disabled="depth >= 5 || ruleCount >= 50"
        @click="add(true)"
      >
        Add group
      </UButton>
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        aria-label="Remove group"
        @click="emit('remove')"
      >
        Remove group
      </UButton>
    </div>
    <template v-if="modelValue.kind === 'group'">
      <AssetFilterNode
        v-for="(node, index) in modelValue.children"
        :key="index"
        :model-value="node"
        :attributes="attributes"
        :options="options"
        :features="features"
        :issues="issues"
        :path="`${path}.children[${index}]`"
        :depth="depth + 1"
        :rule-count="ruleCount"
        @update:model-value="child(index, $event)"
        @remove="child(index)"
      />
    </template>
    <div
      v-else
      class="flex flex-wrap items-center gap-2"
    >
      <USelectMenu
        :model-value="selectedField"
        :items="fieldOptions"
        value-key="value"
        aria-label="Rule field"
        class="w-full sm:w-56"
        @update:model-value="chooseField($event)"
      />
      <USelect
        :model-value="modelValue.operator"
        :items="ruleOperators(modelValue).map(value => ({ label: operatorLabels[value], value }))"
        aria-label="Rule comparison"
        @update:model-value="operator($event)"
      />
      <template v-if="!['empty', 'not_empty'].includes(modelValue.operator)">
        <USelectMenu
          v-if="['any', 'all'].includes(modelValue.operator)"
          :model-value="modelValue.values || []"
          :items="values"
          value-key="value"
          multiple
          aria-label="Rule values"
          placeholder="Select values"
          class="w-full sm:w-64"
          @update:model-value="patchRule({ values: $event })"
        />
        <select
          v-else-if="modelValue.data_type === 'boolean'"
          :value="String(modelValue.value)"
          aria-label="Rule value"
          class="rounded-md border border-default bg-default p-2"
          @change="patchRule({ value: ($event.target as HTMLSelectElement).value === 'true' })"
        >
          <option value="true">
            True
          </option>
          <option value="false">
            False
          </option>
        </select>
        <input
          v-else
          :value="modelValue.value"
          :type="modelValue.data_type === 'number' ? 'number' : modelValue.data_type === 'date' ? 'date' : 'text'"
          step="any"
          aria-label="Rule value"
          placeholder="Value"
          class="min-w-0 flex-1 rounded-md border border-default bg-default p-2 text-default"
          @input="setScalar($event)"
        >
        <input
          v-if="modelValue.operator === 'between'"
          :value="modelValue.upper"
          :type="modelValue.data_type === 'date' ? 'date' : 'number'"
          step="any"
          aria-label="Upper bound"
          placeholder="Upper bound"
          class="min-w-0 flex-1 rounded-md border border-default bg-default p-2 text-default"
          @input="setScalar($event, true)"
        >
      </template>
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        aria-label="Remove rule"
        @click="emit('remove')"
      >
        Remove rule
      </UButton>
      <p
        v-if="stale"
        class="w-full text-sm text-error"
      >
        This attribute is unavailable or its type changed. Choose the field again or remove this rule.
      </p>
    </div>
    <p
      v-for="(issue, index) in ownIssues"
      :key="index"
      role="alert"
      class="text-sm text-error"
    >
      {{ issue.message }}
    </p>
  </fieldset>
</template>
