package api

import (
	"testing"
)

func TestFieldsBuilder(t *testing.T) {
	f := NewFields("id", "summary").
		Add("created").
		Nested("project", "id", "name").
		Nested("reporter", "login", "name")

	got := f.String()
	want := "id,summary,created,project(id,name),reporter(login,name)"
	if got != want {
		t.Errorf("fields = %q, want %q", got, want)
	}
}

func TestIssueListFields(t *testing.T) {
	f := IssueList()
	got := f.String()
	if got == "" {
		t.Error("IssueList fields should not be empty")
	}
	if !contains(got, "idReadable") {
		t.Error("IssueList should contain idReadable")
	}
	if !contains(got, "customFields") {
		t.Error("IssueList should contain customFields")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
