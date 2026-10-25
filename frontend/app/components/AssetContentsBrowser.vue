<script setup lang="ts">
import { formatPurchasePrice } from '~/utils/currency'
import type { Asset, AssetsResponse } from '~/types/api'

const props = defineProps<{ asset: Asset }>()
const emit = defineEmits<{ changed: [] }>()
const { settings } = useOrganizationSettings()
const apiFetch = useApiFetch()
const toast = useToast()
const active = ref(props.asset.id)
const expanded = ref(new Set<string>())
const hierarchyOpen = ref(false)
const query = ref('')
const nested = ref(false)
const view = ref<'grid' | 'list'>('grid')
const offset = ref(0)
const results = ref<AssetsResponse | null>(null)
const loading = ref(false)
const failed = ref(false)
const known = reactive<Record<string, Asset>>({})
const branches = reactive<Record<string, { assets: Asset[], total: number, loading: boolean, error: boolean, offset: number }>>({})
const cache = new Map<string, Promise<AssetsResponse>>()
let generation = 0
let requestId = 0
let selectionId = 0
const pageSize = 24
const selected = ref<string | null>(null)
const selectedAsset = ref<Asset | null>(null)
const pickerOpen = ref(false)
const linking = ref(false)
const current = computed(() => known[active.value] || props.asset)
const breadcrumb = computed(() => {
  const path: Asset[] = []
  const seen = new Set<string>()
  let asset: Asset | undefined = current.value
  while (asset && !seen.has(asset.id)) {
    path.unshift(asset)
    seen.add(asset.id)
    if (asset.id === props.asset.id) break
    asset = asset.parent_id ? known[asset.parent_id] : undefined
  }
  return path
})
const treeRows = computed(() => {
  const rows: { asset: Asset, depth: number, footer: boolean }[] = []
  function visit(asset: Asset, depth: number, seen: Set<string>) {
    if (seen.has(asset.id)) return
    rows.push({ asset, depth, footer: false })
    if (!expanded.value.has(asset.id)) return
    const next = new Set(seen).add(asset.id)
    for (const child of branches[asset.id]?.assets || []) visit(child, depth + 1, next)
    rows.push({ asset, depth, footer: true })
  }
  visit(known[props.asset.id] || props.asset, 0, new Set())
  return rows
})
function request(parent: string, start = 0, search = '', descendants = false) {
  const params = new URLSearchParams({ [descendants ? 'descendant_of' : 'parent_id']: parent, limit: String(pageSize), offset: String(start) })
  if (search.trim()) params.set('q', search.trim())
  const url = `/api/assets?${params}`
  let pending = cache.get(url)
  if (!pending) {
    pending = apiFetch<AssetsResponse>(url).catch((error) => {
      if (cache.get(url) === pending) cache.delete(url)
      throw error
    })
    if (cache.size >= 100) cache.delete(cache.keys().next().value!)
    cache.set(url, pending)
  }
  return pending
}
async function loadBranch(id: string, more = false) {
  branches[id] ||= { assets: [], total: 0, loading: false, error: false, offset: 0 }
  const branch = branches[id]!
  if (branch.loading || (!more && branch.offset > 0 && !branch.error)) return
  const epoch = generation
  branch.loading = true
  branch.error = false
  try {
    const data = await request(id, more ? branch.offset : 0)
    if (epoch !== generation) return
    branch.assets = more
      ? [...new Map([...branch.assets, ...(data.assets || [])].map(asset => [asset.id, asset])).values()]
      : data.assets || []
    branch.offset = (more ? branch.offset : 0) + (data.assets?.length || 0)
    branch.total = data.total
    if (known[id]) known[id]!.child_count = data.total
    for (const asset of branch.assets) known[asset.id] = asset
  } catch {
    if (epoch === generation) branch.error = true
  } finally {
    branch.loading = false
  }
}
async function loadResults() {
  const ticket = ++requestId
  loading.value = true
  failed.value = false
  try {
    const data = active.value !== props.asset.id && current.value.child_count === 0
      ? { assets: [], total: 0, limit: pageSize, offset: 0 }
      : await request(active.value, offset.value, query.value, nested.value)
    if (ticket !== requestId) return
    results.value = data
    for (const asset of data.assets || []) known[asset.id] = asset
  } catch (error: unknown) {
    if (ticket !== requestId) return
    if ((error as { statusCode?: number }).statusCode === 404 && active.value !== props.asset.id) {
      active.value = props.asset.id
      expanded.value = new Set([props.asset.id])
      loadBranch(props.asset.id)
    } else failed.value = true
  } finally {
    if (ticket === requestId) loading.value = false
  }
}
async function select(asset: Asset) {
  const selection = ++selectionId
  const epoch = generation
  known[asset.id] = asset
  for (const ancestor of asset.containment_path || []) {
    if (!known[ancestor.id]) known[ancestor.id] = { ...props.asset, id: ancestor.id, name: ancestor.name, parent_id: undefined }
  }
  const path = asset.containment_path
  if (path?.length) {
    for (let i = 1; i < path.length; i++) known[path[i]!.id]!.parent_id = path[i - 1]!.id
    for (const ancestor of path) {
      expanded.value.add(ancestor.id)
      await loadBranch(ancestor.id)
      if (selection !== selectionId || epoch !== generation) return
      // Ensure a selected result is visible even when its sibling page hasn't loaded.
      const child = ancestor.id === path.at(-1)?.id ? asset : known[path[path.findIndex(item => item.id === ancestor.id) + 1]!.id]!
      if (!branches[ancestor.id]!.assets.some(item => item.id === child.id)) branches[ancestor.id]!.assets.push(child)
    }
  }
  if (selection !== selectionId || epoch !== generation) return
  requestId++
  results.value = null
  loading.value = true
  active.value = asset.id
  query.value = ''
  offset.value = 0
  pickerOpen.value = false
  selected.value = null
  selectedAsset.value = null
  hierarchyOpen.value = false
  loadResults()
  if ((asset.child_count || 0) > 0) {
    expanded.value.add(asset.id)
    await loadBranch(asset.id)
  }
}
function toggle(asset: Asset) {
  if (expanded.value.has(asset.id)) expanded.value.delete(asset.id)
  else {
    expanded.value.add(asset.id)
    loadBranch(asset.id)
  }
}
async function link() {
  if (!selectedAsset.value || linking.value) return
  linking.value = true
  const target = active.value
  try {
    await apiFetch(`/api/assets/${selectedAsset.value.id}/parent`, { method: 'PATCH', body: JSON.stringify({ parent_id: target }) })
    pickerOpen.value = false
    selected.value = null
    selectedAsset.value = null
    generation++
    cache.clear()
    expanded.value.add(target)
    const opened = [...expanded.value]
    for (const id of Object.keys(branches)) Reflect.deleteProperty(branches, id)
    await Promise.all(opened.map(id => loadBranch(id)))
    offset.value = 0
    await loadResults()
    emit('changed')
  } catch (error: unknown) {
    toast.add({ title: (error as { data?: { error?: string } }).data?.error || 'Failed to link asset', color: 'error' })
  } finally { linking.value = false }
}
watch([active, nested], () => {
  offset.value = 0
  loadResults()
})
watch(query, (_, __, onCleanup) => {
  const timer = setTimeout(() => {
    offset.value = 0
    loadResults()
  }, 250)
  onCleanup(() => clearTimeout(timer))
})
watch(offset, loadResults)
watch(() => props.asset.id, () => {
  generation++
  cache.clear()
  for (const id of Object.keys(branches)) Reflect.deleteProperty(branches, id)
  for (const id of Object.keys(known)) Reflect.deleteProperty(known, id)
  known[props.asset.id] = { ...props.asset }
  active.value = props.asset.id
  expanded.value = new Set([props.asset.id])
  offset.value = 0
  query.value = ''
  nested.value = false
  pickerOpen.value = false
  loadBranch(props.asset.id)
  loadResults()
}, { immediate: true })
watch(() => props.asset, (asset) => {
  known[asset.id] = { ...asset, child_count: branches[asset.id]?.total ?? asset.child_count }
})
function price(asset: Asset) {
  return asset.purchase_price == null ? 'Price unknown' : formatPurchasePrice(asset.purchase_price * asset.quantity, settings.value.currency)
}
</script>

<template>
  <section
    class="attic-panel overflow-hidden rounded-[20px] text-default"
    aria-label="Contents"
  >
    <header class="border-b border-mist-100 bg-mist-50/60 px-5 py-4 dark:border-mist-700 dark:bg-mist-800/50">
      <h2 class="flex items-center gap-2 text-base font-bold">
        <UIcon
          name="i-lucide-boxes"
          class="size-5 text-attic-500"
        />Contents
      </h2>
      <p class="mt-1 text-xs text-muted">
        Browse what is inside {{ asset.name }}.
      </p>
    </header>
    <div class="lg:grid lg:grid-cols-[260px_minmax(0,1fr)]">
      <aside
        class="border-b border-mist-100 bg-mist-50/40 dark:border-mist-700 dark:bg-mist-800/20 lg:border-b-0 lg:border-r"
        aria-label="Asset hierarchy"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between p-4 text-sm font-bold lg:hidden"
          :aria-expanded="hierarchyOpen"
          aria-controls="contents-hierarchy"
          @click="hierarchyOpen = !hierarchyOpen"
        >
          Browse hierarchy<UIcon :name="hierarchyOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" />
        </button>
        <div
          id="contents-hierarchy"
          class="overflow-auto p-3 lg:block lg:max-h-[640px]"
          :class="hierarchyOpen ? 'block max-h-80' : 'hidden'"
        >
          <p class="mb-3 hidden px-2 text-xs font-bold uppercase tracking-wider text-muted lg:block">
            Hierarchy
          </p>
          <template
            v-for="row in treeRows"
            :key="`${row.asset.id}-${row.footer}`"
          >
            <div
              v-if="!row.footer"
              class="flex min-w-max items-center gap-1 rounded-lg py-1 pr-2"
              :style="{ paddingLeft: `${row.depth * 16 + 4}px` }"
              :class="active === row.asset.id ? 'bg-attic-50 text-attic-500 dark:bg-attic-500/15' : 'hover:bg-mist-100 dark:hover:bg-mist-700'"
            >
              <button
                v-if="row.asset.id === asset.id || row.asset.child_count"
                type="button"
                class="flex size-7 shrink-0 items-center justify-center rounded focus-visible:outline-2 focus-visible:outline-attic-500"
                :aria-expanded="expanded.has(row.asset.id)"
                :aria-label="`${expanded.has(row.asset.id) ? 'Collapse' : 'Expand'} ${row.asset.name}`"
                @click="toggle(row.asset)"
              >
                <UIcon
                  :name="expanded.has(row.asset.id) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
                  class="size-4"
                />
              </button>
              <span
                v-else
                class="size-7 shrink-0"
              />
              <button
                type="button"
                class="flex flex-1 items-center gap-2 rounded py-1 text-left text-sm focus-visible:outline-2 focus-visible:outline-attic-500"
                :aria-pressed="active === row.asset.id"
                :aria-label="`Browse ${row.asset.name}`"
                @click="select(row.asset)"
              >
                <UIcon
                  name="i-lucide-package"
                  class="size-4 shrink-0"
                /><span
                  class="max-w-40 truncate"
                  :title="row.asset.name"
                >{{ row.asset.name }}</span><span
                  v-if="row.asset.child_count"
                  class="ml-auto text-xs text-muted"
                >{{ row.asset.child_count }}</span>
              </button>
            </div>
            <div
              v-if="row.footer && expanded.has(row.asset.id) && branches[row.asset.id]?.loading"
              role="status"
              class="px-3 py-2 text-xs text-muted"
            >
              Loading {{ row.asset.name }}…
            </div>
            <div
              v-if="row.footer && expanded.has(row.asset.id) && branches[row.asset.id]?.error"
              role="alert"
              class="px-3 py-2 text-xs"
            >
              Could not load {{ row.asset.name }}. <UButton
                variant="link"
                size="xs"
                @click="loadBranch(row.asset.id, (branches[row.asset.id]?.offset || 0) > 0)"
              >
                Retry
              </UButton>
            </div>
            <UButton
              v-if="row.footer && expanded.has(row.asset.id) && !branches[row.asset.id]?.error && (branches[row.asset.id]?.total || 0) > (branches[row.asset.id]?.offset || 0)"
              variant="link"
              size="xs"
              :loading="branches[row.asset.id]?.loading"
              :aria-label="`Load more contents of ${row.asset.name}`"
              @click="loadBranch(row.asset.id, true)"
            >
              Load more in {{ row.asset.name }}
            </UButton>
          </template>
        </div>
      </aside>
      <div class="min-w-0 space-y-4 p-4 sm:p-5">
        <nav
          aria-label="Contents path"
          class="flex flex-wrap items-center gap-1 text-xs text-muted"
        >
          <template
            v-for="(item, index) in breadcrumb"
            :key="item.id"
          >
            <UIcon
              v-if="index"
              name="i-lucide-chevron-right"
              class="size-3"
            /><button
              type="button"
              class="rounded px-1 py-1 hover:text-attic-500 focus-visible:outline-2 focus-visible:outline-attic-500"
              :aria-current="item.id === active ? 'location' : undefined"
              @click="select(item)"
            >
              {{ item.name }}
            </button>
          </template>
        </nav>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="break-words text-lg font-bold">
              {{ current.name }}
            </h3><NuxtLink
              :to="`/assets/${active}`"
              class="text-xs font-medium text-attic-500 hover:underline"
            >View asset ↗</NuxtLink>
          </div>
          <div class="flex flex-wrap gap-2">
            <UButton
              :to="{ path: '/assets/new', query: { parent_id: active } }"
              size="sm"
              variant="soft"
              icon="i-lucide-plus"
            >
              Add asset
            </UButton><UButton
              size="sm"
              variant="ghost"
              icon="i-lucide-link"
              :disabled="linking"
              :aria-expanded="pickerOpen"
              @click="pickerOpen = !pickerOpen"
            >
              Link existing asset
            </UButton>
          </div>
        </div>
        <div
          v-if="pickerOpen"
          class="space-y-3 rounded-xl border border-mist-200 p-4 dark:border-mist-700"
        >
          <AssetPicker
            v-model="selected"
            label="Content asset"
            :exclude-ancestors-of="active"
            @resolved="selectedAsset = $event"
          />
          <div class="flex gap-2">
            <UButton
              :disabled="!selectedAsset || linking"
              :loading="linking"
              @click="link"
            >
              Link asset
            </UButton><UButton
              variant="ghost"
              :disabled="linking"
              @click="pickerOpen = false; selected = null; selectedAsset = null"
            >
              Cancel
            </UButton>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-3 rounded-xl bg-mist-50 p-3 dark:bg-mist-800/50">
          <label class="min-w-40 flex-1"><span class="sr-only">Search these contents</span><input
            v-model="query"
            type="search"
            placeholder="Search these contents…"
            class="w-full rounded-lg border border-mist-200 bg-white px-3 py-2 text-sm focus-visible:outline-attic-500 dark:border-mist-600 dark:bg-mist-900"
          ></label>
          <label class="flex items-center gap-2 text-xs"><input
            v-model="nested"
            type="checkbox"
            class="accent-attic-500"
          >Include nested contents</label>
          <div
            class="flex gap-1"
            aria-label="Results layout"
          >
            <UButton
              icon="i-lucide-layout-grid"
              size="sm"
              :variant="view === 'grid' ? 'soft' : 'ghost'"
              aria-label="Grid view"
              :aria-pressed="view === 'grid'"
              @click="view = 'grid'"
            /><UButton
              icon="i-lucide-list"
              size="sm"
              :variant="view === 'list' ? 'soft' : 'ghost'"
              aria-label="List view"
              :aria-pressed="view === 'list'"
              @click="view = 'list'"
            />
          </div>
        </div>
        <p
          v-if="loading"
          role="status"
          class="py-8 text-center text-sm text-muted"
        >
          Loading contents…
        </p>
        <div
          v-else-if="failed"
          role="alert"
          class="py-8 text-center"
        >
          Contents could not be loaded. <UButton
            variant="link"
            @click="loadResults"
          >
            Try again
          </UButton>
        </div>
        <template v-else>
          <p
            class="text-xs text-muted"
            aria-live="polite"
          >
            {{ results?.total || 0 }} {{ nested ? 'nested' : 'direct' }} {{ results?.total === 1 ? 'asset' : 'assets' }}
          </p>
          <div
            v-if="!results?.total"
            class="rounded-xl border border-dashed border-mist-200 px-4 py-10 text-center dark:border-mist-700"
          >
            <UIcon
              name="i-lucide-package-open"
              class="mb-3 size-8 text-muted"
            /><h4 class="font-bold">
              {{ query.trim() ? 'No matching assets' : 'No contents yet' }}
            </h4><p class="mt-1 text-sm text-muted">
              {{ query.trim() ? 'Try another search or include nested contents.' : `Add a new asset or link an existing one inside ${current.name}.` }}
            </p><UButton
              v-if="query.trim()"
              class="mt-3"
              variant="soft"
              @click="query = ''"
            >
              Clear search
            </UButton>
          </div>
          <ul
            v-else
            :class="view === 'grid' ? 'grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3' : 'space-y-2'"
            aria-label="Asset results"
          >
            <li
              v-for="child in results?.assets"
              :key="child.id"
              class="overflow-hidden rounded-xl border border-mist-200 dark:border-mist-700"
              :class="view === 'list' ? 'flex items-center gap-3 p-3' : 'flex flex-col'"
            >
              <NuxtLink
                :to="`/assets/${child.id}`"
                class="flex shrink-0 items-center justify-center overflow-hidden bg-mist-50 dark:bg-mist-800"
                :class="view === 'grid' ? 'aspect-[2/1]' : 'size-14 rounded-lg'"
                tabindex="-1"
                aria-hidden="true"
              ><img
                v-if="child.main_attachment_url"
                :src="child.main_attachment_url"
                alt=""
                loading="lazy"
                class="size-full object-cover"
              ><UIcon
                v-else
                name="i-lucide-package"
                class="size-8 text-mist-300"
              /></NuxtLink>
              <div
                class="min-w-0 flex-1"
                :class="view === 'grid' ? 'p-4' : ''"
              >
                <NuxtLink
                  :to="`/assets/${child.id}`"
                  class="break-words text-sm font-bold hover:text-attic-500"
                >{{ child.name }}</NuxtLink><p
                  v-if="nested && child.containment_path?.length"
                  class="mt-1 break-words text-xs text-muted"
                >
                  Inside {{ child.containment_path.map(item => item.name).join(' › ') }}
                </p><p class="mt-2 text-xs text-muted">
                  Qty {{ child.quantity }} · {{ price(child) }}
                </p><p
                  v-if="child.category"
                  class="mt-1 text-xs text-muted"
                >
                  {{ child.category.name }}
                </p><UButton
                  v-if="child.child_count"
                  class="mt-2"
                  size="xs"
                  variant="link"
                  :aria-label="`Browse contents of ${child.name}`"
                  @click="select(child)"
                >
                  {{ child.child_count }} inside <UIcon name="i-lucide-arrow-right" />
                </UButton>
              </div>
            </li>
          </ul>
          <div
            v-if="(results?.total || 0) > pageSize"
            class="flex items-center justify-between gap-2"
          >
            <UButton
              variant="ghost"
              :disabled="offset === 0"
              @click="offset -= pageSize"
            >
              Previous contents
            </UButton><span class="text-xs text-muted">{{ offset + 1 }}–{{ Math.min(offset + pageSize, results?.total || 0) }} of {{ results?.total }}</span><UButton
              variant="ghost"
              :disabled="offset + pageSize >= (results?.total || 0)"
              @click="offset += pageSize"
            >
              Next contents
            </UButton>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>
