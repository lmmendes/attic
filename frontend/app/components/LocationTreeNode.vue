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
      class="flex items-center gap-2 p-2 rounded-lg cursor-pointer group transition-all"
      :class="[
        isSelected
          ? 'bg-attic-500/10 border border-attic-500/20'
          : 'hover:bg-mist-50 dark:hover:bg-mist-700/50'
      ]"
      @click="emit('select', node.location)"
    >
      <!-- Expand/collapse toggle -->
      <button
        v-if="hasChildrenComputed"
        class="p-0.5 -ml-0.5 rounded hover:bg-mist-200 dark:hover:bg-mist-600 transition-colors"
        @click.stop="emit('toggle', node.location.id)"
      >
        <UIcon
          :name="isExpanded ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
          class="w-4 h-4 transition-transform"
          :class="isSelected ? 'text-attic-500' : 'text-mist-400 group-hover:text-attic-500'"
        />
      </button>
      <span
        v-else
        class="w-5 flex justify-center"
      >
        <div class="size-1.5 rounded-full bg-mist-300 dark:bg-mist-500" />
      </span>

      <!-- Location icon -->
      <UIcon
        :name="getIcon(node.location)"
        class="w-5 h-5 transition-colors"
        :class="isSelected ? 'text-attic-500' : 'text-mist-400'"
      />

      <!-- Location name -->
      <span
        class="text-sm flex-1 truncate"
        :class="[
          isSelected
            ? 'font-bold text-attic-500'
            : 'font-medium text-mist-950 dark:text-white'
        ]"
      >
        {{ node.location.name }}
      </span>

      <!-- Add child button (shown on hover) -->
      <button
        class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-mist-200 dark:hover:bg-mist-600 transition-all"
        title="Add sub-location"
        @click.stop="emit('addChild', node.location.id)"
      >
        <UIcon
          name="i-lucide-plus"
          class="w-3.5 h-3.5 text-mist-400 hover:text-attic-500"
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
