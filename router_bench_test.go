package gemrouter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
	ALL: go test -run=^$ -bench=. -benchmem
	ROUTING: go test -run=^$ -bench=Router_ -benchmem
	JSON: go test -run=^$ -bench=JSON -benchmem
	FULLSTACK: go test -run=^$ -bench=Full -benchmem
*/

func newBenchRouter() *GemRouter {
	return BasicGemRouter()
}

func BenchmarkRouter_Ping(b *testing.B) {
	r := newBenchRouter()

	r.GET("/ping", func(ctx *GemContext) {
		ctx.OK()
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Param(b *testing.B) {
	r := newBenchRouter()

	r.GET("/users/{id}", func(ctx *GemContext) {
		_ = ctx.Param("id")
		ctx.OK()
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_MiddlewareChain(b *testing.B) {
	r := newBenchRouter()

	r.Use(func(next GemHandler) GemHandler {
		return func(ctx *GemContext) {
			ctx.Store.Set("key", "value")
			next(ctx)
		}
	})

	r.GET("/ping", func(ctx *GemContext) {
		ctx.OK()
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_JSON(b *testing.B) {
	r := newBenchRouter()

	r.GET("/json", func(ctx *GemContext) {
		ctx.ToJSON(200, map[string]any{
			"hello": "world",
			"num":   123,
			"ok":    true,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_FullStack(b *testing.B) {
	r := newBenchRouter()

	r.Use(func(next GemHandler) GemHandler {
		return func(ctx *GemContext) {
			ctx.Set("trace", "id")
			next(ctx)
		}
	})

	r.GET("/users/{id}", func(ctx *GemContext) {
		ctx.ToJSON(200, map[string]any{
			"id":   ctx.Param("id"),
			"name": "test",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_ParallelPing(b *testing.B) {
	r := newBenchRouter()

	r.GET("/ping", func(ctx *GemContext) {
		ctx.OK()
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			r.mux.ServeHTTP(w, req)
		}
	})
}
