package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Olyxz16/ytcli/internal/model"
)

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

	var articles []model.Article
	if err := c.doJSON(ctx, "GET", "/api/articles", q, nil, &articles); err != nil {
		return nil, err
	}
	return articles, nil
}

func (c *Client) GetArticle(ctx context.Context, id string, withComments bool) (*model.Article, error) {
	q := url.Values{}
	if withComments {
		q.Set("fields", ArticleDetailWithComments().String())
	} else {
		q.Set("fields", ArticleDetail().String())
	}

	var article model.Article
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/articles/%s", url.PathEscape(id)), q, nil, &article); err != nil {
		return nil, err
	}
	return &article, nil
}

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

	var created model.Article
	if err := c.doJSON(ctx, "POST", "/api/articles", q, payload, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateArticle(ctx context.Context, id string, updates map[string]interface{}) (*model.Article, error) {
	q := url.Values{}
	q.Set("fields", ArticleDetail().String())

	var updated model.Article
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/articles/%s", url.PathEscape(id)), q, updates, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteArticle(ctx context.Context, id string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/articles/%s", url.PathEscape(id)), nil, nil)
	return err
}