import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { h } from 'vue'

const LocationTreeNode = {
  name: 'LocationTreeNode',
  props: {
    node: { type: Object, required: true },
    selectedId: { type: String, default: undefined },
    expandedNodes: { type: Set, required: true },
    getIcon: { type: Function, required: true },
    hasChildren: { type: Function, required: true }
  },
  emits: ['select', 'toggle', 'addChild'],
  setup(props: any, { emit }: { emit: (event: string, ...args: any[]) => void }) {
    const isExpanded = () => props.expandedNodes.has(props.node.location.id)
    const isSelected = () => props.selectedId === props.node.location.id
    const hasChildrenComputed = () => props.node.children.length > 0

    return () => h('div', { class: 'tree-item' }, [
      h('div', {
        class: ['node-row', { selected: isSelected() }],
        onClick: () => emit('select', props.node.location)
      }, [
        hasChildrenComputed() ? h('button', {
          class: 'toggle-btn',
          onClick: (e: Event) => {
            e.stopPropagation()
            emit('toggle', props.node.location.id)
          }
        }, isExpanded() ? 'collapse' : 'expand') : null,
        h('span', { class: 'icon' }, props.getIcon(props.node.location)),
        h('span', { class: 'name' }, props.node.location.name),
        h('button', {
          class: 'add-child-btn',
          onClick: (e: Event) => {
            e.stopPropagation()
            emit('addChild', props.node.location.id)
          }
        }, '+')
      ]),
      hasChildrenComputed() && isExpanded() ? h('div', { class: 'children' },
        props.node.children.map((child: any) =>
          h(LocationTreeNode, {
            key: child.location.id,
            node: child,
            selectedId: props.selectedId,
            expandedNodes: props.expandedNodes,
            getIcon: props.getIcon,
            hasChildren: props.hasChildren,
            onSelect: (loc: any) => emit('select', loc),
            onToggle: (id: string) => emit('toggle', id),
            onAddChild: (id: string) => emit('addChild', id)
          })
        )
      ) : null
    ])
  }
}

describe('LocationTreeNode', () => {
  const createMockNode = (overrides = {}) => ({
    location: {
      id: 'loc-1',
      name: 'Living Room',
      description: 'Main living area',
      parent_id: null
    },
    children: [],
    level: 0,
    ...overrides
  })

  const defaultProps = {
    node: createMockNode(),
    expandedNodes: new Set<string>(),
    getIcon: () => 'i-lucide-home',
    hasChildren: () => false
  }

  it('renders location name', () => {
    const wrapper = mount(LocationTreeNode, {
      props: defaultProps
    })

    expect(wrapper.text()).toContain('Living Room')
  })

  it('emits select event when node is clicked', async () => {
    const wrapper = mount(LocationTreeNode, {
      props: defaultProps
    })

    await wrapper.find('.node-row').trigger('click')

    expect(wrapper.emitted('select')).toBeTruthy()
    expect(wrapper.emitted('select')![0]).toEqual([defaultProps.node.location])
  })

  it('applies selected class when node is selected', () => {
    const wrapper = mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        selectedId: 'loc-1'
      }
    })

    expect(wrapper.find('.node-row').classes()).toContain('selected')
  })

  it('does not show expand button for nodes without children', () => {
    const wrapper = mount(LocationTreeNode, {
      props: defaultProps
    })

    expect(wrapper.find('.toggle-btn').exists()).toBe(false)
  })

  it('shows expand button for nodes with children', () => {
    const nodeWithChildren = createMockNode({
      children: [
        createMockNode({
          location: { id: 'loc-2', name: 'Bookshelf', parent_id: 'loc-1' },
          level: 1
        })
      ]
    })

    const wrapper = mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        node: nodeWithChildren
      }
    })

    expect(wrapper.find('.toggle-btn').exists()).toBe(true)
  })

  it('emits toggle event when expand button is clicked', async () => {
    const nodeWithChildren = createMockNode({
      children: [
        createMockNode({
          location: { id: 'loc-2', name: 'Bookshelf', parent_id: 'loc-1' },
          level: 1
        })
      ]
    })

    const wrapper = mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        node: nodeWithChildren
      }
    })

    await wrapper.find('.toggle-btn').trigger('click')

    expect(wrapper.emitted('toggle')).toBeTruthy()
    expect(wrapper.emitted('toggle')![0]).toEqual(['loc-1'])
  })

  it('shows children when node is expanded', () => {
    const nodeWithChildren = createMockNode({
      children: [
        createMockNode({
          location: { id: 'loc-2', name: 'Bookshelf', parent_id: 'loc-1' },
          level: 1
        })
      ]
    })

    const wrapper = mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        node: nodeWithChildren,
        expandedNodes: new Set(['loc-1'])
      }
    })

    expect(wrapper.find('.children').exists()).toBe(true)
    expect(wrapper.text()).toContain('Bookshelf')
  })

  it('hides children when node is collapsed', () => {
    const nodeWithChildren = createMockNode({
      children: [
        createMockNode({
          location: { id: 'loc-2', name: 'Bookshelf', parent_id: 'loc-1' },
          level: 1
        })
      ]
    })

    const wrapper = mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        node: nodeWithChildren,
        expandedNodes: new Set()
      }
    })

    expect(wrapper.find('.children').exists()).toBe(false)
  })

  it('emits addChild event when add button is clicked', async () => {
    const wrapper = mount(LocationTreeNode, {
      props: defaultProps
    })

    await wrapper.find('.add-child-btn').trigger('click')

    expect(wrapper.emitted('addChild')).toBeTruthy()
    expect(wrapper.emitted('addChild')![0]).toEqual(['loc-1'])
  })

  it('calls getIcon function with location', () => {
    const mockGetIcon = vi.fn().mockReturnValue('i-lucide-folder')

    mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        getIcon: mockGetIcon
      }
    })

    expect(mockGetIcon).toHaveBeenCalledWith(defaultProps.node.location)
  })

  it('renders nested children recursively', () => {
    const deeplyNestedNode = createMockNode({
      children: [
        createMockNode({
          location: { id: 'loc-2', name: 'Shelf 1', parent_id: 'loc-1' },
          level: 1,
          children: [
            createMockNode({
              location: { id: 'loc-3', name: 'Box A', parent_id: 'loc-2' },
              level: 2
            })
          ]
        })
      ]
    })

    const wrapper = mount(LocationTreeNode, {
      props: {
        ...defaultProps,
        node: deeplyNestedNode,
        expandedNodes: new Set(['loc-1', 'loc-2'])
      }
    })

    expect(wrapper.text()).toContain('Living Room')
    expect(wrapper.text()).toContain('Shelf 1')
    expect(wrapper.text()).toContain('Box A')
  })
})
