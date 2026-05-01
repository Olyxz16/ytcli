# Sync Mechanism

ytcli supports bidirectional synchronization between the local SQLite store and a remote YouTrack instance.

## Concepts

- **Pull**: Fetch remote issues and merge them into the local database.
- **Push**: Send local pending changes to the remote server.
- **Sync Status**: Every local issue has a `sync_status`:
  - `local` — created locally, never synced
  - `synced` — matches the remote state
  - `modified` — changed locally since last sync
  - `conflict` — both local and remote changed (manual resolution required)

## Pull Algorithm

1. Build a query: `project: {PROJ}` if a project is configured.
2. Fetch all matching remote issues via `service.ListIssues` (auto-paginated).
3. For each remote issue:
   - If no local match exists → insert as `synced`.
   - If local status is `modified` or `conflict` → skip (preserve local changes).
   - Otherwise → update local copy and mark as `synced`.

## Push Algorithm

1. Read all `pending` queue items with `attempts < 3`.
2. For each item:
   - Mark as `in_progress`.
   - Deserialize the JSON payload.
   - Execute the appropriate API call based on `operation`.
   - On success → mark as `completed`.
   - On failure → increment `attempts`, store error, mark as `pending`.
3. Prune `completed` items older than 24 hours.

## Queue Operations

| Operation | Action |
|---|---|
| `create` | Create issue on remote, then update local `remote_id` and `remote_db_id` |
| `update` | Call `UpdateIssue` with payload, then mark local as `synced` |
| `comment` | Call `AddComment` with text |
| `state` | Execute YouTrack command `State: <state>` |
| `tag` | Execute YouTrack command `tag <name>` |
| `untag` | Execute YouTrack command `untag <name>` |

## Conflict Handling

Currently, conflicts are detected but not automatically resolved. If an issue is locally `modified` when a Pull runs, the remote update is skipped. The user must manually reconcile (e.g., via `edit` or `sync --push` to force).

Future improvements could include:
- ETag-based optimistic concurrency
- Automatic merge for non-overlapping field changes
- A dedicated `resolve` command
