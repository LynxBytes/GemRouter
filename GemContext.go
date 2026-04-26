package gemrouter

import (
	"encoding/json"
	"log"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.written {
		return
	}
	rw.status = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

type GemContext struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Keys    map[string]any
	rw      *responseWriter
}

func (context *GemContext) StatusCode() int {
	return context.rw.status
}

func (context *GemContext) Status(code int) {
	context.Writer.WriteHeader(code)
}

func (context *GemContext) String(code int, text string) {
	context.Writer.WriteHeader(code)
	if _, err := context.Writer.Write([]byte(text)); err != nil {
		log.Printf("String write error: %v", err)
	}
}

func (context *GemContext) JSON(data any) error {
	defer context.Request.Body.Close()
	return json.NewDecoder(context.Request.Body).Decode(data)
}

func (context *GemContext) ToJSON(code int, data any) {
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(code)
	if err := json.NewEncoder(context.Writer).Encode(data); err != nil {
		log.Printf("ToJSON encode error: %v", err)
	}
}

func (context *GemContext) NoContent(code int) {
	context.Writer.WriteHeader(code)
}

func (context *GemContext) OK() {
	context.ToJSON(http.StatusOK, map[string]string{
		"message": "ok",
	})
}

func (context *GemContext) Query(key string) string {
	return context.Request.URL.Query().Get(key)
}

func (context *GemContext) Header(key string) string {
	return context.Request.Header.Get(key)
}

func (context *GemContext) Method() string {
	return context.Request.Method
}

func (context *GemContext) Path() string {
	return context.Request.URL.Path
}

func (context *GemContext) Param(key string) string {
	return context.Request.PathValue(key)
}

func (context *GemContext) Set(key string, val any) {
	context.Keys[key] = val
}

func (context *GemContext) Get(key string) (any, bool) {
	val, ok := context.Keys[key]
	return val, ok
}
