# RFC 009: Select Attributes

> **Status**: Implemented
> **Created**: 2026-09-08
> **Updated**: 2026-09-12
> **Author**: @lmmendes
> **Related issue**: [#61](https://github.com/lmmendes/attic/issues/61)

## Summary

Add one `select` attribute type with single-select and multi-select modes. Administrators define options and can add, rename, reorder, and delete them. Members choose from those options when editing assets.

Renaming or deleting an option or field includes an impact preview. When assets are affected, a confirmation modal shows the exact number before applying the change. Deleting an option removes it from all affected assets; options are not archived.

This document records the implemented feature and its behavior.

## Goals and Boundaries

- Deliver single-select and multi-select together using existing reusable organization-level attributes and category inheritance.
- Store canonical strings and arrays of strings in the existing asset attributes JSON object.
- Validate selection cardinality and allowed choices on the server.
- Allow administrators to manage options, with impact confirmation for existing assets.
- Apply impact confirmation to field display-name changes, key changes, and deletion.
- Defer member suggestions, approval workflows, archiving, automatic merging, external option synchronization, and attribute-value filtering.
- Permit data-type or selection-mode changes only when no asset contains the field key. Conversion of populated fields is deferred.

## User Experience

### Field and option management

Add **Select** to the existing attribute types. Selecting it reveals **Selection mode**, defaulting to **Single**, with **Multiple** as the alternative, and an option editor.

Each option has a label, canonical stored value, and position. Generate its initial value from its label using the existing attribute-key normalization convention. Allow explicit editing; require manual input when normalization produces an empty value. After creation, label edits do not silently regenerate the stored value. Keep stored-value editing secondary to the label editor.

Administrators can add, rename, reorder, and delete options. Members can read and select options but cannot change the vocabulary or field definition.

Single-select renders as a searchable dropdown. Multi-select renders as a searchable control with removable selected chips. Optional fields can be cleared. Required single-select fields need one option; required multi-select fields need at least one. Asset detail views resolve stored values to current labels.

### Impact confirmation modals

Fetch a server preview before applying renames or deletions. Show the option or field name, old and new labels for renames, and the affected-asset count. Count each asset once, regardless of its quantity or number of selections.

| Operation | Example message | Confirm button |
|---|---|---|
| Delete option | This option is used by 37 assets and will be removed from all of them. | Delete option |
| Rename option label | This option is used by 37 assets and will be displayed as “Commodore International” in all of them. | Rename option |
| Change stored option value | This option is used by 37 assets. Its stored value will be changed in all of them. | Update option |
| Rename field | This field is used by 37 assets and will be displayed as “Manufacturer” in all of them. | Rename field |
| Change field key | This field is used by 37 assets. Its stored key will be changed in all of them. | Update field |
| Delete field | This field is used by 37 assets and will be removed from all of them. | Delete field |

Use singular wording for one asset. The attribute editor keeps field and option edits in one draft and applies them only when the administrator presses **Save Changes**. Combine all option additions, renames, stored-value changes, deletions, and reordering into that one confirmation and transaction; there are no value-level save actions or confirmation dialogs. Explain that deletions cannot be undone through this flow. For multi-select option deletion, explain that other selections remain. When required fields become empty, also show how many assets will need replacement selections.

Always confirm deletion, including at zero usage with “No assets use this option” or “No assets use this field.” Skip rename confirmation at zero usage. Adding and reordering options need no impact confirmation.

The modal has **Cancel** and the operation-specific button, with destructive styling for deletion. Disable confirmation while loading or saving. Preserve the modal on errors. If the preview becomes stale, refresh the impact and require a new confirmation. Canceling changes nothing.

### Counting affected assets

For option changes, count assets containing the exact canonical value under the field key, as either a scalar or array member. For field changes, count assets containing the key, including empty values. Include all categories, inherited uses, and values remaining after category changes.

Preview counts and mutations use the same organization-scoped asset set. Include soft-deleted assets to avoid obsolete references on restoration; show their count separately when nonzero and explain that the total includes deleted assets. Renaming a field also changes category forms using its definition; those forms are not counted as assets.

## Data Model and Behavior

### Definitions and asset values

Extend `AttributeDataType` with `select`. Add nullable `selection_mode`, required to be `single` or `multiple` for select fields and null for other types. Default omitted mode to `single` when creating a select field.

Add an `attribute_options` table with UUID `id`, `attribute_id`, `label`, `value`, `sort_order`, and creation/update timestamps. Enforce the attribute foreign key and uniqueness of `(attribute_id, value)`. Trim labels and values; require nonempty strings up to 255 characters. Reject duplicate labels ignoring case within a field; match canonical values exactly. Option IDs remain stable through renames. There are no suggestion statuses or archive states.

Assets store canonical strings, not option IDs:

```json
{
  "vendor": "commodore",
  "platforms": ["linux", "macos"]
}
```

Single-select values are strings; multi-select values are arrays of unique strings. Clear optional fields by omitting their key. Accept an empty array as clearing optional multi-select and normalize it to an absent key. Reject nulls, unknown values, duplicate entries, and wrong types.

Return mode and options in attribute responses and directly assigned/inherited category attribute responses. Extend frontend asset types to support string arrays. This capability requires updated clients; older scalar-only clients do not automatically support multi-select.

### Rename and delete semantics

| Change | Definition update | Asset update |
|---|---|---|
| Option label rename | Update label; retain canonical value | No JSON rewrite; rendered labels change |
| Option value rename | Update value; retain option ID | Replace exact old value in scalars and arrays |
| Option deletion | Permanently delete option | Remove matching scalar key or array member; omit key if array becomes empty |
| Field display-name rename | Update name | No JSON rewrite; rendered names change |
| Field key rename | Update key; retain field ID and category assignments | Move JSON key, preserving its value |
| Field deletion | Soft-delete definition using existing lifecycle; remove category assignments and options | Remove field key from all affected assets |

Commit definition changes and asset rewrites in one transaction. Update timestamps on changed rows. Preserve unrelated data and remaining selections. Reject collisions with another option value or field key without merging. Reject a key rename if any affected asset already contains the destination key, even when no definition owns it. Retain retired keys after field deletion or key rename, reject their reuse, and ask stale clients that submit them to reload before saving.

Confirmed option deletion may empty a required field. Allow the deletion, report that consequence in the preview, and require a replacement on the next normal asset save. Deleting the entire field removes its category requirements.

Protect plugin-owned definitions from user renames, type changes, option changes, and deletion in this workflow. Keep reserved plugin keys unavailable to custom fields.

## API and Validation

### Attribute and option interfaces

Extend attribute creation with `selection_mode` and initial `options`, saved atomically. Extend updates with optional `key`, `selection_mode`, and the complete ordered `options` list; omitted properties retain their existing values. When `options` is present, reconcile additions, edits, deletions, and ordering atomically with the field update and any required asset rewrites. Reject select-only configuration on other types.

Option objects expose `id`, `label`, `value`, and `sort_order`. The attribute editor reads options as part of the attribute and submits the complete draft through the field update. The option-specific mutation endpoints may remain for API compatibility, but the admin UI does not use them:

```text
GET    /api/attributes/{id}/options
POST   /api/attributes/{id}/options
PATCH  /api/attributes/{id}/options/{option_id}
DELETE /api/attributes/{id}/options/{option_id}
PUT    /api/attributes/{id}/options/order
POST   /api/attributes/{id}/impact-preview
```

The order endpoint accepts the complete ordered list of current option IDs; reject missing, duplicate, and foreign IDs. The preferred update contract is the complete options list on `PUT /api/attributes/{id}`. Definition/option writes and previews require administrator authorization. Verify organization ownership and plugin restrictions for every operation.

### Preview and confirmation contract

The preview request identifies the action (`update_attribute`, `delete_attribute`, `update_option`, or `delete_option`), optional option ID, and exact proposed changes. For saves from the attribute editor, `update_attribute` includes the complete option draft and the preview reports the distinct assets affected by any contained option change. Validate the proposal and return affected-asset count, soft-deleted count, required-fields-left-empty count where relevant, current/proposed display values, and an opaque confirmation token.

Rename and delete mutations carry that token in an `X-Impact-Token` header. Require it even at zero usage; the UI can submit zero-impact renames immediately after preview. Preview is read-only and does not reserve data.

Persist per-organization signing material in the database so tokens work across restarts and replicas. Bind the signed token to organization, administrator, action, target definition/version, exact proposal, and affected asset IDs and relevant values. Expire it after ten minutes. Recompute and compare at commit in the transaction. Comparing only counts is insufficient: a different set can have the same count.

Return `409 Conflict` for missing, expired, or stale confirmation with a machine-readable indication that a fresh preview is needed. The client fetches a fresh preview and requests confirmation again when applicable. Invalid changes return `400`, authorization failures `403`, and missing or foreign targets `404`. Document payloads and headers in OpenAPI.

Coordinate normal asset writes, imports, and definition changes through a shared per-organization transaction lock, acquired before reading definitions or validating values and held through commit. This prevents references appearing between validation and mutation. Preview reads hold the same organization lock across definition and asset reads. Finer-grained locking can follow if measured contention warrants it.

### Asset validation

On asset create/update and import, load effective category attributes, including inherited ones. Validate select values against current options and mode and enforce required select fields. API callers receive the same validation as UI users.

Current asset handlers do not provide comprehensive attribute-schema validation. Implement select validation explicitly without introducing unrelated stricter validation of legacy non-select data. Existing plugin imports remain unchanged unless they supply select values, which must reference predefined options and pass the same checks.

## Compatibility and Delivery

- Add a new migration; leave historical migrations intact. Existing fields and values need no conversion.
- Block populated type/mode changes, including values on soft-deleted assets.
- Key renaming and value cleanup on field deletion extend existing behavior and require the preview contract. Coordinate frontend/backend delivery; older clients attempting these mutations receive confirmation-required conflicts.
- Delivery includes schema and transactional operations, API/OpenAPI changes, field management and asset rendering, and integration coverage.

## Acceptance Tests

- Create both modes; manage options; verify category inheritance and label rendering.
- Verify valid scalar/array saves and optional clearing; reject unknown/deleted choices, wrong types, duplicates, empty required selections, and inappropriate configuration.
- Verify label rename changes presentation without rewriting JSON; canonical rename updates scalars and arrays without touching unrelated data.
- Verify option deletion clears scalars, preserves other array choices, omits emptied keys, and reports emptied required fields.
- Verify field name changes, key migrations/collisions, and complete field deletion with category cleanup.
- Verify zero/one/many counts, quantities greater than one, inherited uses, changed categories, and soft-deleted assets.
- Verify dialog wording, old/new labels, cancellation, loading/errors, zero-impact behavior, and confirmation before affected renames or any deletion.
- Verify missing/expired/wrong-action/wrong-user/stale tokens, including same-count/different-set changes and concurrent writes/imports.
- Verify rollback leaves definitions and assets intact on failure; no partial rename or deletion.
- Verify admin permissions, organization isolation, protected plugin definitions, and legacy non-select behavior.

## Decisions for Review

Both selection modes and admin-only vocabulary management ship together. Option deletion clears values everywhere. Renames show impact whether they change presentation or stored data. Field operations use the same modal pattern. Suggestions and archiving are deferred.

Implementation defaults to review: defer populated type/mode conversions; include soft-deleted assets in counts and mutations; allow confirmed option deletion to empty required fields; expire previews after ten minutes.
