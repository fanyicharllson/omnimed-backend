// Package logger configures the gateway's structured (JSON) logger,
// built on the standard library's slog so no extra logging dependency
// is required.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a structured JSON logger. levelName is case-insensitive
// and one of "debug", "info", "warn", "error"; unrecognized values
// fall back to "info".
func New(levelName, environment string) *slog.Logger {
	level := parseLevel(levelName)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	l := slog.New(handler).With(
		slog.String("service", "gateway"),
		slog.String("environment", environment),
	)

	slog.SetDefault(l)
	return l
}

func parseLevel(levelName string) slog.Level {
	switch strings.ToLower(levelName) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
