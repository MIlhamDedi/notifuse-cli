package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const EnvConfigPath = "NOTIFUSE_CONFIG"

type File struct {
	DefaultProfile string             `yaml:"default_profile,omitempty"`
	Profiles       map[string]Profile `yaml:"profiles,omitempty"`
}

type Profile struct {
	Endpoint              string   `yaml:"endpoint"`
	WorkspaceID           string   `yaml:"workspace_id"`
	APIKeyRef             string   `yaml:"api_key_ref"`
	MaxRecipients         int      `yaml:"max_recipients,omitempty"`
	AllowedTestRecipients []string `yaml:"allowed_test_recipients,omitempty"`
}

func DefaultPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(EnvConfigPath)); configured != "" {
		return expandHome(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "notifuse-cli", "config.yaml"), nil
}

func Load(path string) (File, error) {
	var file File
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{Profiles: map[string]Profile{}}, nil
		}
		return file, err
	}
	if err := yaml.Unmarshal(body, &file); err != nil {
		return file, err
	}
	if file.Profiles == nil {
		file.Profiles = map[string]Profile{}
	}
	return file, nil
}

func Save(path string, file File) error {
	if file.Profiles == nil {
		file.Profiles = map[string]Profile{}
	}
	body, err := yaml.Marshal(file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

func (f File) Resolve(name string) (string, Profile, error) {
	if name == "" {
		name = f.DefaultProfile
	}
	if name == "" {
		return "", Profile{}, fmt.Errorf("profile is required; set default_profile or pass --profile")
	}
	profile, ok := f.Profiles[name]
	if !ok {
		return "", Profile{}, fmt.Errorf("unknown profile %q", name)
	}
	if strings.TrimSpace(profile.Endpoint) == "" {
		return "", Profile{}, fmt.Errorf("profile %q has empty endpoint", name)
	}
	if strings.TrimSpace(profile.WorkspaceID) == "" {
		return "", Profile{}, fmt.Errorf("profile %q has empty workspace_id", name)
	}
	if strings.TrimSpace(profile.APIKeyRef) == "" {
		return "", Profile{}, fmt.Errorf("profile %q has empty api_key_ref", name)
	}
	return name, profile, nil
}

func (p Profile) AllowsRecipient(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, allowed := range p.AllowedTestRecipients {
		if email == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func expandHome(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}
