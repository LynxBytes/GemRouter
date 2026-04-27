package gemrouter

import (
	"fmt"
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
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Param(b *testing.B) {
	r := newBenchRouter()
	r.GET("/users/:id", func(ctx *GemContext) {
		_ = ctx.Param("id")
		ctx.OK()
	})
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
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
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.mux.ServeHTTP(w, req)
	}
}

type jsonResponse struct {
	Hello string `json:"hello"`
	Num   int    `json:"num"`
	Ok    bool   `json:"ok"`
}

type userResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func BenchmarkRouter_JSON(b *testing.B) {
	r := newBenchRouter()
	r.GET("/json", func(ctx *GemContext) {
		ctx.ToJSON(200, jsonResponse{Hello: "world", Num: 123, Ok: true})
	})
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
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
	r.GET("/users/:id", func(ctx *GemContext) {
		ctx.ToJSON(200, userResponse{ID: ctx.Param("id"), Name: "test"})
	})
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
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
		w := httptest.NewRecorder()
		for pb.Next() {
			w.Body.Reset()
			r.mux.ServeHTTP(w, req)
		}
	})
}

func BenchmarkRouter_NoContent(b *testing.B) {
	r := newBenchRouter()
	r.GET("/nc", func(ctx *GemContext) {
		ctx.NoContent(200)
	})
	req := httptest.NewRequest(http.MethodGet, "/nc", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.mux.ServeHTTP(w, req)
	}
}

func benchmarkNRoutes(b *testing.B, n int) {
	r := newBenchRouter()
	for i := range n {
		path := fmt.Sprintf("/route%d/:id", i)
		r.GET(path, func(ctx *GemContext) { ctx.NoContent(200) })
	}
	// hit the last route (worst case for linear scan)
	target := fmt.Sprintf("/route%d/42", n-1)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.mux.ServeHTTP(w, req)
	}
}

func BenchmarkRoutes_10(b *testing.B)  { benchmarkNRoutes(b, 10) }
func BenchmarkRoutes_100(b *testing.B) { benchmarkNRoutes(b, 100) }
func BenchmarkRoutes_500(b *testing.B) { benchmarkNRoutes(b, 500) }
