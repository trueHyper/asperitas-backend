package logger

import (
	"log/slog"
	"os"
)

func Load() *slog.Logger {
	opts := &slog.HandlerOptions{}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
