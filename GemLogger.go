package gemrouter

import (
	"fmt"
	"log"
	"strings"
)

type GemLogger interface {
	Info(msg string, fields map[string]any)
	Debug(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

type stdLogger struct{}

func (stdLogger) Debug(msg string, fields map[string]any) {
	log.Printf("[GemRouter] DEBUG %s %s", msg, formatFields(fields))
}

func (stdLogger) Info(msg string, fields map[string]any) {
	log.Printf("[GemRouter] INFO %s %s", msg, formatFields(fields))
}

func (stdLogger) Warn(msg string, fields map[string]any) {
	log.Printf("[GemRouter] WARN %s %s", msg, formatFields(fields))
}

func (stdLogger) Error(msg string, fields map[string]any) {
	log.Printf("[GemRouter] ERROR %s %s", msg, formatFields(fields))
}

func formatFields(fields map[string]any) string {
	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, " ")
}

var defaultGemLogger GemLogger = stdLogger{}
