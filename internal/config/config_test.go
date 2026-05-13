package config

import (
	"testing"
)

func TestResolveWithProvider(t *testing.T) {
	local := &LocalConfig{
		Provider: ProviderConfig{
			Name: "youtrack",
			URL:  "https://work.youtrack.cloud",
		},
		Project:      "PROJ",
		DefaultQuery: "#Unresolved",
	}
	private := &LocalPrivateConfig{
		CurrentTask: "PROJ-42",
	}

	merged, err := Resolve(local, private)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if merged.ProviderName != "youtrack" {
		t.Errorf("expected provider name youtrack, got %s", merged.ProviderName)
	}
	if merged.ProviderURL != "https://work.youtrack.cloud" {
		t.Errorf("unexpected URL: %s", merged.ProviderURL)
	}
	if merged.Project != "PROJ" {
		t.Errorf("expected project PROJ, got %s", merged.Project)
	}
	if merged.CurrentTask != "PROJ-42" {
		t.Errorf("expected current task PROJ-42, got %s", merged.CurrentTask)
	}
}

func TestResolveNoProvider(t *testing.T) {
	merged, err := Resolve(&LocalConfig{}, &LocalPrivateConfig{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if merged.ProviderName != "" {
		t.Errorf("expected empty provider name, got %s", merged.ProviderName)
	}
	if merged.ProviderURL != "" {
		t.Errorf("expected empty URL, got %s", merged.ProviderURL)
	}
}

func TestResolveBackwardsCompatInstance(t *testing.T) {
	local := &LocalConfig{
		Instance:    "work",
		InstanceURL: "https://work.youtrack.cloud",
	}
	merged, err := Resolve(local, nil)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if merged.ProviderName != "work" {
		t.Errorf("expected provider name work, got %s", merged.ProviderName)
	}
	if merged.ProviderURL != "https://work.youtrack.cloud" {
		t.Errorf("unexpected URL: %s", merged.ProviderURL)
	}
}