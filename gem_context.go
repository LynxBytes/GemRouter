package gem

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/julienschmidt/httprouter"
)

const maxRequestBodySize = 4 << 20 // 4 MB

type GemContext struct {
	Writer            http.ResponseWriter
	Request           *http.Request
	Store             *ContextStore
	Logger            *slog.Logger
	Pattern           string
	params            httprouter.Params
	rw                *responseWriter
	rwBuf             responseWriter
	trustProxy        bool
	responseFormatter ResponseFormatter
	errorFormatter    ErrorFormatter
}

func NewTestContext(w http.ResponseWriter, r *http.Request) *GemContext {
	ctx := &GemContext{
		Request:           r,
		Store:             &ContextStore{},
		Logger:            slog.Default(),
		responseFormatter: defaultResponseFormatter,
		errorFormatter:    defaultErrorFormatter,
	}
	ctx.rwBuf.ResponseWriter = w
	ctx.rw = &ctx.rwBuf
	ctx.Writer = ctx.rw
	return ctx
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
		context.Logger.Error("write error", slog.Any("error", err))
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
		context.Logger.Error("json encode error", slog.Any("error", err))
		return
	}
	context.Writer.Header().Set("Content-Type", "application/json")
	context.Writer.WriteHeader(code)
	_, _ = context.Writer.Write(b)
}

func (context *GemContext) NoContent(code int) {
	context.Writer.WriteHeader(code)
}

func (context *GemContext) Success(code int, data any) {
	finalCode, finalData := context.responseFormatter(code, data)
	context.ToJSON(finalCode, finalData)
}

func (context *GemContext) Fail(code int, errs ...any) {
	finalCode, finalData := context.errorFormatter(code, errs)
	context.ToJSON(finalCode, finalData)
}

var (
	okBody               = []byte(`{"message":"ok"}` + "\n")
	notFoundBody         = []byte(`{"error":"not found"}` + "\n")
	methodNotAllowedBody = []byte(`{"error":"method not allowed"}` + "\n")
	methodNotFoundBody   = []byte(`{"error":"method not found"}` + "\n")
)

var defaultMethodNotAllowed GemHandler = func(ctx *GemContext) {
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.Writer.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = ctx.Writer.Write(methodNotAllowedBody)
}

var defaultMethodNotFound GemHandler = func(ctx *GemContext) {
	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.Writer.WriteHeader(http.StatusNotFound)
	_, _ = ctx.Writer.Write(methodNotFoundBody)
}

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
	return context.params.ByName(key)
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
