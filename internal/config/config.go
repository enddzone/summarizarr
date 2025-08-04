package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	LogLevel slog.Level
}

// New creates a new Config from environment variables.
func New() *Config {
	return &Config{
		LogLevel: parseLogLevel(os.Getenv("LOG_LEVEL")),
	}
}

func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}