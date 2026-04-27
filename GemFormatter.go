package gem

type ResponseFormatter func(code int, data any) (int, any)

type ErrorFormatter func(code int, errs []any) (int, any)

var defaultResponseFormatter ResponseFormatter = func(code int, data any) (int, any) {
	return code, data
}

var defaultErrorFormatter ErrorFormatter = func(code int, errs []any) (int, any) {
	if len(errs) == 1 {
		if msg, ok := errs[0].(string); ok {
			return code, JSON{"error": msg}
		}
		return code, JSON{"errors": errs[0]}
	}
	return code, JSON{"errors": errs}
}
