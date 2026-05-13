package youtrack

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/tkt/internal/model"
)

// ListTags returns all tags accessible to the current user.
func (c *Client) ListTags(ctx context.Context) ([]model.Tag, error) {
	q := url.Values{}
	q.Set("fields", "id,name")

	var ytTags []ytTag
	if err := c.doJSON(ctx, "GET", "/api/tags", q, nil, &ytTags); err != nil {
		return nil, err
	}

	tags := make([]model.Tag, len(ytTags))
	for i, t := range ytTags {
		tags[i] = model.Tag{ID: t.ID, Name: t.Name}
	}
	return tags, nil
}

// AddTagToIssue adds a tag to an issue.
func (c *Client) AddTagToIssue(ctx context.Context, issueID string, tag model.Tag) error {
	payload := map[string]interface{}{
		"name": tag.Name,
	}
	if tag.ID != "" {
		payload["id"] = tag.ID
	}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/issues/%s/tags", url.PathEscape(issueID)), nil, payload)
	return err
}

// RemoveTagFromIssue removes a tag from an issue.
func (c *Client) RemoveTagFromIssue(ctx context.Context, issueID, tagID string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/issues/%s/tags/%s", url.PathEscape(issueID), url.PathEscape(tagID)), nil, nil)
	return err
}