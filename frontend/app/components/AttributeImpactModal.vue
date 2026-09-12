<script setup lang="ts">
interface Impact {
  affected_assets: number
  deleted_assets: number
  required_fields_left_empty: number
  current: string
  proposed: string
  confirmation_token: string
}
interface Operation {
  attributeId: string
  action: 'update_attribute' | 'delete_attribute' | 'update_option' | 'delete_option'
  optionId?: string
  changes?: Record<string, unknown>
  deletedOptionCount?: number
}
const apiFetch = useApiFetch()
const open = ref(false)
const busy = ref(false)
const error = ref('')
const impact = ref<Impact | null>(null)
const operation = ref<Operation | null>(null)
let onSuccess: (() => void | Promise<void>) | undefined
const deleting = computed(() => operation.value?.action.startsWith('delete') ?? false)
const hasOptionDeletions = computed(() => (operation.value?.deletedOptionCount || 0) > 0)
const hasOptionDraft = computed(() => Array.isArray(operation.value?.changes?.options))
const destructive = computed(() => deleting.value || hasOptionDeletions.value)
const noun = computed(() => operation.value?.action.endsWith('option') ? 'option' : 'field')
const title = computed(() => `${deleting.value ? 'Delete' : 'Update'} ${noun.value}`)
const message = computed(() => {
  const value = impact.value
  if (!value) return 'Loading affected assets…'
  const n = value.affected_assets
  if (hasOptionDeletions.value) {
    if (n === 0) return 'No assets use the options you deleted. Your other option changes will also be saved.'
    return `${n} ${n === 1 ? 'asset uses' : 'assets use'} an option you changed. Deleted selections will be removed, and renamed values will be updated automatically.`
  }
  const usage = n === 0 ? `No assets use this ${noun.value}.` : `This ${noun.value} is used by ${n} ${n === 1 ? 'asset' : 'assets'}`
  if (deleting.value) return n === 0 ? usage : `${usage} and will be removed from ${n === 1 ? 'it' : 'all of them'}.`
  if (n === 0) return usage
  if (value.current !== value.proposed) return `${usage} and will be displayed as “${value.proposed}” in ${n === 1 ? 'it' : 'all of them'}.`
  return `${usage}. The field configuration or stored value will be updated for ${n === 1 ? 'it' : 'all of them'}.`
})
async function preview() {
  const op = operation.value!
  impact.value = await apiFetch<Impact>(`/api/attributes/${op.attributeId}/impact-preview`, {
    method: 'POST',
    body: JSON.stringify({ action: op.action, option_id: op.optionId, changes: op.changes || {} })
  })
  return impact.value
}
async function run(op: Operation, success: () => void | Promise<void>) {
  if (busy.value) return
  operation.value = JSON.parse(JSON.stringify(op))
  onSuccess = success
  impact.value = null
  error.value = ''
  open.value = true
  busy.value = true
  try {
    const result = await preview()
    busy.value = false
    if (!destructive.value && result.affected_assets === 0) await confirm()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busy.value = false
  }
}
async function confirm() {
  if (busy.value) return
  if (!impact.value) {
    busy.value = true
    try {
      await preview()
      error.value = ''
    } catch (err) {
      error.value = errorMessage(err)
    } finally {
      busy.value = false
    }
    return
  }
  busy.value = true
  error.value = ''
  try {
    const op = operation.value!
    const path = `/api/attributes/${op.attributeId}${op.optionId ? '/options/' + op.optionId : ''}`
    await apiFetch(path, {
      method: deleting.value ? 'DELETE' : op.optionId ? 'PATCH' : 'PUT',
      headers: { 'X-Impact-Token': impact.value.confirmation_token },
      ...(deleting.value ? {} : { body: JSON.stringify(op.changes || {}) })
    })
    open.value = false
    await onSuccess?.()
  } catch (err) {
    const failure = err as { data?: { code?: string } }
    if (failure.data?.code === 'impact_preview_required') {
      impact.value = null
      try {
        await preview()
        error.value = 'The affected assets or field changed. Review the updated details and confirm again.'
      } catch (refreshError) { error.value = errorMessage(refreshError) }
    } else error.value = errorMessage(err)
  } finally { busy.value = false }
}
function errorMessage(err: unknown) {
  return (err as { data?: { error?: string } }).data?.error || 'Unable to apply this change. Please try again.'
}
defineExpose({ run })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="title"
    description="Review the affected assets before applying this change."
    :dismissible="!busy"
    :close="!busy"
  >
    <template #body>
      <div class="space-y-3">
        <p
          v-if="impact"
          class="font-semibold"
        >
          {{ impact.current }}<span v-if="!deleting && impact.current !== impact.proposed"> → {{ impact.proposed }}</span>
        </p>
        <p>{{ message }}</p>
        <p v-if="impact?.deleted_assets">
          Includes {{ impact.deleted_assets }} deleted assets.
        </p>
        <p v-if="impact?.required_fields_left_empty && !hasOptionDraft">
          {{ impact.required_fields_left_empty }} {{ impact.required_fields_left_empty === 1 ? 'asset will have no selection' : 'assets will have no selection' }}. Because this attribute is required, {{ impact.required_fields_left_empty === 1 ? 'it must be updated' : 'they must be updated' }} the next time {{ impact.required_fields_left_empty === 1 ? 'it is' : 'they are' }} saved.
        </p>
        <p v-if="deleting">
          This cannot be undone through this dialog.
        </p>
        <p v-if="deleting && noun === 'option'">
          Other selections in multi-select fields will be retained.
        </p>
        <p v-if="!deleting && !hasOptionDraft && (operation?.changes?.value || operation?.changes?.key)">
          Any changed stored values or keys will also be updated on the affected assets.
        </p>
        <p
          v-if="error"
          role="alert"
          class="text-red-600"
        >
          {{ error }}
        </p>
      </div>
    </template>
    <template #footer>
      <UButton
        color="neutral"
        variant="ghost"
        :disabled="busy"
        @click="open = false"
      >
        Cancel
      </UButton>
      <UButton
        :color="destructive ? 'error' : 'primary'"
        :loading="busy"
        @click="confirm"
      >
        {{ impact ? title : 'Retry preview' }}
      </UButton>
    </template>
  </UModal>
</template>
