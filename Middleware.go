package gemrouter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type Middleware func(GemHandler) GemHandler

func Recovery(next GemHandler) GemHandler {
	return func(ctx *GemContext) {
		defer func() {
			if r := recover(); r != nil {
				ctx.Logger.Error("panic recovered", map[string]any{
					"error": r,
					"stack": string(debug.Stack()),
				})
				if !ctx.rw.written {
					ctx.ToJSON(500, map[string]string{"error": "internal server error"})
				}
			}
		}()
		next(ctx)
	}
}

func Logger(next GemHandler) GemHandler {
	return func(ctx *GemContext) {
		start := time.Now()
		reqID := newRequestID()
		ctx.Store.RequestID = reqID

		ctx.Logger.Info("→", map[string]any{
			"request_id": reqID,
			"method":     ctx.Request.Method,
			"path":       ctx.Request.URL.Path,
			"ip":         clientIP(ctx.Request, ctx.trustProxy),
			"user_agent": ctx.Request.UserAgent(),
		})

		next(ctx)

		ctx.Logger.Info("←", map[string]any{
			"request_id": reqID,
			"status":     ctx.StatusCode(),
			"latency":    time.Since(start).String(),
		})
	}
}

func Timeout(d time.Duration) Middleware {
	return func(next GemHandler) GemHandler {
		return func(ctx *GemContext) {
			c, cancel := context.WithTimeout(ctx.Request.Context(), d)
			defer cancel()
			ctx.Request = ctx.Request.WithContext(c)
			next(ctx)
		}
	}
}

type corsRuntime struct {
	allowAll      bool
	origins       map[string]struct{}
	allowMethods  string
	allowHeaders  string
	exposeHeaders string
	maxAge        string
}

func buildCorsRuntime(cfg *CorsConfig) *corsRuntime {
	rt := &corsRuntime{
		origins: make(map[string]struct{}, len(cfg.AllowOrigins)),
	}
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			rt.allowAll = true
			continue
		}
		rt.origins[o] = struct{}{}
	}
	if len(cfg.AllowMethods) > 0 {
		rt.allowMethods = strings.Join(cfg.AllowMethods, ", ")
	}
	if len(cfg.AllowHeaders) > 0 {
		rt.allowHeaders = strings.Join(cfg.AllowHeaders, ", ")
	}
	if len(cfg.ExposeHeaders) > 0 {
		rt.exposeHeaders = strings.Join(cfg.ExposeHeaders, ", ")
	}
	if cfg.MaxAge > 0 {
		rt.maxAge = strconv.Itoa(cfg.MaxAge)
	}
	return rt
}

func Cors(cfg *CorsConfig) Middleware {
	if cfg == nil {
		cfg = defaultCors
	}
	rt := buildCorsRuntime(cfg)

	return func(next GemHandler) GemHandler {
		return func(ctx *GemContext) {
			req := ctx.Request
			origin := req.Header.Get("Origin")

			if origin == "" {
				next(ctx)
				return
			}

			h := ctx.Writer.Header()
			h.Add("Vary", "Origin")

			isPreflight := req.Method == http.MethodOptions &&
				req.Header.Get("Access-Control-Request-Method") != ""

			if isPreflight {
				h.Add("Vary", "Access-Control-Request-Method")
				h.Add("Vary", "Access-Control-Request-Headers")
			}

			allowed := rt.allowAll
			if !allowed {
				_, allowed = rt.origins[origin]
			}

			if !allowed {
				if isPreflight {
					ctx.Writer.WriteHeader(http.StatusForbidden)
					return
				}
				next(ctx)
				return
			}

			if rt.allowAll {
				if cfg.AllowCredentials {
					h.Set("Access-Control-Allow-Origin", origin)
				} else {
					h.Set("Access-Control-Allow-Origin", "*")
				}
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
			}

			if rt.allowMethods != "" {
				h.Set("Access-Control-Allow-Methods", rt.allowMethods)
			}
			if rt.allowHeaders != "" {
				h.Set("Access-Control-Allow-Headers", rt.allowHeaders)
			} else if isPreflight {
				if reqH := req.Header.Get("Access-Control-Request-Headers"); reqH != "" {
					h.Set("Access-Control-Allow-Headers", reqH)
				}
			}
			if rt.exposeHeaders != "" {
				h.Set("Access-Control-Expose-Headers", rt.exposeHeaders)
			}
			if cfg.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			if isPreflight {
				if rt.maxAge != "" {
					h.Set("Access-Control-Max-Age", rt.maxAge)
				}
				ctx.Writer.WriteHeader(http.StatusNoContent)
				return
			}

			next(ctx)
		}
	}
}

func WithPrometheus(metricsPath string) GemConfig {
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	return func(r *GemRouter) {
		m := newGemMetrics()
		r.middlewares = append(r.middlewares, m.middleware())
		handler := m.handler()
		r.mux.HandleFunc("GET "+metricsPath, func(w http.ResponseWriter, req *http.Request) {
			ctx := r.newContext(w, req)
			defer r.releaseContext(ctx)
			handler(ctx)
		})
	}
}

func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			return strings.TrimSpace(strings.Split(ip, ",")[0])
		}
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
