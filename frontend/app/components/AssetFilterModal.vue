<script setup lang="ts">
import type { Attribute, FilterCriteria, FilterIssue, FilterNode, OrganizationFeatures, SavedFilter } from '~/types/api'
import { copyCriteria, countRules, criteriaKeys, criteriaLabels, issueLabel, normalizeIssuePath, validateExpression } from '~/utils/assetFilters'

const props = defineProps<{
  criteria: FilterCriteria
  selected?: SavedFilter
  attributes: Attribute[]
  options: Record<string, { label: string, value: string }[]>
  features: OrganizationFeatures
  issues: FilterIssue[]
  message: string
  busy: boolean
  save: (name: string, criteria: FilterCriteria, update: boolean) => Promise<boolean>
}>()
const emit = defineEmits<{ apply: [criteria: FilterCriteria] }>()
const open = ref(false)
const mode = ref<'advanced' | 'edit'>('advanced')
const draft = ref<FilterCriteria>({ version: 1 })
const name = ref('')
const localIssues = ref<FilterIssue[]>([])
const nameError = ref('')
const allIssues = computed(() => [...localIssues.value, ...props.issues])
const topRows = computed(() => criteriaKeys.filter(key => draft.value[key]).map((key) => {
  const value = draft.value[key]!
  const field = key === 'collection_id' ? 'collections' : key.replace('_id', '')
  const option = props.options[field]?.find(option => option.value === value)
  const unavailable = key === 'q' ? false : key === 'attribute_q' ? !props.features.attributes : !option
  return { key, label: criteriaLabels[key], value: option?.label || value, unavailable }
}))
function removeTop(key: typeof criteriaKeys[number]) {
  Reflect.deleteProperty(draft.value, key)
  localIssues.value = localIssues.value.filter(issue => normalizeIssuePath(issue.path) !== key)
}
const title = computed(() => mode.value === 'edit' ? 'Edit saved search' : 'Advanced search')
function show(nextMode: 'advanced' | 'edit' = 'advanced') {
  mode.value = nextMode
  draft.value = copyCriteria(props.criteria)
  name.value = props.selected?.name || ''
  localIssues.value = []
  nameError.value = ''
  open.value = true
}
function validate() {
  localIssues.value = validateExpression(draft.value.expression)
  if (countRules(draft.value.expression) + topRows.value.length > 50) {
    localIssues.value.push({ path: 'expression', message: 'Use at most 50 rules, including current page criteria.' })
  }
  for (const row of topRows.value) {
    if (row.unavailable) localIssues.value.push({ path: row.key, message: 'This criterion is unavailable. Remove it to continue.' })
  }
  function visit(node?: FilterNode, path = 'expression') {
    if (!node) return
    if (node.kind === 'group') {
      node.children.forEach((child, i) => visit(child, `${path}.children[${i}]`))
      return
    }
    const add = (message: string) => localIssues.value.push({ path, message })
    if (node.field === 'attribute') {
      const attr = props.attributes.find(a => a.id === node.attribute_id)
      if (!attr || attr.data_type !== node.data_type || (attr.data_type === 'select' && (attr.selection_mode || 'single') !== node.selection_mode)) add('Choose an available attribute with its current type or remove this rule.')
      if (attr?.data_type === 'select' && node.values?.some(id => !attr.options?.some(o => o.id === id))) add('Remove unavailable options or choose replacements.')
    } else if (node.field === 'attribute_q') {
      if (!props.features.attributes) add('Attribute search is disabled.')
    } else if (node.field !== 'q') {
      if (!props.options[node.field] || node.values?.some(id => !props.options[node.field]?.some(o => o.value === id))) add('Choose available values or remove this rule.')
    }
  }
  visit(draft.value.expression)
  return localIssues.value.length === 0
}
function apply() {
  if (!validate()) return
  emit('apply', copyCriteria(draft.value))
  open.value = false
}
async function saveDraft(update: boolean) {
  nameError.value = name.value.trim() ? '' : 'Enter a filter name.'
  if (!validate() || nameError.value) return
  if (await props.save(name.value, copyCriteria(draft.value), update)) open.value = false
}
defineExpose({ show })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="title"
    description="Combine rules with AND / OR. Saved searches are private to you."
    :dismissible="!busy"
    :close="!busy"
    :ui="{ content: 'sm:max-w-4xl' }"
  >
    <template #body>
      <div class="space-y-4">
        <p class="text-sm text-muted">
          These rules also match all current search and page filters. Saving includes those criteria, without pagination.
        </p>
        <div
          v-if="topRows.length"
          class="space-y-2"
          aria-label="Current page criteria"
        >
          <div
            v-for="row in topRows"
            :key="row.key"
            class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-default p-2"
          >
            <span class="min-w-0 break-words text-sm"><strong>{{ row.label }}:</strong> {{ row.value }} <span
              v-if="row.unavailable"
              class="text-error"
            >(unavailable)</span></span>
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              :aria-label="`Remove ${row.label} criterion`"
              @click="removeTop(row.key)"
            >
              Remove
            </UButton>
            <p
              v-for="(issue, index) in allIssues.filter(issue => normalizeIssuePath(issue.path).split('.')[0] === row.key)"
              :key="index"
              class="w-full text-sm text-error"
            >
              {{ issue.message }}
            </p>
          </div>
        </div>
        <AssetFilterNode
          v-if="draft.expression"
          v-model="draft.expression"
          :attributes="attributes"
          :options="options"
          :features="features"
          :issues="allIssues"
          :rule-count="countRules(draft.expression) + topRows.length"
          @remove="draft.expression = undefined"
        />
        <UButton
          v-else
          variant="soft"
          @click="draft.expression = { kind: 'group', match: 'all', children: [{ kind: 'rule', field: 'q', operator: 'search', value: '' }] }"
        >
          Add rules
        </UButton>
        <UFormField
          label="Filter name"
          :error="nameError || undefined"
        >
          <UInput
            v-model="name"
            aria-label="Filter name"
            placeholder="e.g. Retro computers"
            class="w-full"
          />
        </UFormField>
        <p
          v-if="message"
          role="alert"
          class="text-error"
        >
          {{ message }}
        </p>
        <ul
          v-if="allIssues.length"
          class="list-disc pl-5 text-sm text-error"
          aria-label="Filter issues"
        >
          <li
            v-for="(issue, index) in allIssues"
            :key="index"
          >
            {{ issueLabel(issue, draft, attributes) }}: {{ issue.message }}
          </li>
        </ul>
      </div>
    </template>
    <template #footer>
      <div class="flex flex-wrap gap-2">
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="busy"
          @click="open = false"
        >
          Cancel
        </UButton>
        <UButton
          variant="outline"
          :disabled="busy"
          @click="apply"
        >
          Apply
        </UButton>
        <UButton
          :loading="busy"
          @click="saveDraft(false)"
        >
          Save as new
        </UButton>
        <UButton
          v-if="selected"
          :loading="busy"
          @click="saveDraft(true)"
        >
          Update saved filter
        </UButton>
      </div>
    </template>
  </UModal>
</template>
