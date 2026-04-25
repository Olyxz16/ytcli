# ytcli

A fast, **offline-first** command-line issue tracker that optionally syncs to JetBrains YouTrack.

Think of it as a local TODO list with superpowers: instant creation, full-text search, and state management — no server needed. When you want to collaborate, `ytcli sync` pushes your tickets to YouTrack and pulls updates back.

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

## Usage

ytcli has two operating modes. You can use either or both at the same time.

---

### Mode 1: Fully Offline (Standalone)

No YouTrack account, no internet, no problem. ytcli works as a local issue tracker backed by SQLite.

```bash
# Initialize a project in the current directory
ytcli init

# Create issues instantly — no network, no latency
ytcli add "Fix login bug" -d "OAuth token exchange fails" -P Critical --tag bug
ytcli add "Add dark mode" --tag ui
ytcli add "Refactor API" --state "In Progress"

# List all issues
ytcli list
ytcli ls                          # same thing, shorter

# Search with full-text search (FTS5)
ytcli list "OAuth"
ytcli list --state "In Progress"  # filter by state

# Show issue detail
ytcli show '#1'

# Edit fields
ytcli edit '#1' -s "Fix OAuth login bug" -d "Updated description"

# Change state
ytcli state '#1' "In Progress"
ytcli done '#3'                   # shortcut to Done

# Add comments
ytcli comment '#1' "Found the root cause"
ytcli comments '#1'               # list all comments

# Tags
ytcli tag-add '#1' urgent
ytcli tag-remove '#1' urgent
ytcli tags                        # list all tags

# Open in $EDITOR
ytcli open '#1'

# Delete
ytcli delete '#1'
```

**What's happening:** Every command above writes to `.ytcli/store.db`, a local SQLite database. Nothing leaves your machine.

---

### Mode 2: YouTrack Sidecar (Sync)

Connect your local project to a YouTrack instance. Work offline, sync when ready.

```bash
# Step 1: Configure your YouTrack instance
ytcli config setup
# (interactive prompt: name, URL, token)

# Step 2: Set the project for this directory
ytcli config set project PROJ

# Step 3: Pull existing YouTrack issues into your local store
ytcli sync --pull

# Step 4: Work locally as usual — everything is instant and offline
ytcli add "New ticket from CLI"
ytcli edit '#1' -s "Updated summary"
ytcli comment '#1' "Left a comment"

# Step 5: Sync when you're ready
ytcli sync                        # pull remote changes + push local changes
ytcli sync --push                 # push only (skip pulling)
ytcli sync --pull                 # pull only (skip pushing)
ytcli sync --dry-run              # preview what would happen

# Check sync status anytime
ytcli sync --status
```

**ID mapping:** After sync, local `#1` also resolves as `PROJ-42`. Both refer to the same issue.

```bash
ytcli show '#1'        # works
ytcli show PROJ-42     # same issue, also works
```

---

### Mode 3: Direct YouTrack API (No Local Store)

For quick one-off operations against YouTrack without touching the local database:

```bash
# List/search remote issues (always hits the API)
ytcli issues "#Unresolved for: me"
ytcli issues "project: PROJ #Unresolved" --limit 20

# Create directly on YouTrack
ytcli create PROJ -s "Bug found" -d "Details here" -P Critical

# Apply YouTrack commands (the fast way)
ytcli cmd PROJ-42 "State: In Progress for: me"
ytcli cmd PROJ-42 "Priority: Critical" --silent

# Add comment directly
ytcli comment PROJ-42 "Working on this"

# Open in browser
ytcli open PROJ-42
```

These commands bypass the local store and hit the YouTrack REST API directly.

---

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

Tokens are stored in your OS keyring. Fallback to `~/.config/ytcli/credentials.yml` with 0600 permissions.

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

Created automatically by `ytcli init`.

---

## Command Reference

### Local (offline-first)

| Command | Description |
|---|---|
| `ytcli init` | Initialize local project with SQLite store |
| `ytcli add "Summary"` | Create a local issue |
| `ytcli list [query]` | List/search local issues (FTS5) |
| `ytcli ls` | Alias for `list` |
| `ytcli show <id>` | Show issue detail |
| `ytcli edit <id>` | Edit an issue |
| `ytcli state <id> <state>` | Change state |
| `ytcli done <id>` | Mark as done |
| `ytcli comment <id> <text>` | Add comment |
| `ytcli comments <id>` | List comments |
| `ytcli open <id>` | Open in editor or browser |
| `ytcli tag-add <id> <tag>` | Add tag |
| `ytcli tag-remove <id> <tag>` | Remove tag |
| `ytcli tags` | List tags |
| `ytcli delete <id>` | Delete issue |

### Sync

| Command | Description |
|---|---|
| `ytcli sync` | Bidirectional sync (pull then push) |
| `ytcli sync --status` | Show sync status |
| `ytcli sync --pull` | Pull remote changes only |
| `ytcli sync --push` | Push local changes only |
| `ytcli sync --dry-run` | Preview without applying |

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

---

## Global Flags

| Flag | Description |
|---|---|
| `-i, --instance` | Target instance name |
| `-o, --output` | Output format: `table`, `json`, `wide`, `markdown` |
| `-q, --quiet` | Minimal output (IDs only, implies JSON) |

---

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

---

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
