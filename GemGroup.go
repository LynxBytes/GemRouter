package gem

import (
	"net/http"
	"slices"
)

type GemGroup struct {
	prefix      string
	router      *GemRouter
	middlewares []Middleware
}

func (g *GemGroup) Use(m Middleware) {
	g.middlewares = append(g.middlewares, m)
}

func (g *GemGroup) Group(prefix string, middlewares ...Middleware) *GemGroup {
	return &GemGroup{
		prefix:      g.prefix + prefix,
		router:      g.router,
		middlewares: slices.Concat(g.middlewares, middlewares),
	}
}

func (g *GemGroup) GET(pattern string, handler GemHandler) {
	g.router.handle(http.MethodGet, g.prefix+pattern, handler, g.middlewares...)
}

func (g *GemGroup) POST(pattern string, handler GemHandler) {
	g.router.handle(http.MethodPost, g.prefix+pattern, handler, g.middlewares...)
}

func (g *GemGroup) PUT(pattern string, handler GemHandler) {
	g.router.handle(http.MethodPut, g.prefix+pattern, handler, g.middlewares...)
}

func (g *GemGroup) PATCH(pattern string, handler GemHandler) {
	g.router.handle(http.MethodPatch, g.prefix+pattern, handler, g.middlewares...)
}

func (g *GemGroup) DELETE(pattern string, handler GemHandler) {
	g.router.handle(http.MethodDelete, g.prefix+pattern, handler, g.middlewares...)
}
