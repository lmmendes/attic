# RFC 011: Asset Containment and Component Hierarchy

> **Status**: Implemented in the working tree; not yet released
> **Created**: 2026-10-09
> **Related issue**: [#111](https://github.com/lmmendes/attic/issues/111)

---

## Summary

Allow an asset to contain other assets. Each asset can have one immediate parent,
and a parent can contain multiple assets at arbitrary nesting depths. This supports
both assemblies, such as Computer → Motherboard → RAM, and storage containers,
such as Crate → Box → Collectible.

Containment preserves each asset's identity, attachments, warranty, and other
metadata. The asset detail page presents a recursive, expandable Contents tree
so users can see which component belongs to which parent.

## Motivation

Locations describe where belongings are stored, and collections group related
belongings. Neither expresses that one asset is physically inside or part of
another. A flat contents list also obscures relationships: RAM inside a
motherboard must appear beneath the motherboard when viewing the computer.

## Goals

- Create an asset inside an existing asset or link an existing asset to a parent.
- Display nested contents without losing intermediate parent relationships.
- Move and detach assets while preserving their descendants and metadata.
- Synchronize the physical location of a parent and its entire subtree.
- Calculate the combined purchase value of an asset and its descendants.
- Prevent cycles, cross-workspace relationships, and partial hierarchy updates.
- Preserve compatibility with existing assets and API clients.

## Non-goals

- Multiple parents, arbitrary dependency graphs, or bill-of-materials quantities.
- Drag-and-drop reparenting or manual component ordering.
- A separate container asset type.
- Replacing locations or collections.
- Multiplying a child's quantity by quantities of its ancestors.

## Relationship Rules

An asset with no parent is a root. A contained asset has one immediate parent;
all relationships reference active assets in the same workspace. Self-containment
and indirect cycles are rejected. Linking an asset already inside another asset
moves it, together with its subtree, to the new parent.

A contained asset uses its parent's location. Attaching an asset or changing a
container's location synchronizes all descendants, including when the parent's
location is unset. An explicit conflicting location is rejected. Detaching an
asset retains its current location and leaves its descendants attached to it.
When Locations is disabled, responses continue to respect the feature setting.

Deleting a parent detaches its immediate children without deleting them. Their
locations, descendants, attachments, and warranties remain intact. A later hard
deletion also clears parent references through the foreign key.

## User Experience

### Create and edit

The create and edit forms include a searchable parent picker. Creating from a
parent's **Add asset** action opens `/assets/new?parent_id=<id>` with the parent
preselected. Contained assets show their inherited location.

On the detail page, **Inside** links to the immediate parent and **Detach** removes
that relationship. **Link existing asset** searches existing assets and attaches
the selection. Pickers exclude relationships that would create a cycle, while
the server remains authoritative.

In the edit form, additions and removals are staged. Saving applies the asset
changes and contents changes together; cancelling discards the staged changes.
Removing a component detaches its whole subtree.

The flat assets list retains individual asset rows and indicates their immediate
parent with a link.

### Contents browser

The detail page shows a full-width Contents browser below the asset overview,
with a compact hierarchy rail and an asset results pane. The current asset is
selected and expanded; other branches start collapsed. Selecting a name browses
its contents, while the chevron independently expands its branch. Branches load
on demand and retain siblings with Load more; leaves do not fetch children.

Results default to direct contents, with an Include nested contents toggle for
all descendants. Search applies within the selected scope. Results paginate in
batches of 24 and support image cards or a compact list. Nested results display
their ancestor path. Breadcrumbs navigate within this browser; asset links open
individual detail pages. Add asset and Link existing asset target the selected
container. On mobile, the hierarchy collapses above the results.

Loading and retry states belong to the affected branch or results pane. The
browser supports keyboard selection, accessible expansion states, dark mode,
and long names. Edit-mode Contents remains a direct-child list: additions and
detachments apply only on Save, and Undo restores staged removals. Detached
assets keep their descendants.

### Combined value

The detail page displays the total purchase value of the asset and every active
descendant. Each record contributes `purchase_price × quantity` once. A missing
price contributes no value and increments the unpriced asset count; a price of
zero is valid. An incomplete-total notice appears when any record is unpriced.

Existing inventory statistics continue counting individual records using their
own values. They do not add subtree rollups and therefore do not double-count
components.

## Data Model

Migration `000025_asset_containment` adds:

```sql
ALTER TABLE assets ADD COLUMN parent_id UUID;
ALTER TABLE assets ADD CONSTRAINT assets_parent_fk
    FOREIGN KEY (parent_id) REFERENCES assets(id) ON DELETE SET NULL;
ALTER TABLE assets ADD CONSTRAINT assets_not_own_parent
    CHECK (parent_id IS DISTINCT FROM id);
CREATE INDEX idx_assets_parent ON assets(organization_id, parent_id)
    WHERE deleted_at IS NULL;
```

Existing assets start with a null parent. Workspace membership and indirect-cycle
validation are enforced in repository transactions. The down migration removes
the containment index, constraints, and column without deleting asset records.

## API Design

All operations require authentication and apply the current workspace scope.

| Operation | Containment behavior |
|---|---|
| `POST /api/assets` | Accepts optional `parent_id` |
| `PUT /api/assets/{id}` | Accepts `parent_id`, `add_child_ids`, and `remove_child_ids` |
| `PATCH /api/assets/{id}/parent` | Changes only the parent relationship and inherited location |
| `GET /api/assets?parent_id=<id>` | Lists immediate children using existing pagination |
| `GET /api/assets?descendant_of=<id>` | Lists descendants excluding the root, with ordered `containment_path` references; mutually exclusive with `parent_id` |
| `GET /api/assets?exclude_subtree_of=<id>` | Excludes the asset and its descendants for parent selection |
| `GET /api/assets?exclude_ancestors_of=<id>` | Excludes the asset and its ancestors for content selection |

For updates, an omitted `parent_id` preserves the current parent. Explicit
`null` detaches it. The focused PATCH endpoint requires the field:

```json
{ "parent_id": null }
```

`add_child_ids` and `remove_child_ids` are UUID arrays of relationship changes,
not replacement snapshots. A child cannot appear in both arrays. Additions can
move existing assets from other parents. Removals require that the asset is still
a direct child of the edited parent.

Asset responses include `parent_id` and a shallow `parent` reference containing
its ID and name when present. List responses populate `child_count`; detail
responses include `containment_summary` with `total_value` and
`unpriced_asset_count`. Responses do not embed the full recursive tree.
The existing OpenAPI specification documents these fields and operations.

Malformed identifiers, unavailable relationship targets, self-containment,
cycles, and conflicting locations return `400`. A missing target asset returns
`404`. A removal whose relationship changed since loading returns `409` with a
reload instruction. Unexpected persistence errors return `500`.

## Transactions and Concurrency

Hierarchy mutations acquire a workspace-scoped PostgreSQL transaction advisory
lock. Asset attribute locks precede the containment lock consistently. Current
parent and location values are read under the lock rather than trusted from a
stale form or an earlier request.

Asset updates, staged additions and removals, final graph validation, and subtree
location synchronization commit in one transaction. Validation checks the final
graph so a valid combined detach/reparent operation can succeed. Any failure
rolls back the entire operation. Serialization prevents concurrent opposite
attachments from creating a cycle.

Recursive queries use `UNION` to avoid revisiting records. Parent references are
loaded in batches, and direct child counts are included in list queries.

## Validation

- HTTP integration tests cover linking, detaching, location propagation, atomic
  updates and rollback, cycles, workspace isolation, disabled Locations,
  concurrent opposing attachments, deletion preservation, and migration reversal.
- Frontend tests cover parent selection, staged editing, nested rendering,
  expand/collapse, leaf request avoidance, and independent branch pagination.
- Browser flows exercise assembly and storage-container hierarchies.
- The redesigned tree was inspected at desktop and mobile sizes, including
  expand/collapse behavior and absence of horizontal overflow.
- Backend tests and vet, frontend tests, lint, type checking, and production
  builds have passed during implementation.

## Alternatives and Tradeoffs

A parent reference reuses the existing asset model and supports both assemblies
and containers without introducing duplicate records. A join table would allow
multiple parents, which conflicts with physical containment and is outside scope.

Persisting inherited locations keeps existing location queries useful, at the
cost of updating descendants when an assembly moves. A workspace-level lock
simplifies integrity under concurrent edits but serializes hierarchy mutations
within that workspace. Per-branch pagination limits response sizes; the number
of requests still grows with the number of expanded parents.
