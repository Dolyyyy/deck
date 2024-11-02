package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Init initializes the structured logger with output to a dedicated file to avoid interfering with TUI.
func Init(debug bool) *slog.Logger {
	once.Do(func() {
		var writer io.Writer = io.Discard

		home, err := os.UserHomeDir()
		if err == nil {
			logDir := filepath.Join(home, ".config", "deck")
			_ = os.MkdirAll(logDir, 0700)
			logFile := filepath.Join(logDir, "deck.log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err == nil {
				writer = f
			}
		}

		level := slog.LevelInfo
		if debug {
			level = slog.LevelDebug
		}

		handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: level,
		})

		defaultLogger = slog.New(handler)
		slog.SetDefault(defaultLogger)
	})

	return defaultLogger
}

// Get returns the initialized default logger or a discard logger.
func Get() *slog.Logger {
	if defaultLogger == nil {
		return Init(false)
	}
	return defaultLogger
}
