package api

import (
	"context"
	"net/url"

	"github.com/Olyxz16/ytcli/internal/model"
)

// Me returns the current user.
func (c *Client) Me(ctx context.Context) (*model.User, error) {
	q := url.Values{}
	q.Set("fields", UserDetail().String())

	var user model.User
	if err := c.doJSON(ctx, "GET", "/api/users/me", q, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers returns users matching a query.
func (c *Client) ListUsers(ctx context.Context, query string) ([]model.User, error) {
	q := url.Values{}
	q.Set("fields", UserDetail().String())
	if query != "" {
		q.Set("query", query)
	}

	var users []model.User
	if err := c.doJSON(ctx, "GET", "/api/users", q, nil, &users); err != nil {
		return nil, err
	}
	return users, nil
}
