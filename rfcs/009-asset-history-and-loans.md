# RFC-009: Asset History and Loans

| Field   | Value       |
|---------|-------------|
| Status  | Proposed    |
| Created | 2026-09-02  |
| Author  | @lmmendes  |

## Summary

Attic currently renders an asset's creation and last-update timestamps as a
hardcoded history. This RFC replaces that placeholder with a unified asset
timeline containing:

- User-created events such as repairs, maintenance, notes, and issues
- The existing asset-created and last-updated entries
- Loan and return activity

Loans are stored as first-class records rather than as a flag on an asset.
Loaned and returned timeline entries are derived from those records, ensuring
that the current loan state and the displayed history cannot disagree.

The first version also highlights loans that are due or overdue inside Attic.
It does not send email or push notifications and does not introduce a
persistent notification inbox.

This RFC addresses [GitHub issue #37](https://github.com/lmmendes/attic/issues/37).

## Goals

- Allow users to add, edit, and delete dated events on an asset.
- Let organizations define reusable event categories.
- Track repeated loans and returns without losing prior loan history.
- Prevent more than one active loan for the same asset.
- Warn users inside Attic when a loan reaches its expected return date.
- Present manual events, loans, returns, creation, and last update in one
  chronological timeline.

## Non-Goals

- Lending part of an asset's quantity or creating multiple simultaneous loans
  for one asset
- Requiring borrowers to have Attic accounts
- Storing borrower email addresses, phone numbers, or other contact details
- Email, push, webhook, or scheduled background notifications
- Read/unread notification state
- Future-dated manual events or reminders for manual events
- An immutable audit log
- Recording which Attic user created an event or loan

## Design

### Source of Truth

An `is_loaned` flag on `assets` would describe only the current state. It would
not retain earlier loans, identify the borrower, or explain when the asset was
returned. It could also drift from separately stored timeline entries.

Instead, the `loans` table is the source of truth:

- An asset is available when it has no loan whose `returned_at` is `NULL`.
- An asset is loaned when it has exactly one loan whose `returned_at` is
  `NULL`.
- Returning an asset sets `returned_at` on the active loan.
- Each completed loan remains available for history and the loan overview.

Loaned and returned timeline items are generated when history is read. They are
not copied into the manual event table.

### Event Categories

Event categories are reusable and scoped to an organization. Each category
has:

- A name
- A Lucide icon identifier
- A color selected from the UI's supported palette
- Creation and update timestamps
- An optional archive timestamp

The migration creates Repair, Maintenance, Note, and Issue categories for each
existing organization. Names must be unique among active categories within an
organization, using case-insensitive comparison.

All authenticated users who can manage assets can also create, edit, and
archive event categories. Archived categories:

- Remain attached to existing events
- Retain their name, icon, and color in history
- Are omitted from the selector for new events
- Cannot be assigned to new events

Editing a category changes how all events using that category are presented.

### Manual Events

A manual event belongs to one asset and one event category. It contains:

- The time at which the activity occurred
- A required description
- Creation and update timestamps

The occurrence time may be in the past or present but not in the future.
Manual events can be edited and permanently deleted. They do not affect an
asset's loan state.

### Loans

A loan contains:

- The asset being loaned
- A required free-form borrower name
- The date and time the asset was loaned
- An optional expected return date
- The date and time it was returned, when applicable
- Creation and update timestamps

The loaned timestamp defaults to the current time and cannot be in the future.
The expected return date is a calendar date and cannot precede the loaned date.
A return timestamp defaults to the current time and cannot precede the loaned
timestamp.

While a loan is active, the borrower, loaned timestamp, and expected return
date can be edited. Once returned, the loan is final and cannot be edited or
returned again. Active and returned loans may both be permanently deleted.
Deleting a loan removes all timeline items derived from it.

Attic enforces one active loan per asset with a partial unique database index,
not only through application validation. Asset quantity does not change this
rule in the first version.

### Loan Status

Loan status is derived rather than stored:

| Condition | Status |
|-----------|--------|
| No active loan | `available` |
| Active loan without a due date | `loaned` |
| Active loan before its due date | `loaned` |
| Active loan on its due date | `due_today` |
| Active loan after its due date | `overdue` |

Date comparisons use the server's current calendar date. Returning, editing,
or deleting a loan changes its status and removes any warning immediately; no
notification cleanup job is required.

### Unified History

The history response merges the following sources and sorts them newest first:

1. The asset's `created_at` timestamp as `asset_created`
2. The asset's current `updated_at` timestamp as `asset_updated`
3. Manual asset events as `event`
4. Each loan's `loaned_at` timestamp as `loaned`
5. Each completed loan's `returned_at` timestamp as `returned`

The asset-created and last-updated items are synthetic and cannot be edited or
deleted. Manual event items expose event edit and delete actions. An active
loan's loaned item can open the loan editor. Deleting either timeline item for
a loan deletes the complete loan and therefore all timeline items derived from
it. Returned loans cannot be edited.

Every history item uses a stable source ID and source type as a tie-breaker
when two items have the same timestamp.

## Database Schema

The implementation adds three tables. The migration number must be the next
available number on the target branch; it is `000013` on `main` at the time of
writing.

```sql
CREATE TABLE event_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(100) NOT NULL,
    color VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_event_categories_active_name
    ON event_categories (organization_id, LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TABLE asset_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES event_categories(id),
    occurred_at TIMESTAMPTZ NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_asset_events_asset_occurred
    ON asset_events (asset_id, occurred_at DESC);

CREATE TABLE loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    borrower_name VARCHAR(255) NOT NULL,
    loaned_at TIMESTAMPTZ NOT NULL,
    due_date DATE,
    returned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (due_date IS NULL OR due_date >= loaned_at::date),
    CHECK (returned_at IS NULL OR returned_at >= loaned_at)
);

CREATE UNIQUE INDEX idx_loans_one_active_per_asset
    ON loans (asset_id)
    WHERE returned_at IS NULL;

CREATE INDEX idx_loans_active_due_date
    ON loans (due_date)
    WHERE returned_at IS NULL AND due_date IS NOT NULL;
```

The migration also adds `updated_at` triggers to the three new tables. Its down
migration removes the tables and their indexes. Deleting an asset cascades its
manual events and loans; deleting or archiving an event category never cascades
to events.

Repositories must validate that an event's asset and category belong to the
current organization because a simple foreign key does not enforce that
cross-table ownership rule.

## HTTP API

All endpoints require authentication and use the current organization scope.
The OpenAPI document is updated alongside implementation.

### Event Categories

```text
GET    /api/event-categories
POST   /api/event-categories
PUT    /api/event-categories/{id}
DELETE /api/event-categories/{id}
```

`GET` returns active categories by default. `DELETE` archives a category and
returns `204 No Content`.

Create and update requests use:

```json
{
  "name": "Maintenance",
  "icon": "i-lucide-wrench",
  "color": "blue"
}
```

Creating or renaming a category to an active duplicate returns `409 Conflict`.

### Manual Events and History

```text
GET    /api/assets/{assetId}/history
POST   /api/assets/{assetId}/events
PUT    /api/events/{eventId}
DELETE /api/events/{eventId}
```

Create and update requests use:

```json
{
  "category_id": "550e8400-e29b-41d4-a716-446655440000",
  "occurred_at": "2026-09-02T14:30:00Z",
  "description": "Replaced the damaged power cable."
}
```

The history endpoint returns a common presentation shape:

```json
[
  {
    "id": "loan-or-event-id",
    "type": "event",
    "occurred_at": "2026-09-02T14:30:00Z",
    "title": "Maintenance",
    "description": "Replaced the damaged power cable.",
    "category": {
      "id": "category-id",
      "name": "Maintenance",
      "icon": "i-lucide-wrench",
      "color": "blue",
      "archived": false
    },
    "editable": true,
    "deletable": true
  }
]
```

`type` is one of `asset_created`, `asset_updated`, `event`, `loaned`, or
`returned`. Category is present only for manual events. Loan-derived entries
include a loan summary so the client does not reconstruct borrower or due-date
text.

### Loans

```text
GET    /api/assets/{assetId}/loan
POST   /api/assets/{assetId}/loan
GET    /api/loans
PUT    /api/loans/{loanId}
DELETE /api/loans/{loanId}
POST   /api/loans/{loanId}/return
```

`GET /api/assets/{assetId}/loan` returns the active loan or `404 Not Found`.
Creating a second active loan returns `409 Conflict`.

Create and update requests use:

```json
{
  "borrower_name": "Alex",
  "loaned_at": "2026-09-02T15:00:00Z",
  "due_date": "2026-09-16"
}
```

The return endpoint accepts an optional timestamp:

```json
{
  "returned_at": "2026-09-10T18:15:00Z"
}
```

Omitting it uses the current time. Returning an already returned loan returns
`409 Conflict`; editing a returned loan returns `409 Conflict`.

`GET /api/loans` returns loans with asset name and active status. It supports:

- `status=active`: every unreturned loan
- `status=attention`: loans due today or overdue
- `status=due_today`
- `status=overdue`
- `status=returned`

### Asset Responses and Filtering

Asset list and detail responses gain an optional `active_loan` summary. Its
status is `loaned`, `due_today`, or `overdue`.

`GET /api/assets` accepts `loan_status`:

- `available`: no active loan
- `loaned`: any active loan, including due and overdue loans
- `due_today`
- `overdue`

Existing clients remain compatible because the new response field is optional
and the new filter is opt-in.

## User Interface

### Asset Detail

The existing Asset History placeholder becomes the unified timeline. It
provides:

- An Add Event action
- Category, timestamp, and description for manual events
- Edit and delete actions for manual events
- Loaned and returned entries with borrower and due-date details
- Edit and delete actions for active loans
- Delete actions for returned loans
- Non-editable Asset Created and Last Updated entries

The detail page also contains a current-loan card. An available asset shows a
Loan Asset action. An active loan shows borrower, loaned date, expected return
date, status, and Edit, Mark Returned, and Delete actions.

All destructive actions require confirmation.

### Asset List

Asset cards or rows show a Loaned, Due Today, or Overdue badge when applicable.
The filters include Available, Loaned, Due Today, and Overdue.

### Dashboard

The dashboard shows the count of loans requiring attention and links to the
filtered Loans page. Due-today and overdue records are visually distinct.

### Loans Page

A Loans entry is added to primary navigation. The page lists borrower, asset,
loaned date, expected return date, returned date, and derived status. Users can
filter active, attention-needed, due-today, overdue, and returned loans and
navigate to the associated asset.

### Event Category Management

An Event Categories page is added beside Attributes and Conditions. It uses
the same access model as asset management and supports creating, editing, and
archiving categories with an icon and palette color picker.

## Validation and Error Handling

- IDs must be valid UUIDs.
- Asset, category, event, and loan records must belong to the current
  organization.
- Borrower and manual-event descriptions are trimmed and cannot be empty.
- Category names are trimmed and unique among active organization categories.
- Icons and colors must come from supported UI values.
- Archived categories cannot be assigned to new or updated events.
- Manual occurrence and loaned timestamps cannot be in the future.
- Due and return dates cannot precede the loaned timestamp.
- Duplicate active loans and invalid lifecycle transitions return
  `409 Conflict`.
- Invalid request values return `400 Bad Request`.
- Missing or out-of-organization records return `404 Not Found` without
  exposing another organization's data.

## Testing

### Database and Repository Tests

- Apply and roll back the migration.
- Seed the four default categories for existing organizations.
- Enforce case-insensitive active category uniqueness.
- Preserve events when their category is archived.
- Cascade events and loans when an asset is deleted.
- Enforce one active loan per asset under concurrent inserts.
- Scope every category, event, loan, and history query to its organization.
- Sort mixed history entries deterministically.
- Calculate loaned, due-today, and overdue boundaries correctly.

### HTTP Tests

- Create, list, edit, and archive event categories.
- Create, edit, and delete manual events.
- Reject future events and archived or cross-organization categories.
- Create, read, edit, return, and delete loans.
- Reject duplicate active loans and edits or repeated returns after completion.
- Verify that deleting a loan removes both derived timeline entries.
- Return a unified timeline with all five item types.
- Filter assets and loan overviews by derived status.

### Frontend Tests

- Render mixed timeline entries in the correct order.
- Add, edit, delete, and confirm deletion of manual events.
- Create, edit, return, and delete loans from asset detail.
- Display loan status badges and asset filters.
- Display dashboard attention counts and links.
- Filter the Loans page and navigate to assets.
- Create, edit, and archive reusable event categories.
- Cover loading, empty, validation, and API error states.

## Security and Privacy

Borrower names are organization data and are visible to every authenticated
user who can view assets. This version intentionally does not collect borrower
contact information. All record lookups must enforce organization ownership,
including update and delete endpoints addressed by record ID.

## Alternatives Considered

### Store an `is_loaned` Flag on Assets

Rejected because it cannot represent repeated loans or borrower details and
can drift from history.

### Copy Loan Actions into Asset Events

Rejected because loan state and event rows would have to be updated and deleted
together. Deriving history from loans provides the same user experience with a
single source of truth.

### Model Loans as Manual Event Categories

Rejected because a category cannot enforce one active loan, lifecycle
transitions, return dates, due status, or borrower details.

### Append-Only History

Rejected for this feature because users must be able to correct or remove
manual events and loans. The timeline is operational history, not an audit log.

## Future Work

- Partial-quantity and simultaneous loans
- Borrower contacts or a reusable borrower directory
- Scheduled reminders before the expected return date
- Email, push, and webhook delivery
- Persistent notification inbox with read/unread state
- Future maintenance events and event reminders
- User attribution and immutable auditing
- Organization-specific timezone configuration

## Related Documents

- [Issue #37: Add custom events to asset history](https://github.com/lmmendes/attic/issues/37)
- [RFC-005: Email and Password Authentication](./005-email-password-auth.md)
