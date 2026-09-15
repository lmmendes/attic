import type { AssetFilters, FilterCriteria, FilterIssue, FilterNode, SavedFilter } from '~/types/api'
import { copyCriteria, criteriaEqual, criteriaKeys, filterFailure, readCriteria } from '~/utils/assetFilters'

export function useAssetFiltering(filters: AssetFilters) {
  const route = useRoute()
  const router = useRouter()
  const apiFetch = useApiFetch()
  const expression = ref<FilterNode>()
  const structured = ref(false)
  const selected = ref<SavedFilter>()
  const selectedId = ref<string>()
  const message = ref('')
  const issues = ref<FilterIssue[]>([])
  const busy = ref(false)
  const routeInvalid = ref(false)
  const savedLoading = ref(false)
  const revision = ref(0)
  const { data: savedFilters, error: savedError, refresh: refreshSaved } = useApi<SavedFilter[]>('/api/saved-filters')
  const criteria = computed<FilterCriteria>(() => {
    const value: FilterCriteria = { version: 1, expression: expression.value }
    for (const key of criteriaKeys) if (filters[key]) value[key] = filters[key]
    return copyCriteria(value)
  })
  const modified = computed(() => !!selected.value && !criteriaEqual(criteria.value, selected.value.criteria))
  let restoring = false
  let generation = 0
  let writtenQuery = ''

  function assign(next: FilterCriteria) {
    for (const key of criteriaKeys) filters[key] = next[key]
    expression.value = copyCriteria(next).expression
    filters.offset = 0
    revision.value++
  }
  function failure(error: unknown) {
    const result = filterFailure(error)
    message.value = result.message
    issues.value = result.issues
  }
  function apply(next: FilterCriteria) {
    structured.value = true
    routeInvalid.value = false
    message.value = ''
    issues.value = []
    assign(next)
  }
  function clear() {
    generation++
    savedLoading.value = false
    selected.value = undefined
    selectedId.value = undefined
    apply({ version: 1 })
    structured.value = false
  }
  function writeRoute() {
    const query = { ...route.query }
    for (const key of criteriaKeys) Reflect.deleteProperty(query, key)
    for (const key of criteriaKeys) if (criteria.value[key]) query[key] = criteria.value[key]
    if (structured.value || selectedId.value || expression.value) query.criteria = JSON.stringify(criteria.value)
    else delete query.criteria
    if (selectedId.value) query.saved_filter_id = selectedId.value
    else delete query.saved_filter_id
    if (filters.offset) query.offset = String(filters.offset)
    else delete query.offset
    writtenQuery = JSON.stringify(query)
    if (writtenQuery !== JSON.stringify(route.query)) void router.replace({ query })
  }
  async function loadSaved(id: string, replace: boolean) {
    const request = ++generation
    savedLoading.value = true
    try {
      const saved = await apiFetch<SavedFilter>(`/api/saved-filters/${encodeURIComponent(id)}`)
      if (request !== generation) return
      selected.value = saved
      selectedId.value = saved.id
      if (replace) apply(saved.criteria)
      if (!modified.value) {
        issues.value = saved.issues || []
        if (issues.value.length) message.value = 'This saved search needs repair. Open Advanced search to fix the listed rules.'
      }
    } catch (error) {
      if (request !== generation) return
      routeInvalid.value = true
      failure(error)
    } finally {
      if (request === generation) savedLoading.value = false
    }
  }
  async function selectSaved(id: string) {
    selected.value = undefined
    selectedId.value = id
    await loadSaved(id, true)
    writeRoute()
  }
  function restore() {
    if (JSON.stringify(route.query) === writtenQuery) return
    restoring = true
    generation++
    savedLoading.value = false
    const parsed = readCriteria(route.query)
    structured.value = typeof route.query.criteria === 'string'
    assign(parsed.criteria)
    message.value = parsed.error || ''
    routeInvalid.value = !!parsed.error
    issues.value = []
    const offset = Number(route.query.offset || 0)
    filters.offset = Number.isSafeInteger(offset) && offset >= 0 ? offset : 0
    const id = typeof route.query.saved_filter_id === 'string' ? route.query.saved_filter_id : undefined
    selectedId.value = id
    selected.value = undefined
    const explicitCriteria = typeof route.query.criteria === 'string' || criteriaKeys.some(key => typeof route.query[key] === 'string')
    if (id && !parsed.error) void loadSaved(id, !explicitCriteria)
    restoring = false
  }
  restore()
  watch(() => route.query, restore, { deep: true, flush: 'sync' })
  watch(criteria, () => {
    if (!restoring) filters.offset = 0
  }, { flush: 'sync' })
  watch([criteria, selectedId, structured, () => filters.offset], () => {
    if (!restoring && !routeInvalid.value && !savedLoading.value) writeRoute()
  })

  async function save(name: string, next: FilterCriteria, update: boolean) {
    if (busy.value || !name.trim()) return false
    busy.value = true
    message.value = ''
    issues.value = []
    try {
      const saved = await apiFetch<SavedFilter>(update && selected.value ? `/api/saved-filters/${selected.value.id}` : '/api/saved-filters', {
        method: update && selected.value ? 'PUT' : 'POST', body: JSON.stringify({ name: name.trim(), criteria: copyCriteria(next) })
      })
      selected.value = saved
      selectedId.value = saved.id
      apply(saved.criteria)
      await refreshSaved()
      return true
    } catch (error) {
      failure(error)
      return false
    } finally { busy.value = false }
  }
  async function rename(name: string) {
    if (!selected.value || busy.value || !name.trim()) return false
    busy.value = true
    message.value = ''
    try {
      selected.value = await apiFetch<SavedFilter>(`/api/saved-filters/${selected.value.id}`, { method: 'PUT', body: JSON.stringify({ name: name.trim() }) })
      await refreshSaved()
      return true
    } catch (error) {
      failure(error)
      return false
    } finally { busy.value = false }
  }
  async function remove() {
    if (!selected.value || busy.value) return false
    busy.value = true
    try {
      await apiFetch(`/api/saved-filters/${selected.value.id}`, { method: 'DELETE' })
      selected.value = undefined
      selectedId.value = undefined
      await refreshSaved()
      return true
    } catch (error) {
      failure(error)
      return false
    } finally { busy.value = false }
  }
  return { criteria, expression, structured, selected, selectedId, modified, message, issues, busy, routeInvalid, savedLoading, revision, savedFilters, savedError, refreshSaved, apply, clear, selectSaved, save, rename, remove }
}
