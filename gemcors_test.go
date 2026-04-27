package gem_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gemrouter "github.com/LynxBytes/GemRouter"
)

func newCorsServer(cfg *gemrouter.CorsConfig) *httptest.Server {
	r := gemrouter.NewGemRouter(
		gemrouter.WithMiddlewares([]gemrouter.Middleware{}),
		gemrouter.WithCors(cfg),
	)
	r.GET("/", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	return httptest.NewServer(r.Handler())
}

func preflightReq(url, origin string) *http.Request {
	req, _ := http.NewRequest(http.MethodOptions, url+"/", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	return req
}

func simpleReq(url, origin string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, url+"/", nil)
	req.Header.Set("Origin", origin)
	return req
}

// --- No Origin ---

func TestCorsNoOriginPassthrough(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("want no ACAO header, got %q", got)
	}
}

// --- Allowed origin ---

func TestCorsAllowedOrigin(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("want 'https://example.com', got %q", got)
	}
}

func TestCorsDisallowedOriginSimpleRequest(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://evil.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("want no ACAO header for disallowed origin, got %q", got)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("simple request should still pass, got %d", resp.StatusCode)
	}
}

// --- Wildcard ---

func TestCorsWildcardWithoutCredentials(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"*"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://anything.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("want '*', got %q", got)
	}
}

func TestCorsWildcardWithCredentialsUsesRealOrigin(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://anything.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("* + credentials is invalid, should not return *")
	}
}

// --- Preflight ---

func TestCorsPreflight204(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(preflightReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
}

func TestCorsPreflightDisallowedOrigin403(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(preflightReq(srv.URL, "https://evil.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

func TestCorsOptionsWithoutACRMIsNotPreflight(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/", nil)
	req.Header.Set("Origin", "https://example.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		t.Fatal("OPTIONS without Access-Control-Request-Method should not be treated as preflight")
	}
}

// --- Vary ---

func TestCorsVaryOriginAlwaysSet(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	found := false
	for _, v := range resp.Header.Values("Vary") {
		if v == "Origin" {
			found = true
		}
	}
	if !found {
		t.Fatal("want Vary: Origin header")
	}
}

func TestCorsVaryExtraHeadersOnPreflight(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{AllowOrigins: []string{"https://example.com"}})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(preflightReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	vary := resp.Header.Values("Vary")
	has := func(v string) bool {
		for _, h := range vary {
			if h == v {
				return true
			}
		}
		return false
	}

	if !has("Access-Control-Request-Method") {
		t.Fatal("want Vary: Access-Control-Request-Method on preflight")
	}
	if !has("Access-Control-Request-Headers") {
		t.Fatal("want Vary: Access-Control-Request-Headers on preflight")
	}
}

// --- Headers individuales ---

func TestCorsAllowMethods(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowMethods: []string{"GET", "POST"},
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("want 'GET, POST', got %q", got)
	}
}

func TestCorsAllowHeaders(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("want 'Content-Type, Authorization', got %q", got)
	}
}

func TestCorsExposeHeaders(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins:  []string{"https://example.com"},
		ExposeHeaders: []string{"X-Request-Id"},
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
		t.Fatalf("want 'X-Request-Id', got %q", got)
	}
}

func TestCorsAllowCredentials(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("want 'true', got %q", got)
	}
}

func TestCorsMaxAge(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins: []string{"https://example.com"},
		MaxAge:       3600,
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(preflightReq(srv.URL, "https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Max-Age"); got != "3600" {
		t.Fatalf("want '3600', got %q", got)
	}
}

// --- nil config guard ---

func TestCorsNilConfigUsesDefault(t *testing.T) {
	r := gemrouter.NewGemRouter(
		gemrouter.WithMiddlewares([]gemrouter.Middleware{}),
		gemrouter.WithCors(nil),
	)
	r.GET("/", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "http://localhost:3000"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("nil config should use defaultCors, got ACAO: %q", got)
	}
}

// --- wildcard + credentials refleja origin real ---

func TestCorsWildcardWithCredentialsReflectsOrigin(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})
	defer srv.Close()

	resp, err := http.DefaultClient.Do(simpleReq(srv.URL, "https://anything.com"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://anything.com" {
		t.Fatalf("want reflected origin 'https://anything.com', got %q", got)
	}
}

// --- dynamic AllowHeaders fallback ---

func TestCorsDynamicAllowHeaders(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins: []string{"https://example.com"},
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "X-Custom-Token" {
		t.Fatalf("want 'X-Custom-Token' from request, got %q", got)
	}
}

func TestCorsStaticAllowHeadersOverridesDynamic(t *testing.T) {
	srv := newCorsServer(&gemrouter.CorsConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowHeaders: []string{"Authorization"},
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "Authorization" {
		t.Fatalf("static AllowHeaders should take priority, got %q", got)
	}
}

// --- double CORS guard ---

func TestWithCorsPanicsIfCalledTwice(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("want panic when WithCors called twice, got none")
		}
	}()

	gemrouter.NewGemRouter(
		gemrouter.WithCors(&gemrouter.CorsConfig{AllowOrigins: []string{"https://a.com"}}),
		gemrouter.WithCors(&gemrouter.CorsConfig{AllowOrigins: []string{"https://b.com"}}),
	)
}

// --- WithCorsDefault ---

func TestWithCorsDefault(t *testing.T) {
	r := gemrouter.NewGemRouter(
		gemrouter.WithMiddlewares([]gemrouter.Middleware{}),
		gemrouter.WithCorsDefault(),
	)
	r.GET("/", func(ctx *gemrouter.GemContext) { ctx.NoContent(http.StatusOK) })
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("want 'http://localhost:3000', got %q", got)
	}
}
