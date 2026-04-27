package gemrouter

import (
	"encoding/json"
	"net/http"
)

const maxRequestBodySize = 4 << 20 // 4 MB

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
	Writer     http.ResponseWriter
	Request    *http.Request
	Keys       map[string]any
	Logger     GemLogger
	rw         *responseWriter
	rwBuf      responseWriter
	trustProxy bool
}

// Copy returns a snapshot of the context safe to use in a goroutine.
// The original context must not be used after the handler returns.
func (context *GemContext) Copy() *GemContext {
	keys := make(map[string]any, len(context.Keys))
	for k, v := range context.Keys {
		keys[k] = v
	}
	return &GemContext{
		Request: context.Request,
		Keys:    keys,
		Logger:  context.Logger,
	}
}

func (context *GemContext) RequestID() string {
	id, ok := context.Keys["request_id"]
	if !ok {
		return ""
	}
	s, _ := id.(string)
	return s
}

func (context *GemContext) StatusCode() int {
	if context.rw.status == 0 {
		return http.StatusOK
	}
	return context.rw.status
}

func (context *GemContext) Status(code int) {
	context.Writer.WriteHeader(code)
}

func (context *GemContext) String(code int, text string) {
	context.Writer.WriteHeader(code)
	if _, err := context.Writer.Write([]byte(text)); err != nil {
		context.Logger.Error("write error", map[string]any{"error": err})
	}
}

func (context *GemContext) FromJSON(data any) error {
	defer context.Request.Body.Close()
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, maxRequestBodySize)
	return json.NewDecoder(context.Request.Body).Decode(data)
}

func (context *GemContext) ToJSON(code int, data any) {
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(code)
	if err := json.NewEncoder(context.Writer).Encode(data); err != nil {
		context.Logger.Error("json encode error", map[string]any{"error": err})
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

func (context *GemContext) NOTFOUND() {
	context.ToJSON(http.StatusNotFound, map[string]string{
		"error": "not found",
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

func (context *GemContext) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	http.SetCookie(context.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

func (context *GemContext) Cookie(name string) (string, error) {
	c, err := context.Request.Cookie(name)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

func (context *GemContext) DeleteCookie(name string) {
	http.SetCookie(context.Writer, &http.Cookie{
		Name:   name,
		Value:  "",
		MaxAge: -1,
	})
}
