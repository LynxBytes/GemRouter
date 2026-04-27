package zaplogger

import (
	"io"
	"log/slog"
	"os"

	gem "github.com/LynxBytes/GemRouter"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func WithZapLogger(z *zap.Logger) gem.GemConfig {
	return gem.WithLogger(slog.New(zapslog.NewHandler(z.Core())))
}

func newZapCore(w zapcore.WriteSyncer, level zapcore.Level) zapcore.Core {
	return zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), w, level)
}

func openLogFile(path string) *os.File {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return os.Stdout
	}
	return f
}

func newLumberjack(cfg gem.LogRotateConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}

func combine(configs ...gem.GemConfig) gem.GemConfig {
	return func(r *gem.GemRouter) {
		for _, c := range configs {
			c(r)
		}
	}
}

func WithZapFileLogger(path string, level zapcore.Level) gem.GemConfig {
	f := openLogFile(path)
	z := zap.New(newZapCore(zapcore.AddSync(f), level))
	configs := []gem.GemConfig{WithZapLogger(z)}
	if f != os.Stdout {
		configs = append(configs, gem.WithLogCloser(f))
	}
	return combine(configs...)
}

func WithZapTeeLogger(path string, level zapcore.Level) gem.GemConfig {
	f := openLogFile(path)
	return func(r *gem.GemRouter) {
		w := zapcore.NewMultiWriteSyncer(zapcore.AddSync(r.ConsoleWriter()), zapcore.AddSync(f))
		z := zap.New(newZapCore(w, level))
		WithZapLogger(z)(r)
		if f != os.Stdout {
			gem.WithLogCloser(f)(r)
		}
	}
}

func WithZapRotateLogger(cfg gem.LogRotateConfig, level zapcore.Level) gem.GemConfig {
	lj := newLumberjack(cfg)
	z := zap.New(newZapCore(zapcore.AddSync(lj), level))
	return combine(WithZapLogger(z), gem.WithLogCloser(lj))
}

func WithZapTeeRotateLogger(cfg gem.LogRotateConfig, level zapcore.Level) gem.GemConfig {
	lj := newLumberjack(cfg)
	w := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lj))
	z := zap.New(newZapCore(w, level))
	return combine(WithZapLogger(z), gem.WithLogCloser(lj))
}

func WithZapTeeWriter(w io.Writer, level zapcore.Level) gem.GemConfig {
	return func(r *gem.GemRouter) {
		ws := zapcore.NewMultiWriteSyncer(zapcore.AddSync(r.ConsoleWriter()), zapcore.AddSync(w))
		z := zap.New(newZapCore(ws, level))
		WithZapLogger(z)(r)
	}
}

func WithZapSplitLogger(path string, level zapcore.Level) gem.GemConfig {
	f := openLogFile(path)
	return func(r *gem.GemRouter) {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(r.ConsoleWriter()),
			level,
		)
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(f),
			level,
		)
		z := zap.New(zapcore.NewTee(consoleCore, fileCore))
		WithZapLogger(z)(r)
		if f != os.Stdout {
			gem.WithLogCloser(f)(r)
		}
	}
}

func WithZapSplitRotateLogger(cfg gem.LogRotateConfig, level zapcore.Level) gem.GemConfig {
	lj := newLumberjack(cfg)
	return func(r *gem.GemRouter) {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(r.ConsoleWriter()),
			level,
		)
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(lj),
			level,
		)
		z := zap.New(zapcore.NewTee(consoleCore, fileCore))
		WithZapLogger(z)(r)
		gem.WithLogCloser(lj)(r)
	}
}
