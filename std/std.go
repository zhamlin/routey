package std

import (
	"net/http"
)

type Mux struct {
	*http.ServeMux
}

func (m Mux) Handle(method, pattern string, handler http.Handler) {
	if method != "" {
		pattern = method + " " + pattern
	}

	m.ServeMux.Handle(pattern, handler)
}

func (m Mux) Param(name string, r *http.Request) string {
	return r.PathValue(name)
}
