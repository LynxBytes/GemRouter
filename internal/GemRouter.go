package internal

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type GemRouter struct {
	mux         *http.ServeMux
	Addr        string
	Port        string
	Middlewares []Middleware
	NotFound    GemHandler
	Health      GemHandler
}

func NewGemRouter() *GemRouter {
	return &GemRouter{
		mux:  http.NewServeMux(),
		Addr: "0.0.0.0",
		Port: ":8080",
		Middlewares: []Middleware{
			Recovery,
			Logger,
		},
		NotFound: func(ctx *GemContext) {
			ctx.NOTFOUND()
		},
		Health: func(ctx *GemContext) {
			ctx.OK()
		},
	}
}

func CustomGemRouter(configs ...GemConfig) *GemRouter {
	r := NewGemRouter()

	for _, opt := range configs {
		opt(r)
	}

	return r
}

func (r *GemRouter) Use(m Middleware) {
	r.Middlewares = append(r.Middlewares, m)
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
	r.mux.HandleFunc("/", func(response http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			rw := &responseWriter{ResponseWriter: response}
			ctx := &GemContext{Writer: rw, Request: req, Keys: make(map[string]any), rw: rw}
			finalHandler := next(r.NotFound, r.Middlewares)
			finalHandler(ctx)
			return
		}
		http.NotFound(response, req)
	})
}

func (r *GemRouter) Run() error {
	log.Println(`                                                             
 ▄████  ▄▄▄▄▄ ▄▄   ▄▄ █████▄   ▄▄▄  ▄▄ ▄▄ ▄▄▄▄▄▄ ▄▄▄▄▄ ▄▄▄▄  
██  ▄▄▄ ██▄▄  ██▀▄▀██ ██▄▄██▄ ██▀██ ██ ██   ██   ██▄▄  ██▄█▄ 
 ▀███▀  ██▄▄▄ ██   ██ ██   ██ ▀███▀ ▀███▀   ██   ██▄▄▄ ██ ██\n`)
	r.GET("/health", r.Health)
	r.NoRoute(r.NotFound)

	srv := &http.Server{Addr: r.Addr + r.Port, Handler: r.mux}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

func (r *GemRouter) Group(prefix string, middlewares ...Middleware) *GemGroup {
	return &GemGroup{prefix: prefix, router: r, middlewares: middlewares}
}

func (r *GemRouter) handle(method, pattern string, handler GemHandler, extra ...Middleware) {
	route := method + " " + pattern

	r.mux.HandleFunc(route, func(response http.ResponseWriter, req *http.Request) {
		rw := &responseWriter{ResponseWriter: response}
		ctx := &GemContext{
			Writer:  rw,
			Request: req,
			Keys:    make(map[string]any),
			rw:      rw,
		}
		all := make([]Middleware, 0, len(r.Middlewares)+len(extra))
		all = append(all, r.Middlewares...)
		all = append(all, extra...)
		finalHandler := next(handler, all)
		finalHandler(ctx)
	})
}

func next(handler GemHandler, middlewares []Middleware) GemHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
