package gemrouter_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gemrouter "github.com/LynxBytes/GemRouter"
)

func newTestServer(setup func(*gemrouter.GemRouter)) *httptest.Server {
	r := gemrouter.NewGemRouter(gemrouter.WithMiddlewares([]gemrouter.Middleware{}))
	setup(r)
	return httptest.NewServer(r.Handler())
}

// --- Constructor ---

func TestNewGemRouter(t *testing.T) {
	r := gemrouter.DefaultGemRouter()
	if r == nil {
		t.Fatal("router should not be nil")
	}
}

func TestCustomGemRouter(t *testing.T) {
	r := gemrouter.NewGemRouter(
		gemrouter.WithMiddleware(func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) { next(ctx) }
		}),
	)
	if r == nil {
		t.Fatal("router should not be nil")
	}
}

// --- HTTP methods ---

func TestGETRoute(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/get", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestPOSTRoute(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.POST("/post", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusCreated) })
	})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/post", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
}

func TestPUTRoute(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.PUT("/put", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/put", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestPATCHRoute(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.PATCH("/patch", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/patch", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestDELETERoute(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.DELETE("/del", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusNoContent) })
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/del", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
}

// --- Middleware ---

func TestMiddlewareExecutes(t *testing.T) {
	order := make([]string, 0, 2)
	ch := make(chan []string, 1)

	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.Use(func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) {
				order = append(order, "mid")
				next(ctx)
			}
		})
		r.GET("/", func(ctx *gemrouter.GemContext) {
			order = append(order, "handler")
			ch <- order
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/") //nolint
	got := <-ch
	if len(got) != 2 || got[0] != "mid" || got[1] != "handler" {
		t.Fatalf("want [mid handler], got %v", got)
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	order := make([]string, 0, 3)
	ch := make(chan []string, 1)

	mid := func(label string) gemrouter.Middleware {
		return func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) {
				order = append(order, label)
				next(ctx)
			}
		}
	}

	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.Use(mid("first"))
		r.Use(mid("second"))
		r.GET("/", func(ctx *gemrouter.GemContext) {
			order = append(order, "handler")
			ch <- order
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/") //nolint
	got := <-ch
	if len(got) != 3 || got[0] != "first" || got[1] != "second" || got[2] != "handler" {
		t.Fatalf("want [first second handler], got %v", got)
	}
}

// --- Group ---

func TestGroupPrefix(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		api := r.Group("/api")
		api.GET("/users", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, "users")
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "users" {
		t.Fatalf("want 'users', got %q", string(body))
	}
}

func TestGroupMiddleware(t *testing.T) {
	ch := make(chan bool, 1)

	srv := newTestServer(func(r *gemrouter.GemRouter) {
		mid := func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) {
				ch <- true
				next(ctx)
			}
		}
		api := r.Group("/api", mid)
		api.GET("/ping", func(ctx *gemrouter.GemContext) {
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/api/ping") //nolint
	if ok := <-ch; !ok {
		t.Fatal("group middleware did not execute")
	}
}

// --- Group.Use ---

func TestGroupUse(t *testing.T) {
	ch := make(chan bool, 1)
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		api := r.Group("/api")
		api.Use(func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) {
				ch <- true
				next(ctx)
			}
		})
		api.GET("/ping", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	})
	defer srv.Close()

	http.Get(srv.URL + "/api/ping") //nolint
	if ok := <-ch; !ok {
		t.Fatal("group middleware added via Use() did not execute")
	}
}

// --- Group.Group nested ---

func TestNestedGroup(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		api := r.Group("/api")
		v1 := api.Group("/v1")
		v1.GET("/users", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, "users")
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "users" {
		t.Fatalf("want 'users', got %q", string(body))
	}
}

func TestNestedGroupInheritsMiddlewares(t *testing.T) {
	ch := make(chan string, 2)
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		api := r.Group("/api", func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) { ch <- "api"; next(ctx) }
		})
		v1 := api.Group("/v1", func(next gemrouter.GemHandler) gemrouter.GemHandler {
			return func(ctx *gemrouter.GemContext) { ch <- "v1"; next(ctx) }
		})
		v1.GET("/ping", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	})
	defer srv.Close()

	http.Get(srv.URL + "/api/v1/ping") //nolint
	got := []string{<-ch, <-ch}
	if got[0] != "api" || got[1] != "v1" {
		t.Fatalf("want [api v1], got %v", got)
	}
}

// --- StatusCode default ---

func TestStatusCodeDefaultOK(t *testing.T) {
	ch := make(chan int, 1)
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ch <- ctx.StatusCode()
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/") //nolint
	if code := <-ch; code != http.StatusOK {
		t.Fatalf("want 200 before any write, got %d", code)
	}
}

// --- Handler ---

func TestHandler(t *testing.T) {
	r := gemrouter.NewGemRouter(gemrouter.WithMiddlewares([]gemrouter.Middleware{}))
	r.GET("/ping", func(ctx *gemrouter.GemContext) {
		ctx.String(http.StatusOK, "pong")
	})

	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	r.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if rr.Body.String() != "pong" {
		t.Fatalf("want 'pong', got %q", rr.Body.String())
	}
}
