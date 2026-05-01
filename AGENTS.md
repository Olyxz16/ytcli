# AGENTS.md

This file documents the agent-compatible interface for `ytcli`. AI agents and automation tools should use `--output json` (or `-o json`) and `--quiet` (or `-q`) flags for machine-readable output.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication error (HTTP 401) |
| 3 | Not found (HTTP 404) or resource not available |
| 4 | Validation error (HTTP 400) |

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--instance` | `-i` | Target YouTrack instance name (overrides project config) |
| `--output` | `-o` | Output format: `table`, `json`, `wide`, `markdown` |
| `--quiet` | `-q` | Minimal output (implies `--output json` unless overridden) |

## Output Modes

- **table** (default): Human-readable terminal output with colors and formatting
- **json**: Pretty-printed JSON to stdout
- **quiet** (`-q`): Minimal output — typically just IDs, one per line; implies `--output json` unless `-o` is explicitly set

When `--output json` is used, errors are also emitted as JSON: `{"error": "...", "message": "..."}`.

## Commands Reference

### Issues

```bash
ytcli issues [query] [flags]
```

Lists issues. Uses `--output json` for structured data.

| Flag | Description |
|------|-------------|
| `--assignee` | Filter by assignee (`me` for current user) |
| `--project` | Filter by project short name |
| `--state` | Filter by state |
| `--limit` | Max results (default 50, 0 = all) |
| `--sort` | Sort expression |

**Quiet mode**: prints one `IDReadable` per line.

### Show Issue

```bash
ytcli show <issue-id>
```

Shows issue detail. Supports local IDs (`#1`) or remote IDs (`PROJ-42`).

**JSON mode**: full issue object.

### Create Issue

```bash
ytcli create -s "Summary" [-d "Description"] [--tag tag1 [--tag tag2]]
```

Creates a remote issue. Requires `--project` flag or project in local config.

**Quiet mode**: prints the created issue ID.
**JSON mode**: full issue object.

### Add Local Issue

```bash
ytcli add -s "Summary" [-d "Description"] [--tag tag1]
```

Creates a local-only issue.

**Quiet mode**: prints the local ID (`#N`).
**JSON mode**: full local issue object.

### Edit Issue

```bash
ytcli edit <issue-id> [-s "Summary"] [-d "Description"] [--state State] [--priority Priority] [-a Assignee]
```

Edits an issue (local first, then remote fallback).

**Quiet mode**: prints the issue ID.

### State / Done

```bash
ytcli state <issue-id> <state>
ytcli done <issue-id>
```

Changes issue state. `done` uses the configured done state.

**Quiet mode**: prints the issue ID.

### Delete

```bash
ytcli delete <issue-id>
```

Deletes an issue (local first, then remote).

**Quiet mode**: prints the deleted ID.
**JSON mode**: `{"id": "...", "deleted": true, "local": true/false}`.

### Tags

```bash
ytcli tags                    # List all tags
ytcli tag-add <issue-id> <tag>    # Add a tag
ytcli tag-remove <issue-id> <tag> # Remove a tag
```

**Quiet mode** (`tags`): prints one tag name per line.
**Quiet mode** (`tag-add`/`tag-remove`): prints the issue ID.
**JSON mode** (`tag-add`): `{"issue": "...", "tag": "...", "added": true}`.
**JSON mode** (`tag-remove`): `{"issue": "...", "tag": "...", "removed": true}`.

### Comments

```bash
ytcli comments <issue-id>     # List comments
ytcli comment <issue-id> <text>  # Add a comment
```

**JSON mode**: full comment objects.

### Link

```bash
ytcli link <issue-id> <target-id> --type <link-type-id>
```

**Quiet mode**: prints the source issue ID.
**JSON mode**: `{"source": "...", "target": "...", "type": "...", "linked": true}`.

### Log Work

```bash
ytcli log <issue-id> <duration> [--type type] [--date YYYY-MM-DD]
```

Duration format: `1h30m`, `2d`, `45m` (d=8h workday).

**Quiet mode**: prints the work item ID.
**JSON mode**: full work item object.

### Sync

```bash
ytcli sync                    # Pull then push
ytcli sync --pull             # Pull only
ytcli sync --push             # Push only
ytcli sync --status           # Show sync status
```

**JSON mode**: `{"Pulled": N, "Pushed": N, "Conflicts": N, "Failed": N, "Errors": [...]}`.
**JSON mode** (`--status`): `{"statuses": {...}, "queue": {"pending": N, "failed": N, "completed": N}}`.

### Version

```bash
ytcli version
```

**JSON mode**: `{"version": "...", "commit": "..."}`.
**Quiet mode**: prints just the version string.

### Config

```bash
ytcli config set <key> <value>  # Set config value
ytcli config get <key>           # Get resolved config value
ytcli config instances           # List configured instances
ytcli config setup               # Interactive setup (not agent-friendly)
```

### Auth

```bash
ytcli auth login [instance] [--token TOKEN]  # Authenticate
ytcli auth whoami                              # Show current user
```

`auth login` supports `--token` for non-interactive auth. When no instance is specified, it uses the project's configured instance.

### Init

```bash
ytcli init
```

Initializes a local project. Interactive — prompts for instance selection and project name if global instances are available.

## ID Resolution

Commands accept multiple ID formats:
- **Local ID**: `#1`, `1` — resolves to local SQLite issue
- **Remote ID**: `PROJ-42` — resolves via YouTrack API
- Resolution order: local database first, then remote API

## Configuration Layers

1. **Global**: `~/.config/ytcli/config.yml` — instances, default instance, output format
2. **Local**: `.ytcli.yml` — instance binding, project, local schema (committed to VCS)
3. **Private**: `.ytcli.local.yml` — current task (gitignored)

Key config values:
- `instance` — binds project to a YouTrack instance
- `project` — project short name for queries and issue creation
- `default_query` — default filter for `ytcli issues`
- `local.states` — valid state values
- `local.priorities` — valid priority values
- `local.done_states` — states that mark an issue as done

## Offline / Local-First

Most commands work offline-first when a local project is initialized (`ytcli init`):
- `add`, `list`, `edit`, `state`, `done`, `comment`, `comments`, `tag-add`, `tag-remove`, `tags`, `delete`, `show`
- Changes to synced issues are queued and pushed with `ytcli sync --push`
- Remote-only commands: `issues`, `create`, `cmd`, `projects`, `auth whoami`