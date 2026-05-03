package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/service"
)

type PullResult struct {
	Pulled  int      `json:"pulled"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

type PushResult struct {
	Pushed  int      `json:"pushed"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

type StatusResult struct {
	NeedsPull bool   `json:"needs_pull"`
	WikiDir   string `json:"wiki_dir"`
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func articleFileName(article model.Article) string {
	if article.IDReadable != "" {
		return fmt.Sprintf("%s-%s.md", article.IDReadable, slugify(article.Summary))
	}
	return fmt.Sprintf("%s-%s.md", article.ID, slugify(article.Summary))
}

func writeArticleFile(dir string, article model.Article) error {
	fileName := articleFileName(article)
	path := filepath.Join(dir, fileName)

	var sb strings.Builder
	sb.WriteString("<!--\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", article.ID))
	sb.WriteString(fmt.Sprintf("idReadable: %s\n", article.IDReadable))
	sb.WriteString(fmt.Sprintf("summary: %s\n", article.Summary))
	if article.Project != nil {
		sb.WriteString(fmt.Sprintf("project: %s\n", article.Project.ShortName))
	}
	if article.ParentArticle != nil {
		sb.WriteString(fmt.Sprintf("parentArticle: %s\n", article.ParentArticle.IDReadable))
	}
	if article.Updated > 0 {
		sb.WriteString(fmt.Sprintf("updated: %d\n", article.Updated))
	}
	sb.WriteString("-->\n\n")
	content := article.Content
	if content == "" {
		content = article.WikifiedContent
	}
	sb.WriteString(content)
	sb.WriteString("\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func parseArticleFile(data []byte) (map[string]string, string, error) {
	frontMatter := make(map[string]string)
	text := string(data)

	if !strings.HasPrefix(text, "<!--\n") {
		return frontMatter, text, nil
	}

	end := strings.Index(text, "\n-->\n")
	if end == -1 {
		return frontMatter, text, nil
	}

	frontText := text[5:end]
	for _, line := range strings.Split(frontText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		frontMatter[key] = value
	}

	bodyStart := end + 5
	content := text[bodyStart:]
	content = strings.TrimRight(content, "\n")
	return frontMatter, content, nil
}

func Pull(ctx context.Context, svc *service.Service, wikiDir string) (*PullResult, error) {
	if err := os.MkdirAll(wikiDir, 0755); err != nil {
		return nil, fmt.Errorf("create wiki directory: %w", err)
	}

	articles, err := svc.ListArticles(ctx, "", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("fetch articles: %w", err)
	}

	result := &PullResult{}
	for _, article := range articles {
		detail, err := svc.GetArticle(ctx, article.IDReadable, false)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("fetch %s: %s", article.IDReadable, err))
			continue
		}

		fileName := articleFileName(*detail)
		path := filepath.Join(wikiDir, fileName)

		if stat, err := os.Stat(path); err == nil {
			localModTime := stat.ModTime()
			remoteUpdated := time.UnixMilli(detail.Updated)
			if !remoteUpdated.After(localModTime) {
				result.Skipped++
				continue
			}
		}

		if err := writeArticleFile(wikiDir, *detail); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write %s: %s", article.IDReadable, err))
			continue
		}
		result.Pulled++
	}

	return result, nil
}

func Push(ctx context.Context, svc *service.Service, wikiDir string) (*PushResult, error) {
	entries, err := os.ReadDir(wikiDir)
	if err != nil {
		return nil, fmt.Errorf("read wiki directory: %w", err)
	}

	result := &PushResult{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			result.Skipped++
			continue
		}

		path := filepath.Join(wikiDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("read %s: %s", entry.Name(), err))
			continue
		}

		frontMatter, content, err := parseArticleFile(data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("parse %s: %s", entry.Name(), err))
			continue
		}

		id, ok := frontMatter["idReadable"]
		if !ok || id == "" {
			result.Skipped++
			continue
		}

		updates := map[string]interface{}{
			"content": content,
		}
		if summary, ok := frontMatter["summary"]; ok && summary != "" {
			updates["summary"] = summary
		}

		if _, err := svc.UpdateArticle(ctx, id, updates); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("update %s: %s", id, err))
			continue
		}

		detail, err := svc.GetArticle(ctx, id, false)
		if err == nil {
			_ = writeArticleFile(wikiDir, *detail)
		}
		result.Pushed++
	}

	return result, nil
}

func Status(ctx context.Context, svc *service.Service, wikiDir string) (*StatusResult, error) {
	result := &StatusResult{
		WikiDir: wikiDir,
	}

	articles, err := svc.ListArticles(ctx, "", 1, 0)
	if err != nil {
		return nil, fmt.Errorf("fetch articles: %w", err)
	}
	if len(articles) == 0 {
		result.NeedsPull = false
		return result, nil
	}

	detail, err := svc.GetArticle(ctx, articles[0].IDReadable, false)
	if err != nil {
		return nil, fmt.Errorf("check article: %w", err)
	}

	fileName := articleFileName(*detail)
	path := filepath.Join(wikiDir, fileName)
	stat, err := os.Stat(path)
	if err != nil {
		result.NeedsPull = true
		return result, nil
	}

	modTime := stat.ModTime()
	updatedTime := time.UnixMilli(detail.Updated)
	result.NeedsPull = updatedTime.After(modTime)
	return result, nil
}