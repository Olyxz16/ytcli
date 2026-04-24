package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// InstanceConfig holds the configuration for a single YouTrack instance.
type InstanceConfig struct {
	URL string `yaml:"url"`
}

// GlobalConfig holds the user's global configuration.
type GlobalConfig struct {
	Instances       map[string]InstanceConfig `yaml:"instances"`
	DefaultInstance string                    `yaml:"default_instance"`
	OutputFormat    string                    `yaml:"output_format"`
}

// LocalConfig holds the commitable per-directory configuration.
type LocalConfig struct {
	Instance     string `yaml:"instance,omitempty"`
	Project      string `yaml:"project,omitempty"`
	DefaultQuery string `yaml:"default_query,omitempty"`
}

// LocalPrivateConfig holds the gitignored per-directory configuration.
type LocalPrivateConfig struct {
	CurrentTask string `yaml:"current_task,omitempty"`
}

// MergedConfig is the effective configuration after resolution.
type MergedConfig struct {
	Instance     string
	InstanceURL  string
	Project      string
	DefaultQuery string
	CurrentTask  string
	OutputFormat string
}

// DefaultGlobalConfig returns a default global configuration.
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Instances:       make(map[string]InstanceConfig),
		OutputFormat:    "table",
	}
}

// ConfigPath returns the path to the global config file.
func ConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "ytcli", "config.yml")
}

// CredentialsPath returns the fallback credentials file path.
func CredentialsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "ytcli", "credentials.yml")
}

// LoadGlobal loads the global configuration from disk.
func LoadGlobal() (*GlobalConfig, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultGlobalConfig(), nil
		}
		return nil, err
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse global config: %w", err)
	}
	if cfg.Instances == nil {
		cfg.Instances = make(map[string]InstanceConfig)
	}
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "table"
	}
	return &cfg, nil
}

// SaveGlobal persists the global configuration to disk.
func SaveGlobal(cfg *GlobalConfig) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
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
	path, err := findLocalConfig(wd, ".ytcli.yml")
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
	return &cfg, path, nil
}

// LoadLocalPrivate loads the gitignored local config from the current directory tree.
func LoadLocalPrivate() (*LocalPrivateConfig, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	path, err := findLocalConfig(wd, ".ytcli.local.yml")
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
	path := filepath.Join(dir, ".ytcli.yml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SaveLocalPrivate persists the gitignored local config.
func SaveLocalPrivate(cfg *LocalPrivateConfig, dir string) error {
	path := filepath.Join(dir, ".ytcli.local.yml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Resolve merges global, local, and private configs into a single effective config.
func Resolve(global *GlobalConfig, local *LocalConfig, private *LocalPrivateConfig) (*MergedConfig, error) {
	m := &MergedConfig{
		Instance:     global.DefaultInstance,
		OutputFormat: global.OutputFormat,
	}

	if local != nil {
		if local.Instance != "" {
			m.Instance = local.Instance
		}
		if local.Project != "" {
			m.Project = local.Project
		}
		if local.DefaultQuery != "" {
			m.DefaultQuery = local.DefaultQuery
		}
	}

	if private != nil {
		if private.CurrentTask != "" {
			m.CurrentTask = private.CurrentTask
		}
	}

	// Resolve instance URL
	if m.Instance != "" {
		inst, ok := global.Instances[m.Instance]
		if !ok {
			return nil, fmt.Errorf("instance %q not found in global config", m.Instance)
		}
		m.InstanceURL = strings.TrimSuffix(inst.URL, "/")
	}

	return m, nil
}
