package gem

import "net/http"

func (r *GemRouter) Handler() http.Handler {
	return r.mux
}
