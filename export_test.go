package gem

import (
	"io"
	"net/http"
)

func (r *GemRouter) Handler() http.Handler {
	return r.mux
}

func (r *GemRouter) LogCloser() io.Closer {
	return r.logCloser
}
