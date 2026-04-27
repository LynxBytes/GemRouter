package gem

// ResponseFormatter transforms a success response before writing.
// Returns the final status code and body to serialize.
type ResponseFormatter func(code int, data any) (int, any)

// ErrorFormatter transforms an error response before writing.
// errs contains one or more errors: strings, ValidationError slices, or any custom type.
// Returns the final status code and body to serialize.
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
