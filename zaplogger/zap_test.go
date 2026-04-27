package zaplogger_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	gem "github.com/LynxBytes/GemRouter"
	"github.com/LynxBytes/GemRouter/zaplogger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func discardZap() *zap.Logger {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(io.Discard),
		zapcore.InfoLevel,
	)
	return zap.New(core)
}

func newZapRouter(cfg gem.GemConfig) *gem.GemRouter {
	r := gem.NewGemRouter(cfg)
	r.GET("/ping", func(ctx *gem.GemContext) { ctx.OK() })
	return r
}

func TestWithZapLogger(t *testing.T) {
	r := newZapRouter(zaplogger.WithZapLogger(discardZap()))
	if r == nil {
		t.Fatal("router should not be nil")
	}
}

func TestWithZapFileLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zap.log")
	newZapRouter(zaplogger.WithZapFileLogger(path, zapcore.InfoLevel))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty")
	}
}

func TestWithZapTeeLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zap_tee.log")
	newZapRouter(zaplogger.WithZapTeeLogger(path, zapcore.InfoLevel))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty")
	}
}

func TestWithZapRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zap_rotate.log")
	cfg := gem.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	newZapRouter(zaplogger.WithZapRotateLogger(cfg, zapcore.InfoLevel))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty")
	}
}

func TestWithZapTeeRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zap_tee_rotate.log")
	cfg := gem.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	newZapRouter(zaplogger.WithZapTeeRotateLogger(cfg, zapcore.InfoLevel))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty")
	}
}

func TestWithZapTeeWriter(t *testing.T) {
	var buf bytes.Buffer
	newZapRouter(zaplogger.WithZapTeeWriter(&buf, zapcore.InfoLevel))
	if buf.Len() == 0 {
		t.Fatal("expected log output in writer")
	}
}

func TestWithZapSplitLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zap_split.log")
	newZapRouter(zaplogger.WithZapSplitLogger(path, zapcore.InfoLevel))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty")
	}
}

func TestWithZapSplitRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zap_split_rotate.log")
	cfg := gem.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	newZapRouter(zaplogger.WithZapSplitRotateLogger(cfg, zapcore.InfoLevel))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("log file is empty")
	}
}
