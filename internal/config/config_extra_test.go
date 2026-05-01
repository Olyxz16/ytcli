package config

import (
	"testing"
)

func TestResolveDefaults(t *testing.T) {
	global := &GlobalConfig{
		Instances: map[string]InstanceConfig{
			"work": {URL: "https://work.youtrack.cloud"},
		},
		OutputFormat: "table",
	}
	merged, err := Resolve(global, nil, nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.Instance != "" {
		t.Errorf("expected empty instance (no default), got %q", merged.Instance)
	}
	if merged.InstanceURL != "" {
		t.Errorf("expected empty URL, got %q", merged.InstanceURL)
	}
}

func TestResolveLocalOverrides(t *testing.T) {
	global := &GlobalConfig{
		Instances: map[string]InstanceConfig{
			"work":     {URL: "https://work.youtrack.cloud"},
			"personal": {URL: "https://personal.myjetbrains.com/youtrack"},
		},
	}
	local := &LocalConfig{Instance: "personal", Project: "PROJ"}
	private := &LocalPrivateConfig{CurrentTask: "PROJ-1"}

	merged, err := Resolve(global, local, private)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.Instance != "personal" {
		t.Errorf("instance = %q", merged.Instance)
	}
	if merged.Project != "PROJ" {
		t.Errorf("project = %q", merged.Project)
	}
	if merged.CurrentTask != "PROJ-1" {
		t.Errorf("current task = %q", merged.CurrentTask)
	}
}

func TestResolveInstanceNotInGlobal(t *testing.T) {
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

func TestResolveTrimsTrailingSlash(t *testing.T) {
	global := &GlobalConfig{
		Instances: map[string]InstanceConfig{
			"work": {URL: "https://work.youtrack.cloud/"},
		},
	}
	local := &LocalConfig{Instance: "work"}
	merged, err := Resolve(global, local, nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.InstanceURL != "https://work.youtrack.cloud" {
		t.Errorf("url = %q", merged.InstanceURL)
	}
}

func TestResolveEmptyConfigs(t *testing.T) {
	global := &GlobalConfig{
		Instances:    map[string]InstanceConfig{},
		OutputFormat: "table",
	}
	merged, err := Resolve(global, &LocalConfig{}, &LocalPrivateConfig{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if merged.InstanceURL != "" {
		t.Errorf("url = %q, want empty", merged.InstanceURL)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestCredentialsPath(t *testing.T) {
	path := CredentialsPath()
	if path == "" {
		t.Error("expected non-empty path")
	}
}