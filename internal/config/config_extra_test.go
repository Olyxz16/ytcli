package config

import (
	"testing"
)

func TestResolveDefaults(t *testing.T) {
	merged, err := Resolve(nil, nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.ProviderName != "" {
		t.Errorf("expected empty provider name, got %q", merged.ProviderName)
	}
	if merged.ProviderURL != "" {
		t.Errorf("expected empty URL, got %q", merged.ProviderURL)
	}
}

func TestResolveLocalProvider(t *testing.T) {
	local := &LocalConfig{
		Provider: ProviderConfig{
			Name: "youtrack",
			URL:  "https://work.youtrack.cloud",
		},
		Project: "PROJ",
	}
	private := &LocalPrivateConfig{CurrentTask: "PROJ-1"}

	merged, err := Resolve(local, private)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.ProviderName != "youtrack" {
		t.Errorf("provider name = %q", merged.ProviderName)
	}
	if merged.Project != "PROJ" {
		t.Errorf("project = %q", merged.Project)
	}
	if merged.CurrentTask != "PROJ-1" {
		t.Errorf("current task = %q", merged.CurrentTask)
	}
}

func TestResolveTrimsTrailingSlash(t *testing.T) {
	local := &LocalConfig{
		Provider: ProviderConfig{
			Name: "work",
			URL:  "https://work.youtrack.cloud/",
		},
	}
	merged, err := Resolve(local, nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.ProviderURL != "https://work.youtrack.cloud" {
		t.Errorf("url = %q", merged.ProviderURL)
	}
}

func TestResolveBackwardsCompat(t *testing.T) {
	local := &LocalConfig{
		Instance:    "work",
		InstanceURL: "https://work.youtrack.cloud",
	}
	merged, err := Resolve(local, nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.ProviderName != "work" {
		t.Errorf("provider name = %q, want work", merged.ProviderName)
	}
	if merged.ProviderURL != "https://work.youtrack.cloud" {
		t.Errorf("url = %q", merged.ProviderURL)
	}
}

func TestCredentialsPath(t *testing.T) {
	path := CredentialsPath()
	if path == "" {
		t.Error("expected non-empty path")
	}
}