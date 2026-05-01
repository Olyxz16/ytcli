# Architecture

ytcli is structured in layers with clear separation between CLI commands, business logic, API clients, and local storage.

## Layers

```
cmd/ytcli          → entry point (main.go)
internal/cmdx      → Cobra commands & flag parsing
internal/tui       → Bubble Tea TUI (placeholder)
internal/render    → Output formatting (table, JSON, markdown, wide)
internal/service   → Business logic & pagination
internal/sync      → Bidirectional sync orchestration
internal/api       → YouTrack REST API client
internal/store     → SQLite local database
internal/model     → Shared data structures
internal/config    → Configuration loading & resolution
internal/local     → Local schema defaults & validation
```

## Data Flow

### Offline Mode

```
User → cmdx → store (SQLite) → render
```

All local commands (`add`, `list`, `edit`, `state`, `comment`, `tag-*`, `delete`) interact directly with the SQLite database in `.ytcli/store.db`. No network calls are made.

### Remote Mode (Direct API)

```
User → cmdx → service → api.Client → YouTrack REST API → render
```

Commands like `issues`, `create`, `cmd`, `projects` bypass the local store and hit the YouTrack API directly.

### Sync Mode

```
User → cmdx → sync.Manager
                    ├── Pull: api → store (merge remote into local)
                    └── Push: store queue → api (send local changes)
```

The sync manager bridges the offline and online worlds.

## Configuration Resolution

Configuration is merged from three sources:

1. **Global** (`~/.config/ytcli/config.yml`) — instances, default instance, output format
2. **Local** (`.ytcli.yml`) — per-directory instance override, project, local schema
3. **LocalPrivate** (`.ytcli.local.yml`) — gitignored local state (e.g., current task)

Resolution order: LocalPrivate > Local > Global. Missing values fall back to the next layer.

## Key Design Decisions

- **SQLite as primary data path**: The local store is the source of truth when working offline. Remote commands are secondary.
- **FTS5 for search**: Full-text search over issues is implemented via SQLite's FTS5 virtual table with triggers keeping the index in sync.
- **Sync queue for durability**: Local changes destined for remote are written to a `sync_queue` table. This ensures changes survive crashes and can be retried.
- **In-memory testing**: All store-layer tests use `:memory:` SQLite databases to avoid filesystem side effects.
