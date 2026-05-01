package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Olyxz16/ytcli/internal/model"
)

func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	client := NewClient(srv.URL, "test-token")
	return client, srv
}

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.com/", "token")
	if c == nil {
		t.Fatal("expected client")
	}
}

func TestDoSuccess(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"1"}`))
	})
	defer srv.Close()

	resp, err := client.do(context.Background(), "GET", "/test", nil, nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestDoJSONSuccess(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"1","summary":"Test"}`))
	})
	defer srv.Close()

	var result model.Issue
	if err := client.doJSON(context.Background(), "GET", "/test", nil, nil, &result); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if result.ID != "1" {
		t.Errorf("id = %q", result.ID)
	}
}

func TestDoJSONError(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`not found`))
	})
	defer srv.Close()

	_, err := client.do(context.Background(), "GET", "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected not found error, got %T", err)
	}
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		code   int
		fn     func(error) bool
		expect bool
	}{
		{http.StatusUnauthorized, IsAuthError, true},
		{http.StatusForbidden, IsForbiddenError, true},
		{http.StatusNotFound, IsNotFoundError, true},
		{http.StatusBadRequest, IsValidationError, true},
		{http.StatusOK, IsAuthError, false},
	}

	for _, tt := range tests {
		err := newAPIError(tt.code, "msg")
		if tt.fn(err) != tt.expect {
			t.Errorf("code %d: expected %v", tt.code, tt.expect)
		}
	}
}

func TestListIssues(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issues" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`[{"id":"1","idReadable":"PROJ-1","summary":"Hello"}]`))
	})
	defer srv.Close()

	issues, err := client.ListIssues(context.Background(), "test", 10, 0, nil)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len = %d", len(issues))
	}
	if issues[0].Summary != "Hello" {
		t.Errorf("summary = %q", issues[0].Summary)
	}
}

func TestGetIssue(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issues/PROJ-1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"id":"1","idReadable":"PROJ-1","summary":"Detail"}`))
	})
	defer srv.Close()

	issue, err := client.GetIssue(context.Background(), "PROJ-1", false)
	if err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if issue.Summary != "Detail" {
		t.Errorf("summary = %q", issue.Summary)
	}
}

func TestCreateIssue(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q", r.Method)
		}
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["summary"] != "New" {
			t.Errorf("summary = %v", payload["summary"])
		}
		w.Write([]byte(`{"id":"1","idReadable":"PROJ-2","summary":"New"}`))
	})
	defer srv.Close()

	issue, err := client.CreateIssue(context.Background(), model.Issue{
		Summary: "New",
		Project: &model.Project{ID: "p1"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if issue.IDReadable != "PROJ-2" {
		t.Errorf("idReadable = %q", issue.IDReadable)
	}
}

func TestUpdateIssue(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"1","summary":"Updated"}`))
	})
	defer srv.Close()

	issue, err := client.UpdateIssue(context.Background(), "PROJ-1", map[string]interface{}{"summary": "Updated"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if issue.Summary != "Updated" {
		t.Errorf("summary = %q", issue.Summary)
	}
}

func TestDeleteIssue(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.DeleteIssue(context.Background(), "PROJ-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestIssueCount(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"count":42}`))
	})
	defer srv.Close()

	count, err := client.IssueCount(context.Background(), "test")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d", count)
	}
}

func TestListProjects(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"p1","shortName":"PROJ","name":"Project"}]`))
	})
	defer srv.Close()

	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("len = %d", len(projects))
	}
	if projects[0].ShortName != "PROJ" {
		t.Errorf("shortName = %q", projects[0].ShortName)
	}
}

func TestGetProject(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"p1","shortName":"PROJ","name":"Project"}`))
	})
	defer srv.Close()

	project, err := client.GetProject(context.Background(), "PROJ")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if project.Name != "Project" {
		t.Errorf("name = %q", project.Name)
	}
}

func TestExecuteCommand(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/commands" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"query":"State: Done","issues":[{"idReadable":"PROJ-1"}]}`))
	})
	defer srv.Close()

	result, err := client.ExecuteCommand(context.Background(), model.CommandResult{
		Query:  "State: Done",
		Issues: []model.Issue{{IDReadable: "PROJ-1"}},
	}, false)
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if result.Query != "State: Done" {
		t.Errorf("query = %q", result.Query)
	}
}

func TestCommandSuggestions(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"query":"State","suggestions":[{"option":"State: Open"}]}`))
	})
	defer srv.Close()

	suggestions, err := client.CommandSuggestions(context.Background(), "State", []string{"PROJ-1"})
	if err != nil {
		t.Fatalf("suggestions: %v", err)
	}
	if len(suggestions.Suggestions) != 1 {
		t.Fatalf("len = %d", len(suggestions.Suggestions))
	}
}

func TestListComments(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"c1","text":"Hello","author":{"login":"a"}}]`))
	})
	defer srv.Close()

	comments, err := client.ListComments(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 || comments[0].Text != "Hello" {
		t.Errorf("unexpected comments: %+v", comments)
	}
}

func TestAddComment(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"c1","text":"New comment"}`))
	})
	defer srv.Close()

	comment, err := client.AddComment(context.Background(), "PROJ-1", "New comment")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if comment.Text != "New comment" {
		t.Errorf("text = %q", comment.Text)
	}
}

func TestListTags(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"t1","name":"bug"}]`))
	})
	defer srv.Close()

	tags, err := client.ListTags(context.Background())
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "bug" {
		t.Errorf("unexpected tags: %+v", tags)
	}
}

func TestAddTagToIssue(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.AddTagToIssue(context.Background(), "PROJ-1", model.Tag{Name: "bug"}); err != nil {
		t.Fatalf("add tag: %v", err)
	}
}

func TestRemoveTagFromIssue(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.RemoveTagFromIssue(context.Background(), "PROJ-1", "t1"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
}

func TestListLinks(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"l1","linkType":"relates to"}]`))
	})
	defer srv.Close()

	links, err := client.ListLinks(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("len = %d", len(links))
	}
}

func TestAddLink(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.AddLink(context.Background(), "PROJ-1", "PROJ-2", "relates"); err != nil {
		t.Fatalf("add link: %v", err)
	}
}

func TestListWorkItems(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"w1","text":"work","duration":{"minutes":30,"presentation":"30m"}}]`))
	})
	defer srv.Close()

	items, err := client.ListWorkItems(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("list work items: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len = %d", len(items))
	}
}

func TestAddWorkItem(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"w1","text":"work"}`))
	})
	defer srv.Close()

	item, err := client.AddWorkItem(context.Background(), "PROJ-1", model.WorkItem{
		Text:     "work",
		Duration: &model.Duration{Minutes: 30, Presentation: "30m"},
	})
	if err != nil {
		t.Fatalf("add work item: %v", err)
	}
	if item.Text != "work" {
		t.Errorf("text = %q", item.Text)
	}
}

func TestMe(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/me" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"login":"john","name":"John"}`))
	})
	defer srv.Close()

	user, err := client.Me(context.Background())
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if user.Login != "john" {
		t.Errorf("login = %q", user.Login)
	}
}

func TestListUsers(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"login":"john"},{"login":"jane"}]`))
	})
	defer srv.Close()

	users, err := client.ListUsers(context.Background(), "")
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("len = %d", len(users))
	}
}

func TestMarshalCustomFields(t *testing.T) {
	fields := []model.CustomField{
		{Name: "Priority", Value: model.BundleElement{Name: "Major"}},
		{Name: "Tags", Value: []model.BundleElement{{Name: "bug"}}},
		{Name: "Assignee", Value: model.User{Login: "john"}},
		{Name: "Estimation", Value: model.Duration{Presentation: "1h"}},
		{Name: "Notes", Value: "text"},
		{Name: "Due", Value: int64(1234567890)},
	}

	result := marshalCustomFields(fields)
	if len(result) != len(fields) {
		t.Fatalf("len = %d", len(result))
	}

	types := map[string]string{
		"Priority":   "SingleEnumIssueCustomField",
		"Tags":       "MultiEnumIssueCustomField",
		"Assignee":   "SingleUserIssueCustomField",
		"Estimation": "PeriodIssueCustomField",
		"Notes":      "TextIssueCustomField",
		"Due":        "DateIssueCustomField",
	}

	for _, r := range result {
		name := r["name"].(string)
		expectedType := types[name]
		if r["$type"] != expectedType {
			t.Errorf("%s $type = %v, want %s", name, r["$type"], expectedType)
		}
	}
}

func TestBoolPtr(t *testing.T) {
	p := BoolPtr(true)
	if p == nil || *p != true {
		t.Error("BoolPtr failed")
	}
}

func TestIntPtr(t *testing.T) {
	p := IntPtr(42)
	if p == nil || *p != 42 {
		t.Error("IntPtr failed")
	}
}

func TestAPIErrorError(t *testing.T) {
	err := newAPIError(500, "server error")
	if err.Error() != "HTTP 500: server error" {
		t.Errorf("error string = %q", err.Error())
	}
}

func TestDoRequestBody(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if payload["key"] != "value" {
			t.Errorf("payload = %v", payload)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	_, err := client.do(context.Background(), "POST", "/test", nil, map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestDoPathEscaping(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issues/PROJ-1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	_, err := client.do(context.Background(), "GET", "/api/issues/PROJ-1", nil, nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestDoQueryValues(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fields") != "id" {
			t.Errorf("fields = %q", r.URL.Query().Get("fields"))
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	q := make(map[string][]string)
	q["fields"] = []string{"id"}
	_, err := client.do(context.Background(), "GET", "/test", q, nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestSetHTTPClient(t *testing.T) {
	c := NewClient("https://example.com", "token")
	custom := &http.Client{}
	c.SetHTTPClient(custom)
	if c.httpClient != custom {
		t.Error("http client not set")
	}
}

func TestMarshalCustomFieldsFallbackType(t *testing.T) {
	fields := []model.CustomField{
		{Name: "Custom", Value: 42.5, Type: "CustomType"},
	}
	result := marshalCustomFields(fields)
	if result[0]["$type"] != "CustomType" {
		t.Errorf("$type = %v, want CustomType", result[0]["$type"])
	}
}

func TestCreateIssueWithDescriptionAndTags(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["description"] != "Desc" {
			t.Errorf("description = %v", payload["description"])
		}
		tags := payload["tags"].([]interface{})
		if len(tags) != 1 {
			t.Errorf("tags len = %d", len(tags))
		}
		w.Write([]byte(`{"id":"1"}`))
	})
	defer srv.Close()

	_, err := client.CreateIssue(context.Background(), model.Issue{
		Summary:     "Test",
		Description: "Desc",
		Project:     &model.Project{ID: "p1"},
		Tags:        []model.Tag{{Name: "bug"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestListIssuesPagination(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("$top") != "10" {
			t.Errorf("$top = %q", r.URL.Query().Get("$top"))
		}
		if r.URL.Query().Get("$skip") != "5" {
			t.Errorf("$skip = %q", r.URL.Query().Get("$skip"))
		}
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	_, err := client.ListIssues(context.Background(), "", 10, 5, []string{"Priority"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestDeleteIssueNotFound(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`issue not found`))
	})
	defer srv.Close()

	err := client.DeleteIssue(context.Background(), "MISSING")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected not found, got %T", err)
	}
}

func TestIssueCountWithQuery(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "project: PROJ" {
			t.Errorf("query = %q", r.URL.Query().Get("query"))
		}
		w.Write([]byte(`{"count":5}`))
	})
	defer srv.Close()

	count, err := client.IssueCount(context.Background(), "project: PROJ")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d", count)
	}
}

func TestMeError(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`auth required`))
	})
	defer srv.Close()

	_, err := client.Me(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAuthError(err) {
		t.Errorf("expected auth error, got %T", err)
	}
}

func TestExecuteCommandSilent(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("muteUpdateNotifications") != "true" {
			t.Errorf("muteUpdateNotifications = %q", r.URL.Query().Get("muteUpdateNotifications"))
		}
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := client.ExecuteCommand(context.Background(), model.CommandResult{Query: "Done"}, true)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestAddWorkItemWithoutType(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if _, ok := payload["type"]; ok {
			t.Error("expected no type field")
		}
		w.Write([]byte(`{"id":"w1"}`))
	})
	defer srv.Close()

	_, err := client.AddWorkItem(context.Background(), "PROJ-1", model.WorkItem{
		Text:     "work",
		Duration: &model.Duration{Minutes: 30},
	})
	if err != nil {
		t.Fatalf("add work item: %v", err)
	}
}

func TestAddWorkItemWithType(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		typeVal := payload["type"].(map[string]interface{})
		if typeVal["id"] != "dev" {
			t.Errorf("type id = %q", typeVal["id"])
		}
		w.Write([]byte(`{"id":"w1"}`))
	})
	defer srv.Close()

	_, err := client.AddWorkItem(context.Background(), "PROJ-1", model.WorkItem{
		Duration: &model.Duration{Minutes: 30},
		Type:     &model.WorkItemType{ID: "dev"},
	})
	if err != nil {
		t.Fatalf("add work item: %v", err)
	}
}

func TestGetProjectPathEscape(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/admin/projects/PROJ-1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"id":"p1"}`))
	})
	defer srv.Close()

	_, err := client.GetProject(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
}

func TestRemoveTagPathEscape(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		expected := "/api/issues/PROJ-1/tags/t%20ag"
		if r.URL.EscapedPath() != expected {
			t.Errorf("path = %q, want %q", r.URL.EscapedPath(), expected)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.RemoveTagFromIssue(context.Background(), "PROJ-1", "t ag"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
}

func TestAddCommentPathEscape(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		expected := "/api/issues/PROJ-1/comments"
		if r.URL.Path != expected {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"id":"c1"}`))
	})
	defer srv.Close()

	_, err := client.AddComment(context.Background(), "PROJ-1", "text")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
}

func TestListUsersWithQuery(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "john" {
			t.Errorf("query = %q", r.URL.Query().Get("query"))
		}
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	_, err := client.ListUsers(context.Background(), "john")
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
}

func TestCommandSuggestionsBody(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		issues := payload["issues"].([]interface{})
		if len(issues) != 1 {
			t.Errorf("issues len = %d", len(issues))
		}
		w.Write([]byte(`{"query":"","suggestions":[]}`))
	})
	defer srv.Close()

	_, err := client.CommandSuggestions(context.Background(), "State", []string{"PROJ-1"})
	if err != nil {
		t.Fatalf("suggestions: %v", err)
	}
}

func TestDoJSONDecodeError(t *testing.T) {
	client, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})
	defer srv.Close()

	var result model.Issue
	err := client.doJSON(context.Background(), "GET", "/test", nil, nil, &result)
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestDoNetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "token")
	_, err := c.do(context.Background(), "GET", "/test", nil, nil)
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestDoInvalidURL(t *testing.T) {
	c := NewClient("http://[::1]:namedport", "token")
	_, err := c.do(context.Background(), "GET", "/test", nil, nil)
	if err == nil {
		t.Fatal("expected URL error")
	}
}

func TestMarshalCustomFieldsUnknownValue(t *testing.T) {
	fields := []model.CustomField{
		{Name: "Custom", Value: struct{ A int }{A: 1}},
	}
	result := marshalCustomFields(fields)
	if _, ok := result[0]["$type"]; ok {
		t.Error("expected no $type for unknown value")
	}
}

func TestIsAuthErrorWithNonAPIError(t *testing.T) {
	if IsAuthError(fmt.Errorf("random")) {
		t.Error("expected false for non-API error")
	}
}

func TestIsNotFoundErrorWithNonAPIError(t *testing.T) {
	if IsNotFoundError(fmt.Errorf("random")) {
		t.Error("expected false for non-API error")
	}
}

func TestIsValidationErrorWithNonAPIError(t *testing.T) {
	if IsValidationError(fmt.Errorf("random")) {
		t.Error("expected false for non-API error")
	}
}

func TestIsForbiddenErrorWithNonAPIError(t *testing.T) {
	if IsForbiddenError(fmt.Errorf("random")) {
		t.Error("expected false for non-API error")
	}
}
