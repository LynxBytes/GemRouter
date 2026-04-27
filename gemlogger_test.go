package gem_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gemrouter "github.com/LynxBytes/GemRouter"
)

func newLoggerRouter(cfg gemrouter.GemConfig) *gemrouter.GemRouter {
	r := gemrouter.NewGemRouter(cfg)
	r.GET("/ping", func(ctx *gemrouter.GemContext) { ctx.OK() })
	return r
}

func TestWithTextLogger(t *testing.T) {
	var buf bytes.Buffer
	newLoggerRouter(gemrouter.WithTextLogger(&buf, slog.LevelInfo))
	if !strings.Contains(buf.String(), "Endpoint registered") {
		t.Fatalf("expected log in buffer, got %q", buf.String())
	}
}

func TestWithJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	newLoggerRouter(gemrouter.WithJSONLogger(&buf, slog.LevelInfo))
	if !strings.Contains(buf.String(), `"msg"`) {
		t.Fatalf("expected JSON log in buffer, got %q", buf.String())
	}
}

func TestWithTextFileLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "text.log")
	r := newLoggerRouter(gemrouter.WithTextFileLogger(path, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), "Endpoint registered") {
		t.Fatalf("expected log in file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithJSONFileLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "json.log")
	r := newLoggerRouter(gemrouter.WithJSONFileLogger(path, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), `"msg"`) {
		t.Fatalf("expected JSON log in file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithTextTeeLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tee_text.log")
	r := newLoggerRouter(gemrouter.WithTextTeeLogger(path, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), "Endpoint registered") {
		t.Fatalf("expected log in tee file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithJSONTeeLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tee_json.log")
	r := newLoggerRouter(gemrouter.WithJSONTeeLogger(path, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), `"msg"`) {
		t.Fatalf("expected JSON log in tee file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithTextRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rotate_text.log")
	cfg := gemrouter.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(gemrouter.WithTextRotateLogger(cfg, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), "Endpoint registered") {
		t.Fatalf("expected log in file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithJSONRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rotate_json.log")
	cfg := gemrouter.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(gemrouter.WithJSONRotateLogger(cfg, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), `"msg"`) {
		t.Fatalf("expected JSON log in file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithTextTeeRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tee_rotate_text.log")
	cfg := gemrouter.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(gemrouter.WithTextTeeRotateLogger(cfg, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), "Endpoint registered") {
		t.Fatalf("expected log in tee rotate file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithJSONTeeRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tee_rotate_json.log")
	cfg := gemrouter.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(gemrouter.WithJSONTeeRotateLogger(cfg, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), `"msg"`) {
		t.Fatalf("expected JSON log in tee rotate file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestOpenLogFileFallback(t *testing.T) {
	r := newLoggerRouter(gemrouter.WithTextFileLogger("/nonexistent/path/file.log", slog.LevelInfo))
	if r.LogCloser() != nil {
		t.Fatal("logCloser should be nil when falling back to stdout")
	}
}

func TestWithSplitLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "split.log")
	r := newLoggerRouter(gemrouter.WithSplitLogger(path, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), `"msg"`) {
		t.Fatalf("expected JSON in file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}

func TestWithSplitRotateLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "split_rotate.log")
	cfg := gemrouter.LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(gemrouter.WithSplitRotateLogger(cfg, slog.LevelInfo))

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(content), `"msg"`) {
		t.Fatalf("expected JSON in file, got %q", string(content))
	}
	if r.LogCloser() == nil {
		t.Fatal("logCloser should not be nil")
	}
}
