package gem

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func TestRaceConcurrentRequests(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.GET("/ping", func(ctx *GemContext) {
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

func TestRaceKeysIsolation(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.GET("/echo/:id", func(ctx *GemContext) {
			id := ctx.Param("id")
			ctx.Set("id", id)
			got := ctx.Get("id")
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

func TestRaceRequestIDUnique(t *testing.T) {
	ids := make(chan string, 50)

	srv := newTestServer(func(r *GemRouter) {
		r.Use(Logger)
		r.GET("/ping", func(ctx *GemContext) {
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

func TestRaceCopyInGoroutine(t *testing.T) {
	const n = 20
	done := make(chan string, n)

	srv := newTestServer(func(r *GemRouter) {
		r.GET("/async", func(ctx *GemContext) {
			ctx.Set("token", "abc123")
			cp := ctx.Copy()
			go func() {
				val := cp.Get("token")
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

func TestRaceMultipleRoutesConcurrent(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.GET("/a", func(ctx *GemContext) {
			ctx.Set("route", "a")
			got := ctx.Get("route")
			if got != "a" {
				t.Errorf("route /a got wrong key: %v", got)
			}
			ctx.NoContent(http.StatusOK)
		})
		r.GET("/b", func(ctx *GemContext) {
			ctx.Set("route", "b")
			got := ctx.Get("route")
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
