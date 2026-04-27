package gem

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newLoggerRouter(cfg GemConfig) *GemRouter {
	r := NewGemRouter(cfg)
	r.GET("/ping", func(ctx *GemContext) { ctx.OK() })
	return r
}

func TestWithTextLogger(t *testing.T) {
	var buf bytes.Buffer
	newLoggerRouter(WithTextLogger(&buf, slog.LevelInfo))
	if !strings.Contains(buf.String(), "Endpoint registered") {
		t.Fatalf("expected log in buffer, got %q", buf.String())
	}
}

func TestWithJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	newLoggerRouter(WithJSONLogger(&buf, slog.LevelInfo))
	if !strings.Contains(buf.String(), `"msg"`) {
		t.Fatalf("expected JSON log in buffer, got %q", buf.String())
	}
}

func TestWithTextFileLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "text.log")
	r := newLoggerRouter(WithTextFileLogger(path, slog.LevelInfo))

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
	r := newLoggerRouter(WithJSONFileLogger(path, slog.LevelInfo))

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
	r := newLoggerRouter(WithTextTeeLogger(path, slog.LevelInfo))

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
	r := newLoggerRouter(WithJSONTeeLogger(path, slog.LevelInfo))

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
	cfg := LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(WithTextRotateLogger(cfg, slog.LevelInfo))

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
	cfg := LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(WithJSONRotateLogger(cfg, slog.LevelInfo))

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
	cfg := LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(WithTextTeeRotateLogger(cfg, slog.LevelInfo))

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
	cfg := LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(WithJSONTeeRotateLogger(cfg, slog.LevelInfo))

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
	r := newLoggerRouter(WithTextFileLogger("/nonexistent/path/file.log", slog.LevelInfo))
	if r.LogCloser() != nil {
		t.Fatal("logCloser should be nil when falling back to stdout")
	}
}

func TestWithSplitLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "split.log")
	r := newLoggerRouter(WithSplitLogger(path, slog.LevelInfo))

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
	cfg := LogRotateConfig{Path: path, MaxSizeMB: 1, MaxBackups: 1}
	r := newLoggerRouter(WithSplitRotateLogger(cfg, slog.LevelInfo))

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
