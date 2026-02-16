// Package logging configures colored structured logging with tint.
// Copy this file to any Go project that uses log/slog.
//
// Usage:
//
//	logging.Setup()                          // INFO level, from LOG_LEVEL env
//	logging.SetupWithLevel(slog.LevelDebug)  // explicit level override
//
// Environment variables:
//
//	LOG_LEVEL: debug, info, warn, error (default: info)
package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// Setup configures colored logging at the level specified by LOG_LEVEL env var
// (default: INFO).
func Setup() {
	SetupWithLevel(levelFromEnv())
}

// SetupWithLevel configures colored logging at the given level.
func SetupWithLevel(level slog.Level) {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	))
}

func levelFromEnv() slog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
