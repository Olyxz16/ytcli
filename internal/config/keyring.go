package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

const keyringService = "tkt"

// SetToken stores a token for the given provider in the OS keyring.
func SetToken(provider, token string) error {
	return keyring.Set(keyringService, provider, token)
}

// GetToken retrieves a token for the given provider from the OS keyring.
// Falls back to credentials file if keyring is unavailable.
func GetToken(provider string) (string, error) {
	token, err := keyring.Get(keyringService, provider)
	if err == nil {
		return token, nil
	}

	// Fallback to file-based credentials
	return getTokenFromFile(provider)
}

// DeleteToken removes a token for the given provider.
func DeleteToken(provider string) error {
	_ = keyring.Delete(keyringService, provider)
	_, _ = deleteTokenFromFile(provider)
	return nil
}

type credentialsFile struct {
	Tokens map[string]string `yaml:"tokens"`
}

func getTokenFromFile(provider string) (string, error) {
	path := CredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no token found for provider %q", provider)
		}
		return "", err
	}
	var creds credentialsFile
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return "", err
	}
	token, ok := creds.Tokens[provider]
	if !ok {
		return "", fmt.Errorf("no token found for provider %q", provider)
	}
	return token, nil
}

func deleteTokenFromFile(provider string) (bool, error) {
	path := CredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var creds credentialsFile
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return false, err
	}
	if _, ok := creds.Tokens[provider]; !ok {
		return false, nil
	}
	delete(creds.Tokens, provider)
	out, err := yaml.Marshal(&creds)
	if err != nil {
		return false, err
	}
	perm := os.FileMode(0600)
	if runtime.GOOS == "windows" {
		perm = 0600
	}
	return true, os.WriteFile(path, out, perm)
}

// SetTokenFile stores a token in the fallback credentials file.
func SetTokenFile(provider, token string) error {
	path := CredentialsPath()
	var creds credentialsFile
	data, err := os.ReadFile(path)
	if err == nil {
		_ = yaml.Unmarshal(data, &creds)
	}
	if creds.Tokens == nil {
		creds.Tokens = make(map[string]string)
	}
	creds.Tokens[provider] = token
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := yaml.Marshal(&creds)
	if err != nil {
		return err
	}
	perm := os.FileMode(0600)
	if runtime.GOOS == "windows" {
		perm = 0600
	}
	return os.WriteFile(path, out, perm)
}