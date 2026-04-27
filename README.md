# GemRouter

GemRouter is a fast and minimal HTTP router for Go, built on top of [httprouter](https://github.com/julienschmidt/httprouter) with a clean middleware chain, structured logging via `log/slog`, and zero-alloc context pooling.

## Features

- Fast radix tree routing via httprouter
- Zero-alloc context pool (`sync.Pool`)
- Structured logging with `log/slog` — plug in any logger (zap, zerolog, etc.)
- Built-in middlewares: CORS, Recovery, Logger, Timeout, Prometheus
- Route groups with per-group middleware
- Graceful shutdown out of the box
- Sonic JSON (3-5x faster than `encoding/json`)
- `JSON` type alias for ergonomic responses

## Installation

```bash
go get github.com/LynxBytes/GemRouter
```

Requires Go 1.22+

## Quick start

```go
package main

import gemrouter "github.com/LynxBytes/GemRouter"

func main() {
    r := gemrouter.DefaultGemRouter()

    r.GET("/ping", func(ctx *gemrouter.GemContext) {
        ctx.ToJSON(200, gemrouter.JSON{"message": "pong"})
    })

    r.Run()
}
```

## Routers

| Constructor | Middlewares | CORS |
|---|---|---|
| `BasicGemRouter()` | CORS, Recovery | ✓ default |
| `DefaultGemRouter()` | CORS, Recovery, Logger | ✓ default |
| `NewGemRouter(configs...)` | Recovery, Logger | configurable |

```go
r := gemrouter.NewGemRouter(
    gemrouter.WithPort("3000"),
    gemrouter.WithCorsDefault(),
    gemrouter.WithJSONLogger(os.Stdout, slog.LevelInfo),
)
```

## HTTP methods

```go
r.GET("/users/:id", handler)
r.POST("/users", handler)
r.PUT("/users/:id", handler)
r.PATCH("/users/:id", handler)
r.DELETE("/users/:id", handler)
```

## Parameters

```go
// path param
r.GET("/users/:id", func(ctx *gemrouter.GemContext) {
    id := ctx.Param("id")
    ctx.ToJSON(200, gemrouter.JSON{"id": id})
})

// query param
r.GET("/search", func(ctx *gemrouter.GemContext) {
    q := ctx.Query("q")
    ctx.String(200, q)
})

// wildcard
r.GET("/files/*path", handler)
```

## JSON

```go
// write
ctx.ToJSON(200, gemrouter.JSON{"user": "mario", "age": 30})

// read
var body CreateUserRequest
if err := ctx.FromJSON(&body); err != nil {
    ctx.ToJSON(400, gemrouter.JSON{"error": err.Error()})
    return
}
```

## Validation

Built-in validator, no dependencies. Supports `required`, `min=N`, `max=N`, `len=N`, `email`.

```go
r.POST("/users", func(ctx *gemrouter.GemContext) {
    var body CreateUserRequest
    if err := ctx.FromJSON(&body); err != nil {
        ctx.ToJSON(400, gemrouter.JSON{"error": err.Error()})
        return
    }

    v := gemrouter.NewValidator().
        Check("name",  body.Name,  "required,min=2,max=50").
        Check("email", body.Email, "required,email").
        Check("age",   body.Age,   "min=18,max=120")

    if !v.Valid() {
        ctx.ToJSON(400, gemrouter.JSON{"errors": v.Errors()})
        return
    }

    ctx.ToJSON(201, body)
})
```

Error response:

```json
{
  "errors": [
    {"field": "email", "message": "must be a valid email"},
    {"field": "age",   "message": "must be at least 18"}
  ]
}
```

| Rule | Types | Description |
|---|---|---|
| `required` | any | not empty or zero |
| `min=N` | string, int, float64 | min length / min value |
| `max=N` | string, int, float64 | max length / max value |
| `len=N` | string | exact length |
| `email` | string | valid email format |

## Middlewares

```go
// global
r.Use(MyMiddleware)

// per route group
api := r.Group("/api", AuthMiddleware)
api.GET("/users", handler)
```

Writing a middleware:

```go
func AuthMiddleware(next gemrouter.GemHandler) gemrouter.GemHandler {
    return func(ctx *gemrouter.GemContext) {
        token := ctx.Header("Authorization")
        if !isValid(token) {
            ctx.ToJSON(401, gemrouter.JSON{"error": "unauthorized"})
            return // chain stops here
        }
        next(ctx)
    }
}
```

## Route groups

```go
api := r.Group("/api")

v1 := api.Group("/v1", AuthMiddleware)
v1.GET("/users", getUsers)
v1.POST("/users", createUser)

v2 := api.Group("/v2", AuthMiddleware, RateLimitMiddleware)
v2.GET("/users", getUsersV2)
```

## Built-in middlewares

```go
// CORS
gemrouter.WithCors(&gemrouter.CorsConfig{
    AllowOrigins: []string{"https://example.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders: []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
})

// Timeout
r.Use(gemrouter.Timeout(5 * time.Second))

// Prometheus metrics
r := gemrouter.NewGemRouter(
    gemrouter.WithPrometheus("/metrics"),
)
```

## Logging

```go
// text (default)
gemrouter.WithTextLogger(os.Stdout, slog.LevelInfo)

// JSON
gemrouter.WithJSONLogger(os.Stdout, slog.LevelInfo)

// custom slog logger
gemrouter.WithLogger(mySlogLogger)

// use logger inside handler
ctx.Logger.Info("user created", slog.String("id", user.ID))
```

## Context store

```go
// set/get arbitrary values across middlewares
ctx.Set("userID", "123")
val, ok := ctx.Get("userID")

// typed fields
ctx.Store.RequestID
ctx.Store.UserID
```

## Cookies

```go
ctx.SetCookie("session", token, 3600, "/", "", true, true)
val, err := ctx.Cookie("session")
ctx.DeleteCookie("session")
```

## Custom handlers

```go
r := gemrouter.NewGemRouter(
    gemrouter.WithNotFound(func(ctx *gemrouter.GemContext) {
        ctx.ToJSON(404, gemrouter.JSON{"error": "not found"})
    }),
    gemrouter.WithMethodNotAllowed(func(ctx *gemrouter.GemContext) {
        ctx.ToJSON(405, gemrouter.JSON{"error": "method not allowed"})
    }),
    gemrouter.WithHealth(func(ctx *gemrouter.GemContext) {
        ctx.ToJSON(200, gemrouter.JSON{"status": "ok"})
    }),
)
```

## Graceful shutdown

Built-in. `Run()` listens for `SIGINT` and `SIGTERM` and shuts down cleanly.

```go
r := gemrouter.NewGemRouter(
    gemrouter.WithShutdownTimeout(10 * time.Second),
)
r.Run() // blocks until signal
```

## Benchmarks

```
BenchmarkRouter_Ping-11          15937568     74 ns/op      16 B/op    1 allocs/op
BenchmarkRouter_Param-11         12995856     93 ns/op      48 B/op    2 allocs/op
BenchmarkRouter_ParallelPing-11  64808960     24 ns/op      16 B/op    1 allocs/op
BenchmarkRouter_NoContent-11     31490965     39 ns/op       0 B/op    0 allocs/op
BenchmarkRoutes_500-11           15444356     76 ns/op      32 B/op    1 allocs/op
```

## License

MIT
