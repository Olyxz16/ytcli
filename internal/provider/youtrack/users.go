package youtrack

import (
	"context"
	"net/url"

	"github.com/Olyxz16/tkt/internal/model"
)

// Me returns the current user.
func (c *Client) Me(ctx context.Context) (*model.User, error) {
	q := url.Values{}
	q.Set("fields", UserDetail().String())

	var ytUser ytUser
	if err := c.doJSON(ctx, "GET", "/api/users/me", q, nil, &ytUser); err != nil {
		return nil, err
	}
	return toUser(&ytUser), nil
}

// ListUsers returns users matching a query.
func (c *Client) ListUsers(ctx context.Context, query string) ([]model.User, error) {
	q := url.Values{}
	q.Set("fields", UserDetail().String())
	if query != "" {
		q.Set("query", query)
	}

	var ytUsers []ytUser
	if err := c.doJSON(ctx, "GET", "/api/users", q, nil, &ytUsers); err != nil {
		return nil, err
	}

	users := make([]model.User, len(ytUsers))
	for i := range ytUsers {
		users[i] = *toUser(&ytUsers[i])
	}
	return users, nil
}