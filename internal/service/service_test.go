package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/model"
)

func newTestService(handler http.HandlerFunc) (*Service, *httptest.Server) {
	srv := httptest.NewServer(handler)
	client := api.NewClient(srv.URL, "test-token")
	return NewServiceWithClient(client), srv
}

func TestNewServiceMissingURL(t *testing.T) {
	_, err := NewService(&config.MergedConfig{})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNewServiceWithClient(t *testing.T) {
	client := api.NewClient("http://localhost", "token")
	svc := NewServiceWithClient(client)
	if svc == nil {
		t.Fatal("expected service")
	}
}

func TestListIssuesPagination(t *testing.T) {
	callCount := 0
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Query().Get("$top") != "50" {
			t.Errorf("$top = %q", r.URL.Query().Get("$top"))
		}
		// Return full page, then empty
		if callCount == 1 {
			items := make([]model.Issue, 50)
			for i := range items {
				items[i] = model.Issue{ID: "x"}
			}
			data, _ := json.Marshal(items)
			w.Write(data)
		} else {
			w.Write([]byte(`[]`))
		}
	})
	defer srv.Close()

	issues, err := svc.ListIssues(context.Background(), "", 0, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(issues) != 50 {
		t.Errorf("len = %d, want 50", len(issues))
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, expected 2", callCount)
	}
}

func TestListIssuesWithTop(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("$top") != "5" {
			t.Errorf("$top = %q", r.URL.Query().Get("$top"))
		}
		w.Write([]byte(`[{"id":"1"}]`))
	})
	defer srv.Close()

	issues, err := svc.ListIssues(context.Background(), "", 5, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("len = %d", len(issues))
	}
}

func TestGetIssue(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"1","idReadable":"PROJ-1","summary":"Test"}`))
	})
	defer srv.Close()

	issue, err := svc.GetIssue(context.Background(), "PROJ-1", false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if issue.Summary != "Test" {
		t.Errorf("summary = %q", issue.Summary)
	}
}

func TestCreateIssueMissingProject(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call API")
	})
	defer srv.Close()

	_, err := svc.CreateIssue(context.Background(), model.Issue{Summary: "Test", Project: &model.Project{}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateIssueMissingSummary(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call API")
	})
	defer srv.Close()

	_, err := svc.CreateIssue(context.Background(), model.Issue{Project: &model.Project{ID: "p1"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateIssueResolveProject(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/admin/projects":
			w.Write([]byte(`[{"id":"p1","shortName":"PROJ","name":"Project"}]`))
		case "/api/issues":
			w.Write([]byte(`{"id":"1","idReadable":"PROJ-1","summary":"Test"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})
	defer srv.Close()

	issue, err := svc.CreateIssue(context.Background(), model.Issue{
		Summary:     "Test",
		Project:     &model.Project{ShortName: "PROJ"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if issue.IDReadable != "PROJ-1" {
		t.Errorf("idReadable = %q", issue.IDReadable)
	}
}

func TestCreateIssueProjectNotFound(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	_, err := svc.CreateIssue(context.Background(), model.Issue{
		Summary:     "Test",
		Project:     &model.Project{ShortName: "MISSING"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateIssue(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"1","summary":"Updated"}`))
	})
	defer srv.Close()

	issue, err := svc.UpdateIssue(context.Background(), "PROJ-1", map[string]interface{}{"summary": "Updated"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if issue.Summary != "Updated" {
		t.Errorf("summary = %q", issue.Summary)
	}
}

func TestDeleteIssue(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.DeleteIssue(context.Background(), "PROJ-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestIssueCount(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"count":10}`))
	})
	defer srv.Close()

	count, err := svc.IssueCount(context.Background(), "test")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 10 {
		t.Errorf("count = %d", count)
	}
}

func TestExecuteCommandMissingQuery(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call API")
	})
	defer srv.Close()

	_, err := svc.ExecuteCommand(context.Background(), "", []string{"PROJ-1"}, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteCommandMissingIDs(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call API")
	})
	defer srv.Close()

	_, err := svc.ExecuteCommand(context.Background(), "State: Done", nil, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteCommand(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"query":"State: Done","issues":[{"idReadable":"PROJ-1"}]}`))
	})
	defer srv.Close()

	result, err := svc.ExecuteCommand(context.Background(), "State: Done", []string{"PROJ-1"}, false)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Query != "State: Done" {
		t.Errorf("query = %q", result.Query)
	}
}

func TestListComments(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"c1","text":"Hello"}]`))
	})
	defer srv.Close()

	comments, err := svc.ListComments(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("len = %d", len(comments))
	}
}

func TestAddCommentMissingText(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call API")
	})
	defer srv.Close()

	_, err := svc.AddComment(context.Background(), "PROJ-1", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddComment(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"c1","text":"Hello"}`))
	})
	defer srv.Close()

	comment, err := svc.AddComment(context.Background(), "PROJ-1", "Hello")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if comment.Text != "Hello" {
		t.Errorf("text = %q", comment.Text)
	}
}

func TestListProjects(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"p1","shortName":"PROJ"}]`))
	})
	defer srv.Close()

	projects, err := svc.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("len = %d", len(projects))
	}
}

func TestGetProject(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"p1","shortName":"PROJ","name":"Project"}`))
	})
	defer srv.Close()

	project, err := svc.GetProject(context.Background(), "PROJ")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if project.Name != "Project" {
		t.Errorf("name = %q", project.Name)
	}
}

func TestMe(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"login":"john","name":"John Doe"}`))
	})
	defer srv.Close()

	user, err := svc.Me(context.Background())
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if user.Login != "john" {
		t.Errorf("login = %q", user.Login)
	}
}

func TestListLinks(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"l1","linkType":"relates to"}]`))
	})
	defer srv.Close()

	links, err := svc.ListLinks(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("len = %d", len(links))
	}
}

func TestAddLink(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.AddLink(context.Background(), "PROJ-1", "PROJ-2", "relates"); err != nil {
		t.Fatalf("add link: %v", err)
	}
}

func TestListWorkItems(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"w1","text":"work"}]`))
	})
	defer srv.Close()

	items, err := svc.ListWorkItems(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len = %d", len(items))
	}
}

func TestAddWorkItem(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"w1","text":"work"}`))
	})
	defer srv.Close()

	item, err := svc.AddWorkItem(context.Background(), "PROJ-1", model.WorkItem{
		Text:     "work",
		Duration: &model.Duration{Minutes: 30},
	})
	if err != nil {
		t.Fatalf("add work item: %v", err)
	}
	if item.Text != "work" {
		t.Errorf("text = %q", item.Text)
	}
}

func TestListTags(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"t1","name":"bug"}]`))
	})
	defer srv.Close()

	tags, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 1 {
		t.Errorf("len = %d", len(tags))
	}
}

func TestAddTagToIssue(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.AddTagToIssue(context.Background(), "PROJ-1", model.Tag{Name: "bug"}); err != nil {
		t.Fatalf("add tag: %v", err)
	}
}

func TestRemoveTagFromIssue(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte(`{"id":"2-42","idReadable":"PROJ-1","summary":"Test","tags":[{"id":"6-1","name":"bug"},{"id":"6-2","name":"t1"}]}`))
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	defer srv.Close()

	if err := svc.RemoveTagFromIssue(context.Background(), "PROJ-1", "t1"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
}

func TestRemoveTagFromIssueNotFound(t *testing.T) {
	svc, srv := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"2-42","idReadable":"PROJ-1","summary":"Test","tags":[{"id":"6-1","name":"bug"}]}`))
	})
	defer srv.Close()

	err := svc.RemoveTagFromIssue(context.Background(), "PROJ-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestNewServiceMissingInstanceURL(t *testing.T) {
	cfg := &config.MergedConfig{}
	_, err := NewService(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}
