# AGENTS.md — tkt CLI Agent Reference

This document describes the agent-compatible interface for `tkt`. AI agents and automation tools should use `--output json` (or `-o json`) and `--quiet` (or `-q`) flags for machine-readable output.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication error (`AuthError`, `NetworkError`) — re-run `tkt remote auth` |
| 3 | Not found (`NotFoundError`) — resource not available |
| 4 | Validation error (`ValidationError`) — invalid input or state transition |

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--instance` | `-i` | Target provider instance name (overrides project config) |
| `--output` | `-o` | Output format: `table`, `json`, `wide`, `markdown` |
| `--quiet` | `-q` | Minimal output (implies `--output json` unless overridden) |

## Output Modes

- **table** (default): Human-readable terminal output with colors and formatting.
- **json**: Pretty-printed JSON to stdout.
- **quiet** (`-q`): Minimal output — typically just IDs, one per line; implies `--output json` unless `-o` is explicitly set.

When `--output json` is used, errors are also emitted as JSON: `{"error": "...", "message": "..."}`.

## ID Resolution

Commands accept multiple ID formats:
- **Local ID**: `#1`, `1` — resolves to local SQLite issue.
- **Remote ID**: `PROJ-42` — resolves via remote API.
- Resolution order: local database first, then remote API fallback.

## Configuration

Config is project-based (no global config):
- `.tktrc.yml` — committed, contains `provider`, `project`, `local` schema.
- `.tkt.local.yml` — gitignored, contains `current_task`.

Key config values:
- `provider.name` — provider identifier (e.g., `youtrack`).
- `provider.url` — provider base URL.
- `project` — project short name for queries and issue creation.
- `default_query` — default filter for `tkt issues`.
- `local.states` — valid state values.
- `local.priorities` — valid priority values.
- `local.done_states` — states that mark an issue as done.

### Config Commands

```bash
tkt config init              # Initialize .tktrc.yml and .tkt.local.yml
tkt config set <key> <value> # Set config value
tkt config get <key>         # Get resolved config value
```

**Set keys**: `provider.name`, `provider.url`, `project`, `default_query`, `wiki_dir`.

## Authentication

```bash
tkt remote auth [provider] [--token TOKEN] [--oauth]
```

Authenticates with the remote provider. Supports `--token` for non-interactive auth. Stores credentials in the OS keyring with fallback to `~/.config/tkt/credentials.yml`.

```bash
tkt remote whoami            # Show current user
```

## Commands Reference

### Local Project

```bash
tkt init                     # Initialize local project (interactive)
```

### Issues

```bash
tkt add [summary]            # Create local-only issue
  -s, --summary <text>       # Issue summary
  -d, --description <text>   # Description
      --state <state>        # Initial state (default: Open)
  -P, --priority <prio>      # Priority (default: Normal)
  -a, --assignee <user>      # Assignee
      --tag <tag>            # Tag (repeatable)

tkt list [query]             # List local issues
      --limit <n>            # Max results (default: 100)
      --state <state>        # Filter by state

tkt issues [query]           # Search remote issues
  -p, --project <proj>       # Filter by project
  -a, --assignee <user>      # Filter by assignee ('me' = current user)
  -s, --state <state>        # Filter by state
      --limit <n>            # Max results (default: 50, 0 = all)
      --sort <expr>          # Sort expression

tkt show <issue-id>          # Show issue detail
      --comments             # Include comments
  -w, --web                  # Open in browser

tkt edit <issue-id>          # Edit an issue
  -s, --summary <text>       # New summary
  -d, --description <text>   # New description
      --state <state>        # New state
  -P, --priority <prio>      # New priority
  -a, --assignee <user>      # New assignee

tkt state <issue-id> <state> # Change issue state
      --force                # Bypass state validation

tkt done <issue-id>          # Mark issue as done (uses configured done state)
      --force                # Bypass state validation

tkt delete <issue-id>        # Delete an issue (local first, then remote)
```

### Remote Commands

```bash
tkt create <project>         # Create remote issue
  -s, --summary <text>       # Required
  -d, --description <text>   # Description (also reads from stdin)
  -t, --type <type>          # Issue type
  -P, --priority <prio>      # Priority
  -a, --assignee <user>      # Assignee ('me' = current user)
      --tag <tag>            # Tag (repeatable)

tkt cmd <issue-id> <command> # Apply YouTrack command
  -c, --comment <text>       # Add comment alongside command
      --silent               # Apply without notifications

tkt projects                 # List projects
tkt project <id>             # Show project details
```

### Comments

```bash
tkt comment <issue-id> <text>  # Add a comment
tkt comments <issue-id>        # List comments
```

### Tags

```bash
tkt tags                       # List all tags
tkt tag-add <issue-id> <tag>   # Add a tag
tkt tag-remove <issue-id> <tag> # Remove a tag
```

### Links & Work

```bash
tkt link <issue-id> <target-id> --type <link-type-id>  # Link two issues

tkt log <issue-id> <duration>  # Log work time
  -t, --type <type>            # Work item type
  -d, --date <YYYY-MM-DD>      # Date (default: today)
```

Duration format: `1h30m`, `2d`, `45m` (d = 8h workday).

### Sync

```bash
tkt sync                       # Bidirectional sync: pull then push
      --dry-run                # Preview changes without applying

tkt remote pull                # Pull remote issues only
tkt remote push                # Push local changes only
tkt remote status              # Show sync status and queue summary
```

**JSON mode** (`--output json`):
- `sync`: `{"Pulled": N, "Pushed": N, "Conflicts": N, "Failed": N, "Errors": [...]}`
- `remote status`: `{"statuses": {...}, "queue": {"pending": N, "failed": N, "completed": N}, "conflicts": N}`

**Quiet mode** (`--quiet`):
- `sync`: prints `Pulled Pushed Conflicts Failed` as space-separated numbers.

### Articles (YouTrack Knowledge Base)

```bash
tkt articles [query]           # List articles
      --limit <n>              # Max results (default: 50)
      --skip <n>               # Offset

tkt wiki                       # Sync wiki articles
      --pull                   # Download articles to local wiki_dir
      --push                   # Upload local edits to YouTrack
      --status                 # Check if local wiki is up to date
```

Wiki directory must be configured with `tkt config set wiki_dir <path>`.

### Other

```bash
tkt open <issue-id>            # Open in browser or $EDITOR
tkt tui                        # Launch interactive TUI

tkt version                    # Show version
  --output json: {"version": "...", "commit": "..."}
  --quiet: prints just the version string
```

## Conflict Resolution

When a sync conflict is detected (local issue modified while remote changed), three strategies are available:

1. **manual** (default): Stores conflict in DB, sets issue status to `conflict`. Resolve via `tkt remote status`.
2. **local-wins**: Keeps local version, skips remote update.
3. **remote-wins**: Overwrites local version with remote.

Use `tkt remote status` to view unresolved conflicts.

## Workflow State Validation

State transitions are validated against the project schema (from `.tktrc.yml` or remote `SchemaConfig`). Invalid transitions return exit code 4.

Bypass validation with `--force`:
```bash
tkt state PROJ-1 "InvalidState" --force
tkt done PROJ-1 --force
```

## Error Hints

The CLI prints actionable hints after errors:

| Error Type | Hint |
|-----------|------|
| `AuthError` | "Re-authenticate with: tkt remote auth" |
| `NetworkError` | "Check network and provider URL in .tktrc.yml" |
| `ValidationError` | "Verify input values and try again" |
| `NotFoundError` | "Check the ID exists on the remote" |
| `ConflictError` | "Resolve with: tkt remote status" |

## Retry Behavior

Sync operations retry automatically on transient network failures:
- Exponential backoff: 1s → 2s → 4s
- Max 3 retries
- No retry on `AuthError`, `ValidationError`, `NotFoundError`

## Offline-First Model

Most commands work offline when a local project is initialized (`tkt init`):
- **Offline**: `add`, `list`, `edit`, `state`, `done`, `comment`, `comments`, `tag-add`, `tag-remove`, `tags`, `delete`, `show`
- **Requires remote**: `issues`, `create`, `cmd`, `projects`, `remote auth`, `remote whoami`
- Changes to synced issues are queued and pushed with `tkt remote push` or `tkt sync`.

## File Locations

| File | Purpose |
|------|---------|
| `.tktrc.yml` | Project config (committed) |
| `.tkt.local.yml` | Private config (gitignored) |
| `.tkt/store.db` | SQLite database |
| `~/.config/tkt/credentials.yml` | Fallback credentials file |

## Agent Best Practices

1. **Always use `--output json`** for machine-readable output.
2. **Use `--quiet`** when you only need IDs (e.g., `tkt add -q` prints `#N`).
3. **Check exit codes**: 2 means auth failed — run `tkt remote auth` before retrying.
4. **Handle conflicts**: After `tkt sync`, check if `Conflicts > 0` in JSON output.
5. **State validation**: Use `--force` only when intentionally bypassing workflow rules.
6. **ID formats**: Use `#N` for local IDs, `PROJ-42` for remote IDs. Commands resolve automatically.
7. **Dry run**: Use `tkt sync --dry-run` to preview changes before applying.
8. **Non-interactive auth**: `tkt remote auth --token TOKEN` avoids prompts.