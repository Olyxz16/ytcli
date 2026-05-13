package youtrack

import (
	"context"
	"net/url"

	"github.com/Olyxz16/tkt/internal/model"
)

// ExecuteCommand applies a YouTrack command to one or more issues.
func (c *Client) ExecuteCommand(ctx context.Context, cmd model.CommandResult, silent bool) (*model.CommandResult, error) {
	q := url.Values{}
	q.Set("fields", "query,issues(id,idReadable,summary)")
	if silent {
		q.Set("muteUpdateNotifications", "true")
	}

	issues := make([]map[string]string, len(cmd.Issues))
	for i, issue := range cmd.Issues {
		if issue.DatabaseID != "" {
			issues[i] = map[string]string{"id": issue.DatabaseID}
		} else {
			issues[i] = map[string]string{"idReadable": issue.ID}
		}
	}

	payload := map[string]interface{}{
		"query":  cmd.Query,
		"issues": issues,
	}

	var result ytCommandResult
	if err := c.doJSON(ctx, "POST", "/api/commands", q, payload, &result); err != nil {
		return nil, err
	}
	return toCommandResult(&result), nil
}

// CommandSuggestions gets autocomplete suggestions for a command string.
func (c *Client) CommandSuggestions(ctx context.Context, cmd string, issueIDs []string) (*ytSearchSuggestions, error) {
	q := url.Values{}
	q.Set("fields", "query,suggestions(option,completionStart,description,prefix,suffix)")

	issues := make([]map[string]string, len(issueIDs))
	for i, id := range issueIDs {
		issues[i] = map[string]string{"idReadable": id}
	}
	payload := map[string]interface{}{
		"query":  cmd,
		"issues": issues,
		"caret":  len(cmd),
	}

	var result ytSearchSuggestions
	if err := c.doJSON(ctx, "POST", "/api/commands/assist", q, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}