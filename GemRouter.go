package gemrouter

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type GemRouter struct {
	mux             *http.ServeMux
	Addr            string
	Port            string
	middlewares     []Middleware
	NotFound        GemHandler
	Health          GemHandler
	shutdownTimeout time.Duration
	corsSet         bool
	logger          GemLogger
	trustProxy      bool
	ctxPool         sync.Pool
}

const banner = `
 ▄████  ▄▄▄▄▄ ▄▄   ▄▄ █████▄   ▄▄▄  ▄▄ ▄▄ ▄▄▄▄▄▄ ▄▄▄▄▄ ▄▄▄▄
██  ▄▄▄ ██▄▄  ██▀▄▀██ ██▄▄██▄ ██▀██ ██ ██   ██   ██▄▄  ██▄█▄
 ▀███▀  ██▄▄▄ ██   ██ ██   ██ ▀███▀ ▀███▀   ██   ██▄▄▄ ██ ██\n`

func DefaultGemRouter() *GemRouter {
	r := &GemRouter{
		mux:             http.NewServeMux(),
		Addr:            "0.0.0.0",
		Port:            "8080",
		shutdownTimeout: 5 * time.Second,
		middlewares:     []Middleware{Cors(defaultCors), Recovery, Logger},
		corsSet:         true,
		logger:          defaultGemLogger,
		NotFound: func(ctx *GemContext) {
			ctx.NOTFOUND()
		},
		Health: func(ctx *GemContext) {
			ctx.OK()
		},
	}
	r.ctxPool = sync.Pool{New: func() any { return &GemContext{Keys: make(map[string]any)} }}
	return r
}

func NewGemRouter(configs ...GemConfig) *GemRouter {
	r := &GemRouter{
		mux:             http.NewServeMux(),
		Addr:            "0.0.0.0",
		Port:            "8080",
		shutdownTimeout: 5 * time.Second,
		middlewares:     []Middleware{Recovery, Logger},
		corsSet:         false,
		logger:          defaultGemLogger,
		NotFound: func(ctx *GemContext) {
			ctx.NOTFOUND()
		},
		Health: func(ctx *GemContext) {
			ctx.OK()
		},
	}
	r.ctxPool = sync.Pool{New: func() any { return &GemContext{Keys: make(map[string]any)} }}

	for _, opt := range configs {
		opt(r)
	}

	if r.corsSet {
		r.handle(http.MethodOptions, "/{path...}", func(_ *GemContext) {})
	}
	return r
}

func (r *GemRouter) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
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
	finalHandler := buildChain(handler, r.middlewares)

	r.mux.HandleFunc("/", func(response http.ResponseWriter, req *http.Request) {
		ctx := r.newContext(response, req)
		defer r.releaseContext(ctx)
		finalHandler(ctx)
	})
}

func (r *GemRouter) Run() error {
	log.Println(banner)

	r.GET("/health", r.Health)
	r.NoRoute(r.NotFound)

	srv := &http.Server{Addr: r.Addr + ":" + r.Port, Handler: r.mux}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

func (r *GemRouter) Group(prefix string, middlewares ...Middleware) *GemGroup {
	return &GemGroup{prefix: prefix, router: r, middlewares: middlewares}
}

func (r *GemRouter) newContext(response http.ResponseWriter, req *http.Request) *GemContext {
	ctx := r.ctxPool.Get().(*GemContext)
	ctx.rwBuf = responseWriter{ResponseWriter: response}
	ctx.rw = &ctx.rwBuf
	ctx.Writer = ctx.rw
	ctx.Request = req
	ctx.Logger = r.logger
	ctx.trustProxy = r.trustProxy
	return ctx
}

func (r *GemRouter) releaseContext(ctx *GemContext) {
	clear(ctx.Keys)
	ctx.Writer = nil
	ctx.Request = nil
	ctx.Logger = nil
	ctx.rw = nil
	ctx.rwBuf.ResponseWriter = nil
	ctx.rwBuf.status = 0
	ctx.rwBuf.written = false
	ctx.trustProxy = false
	r.ctxPool.Put(ctx)
}

func (r *GemRouter) handle(method, pattern string, handler GemHandler, extra ...Middleware) {
	all := make([]Middleware, 0, len(r.middlewares)+len(extra))
	all = append(all, r.middlewares...)
	all = append(all, extra...)
	finalHandler := buildChain(handler, all)

	r.mux.HandleFunc(method+" "+pattern, func(response http.ResponseWriter, req *http.Request) {
		ctx := r.newContext(response, req)
		defer r.releaseContext(ctx)
		finalHandler(ctx)
	})
}

func buildChain(handler GemHandler, middlewares []Middleware) GemHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
