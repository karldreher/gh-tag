package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds user-level persistent settings for gh-tag.
// It is persisted as JSON at ~/.gh-tag/config.json.
type Config struct {
	// Prefix is the tag prefix used when creating and discovering tags.
	// When empty, callers should default to "v".
	// Set via `gh tag prefix --edit`.
	Prefix string `json:"prefix"`
	// OverwriteConfirmed records that the user has acknowledged the risk of
	// --overwrite at least once. When true, subsequent --confirm --overwrite
	// invocations skip the one-time warning.
	OverwriteConfirmed bool `json:"overwrite_confirmed"`
}

// EffectivePrefix returns the tag prefix from config, defaulting to "v".
func EffectivePrefix() (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	if cfg.Prefix == "" {
		return "v", nil
	}
	return cfg.Prefix, nil
}

// ConfigPath returns the absolute path to the config file (~/.gh-tag/config.json).
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".gh-tag", "config.json"), nil
}

// LoadConfig reads the config file and returns the parsed Config.
// If the file does not exist, it returns a zero-value Config and no error.
func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes cfg to the config file, creating the directory if needed.
func SaveConfig(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
