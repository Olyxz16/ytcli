package youtrack

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Olyxz16/tkt/internal/model"
)

// ListArticles searches for articles matching the query.
func (c *Client) ListArticles(ctx context.Context, query string, top, skip int) ([]model.Article, error) {
	q := url.Values{}
	q.Set("fields", ArticleList().String())
	if query != "" {
		q.Set("query", query)
	}
	if top > 0 {
		q.Set("$top", strconv.Itoa(top))
	}
	if skip > 0 {
		q.Set("$skip", strconv.Itoa(skip))
	}

	var ytArticles []ytArticle
	if err := c.doJSON(ctx, "GET", "/api/articles", q, nil, &ytArticles); err != nil {
		return nil, err
	}
	return toArticles(ytArticles), nil
}

// GetArticle retrieves a single article by ID.
func (c *Client) GetArticle(ctx context.Context, id string, withComments bool) (*model.Article, error) {
	q := url.Values{}
	if withComments {
		q.Set("fields", ArticleDetailWithComments().String())
	} else {
		q.Set("fields", ArticleDetail().String())
	}

	var ytArticle ytArticle
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/articles/%s", url.PathEscape(id)), q, nil, &ytArticle); err != nil {
		return nil, err
	}
	return toArticle(&ytArticle), nil
}

// CreateArticle creates a new article.
func (c *Client) CreateArticle(ctx context.Context, article model.Article) (*model.Article, error) {
	q := url.Values{}
	q.Set("fields", ArticleDetail().String())

	payload := map[string]interface{}{
		"summary": article.Summary,
		"project": map[string]string{"id": article.Project.ID},
	}
	if article.Content != "" {
		payload["content"] = article.Content
	}
	if article.ParentArticle != nil {
		payload["parentArticle"] = map[string]string{"id": article.ParentArticle.ID}
	}

	var ytArticle ytArticle
	if err := c.doJSON(ctx, "POST", "/api/articles", q, payload, &ytArticle); err != nil {
		return nil, err
	}
	return toArticle(&ytArticle), nil
}

// UpdateArticle updates an article.
func (c *Client) UpdateArticle(ctx context.Context, id string, updates map[string]interface{}) (*model.Article, error) {
	q := url.Values{}
	q.Set("fields", ArticleDetail().String())

	var ytArticle ytArticle
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/articles/%s", url.PathEscape(id)), q, updates, &ytArticle); err != nil {
		return nil, err
	}
	return toArticle(&ytArticle), nil
}

// DeleteArticle deletes an article.
func (c *Client) DeleteArticle(ctx context.Context, id string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/articles/%s", url.PathEscape(id)), nil, nil)
	return err
}