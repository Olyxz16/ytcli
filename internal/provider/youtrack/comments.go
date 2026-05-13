package youtrack

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Olyxz16/tkt/internal/model"
)

// ListComments returns comments for an issue.
func (c *Client) ListComments(ctx context.Context, issueID string) ([]model.Comment, error) {
	q := url.Values{}
	q.Set("fields", "id,text,created,updated,author(id,login,name,fullName)")

	var ytComments []ytComment
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/issues/%s/comments", url.PathEscape(issueID)), q, nil, &ytComments); err != nil {
		return nil, err
	}

	comments := make([]model.Comment, len(ytComments))
	for i := range ytComments {
		comments[i] = *toComment(&ytComments[i])
	}
	return comments, nil
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(ctx context.Context, issueID string, text string) (*model.Comment, error) {
	q := url.Values{}
	q.Set("fields", "id,text,created,updated,author(id,login,name,fullName)")

	payload := map[string]string{"text": text}
	var ytComment ytComment
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/issues/%s/comments", url.PathEscape(issueID)), q, payload, &ytComment); err != nil {
		return nil, err
	}
	return toComment(&ytComment), nil
}