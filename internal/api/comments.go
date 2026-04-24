package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/ytcli/internal/model"
)

// ListComments returns comments for an issue.
func (c *Client) ListComments(ctx context.Context, issueID string) ([]model.Comment, error) {
	q := url.Values{}
	q.Set("fields", "id,text,created,updated,author(id,login,name,fullName)")

	var comments []model.Comment
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s/comments", url.PathEscape(issueID)), q, nil, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(ctx context.Context, issueID string, text string) (*model.Comment, error) {
	q := url.Values{}
	q.Set("fields", "id,text,created,updated,author(id,login,name,fullName)")

	payload := map[string]string{"text": text}
	var comment model.Comment
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/issues/%s/comments", url.PathEscape(issueID)), q, payload, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}
