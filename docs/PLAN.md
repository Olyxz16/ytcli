# ytcli — YouTrack CLI Architecture Plan

## 1. High-Level Architecture

```
┌─────────────────────────────────────────┐
│  Presentation Layer (swappable)          │
│  ┌───────────┐    ┌───────────────────┐  │
│  │  CLI      │    │  TUI (future)     │  │
│  │  (cobra)  │    │  (bubbletea)      │  │
│  └─────┬─────┘    └────────┬──────────┘  │
│        └────────┬──────────┘              │
│         ┌──────▼──────┐                   │
│         │  Renderer    │  lipgloss,       │
│         │  (shared)    │  glamour, table  │
│         └──────┬──────┘                   │
│         ┌──────▼──────┐                   │
│         │  Service     │  business logic,  │
│         │  Layer       │  validation       │
│         └──────┬──────┘                   │
│         ┌──────▼──────┐                   │
│         │  API Client  │  YouTrack REST   │
│         │  Layer       │  with fields mgmt │
│         └──────┬──────┘                   │
│         ┌──────▼──────┐                   │
│         │  Config      │  global + local   │
│         │  Layer       │  + keyring         │
│         └─────────────┘                   │
└─────────────────────────────────────────┘
```

Key principle: **the Service layer is the boundary**. The CLI and future TUI both call into the same Service functions. The Renderer takes service results and formats them for the appropriate output mode (table, JSON, markdown). This makes the TUI evolution natural — just swap the presentation layer.

---

## 2. Project Structure

```
ytcli/
├── cmd/ytcli/main.go              # Entry point
├── internal/
│   ├── api/                        # YouTrack REST API client
│   │   ├── client.go               # HTTP client, auth, retries, base URL
│   │   ├── fields.go               # Fields syntax builder (critical helper)
│   │   ├── issues.go               # Issue CRUD + search
│   │   ├── comments.go             # Comment operations
│   │   ├── projects.go             # Project operations
│   │   ├── users.go                # User operations (incl /users/me)
│   │   ├── commands.go             # POST /api/commands (the speed superpower)
│   │   ├── tags.go                 # Tag operations
│   │   ├── attachments.go          # Attachment upload/download
│   │   ├── links.go                # Issue link operations
│   │   ├── timetracking.go         # Work items
│   │   ├── agiles.go               # Agile boards & sprints
│   │   ├── articles.go             # Knowledge base
│   │   └── search.go               # Search/Command suggestions
│   ├── config/
│   │   ├── config.go               # Global config (~/.config/ytcli/)
│   │   ├── local.go                # Local project configs (.ytcli.yml + .ytcli.local.yml)
│   │   └── keyring.go             # OS keyring for tokens
│   ├── model/                      # Domain models (shared across layers)
│   │   ├── issue.go                # Issue, IssueCustomField unified model
│   │   ├── project.go              # Project model
│   │   ├── user.go                 # User model
│   │   ├── comment.go              # Comment model
│   │   ├── command.go              # Command result model
│   │   └── customfield.go          # Polymorphic custom field handling
│   ├── service/                    # Business logic layer
│   │   ├── issues.go               # Issue operations (list, get, create, update, delete)
│   │   ├── comments.go             # Comment operations
│   │   ├── commands.go             # Command execution (wraps API + validation)
│   │   ├── projects.go             # Project operations
│   │   ├── timetracking.go         # Time tracking operations
│   │   └── search.go               # Search & suggestions
│   ├── render/                     # Output formatting (shared CLI/TUI)
│   │   ├── issue.go                # Issue detail rendering
│   │   ├── list.go                 # Issue list/table rendering
│   │   ├── comment.go              # Comment rendering
│   │   ├── markdown.go             # Glamour markdown renderer
│   │   ├── json.go                 # JSON output (agent-friendly)
│   │   └── styles.go              # Lipgloss style definitions
│   ├── cmdx/                       # CLI command definitions (cobra)
│   │   ├── root.go                 # Root command + global flags
│   │   ├── issues.go               # `ytcli issues [query]`
│   │   ├── show.go                 # `ytcli show <ISSUE-ID>`
│   │   ├── create.go               # `ytcli create <project> ...`
│   │   ├── edit.go                 # `ytcli edit <ISSUE-ID> ...`
│   │   ├── cmd.go                  # `ytcli cmd <ID> <command>` — YouTrack commands
│   │   ├── comment.go              # `ytcli comment <ID> <text>`
│   │   ├── comments.go             # `ytcli comments <ID>`
│   │   ├── link.go                 # `ytcli link <ID> <target> <type>`
│   │   ├── log.go                  # `ytcli log <ID> <duration>` — time tracking
│   │   ├── projects.go             # `ytcli projects`
│   │   ├── auth.go                 # `ytcli auth login|whoami`
│   │   └── config.go               # `ytcli config init|set|get|instances`
│   └── tui/                        # Future TUI (skeleton only initially)
│       └── app.go
├── .ytcli.yml                      # Example commitable config
├── .ytcli.local.yml                # Example local config (gitignored)
├── go.mod
└── Makefile
```

---

## 3. CLI Command Design

```
ytcli [global flags] <command> [args]

Global flags:
  -i, --instance    Instance name (default: from local config or default_instance)
  -o, --output      Output format: table|json|wide|markdown (default: table)
  -q, --quiet       Minimal output (just IDs, exit codes)
  --fields          Custom field selection override

─── Issue Commands ─────────────────────────────────────────
ytcli issues [query]              List issues (full YouTrack query syntax)
  -p, --project     Filter by project
  -a, --assignee    Filter by assignee ("me" shortcut)
  -s, --state       Filter by state
  --limit           Max results (default: 50, 0 = all)
  --sort            Sort expression (default: "updated desc")

ytcli show <ISSUE-ID>             Show issue detail
  --comments        Include comments in output
  -w, --web         Open in browser

ytcli create <PROJECT>            Create a new issue
  -s, --summary     Summary (required)
  -d, --description  Description (or stdin: echo "desc" | ytcli create PROJ -s "Sum")
  -t, --type         Issue type
  -P, --priority     Priority
  -a, --assignee     Assignee
  --tag              Tags (repeatable)

ytcli edit <ISSUE-ID>             Edit issue fields
  -s, --summary     New summary
  -d, --description  New description

ytcli cmd <ISSUE-ID> <command>    Apply YouTrack command (THE SPEED TOOL)
  -c, --comment     Add comment alongside command
  --silent          Apply without notifications
  # Examples:
  #   ytcli cmd PROJ-42 "State: In Progress for: me"
  #   ytcli cmd PROJ-42 "Priority: Critical"
  #   ytcli cmd PROJ-42 "Fixed" -c "Fixed in v2.3.1"

─── Comment Commands ──────────────────────────────────────
ytcli comment <ISSUE-ID> <text>   Add comment
ytcli comments <ISSUE-ID>         List comments

─── Link Commands ─────────────────────────────────────────
ytcli link <ISSUE-ID> <TARGET-ID> <TYPE>   Link issues

─── Time Tracking ─────────────────────────────────────────
ytcli log <ISSUE-ID> <duration>   Log work time (e.g., "2h30m", "1d")
  -t, --type         Work item type
  -d, --date         Date (default: today)

─── Project Commands ──────────────────────────────────────
ytcli projects                    List projects
ytcli project <ID>                Show project detail

─── Config/Auth Commands ─────────────────────────────────
ytcli auth login [instance]       Set auth token (interactive or --token)
ytcli auth whoami                 Show current user & instance
ytcli config init                 Initialize .ytcli.yml + .ytcli.local.yml in cwd
ytcli config set <key> <value>    Set config value
ytcli config get <key>            Get config value
ytcli config instances            List configured instances

─── Shortcuts ────────────────────────────────────────────
ytcli i [query]    = ytcli issues
ytcli s <ID>       = ytcli show
ytcli n <PROJ>     = ytcli create (new)
ytcli c <ID> <cmd> = ytcli cmd
```

---

## 4. Config File Design

### Global: `~/.config/ytcli/config.yml`

```yaml
instances:
  work:
    url: https://work.youtrack.cloud
  personal:
    url: https://personal.myjetbrains.com/youtrack
default_instance: work
output_format: table
```

Tokens stored in **OS keyring** (not in file). Fallback to `~/.config/ytcli/credentials.yml` (chmod 600) if keyring unavailable.

### Local (commitable): `.ytcli.yml`

```yaml
instance: work
project: PROJ
default_query: "#Unresolved assignee: me sort by: updated"
```

### Local (private, gitignored): `.ytcli.local.yml`

```yaml
current_task: PROJ-42
```

`ytcli config init` generates both files + adds `.ytcli.local.yml` to `.gitignore`.

Config resolution order (later overrides earlier):
1. Global config `~/.config/ytcli/config.yml`
2. Local config `.ytcli.yml` (commitable, in CWD or parent dirs)
3. Local config `.ytcli.local.yml` (gitignored, in CWD or parent dirs)
4. Command-line flags

---

## 5. Agent-Friendly Design

Every command supports:
- `--output json` — structured JSON output for piping/agents
- `--quiet` — only essential output (issue IDs), clean exit codes
- **stdin support** — `echo "description" | ytcli create PROJ -s "Summary"`
- **Exit codes**: 0=success, 1=general error, 2=auth error, 3=not found, 4=validation
- **Structured errors** in JSON mode: `{"error": "not_found", "message": "Issue PROJ-99 not found", "code": 3}`

This means an AI agent can:
```bash
ytcli issues "#Unresolved for: me" --output json --quiet
ytcli create PROJ -s "Bug found" -d "Details here" -o json
ytcli cmd PROJ-42 "State: In Progress" -q
```

---

## 6. Key Challenges & Solutions

| Challenge | Solution |
|---|---|
| **YouTrack `fields` syntax** | Build a `fields.Builder` that composes field strings per operation. Smart defaults per command (list vs detail). |
| **Custom field polymorphism** (`$type`) | Unified `CustomFieldValue` Go type with custom `UnmarshalJSON` that dispatches on `$type`. The model layer normalizes all variants. |
| **Wiki markup in descriptions** | Use `wikifiedDescription` (returns HTML), pipe through a HTML→markdown converter, then render with glamour. Or render HTML directly in terminal with a simple tag stripper for quick views. |
| **Pagination** | Service layer handles auto-pagination. `--limit` flag for user control. Default: 50 results. Cursor-based for activities. |
| **Token storage security** | Primary: OS keyring via `zalando/go-keyring`. Fallback: file with strict permissions. Never in commitable config. |
| **YouTrack version variance** | API client detects version via `/api/admin/globalSettings?fields=version` on first connect. Feature-flag endpoints. |
| **Rate limiting** | Exponential backoff with jitter. Respect `Retry-After` headers. Concurrent request limiting (semaphore). |
| **Attachment binary handling** | Streaming upload/download. Don't buffer entire files in memory. |
| **Search UX** | Leverage `/api/commands/assist` and `/api/search/assist` for autocompletion. Shell completions via cobra. |
| **TUI evolution** | Service layer returns domain models, not formatted strings. Renderer is the only layer that touches lipgloss/glamour. TUI swaps in bubbletea views calling same services. |

---

## 7. Charm Library Usage

| Library | Purpose |
|---|---|
| **cobra** | CLI framework (not Charm, but standard Go) |
| **lipgloss** | All terminal styling: tables, colored labels, borders, status indicators |
| **glamour** | Render markdown descriptions & comments in terminal |
| **huh** | Interactive prompts: `ytcli create` wizard, confirmations, auth login |
| **table** (charmbracelet/bubble-table) | Issue list table rendering with columns |
| **bubbletea** | TUI framework (future) |
| **bubbles** | TUI components: list, viewport, text input (future) |

---

## 8. Implementation Phases

### Phase 1: Foundation (MVP)
- Go module setup + cobra scaffold
- Config layer (global + local)
- Auth (keyring + token)
- API client with fields builder
- `ytcli issues` + `ytcli show` + `ytcli create`
- Renderer with table + JSON output
- Custom field unmarshaling

### Phase 2: Speed Features
- `ytcli cmd` — YouTrack command execution
- `ytcli comment` + `ytcli comments`
- `ytcli edit`
- Shortcuts (i, s, n, c)
- Shell completions
- Search suggestions integration

### Phase 3: Full Feature Set
- `ytcli link`, `ytcli log` (time tracking)
- `ytcli projects` with detail
- Attachment upload/download
- `ytcli config init` with interactive setup (huh)
- Auto-pagination
- Markdown rendering for descriptions

### Phase 4: TUI
- bubbletea app with issue list view
- Issue detail viewport
- Command input with suggestions
- Sprint/agile board view

---

## 9. The "Commands API" as the Secret Weapon

The YouTrack Commands API (`POST /api/commands`) is what makes this tool genuinely fast. Instead of building complex edit flows for every field, a single command like:

```bash
ytcli cmd PROJ-42 "for: me Priority: Critical State: In Progress"
```

Does what would otherwise require multiple API calls and complex field-type handling. The commands API also supports `--comment` for adding a comment simultaneously. This is the primary "speed" mechanism — the user types what they want in YouTrack's natural syntax, and the API handles the rest.

For agents, this is even more powerful: a single string can express complex state transitions.

---

## 10. YouTrack REST API Reference

### Base URL
`{instance_url}/api/`

### Authentication
- Permanent token via `Authorization: Bearer <token>` header
- Tokens managed at: Settings → Personal → Authentication Tokens

### Key Endpoints

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/issues` | GET | List/search issues (query param for YouTrack search syntax) |
| `/api/issues` | POST | Create issue (requires: summary, project.id) |
| `/api/issues/{id}` | GET | Get issue detail |
| `/api/issues/{id}` | POST | Update issue fields |
| `/api/issues/{id}` | DELETE | Delete issue |
| `/api/issues/{id}/comments` | GET | List comments |
| `/api/issues/{id}/comments` | POST | Add comment |
| `/api/issues/{id}/links` | GET/POST | Manage issue links |
| `/api/issues/{id}/tags` | GET/POST | Manage tags |
| `/api/issues/{id}/timeTracking/workItems` | GET/POST | Time tracking |
| `/api/issues/{id}/attachments` | GET/POST | File attachments |
| `/api/commands` | POST | Apply YouTrack commands to issues |
| `/api/commands/assist` | POST | Command suggestions/autocomplete |
| `/api/search/assist` | GET | Search suggestions |
| `/api/admin/projects` | GET | List projects |
| `/api/admin/projects/{id}` | GET | Project detail |
| `/api/users/me` | GET | Current user info |
| `/api/users` | GET | List users |
| `/api/agiles` | GET | List agile boards |
| `/api/issuesGetter/count` | GET | Issue count for query |
| `/api/tags` | GET | List tags |
| `/api/articles` | GET | Knowledge base articles |

### Fields Syntax
YouTrack requires explicit field selection via `?fields=` parameter. Only `id` and `$type` returned by default.

Examples:
- `fields=id,idReadable,summary` — top-level fields
- `fields=id,summary,project(name)` — nested object fields
- `fields=id,customFields(id,name,value(name,login))` — deep nesting

### Pagination
- General: `$top` and `$skip` query parameters (default max 42 items)
- Activities: cursor-based via `cursor`, `cursorBefore`, `cursorAfter`

### Query Syntax
The `query` parameter uses YouTrack search syntax:
- `for: me #Unresolved` — assigned to me, unresolved
- `project: {Sample Project}` — filter by project
- `#Unresolved sort by: updated desc` — sort results
- Full reference: https://www.jetbrains.com/help/youtrack/cloud/search-and-command-attributes.html

### Custom Fields
Custom fields are polymorphic — the `$type` discriminator determines the structure:
- `SingleEnumIssueCustomField` — single enum value
- `MultiEnumIssueCustomField` — multiple enum values
- `StateIssueCustomField` / `StateMachineIssueCustomField` — state fields
- `SingleUserIssueCustomField` — single user
- `MultiUserIssueCustomField` — multiple users
- `SingleOwnedIssueCustomField` — single owned field
- `SingleVersionIssueCustomField` / `MultiVersionIssueCustomField` — version fields
- `DateIssueCustomField` — date field
- `TextIssueCustomField` — text field
- `PeriodIssueCustomField` — period/duration field
