package gemrouter

type GemConfig func(router *GemRouter)

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
		router.Middlewares = middlewares
	}
}

func WithMiddleware(middleware Middleware) GemConfig {
	return func(router *GemRouter) {
		router.Middlewares = append(router.Middlewares, middleware)
	}
}

func WithNotFound(handler GemHandler) GemConfig {
	return func(router *GemRouter) {
		router.NotFound = handler
	}
}

func WithHealth(handler GemHandler) GemConfig {
	return func(router *GemRouter) {
		router.Health = handler
	}
}
