<script setup lang="ts">
import type { Location } from '~/types/api'

interface TreeNode {
  location: Location
  children: TreeNode[]
  level: number
}

const props = defineProps<{
  node: TreeNode
  selectedId?: string
  expandedNodes: Set<string>
  getIcon: (location: Location) => string
  hasChildren: (locationId: string) => boolean
}>()

const emit = defineEmits<{
  select: [location: Location]
  toggle: [locationId: string]
  addChild: [parentId: string]
}>()

const isExpanded = computed(() => props.expandedNodes.has(props.node.location.id))
const isSelected = computed(() => props.selectedId === props.node.location.id)
const hasChildrenComputed = computed(() => props.node.children.length > 0)
</script>

<template>
  <div class="tree-item relative">
    <!-- Tree line for hierarchy -->
    <div
      v-if="node.level > 0"
      class="tree-line"
    />

    <!-- Node row -->
    <div
      class="group flex cursor-pointer items-center gap-2 rounded-xl border border-transparent px-2.5 py-2 transition-all"
      :class="[
        isSelected
          ? 'bg-attic-50 border-attic-100 shadow-sm dark:bg-attic-500/15 dark:border-attic-500/20'
          : 'hover:bg-mist-50 dark:hover:bg-mist-700/50'
      ]"
    >
      <!-- Expand/collapse toggle -->
      <button
        v-if="hasChildrenComputed"
        :aria-label="`${isExpanded ? 'Collapse' : 'Expand'} ${node.location.name}`"
        class="p-0.5 -ml-0.5 rounded hover:bg-mist-200 dark:hover:bg-mist-600 transition-colors"
        @click.stop="emit('toggle', node.location.id)"
      >
        <UIcon
          :name="isExpanded ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
          class="w-4 h-4 transition-transform"
          :class="isSelected ? 'text-attic-500' : 'text-muted group-hover:text-attic-500'"
        />
      </button>
      <span
        v-else
        class="w-5 flex justify-center"
      >
        <div class="size-1.5 rounded-full bg-mist-300 dark:bg-mist-500" />
      </span>

      <!-- Location selection -->
      <div
        role="button"
        tabindex="0"
        :aria-pressed="isSelected"
        :aria-label="`View ${node.location.name}`"
        class="flex min-w-0 flex-1 items-center gap-2 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-attic-500"
        @click="emit('select', node.location)"
        @keydown.enter.prevent="emit('select', node.location)"
        @keydown.space.prevent="emit('select', node.location)"
      >
        <UIcon
          :name="getIcon(node.location)"
          class="w-5 h-5 transition-colors"
          :class="isSelected ? 'text-attic-500' : 'text-muted'"
        />

        <span
          class="min-w-0 flex-1 truncate text-sm"
          :class="[
            isSelected
              ? 'font-bold text-attic-500'
              : 'font-medium text-mist-950 dark:text-white'
          ]"
        >
          {{ node.location.name }}
        </span>
      </div>

      <!-- Add child button (shown on hover) -->
      <button
        class="rounded-lg p-1 opacity-0 transition-all hover:bg-mist-200 focus:opacity-100 group-hover:opacity-100 dark:hover:bg-mist-600"
        title="Add sub-location"
        @click.stop="emit('addChild', node.location.id)"
      >
        <UIcon
          name="i-lucide-plus"
          class="w-3.5 h-3.5 text-muted hover:text-attic-500"
        />
      </button>
    </div>

    <!-- Children (nested) -->
    <div
      v-if="hasChildrenComputed && isExpanded"
      class="pl-5 flex flex-col mt-0.5"
    >
      <LocationTreeNode
        v-for="child in node.children"
        :key="child.location.id"
        :node="child"
        :selected-id="selectedId"
        :expanded-nodes="expandedNodes"
        :get-icon="getIcon"
        :has-children="hasChildren"
        @select="emit('select', $event)"
        @toggle="emit('toggle', $event)"
        @add-child="emit('addChild', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.tree-line::before {
  content: '';
  position: absolute;
  left: 11px;
  top: 0;
  bottom: 0;
  width: 1px;
  background-color: #e5e7eb;
  z-index: 0;
}

:root.dark .tree-line::before {
  background-color: #4b5563;
}

.tree-item:last-child > .tree-line::before {
  height: 24px;
}
</style>
