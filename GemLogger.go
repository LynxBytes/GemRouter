package gemrouter

import (
	"io"
	"log/slog"
	"os"
)

var defaultGemLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func WithTextLogger(w io.Writer, level slog.Level) GemConfig {
	return WithLogger(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})))
}

func WithJSONLogger(w io.Writer, level slog.Level) GemConfig {
	return WithLogger(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})))
}
