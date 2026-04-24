package local

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/store"
)

// ResolveID parses a user-provided ID string and returns the local database row ID.
// Supports: "#1", "1", "PROJ-42" (if project prefix matches and remote_id exists locally)
func ResolveID(db *sql.DB, input string, localConfig *config.LocalConfig) (int64, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "#")

	// Pure numeric → local ID
	if n, err := strconv.ParseInt(input, 10, 64); err == nil {
		issue, err := store.GetIssue(db, n)
		if err != nil {
			return 0, err
		}
		if issue == nil {
			return 0, fmt.Errorf("issue #%d not found locally", n)
		}
		return issue.ID, nil
	}

	// Project-prefixed → look up by remote_id
	// e.g. "PROJ-42" → search store.issues WHERE remote_id = "PROJ-42"
	issue, err := store.GetIssueByRemoteID(db, input)
	if err != nil {
		return 0, err
	}
	if issue != nil {
		return issue.ID, nil
	}

	return 0, fmt.Errorf("issue %q not found locally", input)
}

// FormatID returns the best display ID for an issue.
// Prefers remote_id if available, otherwise returns local #id.
func FormatID(issue *store.LocalIssue) string {
	if issue.RemoteID != nil && *issue.RemoteID != "" {
		return *issue.RemoteID
	}
	return fmt.Sprintf("#%d", issue.ID)
}

// IsLocalID checks if input looks like a local ID (#N or N).
func IsLocalID(input string) bool {
	input = strings.TrimPrefix(strings.TrimSpace(input), "#")
	_, err := strconv.ParseInt(input, 10, 64)
	return err == nil
}
