# tkt - Tiket

A fast, **offline-first** command-line issue tracker that optionally syncs to JetBrains YouTrack.

Think of it as a local TODO list with superpowers: instant creation, full-text search, and state management — no server needed. When you want to collaborate, `tkt sync` pushes your tickets to YouTrack and pulls updates back.

## Installation

```bash
go install github.com/Olyxz16/tkt/cmd/tkt@latest
```

Or clone and build:

```bash
git clone https://github.com/Olyxz16/tkt.git
cd tkt
make build
```

## Usage

tkt has two operating modes. You can use either or both at the same time.

---

### Mode 1: Fully Offline (Standalone)

No YouTrack account, no internet, no problem. tkt works as a local issue tracker backed by SQLite.

```bash
# Initialize a project in the current directory
tkt init

# Create issues instantly — no network, no latency
tkt add "Fix login bug" -d "OAuth token exchange fails" -P Critical --tag bug
tkt add "Add dark mode" --tag ui
tkt add "Refactor API" --state "In Progress"

# List all issues
tkt list
tkt ls                          # same thing, shorter

# Search with full-text search (FTS5)
tkt list "OAuth"
tkt list --state "In Progress"  # filter by state

# Show issue detail
tkt show '#1'

# Edit fields
tkt edit '#1' -s "Fix OAuth login bug" -d "Updated description"

# Change state
tkt state '#1' "In Progress"
tkt done '#3'                   # shortcut to Done

# Add comments
tkt comment '#1' "Found the root cause"
tkt comments '#1'               # list all comments

# Tags
tkt tag-add '#1' urgent
tkt tag-remove '#1' urgent
tkt tags                        # list all tags

# Open in $EDITOR
tkt open '#1'

# Delete
tkt delete '#1'
```

**What's happening:** Every command above writes to `.tkt/store.db`, a local SQLite database. Nothing leaves your machine.

---

### Mode 2: YouTrack Sidecar (Sync)

Connect your local project to a YouTrack instance. Work offline, sync when ready.

```bash
# Step 1: Configure your YouTrack instance
tkt config setup
# (interactive prompt: name, URL, token)

# Step 2: Set the project for this directory
tkt config set project PROJ

# Step 3: Pull existing YouTrack issues into your local store
tkt sync --pull

# Step 4: Work locally as usual — everything is instant and offline
tkt add "New ticket from CLI"
tkt edit '#1' -s "Updated summary"
tkt comment '#1' "Left a comment"

# Step 5: Sync when you're ready
tkt sync                        # pull remote changes + push local changes
tkt sync --push                 # push only (skip pulling)
tkt sync --pull                 # pull only (skip pushing)
tkt sync --dry-run              # preview what would happen

# Check sync status anytime
tkt sync --status
```

**ID mapping:** After sync, local `#1` also resolves as `PROJ-42`. Both refer to the same issue.

```bash
tkt show '#1'        # works
tkt show PROJ-42     # same issue, also works
```

---

### Mode 3: Direct YouTrack API (No Local Store)

For quick one-off operations against YouTrack without touching the local database:

```bash
# List/search remote issues (always hits the API)
tkt issues "#Unresolved for: me"
tkt issues "project: PROJ #Unresolved" --limit 20

# Create directly on YouTrack
tkt create PROJ -s "Bug found" -d "Details here" -P Critical

# Apply YouTrack commands (the fast way)
tkt cmd PROJ-42 "State: In Progress for: me"
tkt cmd PROJ-42 "Priority: Critical" --silent

# Add comment directly
tkt comment PROJ-42 "Working on this"

# Open in browser
tkt open PROJ-42
```

These commands bypass the local store and hit the YouTrack REST API directly.

---

## Configuration

### Global config (`~/.config/tkt/config.yml`)

```yaml
instances:
  work:
    url: https://work.youtrack.cloud
  personal:
    url: https://personal.myjetbrains.com/youtrack
default_instance: work
output_format: table
```

Tokens are stored in your OS keyring. Fallback to `~/.config/tkt/credentials.yml` with 0600 permissions.

### Local config (per directory)

**`.tkt.yml`** (safe to commit):

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

**`.tkt.local.yml`** (gitignored):

```yaml
current_task: PROJ-42
```

Created automatically by `tkt init`.

---

## Command Reference

### Local (offline-first)

| Command | Description |
|---|---|
| `tkt init` | Initialize local project with SQLite store |
| `tkt add "Summary"` | Create a local issue |
| `tkt list [query]` | List/search local issues (FTS5) |
| `tkt ls` | Alias for `list` |
| `tkt show <id>` | Show issue detail |
| `tkt edit <id>` | Edit an issue |
| `tkt state <id> <state>` | Change state |
| `tkt done <id>` | Mark as done |
| `tkt comment <id> <text>` | Add comment |
| `tkt comments <id>` | List comments |
| `tkt open <id>` | Open in editor or browser |
| `tkt tag-add <id> <tag>` | Add tag |
| `tkt tag-remove <id> <tag>` | Remove tag |
| `tkt tags` | List tags |
| `tkt delete <id>` | Delete issue |

### Sync

| Command | Description |
|---|---|
| `tkt sync` | Bidirectional sync (pull then push) |
| `tkt sync --status` | Show sync status |
| `tkt sync --pull` | Pull remote changes only |
| `tkt sync --push` | Push local changes only |
| `tkt sync --dry-run` | Preview without applying |

### Remote (direct API)

| Command | Description | Alias |
|---|---|---|
| `tkt issues [query]` | List/search YouTrack issues | `i`, `search` |
| `tkt create <project>` | Create directly on YouTrack | `n` |
| `tkt cmd <id> <command>` | Apply YouTrack command | `c` |
| `tkt projects` | List projects | |
| `tkt project <id>` | Show project details | |
| `tkt log <id> <duration>` | Log work time | |
| `tkt link <id> <target>` | Link issues | |

### Config & Auth

| Command | Description |
|---|---|
| `tkt auth login [instance]` | Authenticate |
| `tkt auth whoami` | Show current user |
| `tkt config setup` | Interactive setup |
| `tkt config init` | Init local configs |
| `tkt config set <key> <value>` | Set config value |
| `tkt config get <key>` | Get config value |
| `tkt config instances` | List instances |
| `tkt completion <shell>` | Shell completions |
| `tkt version` | Version info |

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
tkt issues "#Unresolved for: me" --output json --quiet

# Create locally and get back the ID
tkt add "New bug" -o json --quiet

# Apply a command silently
tkt cmd PROJ-42 "Fixed" -q
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
│                    tkt binary                          │
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
