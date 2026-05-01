package config

import (
	"testing"
)

func TestResolveWithLocalInstance(t *testing.T) {
	global := &GlobalConfig{
		Instances: map[string]InstanceConfig{
			"work":     {URL: "https://work.youtrack.cloud"},
			"personal": {URL: "https://personal.myjetbrains.com/youtrack"},
		},
		OutputFormat: "table",
	}
	local := &LocalConfig{
		Instance:     "personal",
		Project:      "PROJ",
		DefaultQuery: "#Unresolved",
	}
	private := &LocalPrivateConfig{
		CurrentTask: "PROJ-42",
	}

	merged, err := Resolve(global, local, private)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if merged.Instance != "personal" {
		t.Errorf("expected instance personal, got %s", merged.Instance)
	}
	if merged.InstanceURL != "https://personal.myjetbrains.com/youtrack" {
		t.Errorf("unexpected URL: %s", merged.InstanceURL)
	}
	if merged.Project != "PROJ" {
		t.Errorf("expected project PROJ, got %s", merged.Project)
	}
	if merged.CurrentTask != "PROJ-42" {
		t.Errorf("expected current task PROJ-42, got %s", merged.CurrentTask)
	}
	if merged.OutputFormat != "table" {
		t.Errorf("expected output table, got %s", merged.OutputFormat)
	}
}

func TestResolveNoLocalInstance(t *testing.T) {
	global := &GlobalConfig{
		Instances: map[string]InstanceConfig{
			"work": {URL: "https://work.youtrack.cloud"},
		},
	}
	merged, err := Resolve(global, &LocalConfig{}, &LocalPrivateConfig{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if merged.Instance != "" {
		t.Errorf("expected empty instance (no default), got %s", merged.Instance)
	}
	if merged.InstanceURL != "" {
		t.Errorf("expected empty URL, got %s", merged.InstanceURL)
	}
}

func TestResolveInstanceNotFound(t *testing.T) {
	global := &GlobalConfig{
		Instances: map[string]InstanceConfig{
			"work": {URL: "https://work.youtrack.cloud"},
		},
	}
	local := &LocalConfig{Instance: "missing"}
	_, err := Resolve(global, local, nil)
	if err == nil {
		t.Fatal("expected error for missing instance")
	}
}