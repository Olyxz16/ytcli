# Package Reference

## `internal/api`

HTTP client for the YouTrack REST API. All methods live on `*Client` and return domain models from `internal/model`.

Key types:
- `Client` — authenticated HTTP client with `do` and `doJSON` helpers
- `Fields` — builder for the `fields` query parameter
- `APIError` — structured error with status code helpers (`IsAuthError`, `IsNotFoundError`, etc.)

## `internal/config`

Configuration loading and resolution.

Key types:
- `GlobalConfig` — instances, default instance, output format
- `LocalConfig` — per-directory settings (safe to commit)
- `LocalPrivateConfig` — per-directory private state (gitignored)
- `MergedConfig` — effective config after resolution

Token storage uses the OS keyring (`zalando/go-keyring`) with a fallback to `~/.config/ytcli/credentials.yml` (0600 permissions).

## `internal/local`

Local schema defaults and validation.

Key types:
- `DefaultSchema` — default states, priorities, done states
- `GetSchema` — returns effective schema
- `ValidateState`, `ValidatePriority`, `IsDoneState` — schema helpers

## `internal/model`

Shared domain models.

Key types:
- `Issue`, `Project`, `User`, `Comment`, `Tag`, `IssueLink`, `WorkItem`
- `CustomField` — polymorphic JSON unmarshaling for YouTrack custom fields
- `CommandResult`, `SearchSuggestions`

## `internal/render`

Output formatting. Supports four modes:
- `table` — human-readable terminal output (default)
- `json` — indented JSON
- `wide` — reserved for future use (falls back to table)
- `markdown` — reserved for future use

Also provides `JSONQuiet` for machine-readable minimal output.

## `internal/service`

Business logic layer. Wraps `api.Client` with validation and auto-pagination.

Key types:
- `Service` — high-level operations like `ListIssues`, `CreateIssue`, `ExecuteCommand`
- `NewServiceWithClient` — allows injecting a mocked client for tests

## `internal/store`

SQLite persistence layer.

Key types:
- `LocalIssue`, `LocalComment`, `QueueItem`
- `Open`, `CreateIssue`, `UpdateIssue`, `ListIssues`, `DeleteIssue`
- `CreateComment`, `ListComments`, `CountComments`
- `CreateTag`, `AddIssueTag`, `RemoveIssueTag`, `GetIssueTags`
- `Enqueue`, `DequeuePending`, `MarkInProgress`, `MarkCompleted`, `MarkFailed`

## `internal/sync`

Sync orchestration.

Key types:
- `Manager` — coordinates Pull and Push
- `Result` — counts of pulled, pushed, conflicts, and failures

## `internal/cmdx`

Cobra command definitions. One file per command group. `root.go` holds shared flags, `buildService`, and `handleError`.

## `internal/tui`

Placeholder for a future Bubble Tea TUI. Currently unimplemented.
