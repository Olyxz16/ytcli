# ytcli

A fast, agent-friendly, **offline-first** command-line interface for JetBrains YouTrack.

## Features

- **Offline-first ticketing**: All daily operations hit a local SQLite database. Zero latency. Works on a plane.
- **Sync on demand**: `ytcli sync` bridges your local tickets to YouTrack when you're ready to collaborate.
- **Standalone mode**: No YouTrack account needed. Use ytcli as a local TODO tracker for small projects.
- **Speed-first design**: Built in Go for instant startup and execution.
- **Agent-friendly**: Structured JSON output, meaningful exit codes, stdin support.
- **YouTrack Commands API**: Apply complex state changes with natural syntax.
- **Shell completions**: Bash, Zsh, Fish, PowerShell.
- **Interactive setup**: Guided configuration with Charm's `huh` forms.

## Installation

```bash
go install github.com/Olyxz16/ytcli/cmd/ytcli@latest
```

Or clone and build:

```bash
git clone https://github.com/Olyxz16/ytcli.git
cd ytcli
make build
```

## Quick Start

### Local-only mode (no YouTrack needed)

```bash
# Initialize a local project
ytcli init

# Add issues instantly — no network required
ytcli add "Fix login bug" -d "OAuth token exchange fails" -P Critical
ytcli add "Add dark mode" --tag ui
ytcli add "Refactor API" --state "In Progress"

# List all local issues
ytcli list
ytcli ls

# Search locally with full-text search
ytcli list "OAuth"

# Show issue detail
ytcli show '#1'

# Edit, change state, mark done
ytcli edit '#1' -s "Fix OAuth login bug"
ytcli state '#1' "In Progress"
ytcli done '#3'

# Add comments
ytcli comment '#1' "Investigating the flow"

# Open in $EDITOR (unsynced) or browser (synced)
ytcli open '#1'

# Check sync status
ytcli sync --status
```

### Connect to YouTrack for collaboration

```bash
# Configure a YouTrack instance
ytcli config setup

# Pull remote issues into local store
ytcli sync --pull

# Push local changes to YouTrack
ytcli sync --push

# Full bidirectional sync
ytcli sync
```

### Direct YouTrack API commands (online)

```bash
# List remote issues
ytcli issues "#Unresolved for: me"

# Apply a command directly
ytcli cmd PROJ-42 "State: In Progress for: me"

# Create directly on YouTrack
ytcli create PROJ -s "Bug found" -d "Details here" -P Critical
```

## How It Works

ytcli uses a **local-first** architecture:

1. **Local SQLite database** (`.ytcli/store.db`) is the primary data store
2. All `add`, `list`, `show`, `edit`, `state`, `comment`, `tag` commands operate instantly on the local DB
3. `ytcli sync` performs bidirectional sync with YouTrack:
   - **Pull**: Fetch remote issues, update local copies
   - **Push**: Send local creates/edits/comments to YouTrack
4. Issues get hybrid IDs: `#1` locally, `PROJ-42` after sync. Both resolve to the same issue.

## Configuration

### Global config (`~/.config/ytcli/config.yml`)

```yaml
instances:
  work:
    url: https://work.youtrack.cloud
  personal:
    url: https://personal.myjetbrains.com/youtrack
default_instance: work
output_format: table
```

Tokens are stored in your OS keyring (fallback to `~/.config/ytcli/credentials.yml` with 0600 permissions).

### Local config (per directory)

**`.ytcli.yml`** (safe to commit):

```yaml
instance: work
project: PROJ
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
```

**`.ytcli.local.yml`** (gitignored):

```yaml
current_task: PROJ-42
```

Initialize both with:

```bash
ytcli init
```

## Commands

### Local (offline-first)

| Command | Description |
|---|---|
| `ytcli init` | Initialize local project with SQLite store |
| `ytcli add "Summary"` | Create a local issue |
| `ytcli list [query]` | List local issues (FTS5 search) |
| `ytcli ls` | Alias for list |
| `ytcli show <#id\|remote-id>` | Show issue detail (local first, remote fallback) |
| `ytcli edit <id>` | Edit an issue |
| `ytcli state <id> <state>` | Change issue state |
| `ytcli done <id>` | Mark as done |
| `ytcli comment <id> <text>` | Add comment |
| `ytcli comments <id>` | List comments |
| `ytcli open <id>` | Open in editor or browser |
| `ytcli tag-add <id> <tag>` | Add tag |
| `ytcli tag-remove <id> <tag>` | Remove tag |
| `ytcli tags` | List tags |
| `ytcli delete <id>` | Delete issue |
| `ytcli sync` | Bidirectional sync with YouTrack |
| `ytcli sync --status` | Show sync status |
| `ytcli sync --pull` | Pull remote changes only |
| `ytcli sync --push` | Push local changes only |

### Remote (direct API)

| Command | Description | Alias |
|---|---|---|
| `ytcli issues [query]` | List/search YouTrack issues | `i`, `search` |
| `ytcli create <project>` | Create directly on YouTrack | `n` |
| `ytcli cmd <id> <command>` | Apply YouTrack command | `c` |
| `ytcli projects` | List projects | |
| `ytcli project <id>` | Show project details | |
| `ytcli log <id> <duration>` | Log work time | |
| `ytcli link <id> <target>` | Link issues | |

### Config & Auth

| Command | Description |
|---|---|
| `ytcli auth login [instance]` | Authenticate |
| `ytcli auth whoami` | Show current user |
| `ytcli config setup` | Interactive setup |
| `ytcli config init` | Init local configs |
| `ytcli config set <key> <value>` | Set config value |
| `ytcli config get <key>` | Get config value |
| `ytcli config instances` | List instances |
| `ytcli completion <shell>` | Shell completions |
| `ytcli version` | Version info |

## Global Flags

| Flag | Description |
|---|---|
| `-i, --instance` | Target instance name |
| `-o, --output` | Output format: `table`, `json`, `wide`, `markdown` |
| `-q, --quiet` | Minimal output (IDs only, implies JSON) |

## Agent Usage

```bash
# Get structured data from remote
ytcli issues "#Unresolved for: me" --output json --quiet

# Create locally and get back the ID
ytcli add "New bug" -o json --quiet

# Apply a command silently
ytcli cmd PROJ-42 "Fixed" -q
```

Exit codes:
- `0` — success
- `1` — general error
- `2` — authentication error
- `3` — not found
- `4` — validation error

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    ytcli binary                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────────┐    │
│  │  CLI/TUI    │  │  Service   │  │  Sync Manager  │    │
│  │  (cobra)    │  │  Layer     │  │                │    │
│  └──────┬─────┘  └──────┬─────┘  └───────┬────────┘    │
│         │               │                │              │
│  ┌──────▼───────────────▼────────────────▼──────────┐   │
│  │              Local Store (SQLite)                  │   │
│  │  ┌────────┐ ┌─────────┐ ┌───────┐ ┌──────────┐  │   │
│  │  │ issues │ │ comments│ │ queue │ │ projects │  │   │
│  │  │ + FTS5 │ │         │ │ (ops) │ │ + schema │  │   │
│  │  └────────┘ └─────────┘ └───────┘ └──────────┘  │   │
│  └──────────────────────────────────────────────────┘   │
│         │                               │               │
│  ┌──────▼──────┐              ┌─────────▼─────────┐    │
│  │  Local-only │              │  YouTrack API      │    │
│  │  (offline)  │              │  (online, sync)    │    │
│  └─────────────┘              └───────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

The local store is the primary data path. Remote commands bypass it and hit the API directly.

## License

MIT
