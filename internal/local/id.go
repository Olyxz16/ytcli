package local

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/store"
)

// ResolveID parses a user-provided ID string and returns the local database row ID.
// Supports: "#1", "1" (local), or provider refs like "PROJ-42"
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

	// Provider-prefixed → look up by provider_ref
	providerName := ""
	if localConfig != nil && localConfig.Provider.Name != "" {
		providerName = localConfig.Provider.Name
	}
	issue, err := store.GetIssueByProviderRef(db, providerName, input)
	if err != nil {
		return 0, err
	}
	if issue != nil {
		return issue.ID, nil
	}

	return 0, fmt.Errorf("issue %q not found locally", input)
}

// FormatID returns the best display ID for an issue.
// Prefers provider_ref if available, otherwise returns local #id.
func FormatID(issue *store.LocalIssue) string {
	if issue.ProviderRef != "" {
		return issue.ProviderRef
	}
	return fmt.Sprintf("#%d", issue.ID)
}

// IsLocalID checks if input looks like a local ID (#N or N).
func IsLocalID(input string) bool {
	input = strings.TrimPrefix(strings.TrimSpace(input), "#")
	_, err := strconv.ParseInt(input, 10, 64)
	return err == nil
}