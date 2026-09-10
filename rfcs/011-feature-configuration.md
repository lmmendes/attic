# RFC 011: Configurable Feature Availability

> **Status**: Implemented
> **Created**: 2026-09-08
> **Author**: @lmmendes
> **Related issue**: [#63](https://github.com/lmmendes/attic/issues/63)

---

## Summary

Allow administrators to disable collections, plugins, conditions, locations, and warranties for their organization. Disabled features disappear from navigation, management pages, asset create and edit forms, dashboards, and inventory filters. All five features remain enabled by default.

## Motivation

Not every Attic installation needs every optional capability. Hiding unused features makes the interface easier to understand, and disabling plugins also prevents their managed categories and attributes from appearing in normal organization flows.

## Goals

- Provide organization-scoped, persistent feature settings.
- Restrict configuration changes to administrators.
- Hide disabled features throughout the user interface.
- Prevent direct API mutations from bypassing disabled UI controls.
- Preserve all existing data so a feature can be enabled again safely.

## Non-goals

- Disabling features outside the five supported feature flags.
- Deleting or migrating data when a feature is disabled.
- Configuring individual import plugins independently.
- Replacing deployment configuration or environment variables.

## Data Model

Each organization has one `feature_configurations` row with boolean columns for `collections_enabled`, `plugins_enabled`, `conditions_enabled`, `locations_enabled`, and `warranties_enabled`. Columns default to `true`, and migrations create rows for existing organizations. The repository also creates a default row on first access so newly created organizations remain compatible.

## API

`GET /api/configuration` returns the current organization's configuration to any authenticated member. The frontend needs read access to render the available capabilities consistently.

`PATCH /api/configuration` accepts any subset of the five boolean properties and requires an administrator session. Omitted properties retain their current value.

When a feature is disabled, the API returns `403 Forbidden` for its create, update, and delete operations. Plugin search and import are also blocked. Asset creation rejects disabled location, condition, and collection fields. Asset updates reject explicit changes to disabled fields while preserving existing values when those fields are omitted.

Read endpoints remain available for compatibility and data recovery. Disabling a feature never removes stored records or asset relationships.

## User Experience

Administrators manage the five switches on `/configuration`, linked from the administrator navigation. Non-administrators cannot access the page or update the API.

Disabled features are removed from desktop and mobile navigation. Their management routes redirect to the dashboard, and their controls are omitted from asset forms and inventory filters. Existing disabled values are hidden from ordinary asset and dashboard views.

When plugins are disabled, import controls and plugin pages are hidden. Plugin-managed categories and attributes are excluded from normal category, attribute, and asset selection flows. Existing imported assets remain stored.

## Compatibility and Rollout

All flags default to enabled, so upgrading preserves the existing interface and API behavior. Re-enabling a feature restores its preserved records and relationships without a migration or repair step.

## Testing

- Verify migration defaults and repository persistence.
- Verify authenticated reads and administrator-only updates.
- Verify feature middleware rejects disabled mutations.
- Verify asset writes reject explicit disabled fields and preserve omitted existing values.
- Verify each switch hides its navigation, forms, filters, routes, and existing-value presentation.
- Verify disabling plugins hides plugin-managed schema and blocks search/import.
