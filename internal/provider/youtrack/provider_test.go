package youtrack

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/provider"
)

func TestWrapErrorNil(t *testing.T) {
	if err := wrapError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestWrapErrorAuth(t *testing.T) {
	apiErr := &APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}
	err := wrapError(apiErr)

	var authErr *provider.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
	if authErr.Provider != "youtrack" {
		t.Errorf("Provider = %q", authErr.Provider)
	}
}

func TestWrapErrorNotFound(t *testing.T) {
	apiErr := &APIError{StatusCode: http.StatusNotFound, Message: "not found"}
	err := wrapError(apiErr)

	var notFoundErr *provider.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	if notFoundErr.Provider != "youtrack" {
		t.Errorf("Provider = %q", notFoundErr.Provider)
	}
}

func TestWrapErrorValidation(t *testing.T) {
	apiErr := &APIError{StatusCode: http.StatusBadRequest, Message: "bad request"}
	err := wrapError(apiErr)

	var valErr *provider.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if valErr.Provider != "youtrack" {
		t.Errorf("Provider = %q", valErr.Provider)
	}
}

func TestWrapErrorNetwork(t *testing.T) {
	apiErr := &APIError{StatusCode: http.StatusInternalServerError, Message: "internal error"}
	err := wrapError(apiErr)

	var netErr *provider.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected NetworkError, got %T: %v", err, err)
	}
	if netErr.Provider != "youtrack" {
		t.Errorf("Provider = %q", netErr.Provider)
	}
	if !netErr.Retry {
		t.Error("expected Retry = true")
	}
}

func TestProviderName(t *testing.T) {
	p := NewProvider("http://localhost", "token")
	if p.Name() != "youtrack" {
		t.Errorf("Name() = %q, want youtrack", p.Name())
	}
}

func TestProviderPingSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/users/me" {
			w.Write([]byte(`{"login":"john","name":"John"}`))
		}
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	if err := p.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestProviderPingAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`unauthorized`))
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	err := p.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var authErr *provider.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
}

func TestProviderAuthDelegatesToPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"login":"john"}`))
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	if err := p.Auth(context.Background()); err != nil {
		t.Fatalf("Auth: %v", err)
	}
}

func TestProviderFetchIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"2-1","idReadable":"PROJ-1","summary":"Test","created":1000,"updated":2000}]`))
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	issues, err := p.FetchIssues(context.Background(), provider.Query{Project: "PROJ"}, 10, 0)
	if err != nil {
		t.Fatalf("FetchIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("len = %d, want 1", len(issues))
	}
	if issues[0].ID != "PROJ-1" {
		t.Errorf("ID = %q", issues[0].ID)
	}
}

func TestProviderFetchIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"2-42","idReadable":"PROJ-1","summary":"Test issue","created":1000,"updated":2000}`))
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	issue, err := p.FetchIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("FetchIssue: %v", err)
	}
	if issue.ID != "PROJ-1" {
		t.Errorf("ID = %q", issue.ID)
	}
	if issue.Summary != "Test issue" {
		t.Errorf("Summary = %q", issue.Summary)
	}
}

func TestProviderCreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"2-1","idReadable":"PROJ-1","summary":"New issue","created":1000,"updated":1000}`))
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	issue := model.Issue{Summary: "New issue", Project: &model.Project{ID: "p1"}}
	created, err := p.CreateIssue(context.Background(), issue)
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if created.ID != "PROJ-1" {
		t.Errorf("ID = %q", created.ID)
	}
}

func TestProviderDeleteIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewProvider(srv.URL, "token")
	if err := p.DeleteIssue(context.Background(), "PROJ-1"); err != nil {
		t.Fatalf("DeleteIssue: %v", err)
	}
}

func TestProviderFetchSchemaReturnsNil(t *testing.T) {
	p := NewProvider("http://localhost", "token")
	schema, err := p.FetchSchema(context.Background())
	if err != nil {
		t.Fatalf("FetchSchema: %v", err)
	}
	if schema != nil {
		t.Errorf("expected nil schema, got %v", schema)
	}
}

func TestIsAuthError(t *testing.T) {
	err := &APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}
	if !IsAuthError(err) {
		t.Error("expected IsAuthError = true")
	}
	if IsNotFoundError(err) {
		t.Error("expected IsNotFoundError = false")
	}
}

func TestIsNotFoundError(t *testing.T) {
	err := &APIError{StatusCode: http.StatusNotFound, Message: "not found"}
	if !IsNotFoundError(err) {
		t.Error("expected IsNotFoundError = true")
	}
	if IsAuthError(err) {
		t.Error("expected IsAuthError = false")
	}
}

func TestIsValidationError(t *testing.T) {
	err := &APIError{StatusCode: http.StatusBadRequest, Message: "bad request"}
	if !IsValidationError(err) {
		t.Error("expected IsValidationError = true")
	}
	if IsAuthError(err) {
		t.Error("expected IsAuthError = false")
	}
}

func TestIsForbiddenError(t *testing.T) {
	err := &APIError{StatusCode: http.StatusForbidden, Message: "forbidden"}
	if !IsForbiddenError(err) {
		t.Error("expected IsForbiddenError = true")
	}
}

func TestNewProviderWithClient(t *testing.T) {
	client := NewClient("http://localhost", "token")
	p := NewProviderWithClient(client)
	if p == nil {
		t.Fatal("expected provider")
	}
	if p.Client() != client {
		t.Error("Client() should return the provided client")
	}
	if p.Name() != "youtrack" {
		t.Errorf("Name() = %q, want youtrack", p.Name())
	}
}