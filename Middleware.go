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
		ctx.Set("request_id", reqID)

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

func Cors(cfg *CorsConfig) Middleware {
	if cfg == nil {
		cfg = defaultCors
	}

	return func(next GemHandler) GemHandler {
		return func(ctx *GemContext) {
			origin := ctx.Request.Header.Get("Origin")
			if origin == "" {
				next(ctx)
				return
			}

			isPreflight := ctx.Request.Method == http.MethodOptions &&
				ctx.Request.Header.Get("Access-Control-Request-Method") != ""

			headers := ctx.Writer.Header()
			addVary(headers, "Origin")
			if isPreflight {
				addVary(headers, "Access-Control-Request-Method", "Access-Control-Request-Headers")
			}

			allowed := false
			for _, allowedOrigin := range cfg.AllowOrigins {
				if allowedOrigin == "*" {
					if cfg.AllowCredentials {
						headers.Set("Access-Control-Allow-Origin", origin)
					} else {
						headers.Set("Access-Control-Allow-Origin", "*")
					}
					allowed = true
					break
				}
				if allowedOrigin == origin {
					headers.Set("Access-Control-Allow-Origin", origin)
					allowed = true
					break
				}
			}

			if !allowed && isPreflight {
				ctx.Writer.WriteHeader(http.StatusForbidden)
				return
			}

			if len(cfg.AllowMethods) > 0 {
				headers.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
			}
			if len(cfg.AllowHeaders) > 0 {
				headers.Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
			} else if reqHeaders := ctx.Request.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
				headers.Set("Access-Control-Allow-Headers", reqHeaders)
			}
			if len(cfg.ExposeHeaders) > 0 {
				headers.Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
			}
			if cfg.AllowCredentials {
				headers.Set("Access-Control-Allow-Credentials", "true")
			}
			if cfg.MaxAge > 0 {
				headers.Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
			}

			if isPreflight {
				ctx.Writer.WriteHeader(http.StatusNoContent)
				return
			}

			next(ctx)
		}
	}
}

func addVary(h http.Header, values ...string) {
	existing := make(map[string]bool)
	for _, v := range h.Values("Vary") {
		existing[strings.TrimSpace(v)] = true
	}
	for _, v := range values {
		if !existing[v] {
			h.Add("Vary", v)
		}
	}
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
