package gem

import (
	"context"
	"errors"
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
	routerVersion     string
	mux               *httprouter.Router
	name              string
	version           string
	Addr              string
	Port              string
	middlewares       []Middleware
	methodNotFound    GemHandler
	methodNotAllowed  GemHandler
	Health            GemHandler
	shutdownTimeout   time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	corsSet           bool
	stdout            *rawModeWriter
	logger            *slog.Logger
	logHandlers       bool
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
		routerVersion:     "v0.0.37",
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
		logHandlers:       true,
		responseFormatter: defaultResponseFormatter,
		errorFormatter:    defaultErrorFormatter,
		methodNotAllowed:  defaultMethodNotAllowed,
		methodNotFound:    defaultMethodNotFound,
		Health:            func(ctx *GemContext) { ctx.OK() },
		ctxPool: sync.Pool{
			New: func() any { return &GemContext{} },
		},
	}
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

func (r *GemRouter) HandleSystemErrors() {
	methodNotAllowedFinal := buildChain(r.methodNotAllowed, r.middlewares)
	r.mux.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := r.newContext(w, req)
		defer r.releaseContext(ctx)
		methodNotAllowedFinal(ctx)
	})

	notFoundFinal := buildChain(r.methodNotFound, r.middlewares)
	r.mux.NotFound = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := r.newContext(w, req)
		defer r.releaseContext(ctx)
		notFoundFinal(ctx)
	})
}

func (r *GemRouter) Run() error {
	slog.Info(banner)
	r.logger.Info("Starting server " + r.name + "@" + r.version)

	r.GET("/health", r.Health)
	r.HandleSystemErrors()

	addr := r.Addr + ":" + r.Port

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM)
	defer signal.Stop(termCh)

	stopEventCh := make(chan struct{}, 1)
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))

	var restoreTerm func()

	if isTTY {
		if old, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
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
					if b[0] == 0x03 || b[0] == 0x04 {
						stopEventCh <- struct{}{}
					}
				}
			}()
		}
	} else {
		intCh := make(chan os.Signal, 1)
		signal.Notify(intCh, syscall.SIGINT)
		defer signal.Stop(intCh)
		go func() {
			for range intCh {
				stopEventCh <- struct{}{}
			}
		}()
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

	defer func() {
		if r.logCloser != nil {
			_ = r.logCloser.Close()
		}
	}()

	for {
		select {
		case err := <-errCh:
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err

		case <-termCh:
			return r.doShutdown(srv, restoreTerm)

		case <-stopEventCh:
			sigintCount++
			if sigintCount >= 2 {
				return r.doShutdown(srv, restoreTerm)
			}
			fmt.Fprint(os.Stderr, "\r\nPress Ctrl+C again to stop the server\r\n")
			resetTimer = time.After(3 * time.Second)

		case <-resetTimer:
			sigintCount = 0
			resetTimer = nil
			fmt.Fprint(os.Stderr, "\r\nShutdown cancelled\r\n")
		}
	}
}

func (r *GemRouter) doShutdown(srv *http.Server, restoreTerm func()) error {
	if restoreTerm != nil {
		restoreTerm()
	}

	r.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout)
	defer cancel()

	return srv.Shutdown(ctx)
}

func (r *GemRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
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
	ctx.rwBuf.ResponseWriter = nil
	ctx.rwBuf.status = 0
	ctx.rwBuf.written = false
	ctx.Request = nil
	ctx.rw = nil
	ctx.params = nil
	ctx.store = ctx.store[:0]
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

	if r.logHandlers {
		r.logger.Info("Endpoint registered ✅", "method", method, "endpoint", pattern)
	}
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
