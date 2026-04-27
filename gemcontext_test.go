package gemrouter_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	gemrouter "github.com/LynxBytes/GemRouter"
)

// --- Response methods ---

func TestString(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, "hello")
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Fatalf("want 'hello', got %q", string(body))
	}
}

func TestToJSON(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.ToJSON(http.StatusCreated, map[string]string{"key": "value"})
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("want application/json, got %q", ct)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["key"] != "value" {
		t.Fatalf("want 'value', got %q", result["key"])
	}
}

func TestNoContent(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.DELETE("/", func(ctx *gemrouter.GemContext) {
			ctx.NoContent(http.StatusNoContent)
		})
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
}

func TestOK(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.OK()
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestNOTFOUND(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.NOTFOUND()
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}

func TestStatusCode(t *testing.T) {
	got := make(chan int, 1)
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusAccepted, "")
			got <- ctx.StatusCode()
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/") //nolint
	if code := <-got; code != http.StatusAccepted {
		t.Fatalf("want 202, got %d", code)
	}
}

// --- Request reading ---

func TestQuery(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, ctx.Query("name"))
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/?name=mario")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "mario" {
		t.Fatalf("want 'mario', got %q", string(body))
	}
}

func TestHeader(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, ctx.Header("X-Custom"))
		})
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	req.Header.Set("X-Custom", "gem")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "gem" {
		t.Fatalf("want 'gem', got %q", string(body))
	}
}

func TestMethod(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.POST("/", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, ctx.Method())
		})
	})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != http.MethodPost {
		t.Fatalf("want 'POST', got %q", string(body))
	}
}

func TestPath(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/foo/bar", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, ctx.Path())
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/foo/bar")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "/foo/bar" {
		t.Fatalf("want '/foo/bar', got %q", string(body))
	}
}

func TestParam(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/users/:id", func(ctx *gemrouter.GemContext) {
			ctx.String(http.StatusOK, ctx.Param("id"))
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "42" {
		t.Fatalf("want '42', got %q", string(body))
	}
}

func TestFromJSON(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.POST("/", func(ctx *gemrouter.GemContext) {
			var payload map[string]string
			if err := ctx.FromJSON(&payload); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}
			ctx.String(http.StatusOK, payload["msg"])
		})
	})
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"msg": "hello"})
	resp, err := http.Post(srv.URL+"/", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if string(b) != "hello" {
		t.Fatalf("want 'hello', got %q", string(b))
	}
}

func TestFromJSONBodyTooLarge(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.POST("/", func(ctx *gemrouter.GemContext) {
			var payload map[string]string
			if err := ctx.FromJSON(&payload); err != nil {
				ctx.NoContent(http.StatusRequestEntityTooLarge)
				return
			}
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	big := make([]byte, 5<<20) // 5 MB > 4 MB limit
	for i := range big {
		big[i] = 'a'
	}
	resp, err := http.Post(srv.URL+"/", "application/json", bytes.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413, got %d", resp.StatusCode)
	}
}

// --- Key store ---

func TestSetGet(t *testing.T) {
	got := make(chan any, 1)
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.Set("user", "mario")
			val, ok := ctx.Get("user")
			if !ok {
				got <- nil
				return
			}
			got <- val
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/") //nolint
	val := <-got
	if val != "mario" {
		t.Fatalf("want 'mario', got %v", val)
	}
}

func TestGetMissing(t *testing.T) {
	got := make(chan bool, 1)
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			_, ok := ctx.Get("missing")
			got <- ok
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	http.Get(srv.URL + "/") //nolint
	if ok := <-got; ok {
		t.Fatal("want false for missing key, got true")
	}
}

// --- Cookies ---

func newClientWithJar() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func TestSetCookie(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.SetCookie("session", "abc123", 3600, "/", "", false, true)
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.Value == "abc123" {
			return
		}
	}
	t.Fatal("cookie 'session' not found in response")
}

func TestCookie(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/set", func(ctx *gemrouter.GemContext) {
			ctx.SetCookie("token", "xyz", 3600, "/", "", false, false)
			ctx.NoContent(http.StatusOK)
		})
		r.GET("/get", func(ctx *gemrouter.GemContext) {
			val, err := ctx.Cookie("token")
			if err != nil {
				ctx.String(http.StatusBadRequest, "no cookie")
				return
			}
			ctx.String(http.StatusOK, val)
		})
	})
	defer srv.Close()

	client := newClientWithJar()
	client.Get(srv.URL + "/set") //nolint

	resp, err := client.Get(srv.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "xyz" {
		t.Fatalf("want 'xyz', got %q", string(body))
	}
}

func TestDeleteCookie(t *testing.T) {
	srv := newTestServer(func(r *gemrouter.GemRouter) {
		r.GET("/", func(ctx *gemrouter.GemContext) {
			ctx.DeleteCookie("session")
			ctx.NoContent(http.StatusOK)
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.MaxAge == -1 {
			return
		}
	}
	t.Fatal("expected cookie 'session' with MaxAge=-1")
}
