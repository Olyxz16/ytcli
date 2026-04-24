package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/ytcli/internal/model"
)

// ListTags returns all tags accessible to the current user.
func (c *Client) ListTags(ctx context.Context) ([]model.Tag, error) {
	q := url.Values{}
	q.Set("fields", "id,name")

	var tags []model.Tag
	if err := c.doJSON(ctx, "GET", "/api/tags", q, nil, &tags); err != nil {
		return nil, err
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
