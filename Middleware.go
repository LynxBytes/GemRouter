package gemrouter

import (
	"log"
	"time"
)

type Middleware func(GemHandler) GemHandler

func Recovery(next GemHandler) GemHandler {
	return func(ctx *GemContext) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				ctx.ToJSON(500, map[string]string{"error": "internal server error"})
			}
		}()
		next(ctx)
	}
}

func Logger(next GemHandler) GemHandler {
	return func(ctx *GemContext) {
		start := time.Now()
		log.Printf("➡️ %s %s", ctx.Request.Method, ctx.Request.URL.Path)

		next(ctx)

		log.Printf("⬅️ %s %s %d (%v)",
			ctx.Request.Method,
			ctx.Request.URL.Path,
			ctx.StatusCode(),
			time.Since(start),
		)
	}
}
