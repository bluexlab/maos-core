package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	maoscore "gitlab.com/navyx/ai/maos/maos-core"
)

func main() {
	// Load environment variables from .env file if exists
	if _, err := os.Stat(".env"); err == nil {
		godotenv.Load(".env")
	}

	logger := initLogger()
	app := &App{logger}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		app.Migrate()
	} else {
		app.Run()
	}
}

func initLogger() *slog.Logger {
	// Get log level from environment variable, default to "info"
	logLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	level := slog.LevelInfo
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// Choose handler based on environment
	var handler slog.Handler
	if os.Getenv("DEV") != "" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	// Create logger with additional context
	logger := slog.New(handler).With(
		"service", maoscore.ServiceName,
		"version", maoscore.GetVersion(),
	)

	// Set as default logger
	slog.SetDefault(logger)

	return logger
}
