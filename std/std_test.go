package std_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhamlin/routey/std"
)

func TestParam(t *testing.T) {
	r := std.Mux{&http.ServeMux{}}
	want := "value"

	r.Handle(
		http.MethodGet,
		"/{test}",
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			got := r.Param("test", req)
			if got != want {
				t.Errorf("got path value: %s, want: %s", got, want)
			}
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/"+want, nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
}
