package gem

import (
	"io"
	"log/slog"
	"os"
)

var defaultGemLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.String(a.Key, a.Value.Time().Format("2006-01-02 15:04:05"))
		}
		return a
	},
}))

func WithTextLogger(w io.Writer, level slog.Level) GemConfig {
	return WithLogger(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})))
}

func WithJSONLogger(w io.Writer, level slog.Level) GemConfig {
	return WithLogger(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})))
}
