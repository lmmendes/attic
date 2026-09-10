# RFC 010: Custom Asset Events

> **Status**: Proposed
> **Created**: 2026-09-08
> **Author**: @lmmendes
> **Related issue**: [#37](https://github.com/lmmendes/attic/issues/37)

---

## Summary

Add user-managed events to an asset's history. An event records a title, an
optional description, a Lucide icon, and the date and time when it occurred.
Users can create, edit, and permanently delete events from the asset detail
page.

Custom events share the existing Asset History timeline with the generated
"Asset Created" and "Last Updated" entries. The combined timeline gives users
one place to record repairs, maintenance, issues, and notes.

## Motivation

The asset detail page currently derives its history from the asset's creation
and update timestamps. This shows when the database record changed, but cannot
describe events in the asset's real-world lifecycle. A repaired appliance, a
serviced bicycle, or a damaged collectible all require context that does not
belong in the asset's general notes or current condition.

Custom events make that history explicit while keeping the entry model small.
The title and icon provide visual and semantic cues without imposing an event
taxonomy.

## Goals

- Let authenticated users add timestamped events to an existing asset.
- Let users correct an event after it has been saved.
- Let users permanently delete an event after confirmation.
- Present custom events and generated asset lifecycle entries in one timeline.
- Preserve the event's occurrence time and allow both historical and future timestamps.
- Scope every event operation to the current workspace and parent asset.
- Document the event API in the existing OpenAPI specification.

## Non-goals

- Structured event categories or organization-managed event types.
- Duration tracking.
- Recurring events and maintenance schedules.
- Reminders or notifications for future events.
- Loan, borrower, reservation, or movement tracking.
- Event attachments.
- A global event feed, search, filtering, or reporting.
- Recording or displaying the user who created or edited an event.
- A general-purpose audit log of asset changes.

## Terminology

| Term | Meaning |
|---|---|
| Custom event | A timestamped history entry created and managed by a user |
| Generated entry | The non-editable Asset Created or Last Updated entry derived from the asset |
| Occurrence time | The user-selected date and time at which an event occurred |

## User Experience

### Timeline

The asset detail page retains its **Asset History** section and adds an
**Add event** action to the section header. The timeline contains:

- **Asset Created**, derived from `asset.created_at` and not editable;
- **Last Updated**, derived from `asset.updated_at` and not editable; and
- zero or more custom events, rendered with their selected icon, title,
  optional description, and localized occurrence time.

The combined timeline is ordered by occurrence timestamp descending. Generated
entries use their source timestamps. A stable ID and entry type provide the
final tie-break so the order does not change between renders.

Creating or editing an event does not update the parent asset's `updated_at`.
The Last Updated entry therefore continues to mean that the asset record itself
was modified.

### Creating an event

Selecting **Add event** opens a modal containing:

1. A required title.
2. An optional description.
3. A required date and time, defaulted to the user's current local date and time.
4. An accessible, curated Lucide icon grid, defaulted to
   `i-lucide-calendar`.

Past, current, and future timestamps are accepted. A future event is still an event,
not a reminder: this RFC does not add notifications or a separate scheduled
state.

After a successful save, the modal closes and the timeline refreshes. If the
request fails, the modal remains open, the entered values are preserved, and a
useful error message is displayed.

### Editing and deleting events

Each custom event exposes edit and delete actions. Editing opens the same modal
with the event's values pre-filled. Generated entries do not expose either
action.

Deleting opens a confirmation dialog naming the event. Confirming permanently
deletes the record and refreshes the timeline. Cancelling makes no changes.

## Data Model

Add an `asset_events` table:

```sql
CREATE TABLE asset_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL CHECK (length(trim(title)) > 0),
    description TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 2000),
    icon VARCHAR(100) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_asset_events_asset_occurred_at
    ON asset_events(asset_id, occurred_at DESC, created_at DESC);

CREATE TRIGGER update_asset_events_updated_at
    BEFORE UPDATE ON asset_events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

`occurred_at` is an instant. API clients send RFC3339 date-times and render the
stored value in the viewer's local timezone.

Events inherit their workspace ownership through the parent asset. Repository
queries join the asset and require its `organization_id` to match the current
workspace. Events belonging to soft-deleted assets are not accessible. A later
hard deletion of the asset removes its events through the foreign-key cascade.

The down migration drops the trigger, index, and table in that order.

## Domain Model

```go
type AssetEvent struct {
    ID          uuid.UUID          `json:"id"`
    AssetID     uuid.UUID          `json:"asset_id"`
    Title       string             `json:"title"`
    Description string             `json:"description"`
    Icon        string             `json:"icon"`
    OccurredAt  time.Time          `json:"occurred_at"`
    CreatedAt   time.Time          `json:"created_at"`
    UpdatedAt   time.Time          `json:"updated_at"`
}
```

The API and frontend represent `occurred_at` as an RFC3339 date-time string.

## API Design

All endpoints require authentication. The asset ID and event ID are UUIDs.

```text
GET    /api/assets/{id}/events
POST   /api/assets/{id}/events
PUT    /api/assets/{id}/events/{eventId}
DELETE /api/assets/{id}/events/{eventId}
```

Keeping every operation nested under the asset makes the parent relationship
explicit and ensures update and delete requests cannot address an event through
the wrong asset.

### List events

`GET /api/assets/{id}/events` returns `200 OK` and an array ordered by
`occurred_at DESC`, `created_at DESC`, and `id DESC`. An asset with no custom
events returns `[]`, not `null`. A missing or soft-deleted asset returns
`404 Not Found`.

### Create an event

`POST /api/assets/{id}/events` accepts:

```json
{
  "title": "Replaced drive belt",
  "description": "Installed the manufacturer's replacement part.",
  "icon": "i-lucide-wrench",
  "occurred_at": "2026-09-08T14:30:00Z"
}
```

A successful request returns `201 Created` with the complete event:

```json
{
  "id": "9ceeca85-306a-49dd-a5ab-94004335d454",
  "asset_id": "0b242c81-76ab-4e67-b07c-ad520237c867",
  "title": "Replaced drive belt",
  "description": "Installed the manufacturer's replacement part.",
  "icon": "i-lucide-wrench",
  "occurred_at": "2026-09-08T14:30:00Z",
  "created_at": "2026-09-08T14:42:00Z",
  "updated_at": "2026-09-08T14:42:00Z"
}
```

The response serializer emits `occurred_at` as an RFC3339 timestamp. Clients
display it in the viewer's local timezone.

### Update an event

`PUT /api/assets/{id}/events/{eventId}` accepts the same complete body as
creation. It replaces all editable fields and returns `200 OK` with the updated
event. It returns `404 Not Found` if the asset is unavailable, the event does
not exist, or the event belongs to a different asset or workspace.

### Delete an event

`DELETE /api/assets/{id}/events/{eventId}` permanently removes a matching event
and returns `204 No Content`. The same parent and workspace checks used for
updates apply. A missing or mismatched event returns `404 Not Found`.

## Validation

The server is authoritative and applies identical validation during creation
and update:

- Trim the title and require 1 to 255 Unicode characters.
- Trim the optional description and allow up to 2,000 Unicode characters.
- Require an icon of at most 100 bytes matching
  `^i-lucide-[a-z0-9]+(?:-[a-z0-9]+)*$`.
- Require `occurred_at` to be an RFC3339 date-time with an explicit timezone.
- Accept timestamps before, at, or after the current time.
- Reject unknown JSON shapes or malformed request bodies with `400 Bad Request`
  where supported by the shared decoder conventions.

The curated frontend picker improves usability but is not the authoritative
list of allowed icons. Any syntactically valid Lucide icon name is accepted by
the API, matching the existing collection icon behavior.

## Error Handling

| Situation | Response |
|---|---|
| Invalid asset or event UUID | `400 Bad Request` |
| Invalid body or field value | `400 Bad Request` with a useful validation message |
| Missing or soft-deleted asset | `404 Not Found` |
| Missing event or parent/workspace mismatch | `404 Not Found` |
| Unexpected persistence failure | `500 Internal Server Error` |

Parent and workspace mismatches deliberately use `404` so the API does not
reveal whether an event exists elsewhere.

## Security and Integrity

- Every list, create, update, and delete operation verifies an active parent
  asset in the current workspace.
- Update and delete repository statements include both `asset_id` and event
  `id`; handlers do not rely on a prior unscoped lookup.
- User text is rendered as text, never injected as HTML.
- Icon values are constrained to the Lucide naming format before rendering.
- Database constraints remain a second line of defense for field limits.
- Cascading deletion prevents orphaned event records after a hard asset delete.

## Compatibility

- Existing assets require no backfill and initially have no custom events.
- Existing Asset, Warranty, and Attachment response shapes do not change.
- The generated Asset Created and Last Updated entries remain frontend-derived;
  they are not copied into `asset_events`.
- Clients unaware of the new endpoints continue to function unchanged.
- Event endpoints are additive and do not alter asset create or update behavior.

## Implementation Plan

1. Add the migration, `AssetEvent` domain type, repository, test fixture support,
   and repository tests.
2. Register the repository and nested routes, implement handlers and validation,
   and document the endpoints and schemas in OpenAPI.
3. Add the frontend event type and extract the Asset History timeline into a
   focused component that fetches and mutates custom events.
4. Add the shared create/edit modal, curated accessible icon picker, and delete
   confirmation flow.
5. Add handler and frontend tests, then run backend tests, frontend tests,
   frontend type checking and linting, and the production build.

## Test Plan

### Repository

- Create, retrieve, update, and delete an event.
- Return events in deterministic reverse-chronological order.
- Return an empty slice when an asset has no events.
- Reject or hide events through the wrong asset or workspace.
- Hide events for soft-deleted assets.
- Remove events when their asset is hard-deleted.

### HTTP

- Exercise successful list, create, update, and delete responses.
- Reject malformed UUIDs, JSON, dates, titles, descriptions, and icon names.
- Accept past, current, and future dates.
- Return `404` for missing assets, missing events, and mismatched event/asset
  pairs.
- Return `[]` for a valid asset without custom events.

### Frontend

- Render generated and custom entries in one deterministic timeline.
- Render the selected icon, title, optional description, and localized occurrence time.
- Open the add modal with the current local date and time and default calendar icon.
- Submit create and update payloads and refresh after success.
- Preserve form state and show an error after a failed save.
- Pre-fill all fields when editing an event.
- Require confirmation before deletion and refresh after success.
- Keep generated entries non-editable.
- Verify accessible names and selected state for icon and action controls.
- Verify the timeline and modal remain usable on mobile-width layouts.

## Acceptance Criteria

- An authenticated user can create, edit, and delete custom events from an
  existing asset's detail page.
- Events contain a title, optional description, icon, and occurrence timestamp
  as editable domain fields.
- Custom and generated entries appear together in reverse-chronological order.
- Future event dates are accepted without adding reminder semantics.
- All event operations are scoped to the current workspace and URL asset.
- The API, migration behavior, validation, UI states, and tests are documented
  sufficiently for implementation without additional product decisions.
