// Package config reads the workspace configuration files the harness needs at
// startup. Today it reads only the logging settings; the full agent/model resolver
// (deep-merging the user overlay onto the system base, ADR-0011) is a separate,
// still-deferred milestone. This is the first real slice of that loader: it reads the
// same two files — the system base 00-system.yaml then the user overlay config.yaml —
// from the .cirius-harness directory, with the overlay overriding the base.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the subset of the workspace configuration the binary consumes at startup.
type Config struct {
	Logging Logging `yaml:"logging"`
}

// Logging holds the logging settings. Level is the slog level name
// (debug|info|warn|error); empty means "unset" and the caller applies its default.
type Logging struct {
	Level string `yaml:"level"`
}

// systemFile is the embedded-defaults file; userFile is the optional user overlay.
const (
	systemFile = "00-system.yaml"
	userFile   = "config.yaml"
)

// Load reads the system config then the user overlay from dir (the .cirius-harness
// directory), returning the merged result. A missing file is not an error — defaults
// stand and the user overlay is optional; a present-but-malformed file is. Values set
// by the user overlay override the system base (later read wins per field).
func Load(dir string) (Config, error) {
	var cfg Config
	for _, name := range []string{systemFile, userFile} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Config{}, fmt.Errorf("config: read %s: %w", name, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("config: parse %s: %w", name, err)
		}
	}
	return cfg, nil
}
