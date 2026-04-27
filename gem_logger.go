package gem

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	"gopkg.in/natefinch/lumberjack.v2"
)

type rawModeWriter struct {
	w   io.Writer
	raw atomic.Bool
}

func (rw *rawModeWriter) Write(p []byte) (int, error) {
	if rw.raw.Load() {
		modified := bytes.ReplaceAll(p, []byte{'\n'}, []byte{'\r', '\n'})
		if _, err := rw.w.Write(modified); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	return rw.w.Write(p)
}

type teeHandler struct {
	console slog.Handler
	file    slog.Handler
}

func (h *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.console.Enabled(ctx, level) || h.file.Enabled(ctx, level)
}

func (h *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.console.Handle(ctx, r.Clone()); err != nil {
		return err
	}
	return h.file.Handle(ctx, r.Clone())
}

func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{console: h.console.WithAttrs(attrs), file: h.file.WithAttrs(attrs)}
}

func (h *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{console: h.console.WithGroup(name), file: h.file.WithGroup(name)}
}

type LogRotateConfig struct {
	Path       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

func newLumberjack(cfg LogRotateConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}

func newDefaultLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(a.Key, a.Value.Time().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	}))
}

func WithTextLogger(w io.Writer, level slog.Level) GemConfig {
	return WithLogger(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})))
}

func WithJSONLogger(w io.Writer, level slog.Level) GemConfig {
	return WithLogger(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})))
}

func openLogFile(path string) *os.File {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Warn("gem: could not open log file, falling back to stdout", "path", path, "err", err)
		return os.Stdout
	}
	return f
}

func WithTextFileLogger(path string, level slog.Level) GemConfig {
	f := openLogFile(path)
	return func(r *GemRouter) {
		WithTextLogger(f, level)(r)
		if f != os.Stdout {
			r.logCloser = f
		}
	}
}

func WithJSONFileLogger(path string, level slog.Level) GemConfig {
	f := openLogFile(path)
	return func(r *GemRouter) {
		WithJSONLogger(f, level)(r)
		if f != os.Stdout {
			r.logCloser = f
		}
	}
}

func WithTextTeeLogger(path string, level slog.Level) GemConfig {
	f := openLogFile(path)
	return func(r *GemRouter) {
		WithTextLogger(io.MultiWriter(os.Stdout, f), level)(r)
		if f != os.Stdout {
			r.logCloser = f
		}
	}
}

func WithJSONTeeLogger(path string, level slog.Level) GemConfig {
	f := openLogFile(path)
	return func(r *GemRouter) {
		WithJSONLogger(io.MultiWriter(os.Stdout, f), level)(r)
		if f != os.Stdout {
			r.logCloser = f
		}
	}
}

func WithTextRotateLogger(cfg LogRotateConfig, level slog.Level) GemConfig {
	lj := newLumberjack(cfg)
	return func(r *GemRouter) {
		WithTextLogger(lj, level)(r)
		r.logCloser = lj
	}
}

func WithJSONRotateLogger(cfg LogRotateConfig, level slog.Level) GemConfig {
	lj := newLumberjack(cfg)
	return func(r *GemRouter) {
		WithJSONLogger(lj, level)(r)
		r.logCloser = lj
	}
}

func WithTextTeeRotateLogger(cfg LogRotateConfig, level slog.Level) GemConfig {
	lj := newLumberjack(cfg)
	return func(r *GemRouter) {
		WithTextLogger(io.MultiWriter(os.Stdout, lj), level)(r)
		r.logCloser = lj
	}
}

func WithJSONTeeRotateLogger(cfg LogRotateConfig, level slog.Level) GemConfig {
	lj := newLumberjack(cfg)
	return func(r *GemRouter) {
		WithJSONLogger(io.MultiWriter(os.Stdout, lj), level)(r)
		r.logCloser = lj
	}
}

func WithSplitLogger(path string, level slog.Level) GemConfig {
	f := openLogFile(path)
	opts := &slog.HandlerOptions{Level: level}
	h := &teeHandler{
		console: slog.NewTextHandler(os.Stdout, opts),
		file:    slog.NewJSONHandler(f, opts),
	}

	return func(r *GemRouter) {
		WithLogger(slog.New(h))(r)
		if f != os.Stdout {
			r.logCloser = f
		}
	}
}

func WithSplitRotateLogger(cfg LogRotateConfig, level slog.Level) GemConfig {
	lj := newLumberjack(cfg)
	opts := &slog.HandlerOptions{Level: level}
	h := &teeHandler{
		console: slog.NewTextHandler(os.Stdout, opts),
		file:    slog.NewJSONHandler(lj, opts),
	}

	return func(r *GemRouter) {
		WithLogger(slog.New(h))(r)
		r.logCloser = lj
	}
}
