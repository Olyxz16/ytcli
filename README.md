# ytcli

A fast, agent-friendly command-line interface for JetBrains YouTrack.

## Features

- **Speed-first design**: Built in Go for instant startup and execution
- **Agent-friendly**: Structured JSON output, meaningful exit codes, stdin support
- **YouTrack Commands API**: Apply complex state changes with natural syntax
- **Shell completions**: Bash, Zsh, Fish, PowerShell
- **Interactive setup**: Guided configuration with Charm's `huh` forms
- **TUI-ready architecture**: Service layer cleanly separated for future TUI

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

### 1. Configure an instance

Interactive setup:

```bash
ytcli config setup
```

Or manual:

```bash
ytcli config set instances.work.url https://company.youtrack.cloud
ytcli config set default_instance work
ytcli auth login work --token <your-permanent-token>
```

### 2. Use it

```bash
# List your unresolved issues
ytcli issues "#Unresolved for: me"

# Show issue detail
ytcli show PROJ-42

# Create an issue
ytcli create PROJ -s "Bug found" -d "Details here" -P Critical

# Apply a command (the fast way)
ytcli cmd PROJ-42 "State: In Progress for: me"

# Add a comment
ytcli comment PROJ-42 "Working on this now"

# Open in browser
ytcli open PROJ-42
```

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
default_query: "#Unresolved assignee: me sort by: updated"
```

**`.ytcli.local.yml`** (gitignored):

```yaml
current_task: PROJ-42
```

Initialize both with:

```bash
ytcli config init
```

## Commands

| Command | Description | Alias |
|---|---|---|
| `ytcli issues [query]` | List/search issues | `i` |
| `ytcli show <id>` | Show issue details | `s` |
| `ytcli create <project>` | Create an issue | `n` |
| `ytcli edit <id>` | Edit an issue | |
| `ytcli cmd <id> <command>` | Apply YouTrack command | `c` |
| `ytcli comment <id> <text>` | Add comment | |
| `ytcli comments <id>` | List comments | |
| `ytcli projects` | List projects | |
| `ytcli log <id> <duration>` | Log work time | |
| `ytcli link <id> <target>` | Link issues | |
| `ytcli tags` | List tags | |
| `ytcli tag-add <id> <tag>` | Add tag to issue | |
| `ytcli tag-remove <id> <tag-id>` | Remove tag from issue | |
| `ytcli open <id>` | Open issue in browser | `o` |
| `ytcli auth login [instance]` | Authenticate | |
| `ytcli auth whoami` | Show current user | |
| `ytcli config setup` | Interactive setup | |
| `ytcli config init` | Init local configs | |
| `ytcli completion <shell>` | Shell completions | |

## Global Flags

| Flag | Description |
|---|---|
| `-i, --instance` | Target instance name |
| `-o, --output` | Output format: `table`, `json`, `wide`, `markdown` |
| `-q, --quiet` | Minimal output (IDs only, implies JSON) |

## Agent Usage

```bash
# Get structured data
ytcli issues "#Unresolved for: me" --output json --quiet

# Create and get back the issue ID
ytcli create PROJ -s "New bug" -o json --quiet

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
CLI (cobra)  →  Service Layer  →  API Client  →  YouTrack REST API
                    ↑                ↑
TUI (future)  →  Renderer    →  Config + Keyring
```

The Service layer is the boundary. Both CLI and future TUI call into the same services. The Renderer formats output for the appropriate mode.

## License

MIT
