package gem_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	gemrouter "github.com/LynxBytes/GemRouter"
)

// TestRaceConcurrentRequests dispara N requests concurrentes para verificar
// que el pool no produce data races bajo carga paralela.
func TestRaceConcurrentRequests(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/ping", func(ctx *gemrouter.GemContext) {
			ctx.Set("key", "value")
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/ping")
			if err != nil {
				t.Errorf("request failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
}

// TestRaceKeysIsolation verifica que ctx.Keys no se filtra entre requests concurrentes.
func TestRaceKeysIsolation(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/echo/:id", func(ctx *gemrouter.GemContext) {
			id := ctx.Param("id")
			ctx.Set("id", id)
			got, _ := ctx.Get("id")
			if got != id {
				t.Errorf("key bleed: set %q got %q", id, got)
			}
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("%s/echo/%d", srv.URL, i))
			if err != nil {
				t.Errorf("request failed: %v", err)
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
}

// TestRaceRequestIDUnique verifica que los request IDs son únicos bajo concurrencia.
func TestRaceRequestIDUnique(t *testing.T) {
	ids := make(chan string, 50)

	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.Use(gemrouter.Logger)
		r.GET("/ping", func(ctx *gemrouter.GemContext) {
			ids <- ctx.RequestID()
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/ping")
			if err != nil {
				t.Errorf("request failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[string]bool)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate request ID: %q", id)
		}
		seen[id] = true
	}
}

// TestRaceCopyInGoroutine verifica que ctx.Copy() es seguro para usar en una goroutine
// después de que el handler retorna y el ctx original ha sido reciclado al pool.
func TestRaceCopyInGoroutine(t *testing.T) {
	const n = 20
	done := make(chan string, n)

	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/async", func(ctx *gemrouter.GemContext) {
			ctx.Set("token", "abc123")
			cp := ctx.Copy()
			go func() {
				val, _ := cp.Get("token")
				done <- fmt.Sprintf("%v", val)
			}()
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	for range n {
		resp, err := http.Get(srv.URL + "/async")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	for range n {
		if val := <-done; val != "abc123" {
			t.Fatalf("want abc123, got %q", val)
		}
	}
}

// TestRaceMultipleRoutesConcurrent verifica que requests concurrentes a distintas
// rutas no interfieren entre sí.
func TestRaceMultipleRoutesConcurrent(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/a", func(ctx *gemrouter.GemContext) {
			ctx.Set("route", "a")
			got, _ := ctx.Get("route")
			if got != "a" {
				t.Errorf("route /a got wrong key: %v", got)
			}
			ctx.NoContent(http.StatusOK)
		})
		r.GET("/b", func(ctx *gemrouter.GemContext) {
			ctx.Set("route", "b")
			got, _ := ctx.Get("route")
			if got != "b" {
				t.Errorf("route /b got wrong key: %v", got)
			}
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for range n {
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/a")
			if err != nil {
				t.Errorf("request /a failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/b")
			if err != nil {
				t.Errorf("request /b failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
}
