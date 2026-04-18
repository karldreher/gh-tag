package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigPath verifies that the config file path is rooted under HOME and
// ends with .gh-tag/config.json.
func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() failed: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".gh-tag", "config.json")) {
		t.Errorf("unexpected config path: %s", path)
	}
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("config path %s is not under HOME %s", path, tmpDir)
	}
}

// TestLoadConfig_FileNotExist verifies that a missing config file returns a
// zero-value Config without error.
func TestLoadConfig_FileNotExist(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error for missing file: %v", err)
	}
	if cfg.Prefix != "" {
		t.Errorf("expected empty Prefix, got %q", cfg.Prefix)
	}
}

// TestLoadConfig_Valid verifies that a well-formed JSON config file is parsed
// into the expected Config fields.
func TestLoadConfig_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".gh-tag")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(Config{Prefix: "release-"})
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if cfg.Prefix != "release-" {
		t.Errorf("expected Prefix %q, got %q", "release-", cfg.Prefix)
	}
}

// TestLoadConfig_MalformedJSON verifies that unparseable JSON returns an error.
func TestLoadConfig_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".gh-tag")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() expected error for malformed JSON, got nil")
	}
}

// TestLoadConfig_WrongFieldType verifies that a JSON type mismatch (e.g. number
// where a string is expected) returns a parsing error.
func TestLoadConfig_WrongFieldType(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".gh-tag")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	// prefix must be a string; passing a number violates the schema.
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"prefix": 123}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() expected error for wrong field type, got nil")
	}
	if !strings.Contains(err.Error(), "parsing config file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestLoadConfig_ReadError verifies that an I/O failure (config path is a
// directory) returns an error with the expected prefix.
func TestLoadConfig_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config.json as a directory — ReadFile will fail with a
	// non-IsNotExist error, exercising the "reading config file" branch.
	configPath := filepath.Join(tmpDir, ".gh-tag", "config.json")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error when config.json is a directory")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSaveConfig verifies that a Config written with SaveConfig round-trips
// correctly through LoadConfig.
func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := Config{Prefix: "myprefix-"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() after SaveConfig() failed: %v", err)
	}
	if loaded.Prefix != cfg.Prefix {
		t.Errorf("round-trip mismatch: got %q, want %q", loaded.Prefix, cfg.Prefix)
	}
}

// TestLoadConfig_MissingOverwriteConfirmed verifies that a config file that
// omits overwrite_confirmed deserializes the field to false.
func TestLoadConfig_MissingOverwriteConfirmed(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".gh-tag")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Config file written without overwrite_confirmed — must deserialize to false.
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"prefix":"v"}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if cfg.OverwriteConfirmed != false {
		t.Errorf("expected OverwriteConfirmed=false for config without field, got true")
	}
}

// TestSaveConfig_WriteError verifies that an I/O failure during write (config
// path is a directory) returns an error with the expected prefix.
func TestSaveConfig_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config.json as a directory so MkdirAll succeeds but WriteFile fails.
	configPath := filepath.Join(tmpDir, ".gh-tag", "config.json")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatal(err)
	}

	err := SaveConfig(Config{Prefix: "v"})
	if err == nil {
		t.Error("expected error when config.json is a directory")
	}
	if !strings.Contains(err.Error(), "writing config file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSaveConfig_CreatesDirectory verifies that SaveConfig creates the
// ~/.gh-tag directory when it does not yet exist.
func TestSaveConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Confirm ~/.gh-tag does not exist yet
	configDir := filepath.Join(tmpDir, ".gh-tag")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal("config directory should not exist before SaveConfig")
	}

	if err := SaveConfig(Config{Prefix: "v"}); err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create the config directory")
	}
}

// TestEffectivePrefix covers the default fallback to "v", an explicitly
// configured prefix, load errors, and an explicitly empty prefix stored in
// the config file.
func TestEffectivePrefix(t *testing.T) {
	t.Run("no config returns v", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		got, err := EffectivePrefix()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "v" {
			t.Errorf("got %q, want %q", got, "v")
		}
	})

	t.Run("config with prefix returns it", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		if err := os.MkdirAll(filepath.Join(tmpDir, ".gh-tag"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".gh-tag", "config.json"),
			[]byte(`{"prefix":"release-"}`), 0644,
		); err != nil {
			t.Fatal(err)
		}
		got, err := EffectivePrefix()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "release-" {
			t.Errorf("got %q, want %q", got, "release-")
		}
	})

	t.Run("load error returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		// config.json as a directory forces LoadConfig to fail
		if err := os.MkdirAll(filepath.Join(tmpDir, ".gh-tag", "config.json"), 0755); err != nil {
			t.Fatal(err)
		}
		_, err := EffectivePrefix()
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("config with empty prefix returns v", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		if err := os.MkdirAll(filepath.Join(tmpDir, ".gh-tag"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".gh-tag", "config.json"),
			[]byte(`{"prefix":""}`), 0644,
		); err != nil {
			t.Fatal(err)
		}
		got, err := EffectivePrefix()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "v" {
			t.Errorf("got %q, want %q", got, "v")
		}
	})
}
