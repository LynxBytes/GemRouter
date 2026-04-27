package gem

import (
	"io"
	"log/slog"
	"time"
)

type GemConfig func(router *GemRouter)

var defaultCors = &CorsConfig{
	AllowOrigins: []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:8080",
	},
	AllowMethods: []string{
		"GET",
		"POST",
		"PUT",
		"PATCH",
		"DELETE",
		"OPTIONS",
	},
	AllowHeaders: []string{
		"Content-Type",
		"Authorization",
	},
	ExposeHeaders:    nil,
	AllowCredentials: true,
	MaxAge:           3600,
}

type CorsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

func WithName(name string) GemConfig {
	return func(router *GemRouter) {
		router.name = name
	}
}

func WithVersion(version string) GemConfig {
	return func(router *GemRouter) {
		router.version = version
	}
}

func WithAddr(addr string) GemConfig {
	return func(router *GemRouter) {
		router.Addr = addr
	}
}

func WithPort(port string) GemConfig {
	return func(router *GemRouter) {
		router.Port = port
	}
}

func WithMiddlewares(middlewares []Middleware) GemConfig {
	return func(router *GemRouter) {
		router.middlewares = middlewares
		router.corsSet = false
	}
}

func WithMiddleware(middleware Middleware) GemConfig {
	return func(router *GemRouter) {
		router.middlewares = append(router.middlewares, middleware)
	}
}

func WithMethodNotAllowed(handler GemHandler) GemConfig {
	return func(router *GemRouter) {
		router.MethodNotAllowed = handler
	}
}

func WithHealth(handler GemHandler) GemConfig {
	return func(router *GemRouter) {
		router.Health = handler
	}
}

func WithLogger(l *slog.Logger) GemConfig {
	return func(router *GemRouter) {
		router.logger = l
	}
}

func WithShutdownTimeout(d time.Duration) GemConfig {
	return func(router *GemRouter) {
		router.shutdownTimeout = d
	}
}

func WithCors(cfg *CorsConfig) GemConfig {
	return func(router *GemRouter) {
		if router.corsSet {
			panic("gemrouter: CORS already configured — WithCors called more than once")
		}
		router.corsSet = true
		router.middlewares = append(router.middlewares, Cors(cfg))
	}
}

func WithCorsDefault() GemConfig {
	return WithCors(defaultCors)
}

func WithTrustedProxy() GemConfig {
	return func(router *GemRouter) {
		router.trustProxy = true
	}
}

func WithReadTimeout(d time.Duration) GemConfig {
	return func(router *GemRouter) {
		router.readTimeout = d
	}
}

func WithWriteTimeout(d time.Duration) GemConfig {
	return func(router *GemRouter) {
		router.writeTimeout = d
	}
}

func WithIdleTimeout(d time.Duration) GemConfig {
	return func(router *GemRouter) {
		router.idleTimeout = d
	}
}

func WithLogCloser(c io.Closer) GemConfig {
	return func(router *GemRouter) {
		router.logCloser = c
	}
}

func WithResponseFormatter(f ResponseFormatter) GemConfig {
	return func(router *GemRouter) {
		router.responseFormatter = f
	}
}

func WithErrorFormatter(f ErrorFormatter) GemConfig {
	return func(router *GemRouter) {
		router.errorFormatter = f
	}
}
