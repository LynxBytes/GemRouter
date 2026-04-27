package gemrouter

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
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

type ContextStore struct {
	RequestID string
	UserID    string
	data      map[string]any
}

func (store *ContextStore) Set(key string, val any) {
	if store.data == nil {
		store.data = make(map[string]any, 4)
	}
	store.data[key] = val
}

func (store *ContextStore) Get(key string) (any, bool) {
	if store.data == nil {
		return nil, false
	}
	v, ok := store.data[key]
	return v, ok
}

type GemContext struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	Store      *ContextStore
	Logger     GemLogger
	rw         *responseWriter
	rwBuf      responseWriter
	trustProxy bool
}

func (context *GemContext) Copy() *GemContext {
	var storeCopy *ContextStore

	if context.Store != nil {
		storeCopy = &ContextStore{
			RequestID: context.Store.RequestID,
			UserID:    context.Store.UserID,
			data:      make(map[string]any, len(context.Store.data)),
		}

		for k, v := range context.Store.data {
			storeCopy.data[k] = v
		}
	}

	return &GemContext{
		Request: context.Request,
		Store:   storeCopy,
		Logger:  context.Logger,
	}
}

func (context *GemContext) RequestID() string {
	return context.Store.RequestID
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
	if _, err := io.WriteString(context.Writer, text); err != nil {
		context.Logger.Error("write error", map[string]any{"error": err})
	}
}

func (context *GemContext) FromJSON(data any) error {
	defer context.Request.Body.Close()
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, maxRequestBodySize)
	return sonic.ConfigDefault.NewDecoder(context.Request.Body).Decode(data)
}

func (context *GemContext) ToJSON(code int, data any) {
	b, err := sonic.Marshal(data)
	if err != nil {
		context.Logger.Error("json encode error", map[string]any{"error": err})
		return
	}
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(code)
	_, _ = context.Writer.Write(b)
}

func (context *GemContext) NoContent(code int) {
	context.Writer.WriteHeader(code)
}

var (
	okBody       = []byte(`{"message":"ok"}` + "\n")
	notFoundBody = []byte(`{"error":"not found"}` + "\n")
)

func (context *GemContext) OK() {
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(http.StatusOK)
	_, _ = context.Writer.Write(okBody)
}

func (context *GemContext) NOTFOUND() {
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(http.StatusNotFound)
	_, _ = context.Writer.Write(notFoundBody)
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
	if context.Store == nil {
		context.Store = &ContextStore{}
	}
	context.Store.Set(key, val)
}

func (context *GemContext) Get(key string) (any, bool) {
	if context.Store == nil {
		return nil, false
	}
	return context.Store.Get(key)
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
