package gem

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/term"
)

type GemRouter struct {
	mux               *httprouter.Router
	name              string
	version           string
	Addr              string
	Port              string
	middlewares       []Middleware
	NotFound          GemHandler
	MethodNotAllowed  GemHandler
	Health            GemHandler
	shutdownTimeout   time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	corsSet           bool
	stdout            *rawModeWriter
	logger            *slog.Logger
	logCloser         io.Closer
	trustProxy        bool
	responseFormatter ResponseFormatter
	errorFormatter    ErrorFormatter
	ctxPool           sync.Pool
}

const banner = `
 ▄████  ▄▄▄▄▄ ▄▄   ▄▄ █████▄   ▄▄▄  ▄▄ ▄▄ ▄▄▄▄▄▄ ▄▄▄▄▄ ▄▄▄▄
██  ▄▄▄ ██▄▄  ██▀▄▀██ ██▄▄██▄ ██▀██ ██ ██   ██   ██▄▄  ██▄█▄
 ▀███▀  ██▄▄▄ ██   ██ ██   ██ ▀███▀ ▀███▀   ██   ██▄▄▄ ██ ██`

func newHTTPRouter() *httprouter.Router {
	r := httprouter.New()
	r.HandleOPTIONS = false
	r.HandleMethodNotAllowed = true
	r.RedirectTrailingSlash = true
	r.RedirectFixedPath = true
	return r
}

func newBaseRouter() *GemRouter {
	stdout := &rawModeWriter{w: os.Stdout}
	r := &GemRouter{
		mux:               newHTTPRouter(),
		name:              "GemRouter Server",
		version:           "v0.0.0",
		Addr:              "0.0.0.0",
		Port:              "8080",
		shutdownTimeout:   5 * time.Second,
		readTimeout:       30 * time.Second,
		writeTimeout:      30 * time.Second,
		idleTimeout:       120 * time.Second,
		stdout:            stdout,
		logger:            newDefaultLogger(stdout),
		responseFormatter: defaultResponseFormatter,
		errorFormatter:    defaultErrorFormatter,
		NotFound:          func(ctx *GemContext) { ctx.NOTFOUND() },
		MethodNotAllowed:  defaultMethodNotAllowed,
		Health:            func(ctx *GemContext) { ctx.OK() },
	}
	r.ctxPool = sync.Pool{New: func() any { return &GemContext{Store: &ContextStore{}} }}
	return r
}

func (r *GemRouter) ConsoleWriter() io.Writer {
	return r.stdout
}

func BasicGemRouter() *GemRouter {
	r := newBaseRouter()
	r.middlewares = []Middleware{}
	r.corsSet = true
	return r
}

func DefaultGemRouter() *GemRouter {
	r := newBaseRouter()
	r.middlewares = []Middleware{Cors(defaultCors), Recovery, Logger}
	r.corsSet = true
	return r
}

func NewGemRouter(configs ...GemConfig) *GemRouter {
	r := newBaseRouter()
	r.middlewares = []Middleware{Recovery}

	for _, opt := range configs {
		opt(r)
	}

	if r.corsSet {
		r.handle(http.MethodOptions, "/*path", func(_ *GemContext) {})
	}
	return r
}

func (r *GemRouter) Use(middleware Middleware) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *GemRouter) GET(pattern string, handler GemHandler) {
	r.handle(http.MethodGet, pattern, handler)
}

func (r *GemRouter) POST(pattern string, handler GemHandler) {
	r.handle(http.MethodPost, pattern, handler)
}

func (r *GemRouter) PUT(pattern string, handler GemHandler) {
	r.handle(http.MethodPut, pattern, handler)
}

func (r *GemRouter) PATCH(pattern string, handler GemHandler) {
	r.handle(http.MethodPatch, pattern, handler)
}

func (r *GemRouter) DELETE(pattern string, handler GemHandler) {
	r.handle(http.MethodDelete, pattern, handler)
}

func (r *GemRouter) NoRoute(handler GemHandler) {
	r.NotFound = handler
	notFoundFinal := buildChain(handler, r.middlewares)
	methodNotAllowedFinal := buildChain(r.MethodNotAllowed, r.middlewares)

	r.mux.NotFound = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := r.newContext(w, req)
		defer r.releaseContext(ctx)
		notFoundFinal(ctx)
	})

	r.mux.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := r.newContext(w, req)
		defer r.releaseContext(ctx)
		methodNotAllowedFinal(ctx)
	})
}

func (r *GemRouter) Run() error {
	slog.Info(banner)
	r.logger.Info("Starting server " + r.name + "@" + r.version)

	r.GET("/health", r.Health)
	r.NoRoute(r.NotFound)

	addr := r.Addr + ":" + r.Port

	// SIGTERM always triggers immediate shutdown.
	// SIGINT is only used in non-TTY mode (pipes, Docker, CI).
	termCh := make(chan os.Signal, 1)
	intCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM)
	defer signal.Stop(termCh)

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	var restoreTerm func()

	inputCh := make(chan struct{}, 1)

	if isTTY {
		old, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			r.stdout.raw.Store(true)
			restoreTerm = func() {
				r.stdout.raw.Store(false)
				_ = term.Restore(int(os.Stdin.Fd()), old)
			}
			go func() {
				b := make([]byte, 1)
				for {
					if _, err := os.Stdin.Read(b); err != nil {
						return
					}
					if b[0] == 0x03 || b[0] == 0x04 { // Ctrl+C, Ctrl+D
						inputCh <- struct{}{}
					}
				}
			}()
		}
	} else {
		signal.Notify(intCh, syscall.SIGINT)
		defer signal.Stop(intCh)
	}

	doShutdown := func(srv *http.Server) error {
		if restoreTerm != nil {
			restoreTerm()
		}
		ctx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout)
		defer cancel()
		err := srv.Shutdown(ctx)
		if r.logCloser != nil {
			_ = r.logCloser.Close()
		}
		return err
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r.mux,
		ReadTimeout:  r.readTimeout,
		WriteTimeout: r.writeTimeout,
		IdleTimeout:  r.idleTimeout,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	r.logger.Info("Server up: " + addr)

	sigintCount := 0
	var resetTimer <-chan time.Time

	for {
		select {
		case err := <-errCh:
			if r.logCloser != nil {
				_ = r.logCloser.Close()
			}
			return err

		case <-termCh:
			return doShutdown(srv)

		case <-intCh:
			sigintCount++
			if sigintCount == 1 {
				fmt.Fprint(os.Stderr, "\r\nPress Ctrl+C again to stop the server\r\n")
				resetTimer = time.After(3 * time.Second)
			} else {
				return doShutdown(srv)
			}

		case <-inputCh:
			fmt.Fprint(os.Stderr, "\r\nPress Ctrl+C again to stop the server\r\n")
			sigintCount++
			if sigintCount == 1 {
				resetTimer = time.After(3 * time.Second)
			} else {
				return doShutdown(srv)
			}

		case <-resetTimer:
			sigintCount = 0
			resetTimer = nil
			fmt.Fprint(os.Stderr, "\r\nShutdown cancelled\r\n")
		}
	}
}

func (r *GemRouter) Group(prefix string, middlewares ...Middleware) *GemGroup {
	return &GemGroup{prefix: prefix, router: r, middlewares: middlewares}
}

func (r *GemRouter) newContext(w http.ResponseWriter, req *http.Request) *GemContext {
	ctx := r.ctxPool.Get().(*GemContext)
	ctx.Request = req
	ctx.Logger = r.logger
	ctx.trustProxy = r.trustProxy
	ctx.responseFormatter = r.responseFormatter
	ctx.errorFormatter = r.errorFormatter
	ctx.rwBuf.ResponseWriter = w
	ctx.rwBuf.status = 0
	ctx.rwBuf.written = false
	ctx.rw = &ctx.rwBuf
	ctx.Writer = ctx.rw
	return ctx
}

func (r *GemRouter) releaseContext(ctx *GemContext) {
	if ctx.Store != nil {
		ctx.Store.RequestID = ""
		ctx.Store.UserID = ""
		clear(ctx.Store.data)
	}
	ctx.rwBuf.ResponseWriter = nil
	ctx.rwBuf.status = 0
	ctx.rwBuf.written = false
	ctx.Request = nil
	ctx.rw = nil
	ctx.params = nil
	ctx.Pattern = ""
	r.ctxPool.Put(ctx)
}

func (r *GemRouter) handle(method, pattern string, handler GemHandler, extra ...Middleware) {
	finalHandler := buildChain(handler, r.middlewares, extra...)
	fullPattern := method + " " + pattern

	r.mux.Handle(method, pattern, func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		ctx := r.newContext(w, req)
		ctx.params = ps
		ctx.Pattern = fullPattern
		defer r.releaseContext(ctx)
		finalHandler(ctx)
	})

	r.logger.Info("Endpoint registered ✅", "method", method, "endpoint", pattern)
}

func buildChain(handler GemHandler, base []Middleware, extra ...Middleware) GemHandler {
	for i := len(extra) - 1; i >= 0; i-- {
		handler = extra[i](handler)
	}
	for i := len(base) - 1; i >= 0; i-- {
		handler = base[i](handler)
	}
	return handler
}
