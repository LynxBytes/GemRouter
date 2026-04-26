package gemrouter

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type GemRouter struct {
	mux         *http.ServeMux
	middlewares []Middleware
	notFound    GemHandler
}

func NewGemRouter() *GemRouter {
	return &GemRouter{
		mux: http.NewServeMux(),
		middlewares: []Middleware{
			Recovery,
			Logger,
		},
	}
}

func CustomGemRouter(middlewares []Middleware, notFound GemHandler) *GemRouter {
	return &GemRouter{
		mux:         http.NewServeMux(),
		middlewares: middlewares,
		notFound:    notFound,
	}
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
	r.notFound = handler
	r.mux.HandleFunc("/", func(response http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			rw := &responseWriter{ResponseWriter: response}
			ctx := &GemContext{Writer: rw, Request: req, Keys: make(map[string]any), rw: rw}
			finalHandler := next(r.notFound, r.middlewares)
			finalHandler(ctx)
			return
		}
		http.NotFound(response, req)
	})
}

func (r *GemRouter) Run(addr string) error {
	srv := &http.Server{Addr: addr, Handler: r.mux}

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
		all := make([]Middleware, 0, len(r.middlewares)+len(extra))
		all = append(all, r.middlewares...)
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
