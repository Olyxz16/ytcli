package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProviderConfig holds the configuration for a remote provider.
type ProviderConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// LocalSchema defines states, priorities, and other schema for local issues.
type LocalSchema struct {
	States          []string `yaml:"states,omitempty"`
	Priorities      []string `yaml:"priorities,omitempty"`
	DefaultState    string   `yaml:"default_state,omitempty"`
	DefaultPriority string   `yaml:"default_priority,omitempty"`
	DoneStates      []string `yaml:"done_states,omitempty"`
}

// LocalConfig holds the commitable per-directory configuration.
type LocalConfig struct {
	Provider     ProviderConfig `yaml:"provider,omitempty"`
	Instance     string         `yaml:"instance,omitempty"`
	InstanceURL  string         `yaml:"instance_url,omitempty"`
	Project      string         `yaml:"project,omitempty"`
	DefaultQuery string         `yaml:"default_query,omitempty"`
	WikiDir      string         `yaml:"wiki_dir,omitempty"`
	Local        LocalSchema    `yaml:"local,omitempty"`
}

// LocalPrivateConfig holds the gitignored per-directory configuration.
type LocalPrivateConfig struct {
	CurrentTask string `yaml:"current_task,omitempty"`
}

// MergedConfig is the effective configuration after resolution.
type MergedConfig struct {
	ProviderName string
	ProviderURL  string
	Project      string
	DefaultQuery string
	CurrentTask  string
	OutputFormat string
	WikiDir      string
}

// CredentialsPath returns the credentials file path.
func CredentialsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "tkt", "credentials.yml")
}

// findLocalConfig searches for local config files starting at dir and walking up.
func findLocalConfig(dir string, filename string) (string, error) {
	for {
		candidate := filepath.Join(dir, filename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// LoadLocal loads the commitable local config from the current directory tree.
func LoadLocal() (*LocalConfig, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	path, err := findLocalConfig(wd, ".tktrc.yml")
	if err != nil {
		return &LocalConfig{}, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var cfg LocalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("parse local config: %w", err)
	}
	// Backwards compatibility: instance_url maps to provider.url
	if cfg.Provider.URL == "" && cfg.InstanceURL != "" {
		cfg.Provider.URL = cfg.InstanceURL
	}
	return &cfg, path, nil
}

// LoadLocalPrivate loads the gitignored local config from the current directory tree.
func LoadLocalPrivate() (*LocalPrivateConfig, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	path, err := findLocalConfig(wd, ".tkt.local.yml")
	if err != nil {
		return &LocalPrivateConfig{}, "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var cfg LocalPrivateConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("parse local private config: %w", err)
	}
	return &cfg, path, nil
}

// SaveLocal persists the commitable local config.
func SaveLocal(cfg *LocalConfig, dir string) error {
	path := filepath.Join(dir, ".tktrc.yml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SaveLocalPrivate persists the gitignored local config.
func SaveLocalPrivate(cfg *LocalPrivateConfig, dir string) error {
	path := filepath.Join(dir, ".tkt.local.yml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// NormalizeURL trims trailing slashes from a URL for comparison.
func NormalizeURL(u string) string {
	return strings.TrimSuffix(u, "/")
}

// Resolve merges local and private configs into a single effective config.
// The project config is the source of truth for provider connection details.
func Resolve(local *LocalConfig, private *LocalPrivateConfig) (*MergedConfig, error) {
	m := &MergedConfig{
		OutputFormat: "table",
	}

	if local != nil {
		m.ProviderName = local.Provider.Name
		m.ProviderURL = strings.TrimSuffix(local.Provider.URL, "/")
		m.Project = local.Project
		m.DefaultQuery = local.DefaultQuery
		m.WikiDir = local.WikiDir
		// Backwards compatibility
		if m.ProviderURL == "" && local.InstanceURL != "" {
			m.ProviderURL = strings.TrimSuffix(local.InstanceURL, "/")
		}
		if m.ProviderName == "" && local.Instance != "" {
			m.ProviderName = local.Instance
		}
	}

	if private != nil {
		if private.CurrentTask != "" {
			m.CurrentTask = private.CurrentTask
		}
	}

	return m, nil
}