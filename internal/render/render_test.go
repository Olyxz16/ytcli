package render

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Olyxz16/ytcli/internal/model"
	"github.com/Olyxz16/ytcli/internal/store"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestJSON(t *testing.T) {
	out := captureStdout(func() {
		JSON(map[string]string{"key": "value"})
	})
	var result map[string]string
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q", result["key"])
	}
}

func TestJSONQuiet(t *testing.T) {
	out := captureStdout(func() {
		JSONQuiet(map[string]string{"key": "value"})
	})
	if !strings.Contains(out, `"key"`) {
		t.Errorf("output = %q", out)
	}
}

func TestIssueListJSON(t *testing.T) {
	out := captureStdout(func() {
		IssueList([]model.Issue{{IDReadable: "PROJ-1", Summary: "Test"}}, OutputJSON)
	})
	if !strings.Contains(out, "PROJ-1") {
		t.Errorf("output = %q", out)
	}
}

func TestIssueListTable(t *testing.T) {
	out := captureStdout(func() {
		IssueList([]model.Issue{{IDReadable: "PROJ-1", Summary: "Test"}}, OutputTable)
	})
	if !strings.Contains(out, "PROJ-1") {
		t.Errorf("output = %q", out)
	}
}

func TestIssueListEmpty(t *testing.T) {
	out := captureStdout(func() {
		IssueList([]model.Issue{}, OutputTable)
	})
	if !strings.Contains(out, "No issues found") {
		t.Errorf("output = %q", out)
	}
}

func TestIssueDetailJSON(t *testing.T) {
	out := captureStdout(func() {
		IssueDetail(&model.Issue{IDReadable: "PROJ-1", Summary: "Test"}, OutputJSON)
	})
	if !strings.Contains(out, "PROJ-1") {
		t.Errorf("output = %q", out)
	}
}

func TestIssueDetailText(t *testing.T) {
	out := captureStdout(func() {
		IssueDetail(&model.Issue{IDReadable: "PROJ-1", Summary: "Test", Project: &model.Project{Name: "P"}}, OutputTable)
	})
	if !strings.Contains(out, "PROJ-1") {
		t.Errorf("output = %q", out)
	}
}

func TestCommentListJSON(t *testing.T) {
	out := captureStdout(func() {
		CommentList([]model.Comment{{Text: "Hello"}}, OutputJSON)
	})
	if !strings.Contains(out, "Hello") {
		t.Errorf("output = %q", out)
	}
}

func TestCommentListText(t *testing.T) {
	out := captureStdout(func() {
		CommentList([]model.Comment{{Text: "Hello", Author: &model.User{Login: "john"}}}, OutputTable)
	})
	if !strings.Contains(out, "Hello") {
		t.Errorf("output = %q", out)
	}
}

func TestProjectListJSON(t *testing.T) {
	out := captureStdout(func() {
		ProjectList([]model.Project{{ShortName: "PROJ"}}, OutputJSON)
	})
	if !strings.Contains(out, "PROJ") {
		t.Errorf("output = %q", out)
	}
}

func TestProjectListText(t *testing.T) {
	out := captureStdout(func() {
		ProjectList([]model.Project{{ShortName: "PROJ", Name: "Project"}}, OutputTable)
	})
	if !strings.Contains(out, "PROJ") {
		t.Errorf("output = %q", out)
	}
}

func TestUserJSON(t *testing.T) {
	out := captureStdout(func() {
		User(&model.User{Login: "john"}, OutputJSON)
	})
	if !strings.Contains(out, "john") {
		t.Errorf("output = %q", out)
	}
}

func TestUserText(t *testing.T) {
	out := captureStdout(func() {
		User(&model.User{Login: "john", FullName: "John Doe"}, OutputTable)
	})
	if !strings.Contains(out, "John Doe") {
		t.Errorf("output = %q", out)
	}
}

func TestCommandResultJSON(t *testing.T) {
	out := captureStdout(func() {
		CommandResult(&model.CommandResult{Query: "Done"}, OutputJSON)
	})
	if !strings.Contains(out, "Done") {
		t.Errorf("output = %q", out)
	}
}

func TestCommandResultText(t *testing.T) {
	out := captureStdout(func() {
		CommandResult(&model.CommandResult{Query: "Done"}, OutputTable)
	})
	if !strings.Contains(out, "Done") {
		t.Errorf("output = %q", out)
	}
}

func TestWorkItemListJSON(t *testing.T) {
	out := captureStdout(func() {
		WorkItemList([]model.WorkItem{{Text: "work"}}, OutputJSON)
	})
	if !strings.Contains(out, "work") {
		t.Errorf("output = %q", out)
	}
}

func TestWorkItemListText(t *testing.T) {
	out := captureStdout(func() {
		WorkItemList([]model.WorkItem{{Text: "work", Duration: &model.Duration{Presentation: "1h"}}}, OutputTable)
	})
	if !strings.Contains(out, "work") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalIssueListJSON(t *testing.T) {
	out := captureStdout(func() {
		LocalIssueList([]store.LocalIssue{{ID: 1, Summary: "Test"}}, OutputJSON)
	})
	if !strings.Contains(out, "Test") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalIssueListTable(t *testing.T) {
	out := captureStdout(func() {
		LocalIssueList([]store.LocalIssue{{ID: 1, Summary: "Test", State: "Open", SyncStatus: "local"}}, OutputTable)
	})
	if !strings.Contains(out, "Test") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalIssueListEmpty(t *testing.T) {
	out := captureStdout(func() {
		LocalIssueList([]store.LocalIssue{}, OutputTable)
	})
	if !strings.Contains(out, "No issues found") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalIssueDetailJSON(t *testing.T) {
	out := captureStdout(func() {
		LocalIssueDetail(&store.LocalIssue{ID: 1, Summary: "Test"}, OutputJSON)
	})
	if !strings.Contains(out, "Test") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalIssueDetailText(t *testing.T) {
	out := captureStdout(func() {
		LocalIssueDetail(&store.LocalIssue{ID: 1, Summary: "Test", State: "Open", SyncStatus: "local"}, OutputTable)
	})
	if !strings.Contains(out, "Test") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalCommentListJSON(t *testing.T) {
	out := captureStdout(func() {
		LocalCommentList([]store.LocalComment{{Text: "Hello"}}, OutputJSON)
	})
	if !strings.Contains(out, "Hello") {
		t.Errorf("output = %q", out)
	}
}

func TestLocalCommentListText(t *testing.T) {
	out := captureStdout(func() {
		LocalCommentList([]store.LocalComment{{Text: "Hello", Author: "john"}}, OutputTable)
	})
	if !strings.Contains(out, "Hello") {
		t.Errorf("output = %q", out)
	}
}

func TestHeader(t *testing.T) {
	s := Header("Title")
	if s == "" {
		t.Error("expected non-empty header")
	}
}

func TestErrorJSON(t *testing.T) {
	out := captureStdout(func() {
		Error(os.ErrNotExist, OutputJSON)
	})
	if !strings.Contains(out, "error") {
		t.Errorf("output = %q", out)
	}
}

func TestErrorText(t *testing.T) {
	out := captureStderr(func() {
		Error(os.ErrNotExist, OutputTable)
	})
	if !strings.Contains(out, "Error:") {
		t.Errorf("output = %q", out)
	}
}

func TestFormatTime(t *testing.T) {
	s := formatTime(0)
	if s == "" {
		t.Error("expected non-empty time")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("expected no truncation")
	}
	if len(truncate("hello world", 8)) != 8 {
		t.Errorf("expected length 8, got %d", len(truncate("hello world", 8)))
	}
}

func TestRenderMarkdown(t *testing.T) {
	out, err := RenderMarkdown("# Hello")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}
