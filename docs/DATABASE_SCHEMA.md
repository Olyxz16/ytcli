# Database Schema

The local store uses SQLite (`modernc.org/sqlite`) with a single file per project: `.ytcli/store.db`.

## Tables

### `issues`

The core issue table. Every local or synced issue lives here.

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | Local ID (e.g., `#1`) |
| `remote_id` | TEXT | YouTrack readable ID (e.g., `PROJ-42`) |
| `remote_db_id` | TEXT | YouTrack internal ID |
| `summary` | TEXT NOT NULL | Issue title |
| `description` | TEXT | Issue body |
| `state` | TEXT NOT NULL DEFAULT 'Open' | Local workflow state |
| `priority` | TEXT NOT NULL DEFAULT 'Normal' | Local priority |
| `assignee` | TEXT | Login name of assignee |
| `created_at` | DATETIME | UTC timestamp |
| `updated_at` | DATETIME | UTC timestamp |
| `synced_at` | DATETIME | Last successful sync timestamp |
| `sync_status` | TEXT NOT NULL DEFAULT 'local' | `local`, `modified`, `synced`, `conflict` |
| `remote_etag` | TEXT | Future use for optimistic concurrency |

### `comments`

Comments attached to local issues.

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | Local comment ID |
| `issue_id` | INTEGER FK → issues(id) | Cascade delete |
| `remote_id` | TEXT | YouTrack comment ID |
| `text` | TEXT NOT NULL | Comment body |
| `author` | TEXT NOT NULL DEFAULT 'local' | Author name |
| `created_at` | DATETIME | UTC timestamp |
| `sync_status` | TEXT NOT NULL DEFAULT 'local' | `local`, `synced` |

### `tags`

Tag definitions.

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | Local tag ID |
| `name` | TEXT NOT NULL UNIQUE | Tag name |
| `remote_id` | TEXT | YouTrack tag ID |

### `issue_tags`

Many-to-many join between issues and tags.

| Column | Type | Notes |
|---|---|---|
| `issue_id` | INTEGER FK → issues(id) | Cascade delete |
| `tag_id` | INTEGER FK → tags(id) | |
| PK | (issue_id, tag_id) | Composite primary key |

### `sync_queue`

Pending operations to push to YouTrack.

| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | Queue item ID |
| `operation` | TEXT NOT NULL | `create`, `update`, `comment`, `state`, `tag`, `untag` |
| `entity_type` | TEXT NOT NULL | `issue`, `comment` |
| `local_id` | INTEGER NOT NULL | ID of the affected local entity |
| `payload` | TEXT NOT NULL | JSON payload for the operation |
| `created_at` | DATETIME | UTC timestamp |
| `attempts` | INTEGER DEFAULT 0 | Retry counter |
| `last_error` | TEXT | Last failure message |
| `status` | TEXT NOT NULL DEFAULT 'pending' | `pending`, `in_progress`, `completed`, `failed` |

### `schema_meta`

Tracks the current migration version.

| Column | Type | Notes |
|---|---|---|
| `key` | TEXT PK | e.g., `version` |
| `value` | TEXT | Version number |

## FTS5 Full-Text Search

The `issues_fts` virtual table indexes:
- `summary`
- `description`
- `state`
- `priority`
- `assignee`

It is kept in sync via triggers:
- `issues_ai` — insert into FTS on new issue
- `issues_au` — delete old FTS row, insert new on update
- `issues_ad` — delete from FTS on issue deletion

## Migrations

Migrations are sequential and idempotent. The current version is stored in `schema_meta`. On `store.Open()`, the migration runner checks the version and applies any missing migrations.

Current version: **1**

Migration v1 creates all tables, triggers, and the FTS5 virtual table.
