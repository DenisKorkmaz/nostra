package logging

import (
	"log/slog"
	"os"
)

func New(production bool) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, options)
	if production {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}

	return slog.New(handler)
}
