package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"harness-workspace/services/harness/internal/infra/config"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadMissingFilesDefaultsEmpty(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Logging.Level != "" {
		t.Fatalf("Level = %q, want empty (caller applies default)", cfg.Logging.Level)
	}
}

func TestLoadUserOverridesSystem(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "00-system.yaml", "version: 1\nlogging:\n  level: info\n")

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load (system only): %v", err)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("system Level = %q, want info", cfg.Logging.Level)
	}

	writeFile(t, dir, "config.yaml", "logging:\n  level: debug\n")
	cfg, err = config.Load(dir)
	if err != nil {
		t.Fatalf("Load (with overlay): %v", err)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("overlaid Level = %q, want debug (user overrides system)", cfg.Logging.Level)
	}
}

func TestLoadMalformedIsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "00-system.yaml", "logging: [unclosed\n")
	if _, err := config.Load(dir); err == nil {
		t.Fatal("Load = nil error, want a parse error for malformed yaml")
	}
}
