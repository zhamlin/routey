package tests_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/openapi3"
)

func BenchmarkPathParam(b *testing.B) {
	stdMux := http.NewServeMux()
	stdMux.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		r.PathValue("id")
		w.WriteHeader(http.StatusCreated)
	})

	r := routey.New()
	openAPIRouter, _ := openapi3.NewRouter()

	type Params struct {
		w  http.ResponseWriter
		ID routey.Path[string]
	}

	h := func(p Params) (any, error) {
		p.w.WriteHeader(http.StatusCreated)
		return nil, nil
	}
	routey.Get(r, "/{id}", h)
	routey.Get(openAPIRouter, "/{id}", h)

	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			name:    "std router",
			handler: stdMux,
		},
		{
			name:    "router",
			handler: r,
		},
		{
			name:    "openapi router",
			handler: openAPIRouter,
		},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/123", nil)

			for b.Loop() {
				test.handler.ServeHTTP(resp, req)

				if resp.Code != http.StatusCreated {
					b.Fatal("incorrect status code")
				}
			}
		})
	}
}

func BenchmarkQueryParam(b *testing.B) {
	stdMux := http.NewServeMux()
	stdMux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		_ = values.Get("value")

		w.WriteHeader(http.StatusCreated)
	})

	type Params struct {
		w     http.ResponseWriter
		Value routey.Query[string]
	}

	r := routey.New()

	openAPIRouter := routey.New()
	openapi3.AddSpecToRouter(openAPIRouter, openapi3.AddSpecToRouterOpts{})

	h := func(p Params) (any, error) {
		p.w.WriteHeader(http.StatusCreated)
		return nil, nil
	}

	routey.Get(r, "/", h)
	routey.Get(openAPIRouter, "/", h)

	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			name:    "std router",
			handler: stdMux,
		},
		{
			name:    "router",
			handler: r,
		},
		{
			name:    "openapi router",
			handler: r,
		},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/?value=1", nil)

			for b.Loop() {
				// performance included in benchmark
				req = req.WithContext(b.Context())
				test.handler.ServeHTTP(resp, req)

				if resp.Code != http.StatusCreated {
					b.Fatal("incorrect status code")
				}
			}
		})
	}
}
