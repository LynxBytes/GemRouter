package gem

import (
	"encoding/json"
	"net/http"
	"testing"
)

// --- ctx.Success default ---

func TestSuccessDefault(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.GET("/", func(ctx *GemContext) {
			ctx.Success(http.StatusOK, JSON{"id": 1})
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
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["id"] == nil {
		t.Fatal("expected 'id' field in default success response")
	}
}

// --- ctx.Fail default: single string ---

func TestFailDefaultSingleString(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.GET("/", func(ctx *GemContext) {
			ctx.Fail(http.StatusBadRequest, "invalid input")
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "invalid input" {
		t.Fatalf("want error='invalid input', got %v", body["error"])
	}
}

// --- ctx.Fail default: multiple strings ---

func TestFailDefaultMultipleStrings(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.GET("/", func(ctx *GemContext) {
			ctx.Fail(http.StatusBadRequest, "name required", "email invalid")
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	errs, ok := body["errors"].([]any)
	if !ok {
		t.Fatalf("want 'errors' array, got %T", body["errors"])
	}
	if len(errs) != 2 {
		t.Fatalf("want 2 errors, got %d", len(errs))
	}
}

// --- ctx.Fail default: ValidationError slice ---

func TestFailDefaultValidationErrors(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		r.POST("/", func(ctx *GemContext) {
			v := NewValidator().
				Check("name", "", "required").
				Check("email", "bad", "email")
			if !v.Valid() {
				ctx.Fail(http.StatusUnprocessableEntity, v.Errors())
			}
		})
	})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	errs, ok := body["errors"].([]any)
	if !ok {
		t.Fatalf("want 'errors' array, got %T", body["errors"])
	}
	if len(errs) != 2 {
		t.Fatalf("want 2 validation errors, got %d", len(errs))
	}
	first := errs[0].(map[string]any)
	if first["field"] == nil || first["message"] == nil {
		t.Fatal("expected field and message in ValidationError")
	}
}

// --- WithResponseFormatter ---

func TestWithResponseFormatter(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		WithResponseFormatter(func(code int, data any) (int, any) {
			return code, JSON{"success": true, "data": data}
		})(r)
		r.GET("/", func(ctx *GemContext) {
			ctx.Success(http.StatusOK, JSON{"id": 42})
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["success"] != true {
		t.Fatalf("want success=true, got %v", body["success"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("want data object, got %T", body["data"])
	}
	if data["id"] != float64(42) {
		t.Fatalf("want data.id=42, got %v", data["id"])
	}
}

// --- WithErrorFormatter: custom multi-error shape ---

func TestWithErrorFormatter(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		WithErrorFormatter(func(code int, errs []any) (int, any) {
			return code, JSON{"success": false, "errors": errs, "code": code}
		})(r)
		r.POST("/", func(ctx *GemContext) {
			v := NewValidator().
				Check("name", "", "required").
				Check("email", "bad", "email")
			ctx.Fail(http.StatusUnprocessableEntity, v.Errors())
		})
	})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["success"] != false {
		t.Fatalf("want success=false, got %v", body["success"])
	}
	if body["code"] != float64(422) {
		t.Fatalf("want code=422, got %v", body["code"])
	}
	if body["errors"] == nil {
		t.Fatal("expected 'errors' field")
	}
}

// --- formatter can override status code ---

func TestResponseFormatterOverridesCode(t *testing.T) {
	srv := newTestServer(func(r *GemRouter) {
		WithResponseFormatter(func(_ int, data any) (int, any) {
			return http.StatusAccepted, data
		})(r)
		r.GET("/", func(ctx *GemContext) {
			ctx.Success(http.StatusOK, JSON{})
		})
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("want 202, got %d", resp.StatusCode)
	}
}
