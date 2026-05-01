# ytcli — Offline-First Local Issue Tracker Plan

## Core Concept

ytcli is a local-first issue tracker backed by SQLite. All daily operations hit the local DB instantly. `ytcli sync` bridges to YouTrack on demand for collaboration.

- **Standalone**: Works with zero YouTrack configuration. `ytcli init` in any directory creates a local project.
- **Local-first**: All `add`, `list`, `show`, `edit`, `state`, `comment`, `tag` commands operate on the local SQLite DB.
- **Sync on demand**: `ytcli sync` performs bidirectional sync with YouTrack. Pull remote changes, push local queue.
- **Hybrid IDs**: `#1`, `#2` locally. After sync, same issue also resolves as `PROJ-42`.

## ID Scheme

- **Local**: `#1`, `#2`, `#3` — monotonically increasing, scoped to the project. The `#` prefix is optional: `show 1` works too.
- **After sync**: YouTrack assigns `PROJ-42`. Both `#1` and `PROJ-42` resolve to the same local issue.
- **Single project context** (after `ytcli init`): just `#3`
- **Cross-project**: project prefix disambiguates: `PROJ-3`

Resolution logic:

```
#1  →  look up local issue WHERE id = 1
1   →  same as #1
PROJ-42 → look up local issue WHERE remote_id = "PROJ-42"
       → if not found locally, fall back to remote API
```

## Command Structure

```
─── Local (offline-first, primary interface) ──────────────
ytcli init                         Initialize project (.ytcli/ + store.db)
ytcli add "Summary"               Create local issue
ytcli list [query]                 List local issues (FTS5 search)
ytcli ls                           Alias for list
ytcli show <#id|remote-id>         Show issue detail (local first, remote fallback)
ytcli edit <id> -s "..." -d "..." Edit issue fields (local first)
ytcli state <id> <state>           Change state
ytcli done <id>                    Shortcut: move to Done
ytcli comment <id> <text>          Add comment (local first)
ytcli open <id>                    Open issue in $EDITOR
ytcli tag <id> <tag>               Add tag
ytcli untag <id> <tag>             Remove tag

─── Remote (direct API, online only) ─────────────────────
ytcli issues [query]               List YouTrack issues (direct API)
ytcli remote show <remote-id>      Show remote issue (bypass local DB)
ytcli cmd <remote-id> <command>    Apply YouTrack command (direct API)
ytcli remote create <project> ...  Create directly on YouTrack

─── Sync ──────────────────────────────────────────────────
ytcli sync                         Full bidirectional sync
ytcli sync --push                  Push local changes only
ytcli sync --pull                  Pull remote changes only
ytcli sync --dry-run               Show what would happen
ytcli sync --status                Show sync status of local issues

─── Setup ─────────────────────────────────────────────────
ytcli config setup                 Interactive global config
ytcli config init                  Initialize .ytcli.yml + .ytcli.local.yml
ytcli config set/get               Get/set config values
ytcli auth login                    Authenticate with YouTrack

─── Info ──────────────────────────────────────────────────
ytcli projects                     List projects (local + remote)
ytcli tags                         List tags
ytcli version                      Version info
ytcli completion <shell>           Shell completions
```

## Command Behavior: Local-First with Remote Fallback

Every command resolves IDs locally first. If an ID isn't found locally, it falls back to the remote API.

| Command | Local Found | Local Not Found |
|---|---|---|
| `ytcli show #3` | Show local data | Error: not found |
| `ytcli show PROJ-42` | Show local data (mapped) | Fall back to remote API |
| `ytcli edit PROJ-42 -s "new"` | Edit local + queue sync | Hit API directly |
| `ytcli comment PROJ-42 "text"` | Add local comment + queue sync | Hit API directly |
| `ytcli issues [query]` | Always hits remote API | Always hits remote API |
| `ytcli cmd PROJ-42 "..."` | Always hits remote API | Always hits remote API |

## SQLite Schema (per-project, stored at `.ytcli/store.db`)

```sql
CREATE TABLE issues (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    remote_id TEXT,                    -- YouTrack readable ID (e.g. "PROJ-42"), NULL until synced
    remote_db_id TEXT,                 -- YouTrack database ID (e.g. "2-42"), NULL until synced
    summary TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL DEFAULT 'Open',
    priority TEXT NOT NULL DEFAULT 'Normal',
    assignee TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    synced_at DATETIME,                -- NULL if never synced
    sync_status TEXT NOT NULL DEFAULT 'local',  -- 'local'|'synced'|'modified'|'conflict'|'deleted_remote'
    remote_etag TEXT,                  -- for conflict detection (YouTrack updated timestamp)
    UNIQUE(remote_id)
);

CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    remote_id TEXT,                    -- YouTrack comment ID, NULL until synced
    text TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT 'local',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sync_status TEXT NOT NULL DEFAULT 'local'
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    remote_id TEXT
);

CREATE TABLE issue_tags (
    issue_id INTEGER REFERENCES issues(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id),
    PRIMARY KEY(issue_id, tag_id)
);

CREATE TABLE sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL,              -- 'create'|'update'|'comment'|'state'|'delete'|'tag'|'untag'
    entity_type TEXT NOT NULL,            -- 'issue'|'comment'
    local_id INTEGER NOT NULL,           -- local issue.id or comment.id
    payload TEXT NOT NULL,                -- JSON: fields that changed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    attempts INTEGER DEFAULT 0,
    last_error TEXT,
    status TEXT NOT NULL DEFAULT 'pending'  -- 'pending'|'in_progress'|'failed'|'completed'
);

CREATE TABLE schema_meta (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- FTS5 for fast local search
CREATE VIRTUAL TABLE issues_fts USING fts5(
    summary, description, state, priority, assignee,
    content=issues, content_rowid=id
);

CREATE TRIGGER issues_ai AFTER INSERT ON issues BEGIN
    INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
    VALUES (new.id, new.summary, new.description, new.state, new.priority, new.assignee);
END;

CREATE TRIGGER issues_au AFTER UPDATE ON issues BEGIN
    INSERT INTO issues_fts(issues_fts, rowid, summary, description, state, priority, assignee)
    VALUES ('delete', old.id, old.summary, old.description, old.state, old.priority, old.assignee);
    INSERT INTO issues_fts(rowid, summary, description, state, priority, assignee)
    VALUES (new.id, new.summary, new.description, new.state, new.priority, new.assignee);
END;
```

## Local Schema Configuration (`.ytcli.yml`)

```yaml
instance: work               # optional — can be unset for local-only projects
project: PROJ                 # optional — auto-assigned on sync if not set

local:
  states:
    - Open
    - In Progress
    - Review
    - Done
    - Closed
  priorities:
    - Critical
    - Major
    - Normal
    - Minor
  default_state: Open
  default_priority: Normal
  done_states: [Done, Closed]
```

If `instance` and `project` are unset, `ytcli sync` will prompt to configure them. The tool works fully standalone without them.

## File Structure

```
internal/
├── store/
│   ├── sqlite.go              # DB connection, migrations, initialization
│   ├── issues.go              # Local issue CRUD + search
│   ├── comments.go            # Local comment CRUD
│   ├── tags.go                # Local tag CRUD
│   ├── queue.go               # Sync queue operations
│   └── migrations.go          # Schema version management
├── sync/
│   ├── manager.go             # Sync orchestration
│   ├── pull.go                # Pull remote changes to local DB
│   ├── push.go                # Push queued local changes to YouTrack
│   ├── conflict.go            # Conflict detection & resolution
│   └── merge.go               # Field-level merge logic
├── local/
│   ├── service.go             # Local issue business logic
│   ├── id.go                  # ID resolution (#1, PROJ-42 → local issue)
│   └── config.go              # Local schema defaults (states, priorities)
├── cmdx/
│   ├── root.go                # Modified: detect local DB context
│   ├── init.go                # `ytcli init` — creates .ytcli/ + store.db + config
│   ├── add.go                 # `ytcli add "Summary"`
│   ├── list.go                # `ytcli list` / `ytcli ls`
│   ├── show.go                # Modified: resolves local #id or remote ID
│   ├── edit.go                # Modified: edits local issue, queues sync
│   ├── state.go                # `ytcli state <id> <state>` / `ytcli done <id>`
│   ├── comment.go             # Modified: local-first comments
│   ├── sync.go                # `ytcli sync`
│   ├── remote.go              # `ytcli remote show/create` — direct API calls
│   ├── tag.go                 # Modified: local-first tags
│   ├── open.go                # Modified: open in $EDITOR
│   └── (existing commands: auth, config, version, completion)
├── api/                        # Existing, unchanged
├── config/                     # Existing, unchanged
├── model/                      # Existing, unchanged
├── render/                     # Existing, unchanged
└── service/                    # Existing, unchanged (remote API layer)
```

## Sync Flow

```
ytcli sync
│
├── 1. PULL (remote → local, per project)
│   │   GET /api/issues?query=project:{PROJ}&$top=100&fields=...
│   │   For each remote issue:
│   │   ├── Match by remote_id in local DB
│   │   │   ├── Not found → INSERT (new remote issue, sync_status='synced')
│   │   │   ├── Found, sync_status='synced' → UPDATE with remote changes
│   │   │   ├── Found, sync_status='modified' → CONFLICT DETECT
│   │   │   │   ├── timestamps: local newer → keep local, queue push
│   │   │   │   ├── timestamps: remote newer → overwrite local, warn
│   │   │   │   └── both changed same field → mark 'conflict', surface to user
│   │   │   └── Found, sync_status='conflict' → skip, warn user
│   │   └── Detect remote deletions:
│   │       Local issues with sync_status='synced' that no longer appear
│   │       in remote results → mark 'deleted_remote', surface to user
│   └── Pull remote schema updates (states, priorities, custom fields)
│
├── 2. PUSH (local → remote)
│   │   Process sync_queue in creation order:
│   │   ├── 'create': POST /api/issues → get remote_id, update local record
│   │   ├── 'update': POST /api/issues/{remote_id} with changed fields
│   │   ├── 'comment': POST /api/issues/{remote_id}/comments
│   │   ├── 'state': POST /api/commands with "State: {state}"
│   │   ├── 'tag': POST /api/issues/{remote_id}/tags
│   │   ├── 'untag': DELETE /api/issues/{remote_id}/tags/{tag_id}
│   │   └── 'delete': DELETE /api/issues/{remote_id}
│   │   On success: mark sync_status='synced', remove from queue
│   │   On failure: increment attempts, store error, keep in queue
│   └── Max 3 retry attempts per operation
│
└── 3. RECONCILE
    ├── Set synced_at on all synced issues
    ├── Report summary: "3 pulled, 2 pushed, 1 conflict, 0 failures"
    └── Clean completed queue items older than 24h
```

## State Transitions

```
                    ytcli add
                       │
                       ▼
                  ┌────────┐
          ┌──────►│  local ◄──────┐──────┐
          │       └────┬───┘      │      │
          │            │          │      │
          │     ytcli edit       sync   sync
          │     ytcli state      --pull  --push
          │            │          │      │
          │            ▼          │      │
          │       ┌─────────┐   │      │
          │       │ modified │───┘      │
          │       └────┬─────┘         │
          │            │               │
          │     ytcli edit             │
          │            │               │
          │            ▼               │
          │       ┌─────────┐         │
          │       │ modified │─────────┘
          │       └────┬─────┘
          │            │
          │       sync --push succeeds
          │            │
          │            ▼
          │       ┌────────┐
          └───────│ synced │◄─────── sync --pull
                  └────┬───┘
                       │
                  conflict (rare)
                       │
                       ▼
                  ┌──────────┐
                  │ conflict  │─── user resolves: ytcli edit #3 --resolve
                  └──────────┘
```

## Key Design Decisions

1. **Local operations are instant** — SQLite writes are <1ms. All local commands work offline with zero network.
2. **Remote commands unchanged** — `ytcli issues`, `remote show`, `cmd` all still hit the API directly. They bypass the local DB.
3. **`.ytcli/store.db` is gitignorable** — it's a local cache. The source of truth is either local (for unsynced issues) or YouTrack (for synced ones).
4. **The queue is durable** — even if sync fails mid-way, pending operations survive in SQLite. Next `ytcli sync` picks up where it left off.
5. **Schema comes from `.ytcli.yml`** — local states/priorities are defined there. On first sync, YouTrack's schema is pulled and used to update the local config.
6. **Per-project DB** — each project gets its own `.ytcli/store.db`. Portable, git-ignorable, deletable.
7. **Standalone mode** — `ytcli init` works without any YouTrack instance. Pure local mode for small projects. Add a remote later when you need collaboration.

## Implementation Phases

### Phase A: Local Store + CRUD (MVP)

- SQLite store, migrations, initialization
- `ytcli init` — initialize `.ytcli/` directory + `store.db` + update `.ytcli.yml`
- `ytcli add "Summary"` — create local issue with state/priority/assignee flags
- `ytcli list` / `ytcli ls` — list local issues with state coloring
- `ytcli show #3` — show local issue detail (reuse render package)
- `ytcli edit #3` — edit summary/description/state/priority
- `ytcli state #3 InProgress` — change state
- `ytcli done #3` — shortcut to Done state
- FTS5 search in `ytcli list [query]`

### Phase B: Local Completeness

- `ytcli comment #3 "text"` — local comments
- `ytcli tag #3 bug` / `ytcli untag #3 bug` — local tags
- Rich list rendering with lipgloss tables
- `$EDITOR` integration with `ytcli open #3`
- ID resolution: `#1`, `1`, `PROJ-42` all work
- `ytcli sync --status` — show which issues are local/modified/synced

### Phase C: Sync Engine

- `ytcli sync --pull` — fetch remote issues into local DB
- `ytcli sync --push` — flush sync_queue to YouTrack
- `ytcli sync` — bidirectional (pull then push)
- `ytcli sync --dry-run` — preview what would sync
- Auto-assign remote IDs on push (create gets YouTrack ID back)
- Retry logic for failed sync operations

### Phase D: Conflict Resolution + Polish

- Conflict detection on pull
- Interactive conflict resolution with `huh`
- `ytcli sync --force` flag
- `ytcli remote show/create` commands for direct API access
- Performance optimization (bulk operations, incremental sync with `$updated` filter)
- Display sync status indicators in `ytcli list` output

### Phase E: Advanced Features

- Offline command queueing (queue `ytcli cmd` operations for sync)
- Issue linking between local issues
- `ytcli list --mine` — filter by assignee matching current user
- Watch mode: `ytcli sync --watch` — periodic background sync
- Export/import for project portability