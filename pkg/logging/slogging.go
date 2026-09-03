package logging

import (
	"log/slog"
	"os"
)

func CreateRootSlogger(productionMode bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	if productionMode {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
