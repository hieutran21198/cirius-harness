// Package slogx provides small helpers for constructing and carrying
// structured loggers built on the standard library's log/slog.
package slogx

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Format selects the slog handler used by New.
type Format string

const (
	// FormatText emits human-readable key=value lines (slog.TextHandler).
	FormatText Format = "text"
	// FormatJSON emits one JSON object per record (slog.JSONHandler).
	FormatJSON Format = "json"
)

// ErrInvalidLevel is returned by ParseLevel for an unrecognised level name.
var ErrInvalidLevel = errors.New("slogx: invalid level")

// New builds a *slog.Logger writing to w at the given level using the chosen
// format. An unknown format falls back to FormatText.
func New(w io.Writer, level slog.Level, format Format) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	switch format {
	case FormatJSON:
		h = slog.NewJSONHandler(w, opts)
	default:
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}

// ParseLevel maps a case-insensitive name (debug, info, warn, error) to a
// slog.Level. It returns ErrInvalidLevel for anything else.
func ParseLevel(name string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidLevel, name)
	}
}
