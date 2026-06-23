package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestNewLoggerWritesFileNotConsole is the regression for ADR-0022: when a per-session
// log file is active, logs go to the file ONLY — never to the console writer, which a
// client (e.g. the Pi extension) relays into its own UI.
func TestNewLoggerWritesFileNotConsole(t *testing.T) {
	// t.Setenv registers restoration of the original value; the Unsetenv then guarantees
	// the var is absent during the test so newLogger uses the default per-session path.
	t.Setenv("HARNESS_LOG_FILE", "")
	os.Unsetenv("HARNESS_LOG_FILE")
	t.Setenv("HARNESS_LOG_FORMAT", "")

	stateDir := t.TempDir()
	sessionID := uuid.Must(uuid.NewV7()).String()
	var console bytes.Buffer

	logger, logPath, closeLog, err := newLogger(&console, stateDir, sessionID, slog.LevelInfo)
	if err != nil {
		t.Fatalf("newLogger: %v", err)
	}
	logger.Info("hello", slog.String("k", "v"))
	closeLog()

	want := filepath.Join(stateDir, "logging", sessionID+".log")
	if logPath != want {
		t.Fatalf("logPath = %q, want %q", logPath, want)
	}
	if console.Len() != 0 {
		t.Fatalf("console got %q, want empty (logs must not reach the console)", console.String())
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("log file %q missing the line; got %q", logPath, data)
	}
	if !strings.Contains(string(data), sessionID) {
		t.Fatalf("log file missing the session tag %q; got %q", sessionID, data)
	}
}

// TestNewLoggerConsoleFallbackWhenFileDisabled covers the escape hatch:
// HARNESS_LOG_FILE="-" disables the file, so logs go to the console writer instead.
func TestNewLoggerConsoleFallbackWhenFileDisabled(t *testing.T) {
	t.Setenv("HARNESS_LOG_FILE", "-")
	t.Setenv("HARNESS_LOG_FORMAT", "")

	var console bytes.Buffer
	logger, logPath, closeLog, err := newLogger(&console, t.TempDir(), "sess", slog.LevelInfo)
	if err != nil {
		t.Fatalf("newLogger: %v", err)
	}
	logger.Info("hello")
	closeLog()

	if logPath != "" {
		t.Fatalf("logPath = %q, want empty (file disabled)", logPath)
	}
	if !strings.Contains(console.String(), "hello") {
		t.Fatalf("console missing the line; got %q", console.String())
	}
}

// TestNewLoggerJSONFormat checks HARNESS_LOG_FORMAT=json selects the JSON handler.
func TestNewLoggerJSONFormat(t *testing.T) {
	// As above: register restore, then guarantee absent so the default file path is used.
	t.Setenv("HARNESS_LOG_FILE", "")
	os.Unsetenv("HARNESS_LOG_FILE")
	t.Setenv("HARNESS_LOG_FORMAT", "json")

	stateDir := t.TempDir()
	logger, logPath, closeLog, err := newLogger(&bytes.Buffer{}, stateDir, "sess", slog.LevelInfo)
	if err != nil {
		t.Fatalf("newLogger: %v", err)
	}
	logger.Info("hello", slog.String("k", "v"))
	closeLog()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &rec); err != nil {
		t.Fatalf("log line is not JSON: %v (got %q)", err, data)
	}
	if rec["msg"] != "hello" {
		t.Fatalf("json msg = %v, want hello", rec["msg"])
	}
}
