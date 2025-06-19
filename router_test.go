package routey_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

func expectErrSink(t *testing.T, want any) func(error) {
	t.Helper()

	gotErr := test.WantAfterTest(t, false, true, "expected an error, got none")
	return func(err error) {
		test.WantError(t, err, want)
		*gotErr = true
	}
}

func newRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), method, path, body)
}

func newTestRouter(t *testing.T) *routey.Router {
	t.Helper()

	r := routey.New()
	r.Response = func(w http.ResponseWriter, _ *http.Request, resp extractor.Response) {
		test.NoError(t, resp.Error, "newTestRouter: Response")
	}
	r.ErrorSink = func(err error) {
		test.NoError(t, err, "newTestRouter: ErrorSink")
	}
	return r
}

func compareRespStatus(t *testing.T, r http.Handler, req *http.Request, want int) {
	t.Helper()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := w.Result().StatusCode
	if got != want {
		t.Errorf(
			"Request(url=%q,method=%q): wanted status code: %v, got: %v",
			req.URL, req.Method, want, got,
		)
	}
}

func TestRouter_HandlerFunc(t *testing.T) {
	want := http.StatusCreated
	r := routey.New()
	r.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(want)
	})

	req := newRequest(t, http.MethodGet, "/", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_MethodHandlers(t *testing.T) {
	want := http.StatusCreated
	h := func(w struct{ http.ResponseWriter }) (any, error) {
		w.WriteHeader(want)
		return nil, nil
	}

	path := "/foo"
	r := routey.New()

	routey.Get(r, path, h)
	routey.Put(r, path, h)
	routey.Post(r, path, h)
	routey.Patch(r, path, h)
	routey.Delete(r, path, h)

	methods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodPost,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range methods {
		req := newRequest(t, method, path, nil)
		compareRespStatus(t, r, req, want)
	}
}

func TestRouter_UnknownFieldErrorNested(t *testing.T) {
	type doubleNestedInput struct {
		Value int
	}
	type nestedInput struct {
		Nested doubleNestedInput
	}
	type input struct {
		Nested nestedInput
	}

	testHandler := func(input) (any, error) { return nil, nil }
	r := newTestRouter(t)

	want := `
error: cannot determine how to extract field
| type doubleNestedInput struct {
|     Value int
|           ^^^
|           |
|           cannot extract "int"
| }
`

	var wantErr routey.HandlerError
	r.ErrorSink = expectErrSink(t, &wantErr)

	routey.Get(r, "/", testHandler)
	got := wantErr.Error()

	if !strings.Contains(got, strings.TrimSpace(want)) {
		t.Fatalf("got: %v, wanted: %v", got, want)
	}
}

func TestRouter_HandleValidPathParam(t *testing.T) {
	type Input struct {
		Value routey.Path[int]
	}

	var got int
	fn := func(i Input) (any, error) {
		got = i.Value.Value
		return nil, nil
	}

	r := newTestRouter(t)
	routey.Handle(r, http.MethodGet, "/{value}", fn)

	want := 1
	req := newRequest(t, http.MethodGet, "/"+fmt.Sprintf("%d", want), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != want {
		t.Errorf("got: %v, wanted: %v", got, want)
	}
}

func TestRouter_HandleValidJSONBodyParam(t *testing.T) {
	type obj struct {
		Field string `json:"field"`
	}
	type Input struct {
		Body routey.JSON[obj]
	}

	var got string
	fn := func(i Input) (any, error) {
		got = i.Body.V.Field
		return nil, nil
	}

	r := newTestRouter(t)
	routey.Handle(r, http.MethodGet, "/", fn)

	want := "test"
	input, err := json.Marshal(obj{Field: want})
	test.NoError(t, err)

	req := newRequest(t, http.MethodGet, "/", bytes.NewReader(input))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != want {
		t.Errorf("got: %v, wanted: %v", got, want)
	}
}

func TestRouter_HandleValidQueryParam(t *testing.T) {
	type Input struct {
		Query routey.Query[int]
	}

	var got int
	fn := func(i Input) (any, error) {
		got = i.Query.Value
		return nil, nil
	}

	r := newTestRouter(t)
	routey.Handle(r, http.MethodGet, "/", fn)

	want := 1
	req := newRequest(t, http.MethodGet, "/?query="+fmt.Sprintf("%d", want), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != want {
		t.Errorf("got: %v, wanted: %v", got, want)
	}
}

func TestRouter_UseGlobal(t *testing.T) {
	r := newTestRouter(t)
	want := http.StatusCreated
	wantMW := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(want)
			h.ServeHTTP(w, r)
		})
	}

	r.Use(wantMW)
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	req := newRequest(t, http.MethodGet, "/", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_GroupMiddleware(t *testing.T) {
	r := newTestRouter(t)
	wantMW := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			h.ServeHTTP(w, r)
		})
	}

	r.Group(func(r *routey.Router) {
		r.Use(wantMW)
		r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})
	})

	r.Get("/bar", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	want := http.StatusCreated
	req := newRequest(t, http.MethodGet, "/foo", nil)
	compareRespStatus(t, r, req, want)

	want = http.StatusBadRequest
	req = newRequest(t, http.MethodGet, "/bar", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_MiddlewareOrder(t *testing.T) {
	r := newTestRouter(t)
	gotOrder := []string{}
	mw := func(name string) routey.Middleware {
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotOrder = append(gotOrder, name)
				h.ServeHTTP(w, r)
			})
		}
	}

	global1MW := mw("global-1")
	global2MW := mw("global-2")
	groupMW := mw("group")
	routeMW := mw("route")

	r.Use(global1MW)
	r.Use(global2MW)

	r.Group(func(r *routey.Router) {
		r.Use(groupMW)
		r.With(routeMW).Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})
	})

	want := http.StatusCreated
	req := newRequest(t, http.MethodGet, "/foo", nil)
	compareRespStatus(t, r, req, want)

	wantOrder := []string{"global-1", "global-2", "group", "route"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Errorf("wanted: %v, got: %v", wantOrder, gotOrder)
	}
}

func TestRouter_HandlerWithMiddleware(t *testing.T) {
	r := newTestRouter(t)
	wantMW := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			h.ServeHTTP(w, r)
		})
	}

	r.With(wantMW).Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	r.Get("/bar", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	want := http.StatusCreated
	req := newRequest(t, http.MethodGet, "/foo", nil)
	compareRespStatus(t, r, req, want)

	want = http.StatusBadRequest
	req = newRequest(t, http.MethodGet, "/bar", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_RouteWithHandle(t *testing.T) {
	r := newTestRouter(t)
	want := http.StatusCreated

	fn := func(p struct {
		w http.ResponseWriter
	}) (any, error) {
		p.w.WriteHeader(want)
		return nil, nil
	}

	r.Route("/v1", func(r *routey.Router) {
		routey.Get(r, "/foo", fn)
	})

	req := newRequest(t, http.MethodGet, "/v1/foo", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_Route(t *testing.T) {
	r := newTestRouter(t)
	want := http.StatusCreated

	r.Route("/v1", func(r *routey.Router) {
		r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(want)
		})
	})

	r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(want)
	})

	req := newRequest(t, http.MethodGet, "/v1/foo", nil)
	compareRespStatus(t, r, req, want)

	req = newRequest(t, http.MethodGet, "/foo", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_Mount(t *testing.T) {
	r := newTestRouter(t)
	subRouter := newTestRouter(t)

	want := http.StatusCreated
	subRouter.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(want)
	})
	r.Mount("/v1", subRouter)

	req := newRequest(t, http.MethodGet, "/v1/foo", nil)
	compareRespStatus(t, r, req, want)
}

func TestRouter_MountMiddleware(t *testing.T) {
	gotOrder := []string{}
	mw := func(name string) routey.Middleware {
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotOrder = append(gotOrder, name)
				h.ServeHTTP(w, r)
			})
		}
	}

	globalMW := mw("global")
	subRouterMW := mw("sub router")

	r := newTestRouter(t)
	subRouter := newTestRouter(t)

	r.Use(globalMW)
	subRouter.Use(subRouterMW)

	want := http.StatusCreated
	subRouter.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(want)
	})

	r.Mount("/v1", subRouter)

	req := newRequest(t, http.MethodGet, "/v1/foo", nil)
	compareRespStatus(t, r, req, want)

	wantOrder := []string{"global", "sub router"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Errorf("wanted: %v, got: %v", wantOrder, gotOrder)
	}
}

func TestRouter_HandleInvalidParamErr(t *testing.T) {
	type Input struct {
		Value routey.Query[struct{}]
	}
	fn := func(Input) (any, error) { return nil, nil }

	r := routey.New()
	var want *param.InvalidParamError
	r.ErrorSink = expectErrSink(t, &want)

	routey.Handle(r, http.MethodGet, "/", fn)
}

func TestRouter_RouteInfo(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r := newTestRouter(t)
	routey.Get(r, "/foo", h)

	want := []route.Info{{
		Method:      http.MethodGet,
		FullPattern: "/foo",
		Pattern:     "/foo",
		Params: []param.Info{
			{
				Name:   "query",
				Source: "query",
				Type:   reflect.TypeOf(int(0)),
				Field:  reflect.TypeFor[input]().Field(0),
				Struct: reflect.TypeFor[input](),
			},
		},
		ReturnType: reflect.TypeFor[any](),
		Context:    route.Context{},
	}}

	test.MatchAsJSON(t, r.Routes(), want)
}

func TestRouter_RouteInfoWithRoute(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r := newTestRouter(t)
	r.Route("/v1", func(r *routey.Router) {
		routey.Get(r, "/foo", h)
	})

	want := []route.Info{{
		Method:      http.MethodGet,
		FullPattern: "/v1/foo",
		Pattern:     "/foo",
		Params: []param.Info{
			{
				Name:   "query",
				Source: "query",
				Type:   reflect.TypeOf(int(0)),
				Field:  reflect.TypeFor[input]().Field(0),
				Struct: reflect.TypeFor[input](),
			},
		},
		ReturnType: reflect.TypeFor[any](),
		Context:    route.Context{},
	}}

	test.MatchAsJSON(t, r.Routes(), want)
}

func TestRouter_RouteInfoWithWith(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r := newTestRouter(t)
	routey.Get(r.With(func(http.Handler) http.Handler {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}), "/foo", h)

	want := []route.Info{{
		Method:      http.MethodGet,
		FullPattern: "/foo",
		Pattern:     "/foo",
		Params: []param.Info{
			{
				Name:   "query",
				Source: "query",
				Type:   reflect.TypeOf(int(0)),
				Field:  reflect.TypeFor[input]().Field(0),
				Struct: reflect.TypeFor[input](),
			},
		},
		ReturnType: reflect.TypeFor[any](),
		Context:    route.Context{},
	}}

	test.MatchAsJSON(t, r.Routes(), want)
}

func TestRouter_RouteInfoWithGroup(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r := newTestRouter(t)
	r.Group(func(r *routey.Router) {
		routey.Get(r, "/foo", h)
	})

	want := []route.Info{{
		Method:      http.MethodGet,
		FullPattern: "/foo",
		Pattern:     "/foo",
		Params: []param.Info{
			{
				Name:   "query",
				Source: "query",
				Type:   reflect.TypeOf(int(0)),
				Field:  reflect.TypeFor[input]().Field(0),
				Struct: reflect.TypeFor[input](),
			},
		},
		ReturnType: reflect.TypeFor[any](),
		Context:    route.Context{},
	}}

	test.MatchAsJSON(t, r.Routes(), want)
}

func TestRouter_RouteInfoWithMount(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r := newTestRouter(t)
	subRouter := newTestRouter(t)
	routey.Get(subRouter, "/foo", h)
	r.Mount("/v1", subRouter)

	want := []route.Info{{
		Method:      http.MethodGet,
		FullPattern: "/v1/foo",
		Pattern:     "/foo",
		Params: []param.Info{
			{
				Name:   "query",
				Source: "query",
				Type:   reflect.TypeOf(int(0)),
				Field:  reflect.TypeFor[input]().Field(0),
				Struct: reflect.TypeFor[input](),
			},
		},
		ReturnType: reflect.TypeFor[any](),
		Context:    route.Context{},
	}}

	test.MatchAsJSON(t, r.Routes(), want)
}

func TestRouter_RouteInfoAddCallback(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	r := newTestRouter(t)

	called := test.WantAfterTest(t, false, true, "expected OnRouteAdd to be called")
	r.OnRouteAdd = func(i *route.Info) error {
		want := "/"
		test.Equal(t, i.FullPattern, want)
		*called = true
		return nil
	}
	routey.Get(r, "/", h)
}

func TestRouter_RouteInfoOptions(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	called := test.WantAfterTest(t, false, true, "expected option to be called")
	opt := func(i *route.Info) error {
		want := "/"
		test.Equal(t, i.FullPattern, want)
		*called = true
		return nil
	}

	r := newTestRouter(t)
	routey.Get(r, "/", h, opt)
}

func TestRouter_OptionReturnsError(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }

	err := errors.New("test error")
	opt := func(i *route.Info) error {
		return err
	}

	r := newTestRouter(t)
	var want routey.HandlerError
	r.ErrorSink = expectErrSink(t, &want)

	routey.Get(r, "/", h, opt)
	test.IsError(t, want.Err, err)
}

func TestRouter_UnparsableDefaultValue(t *testing.T) {
	r := routey.New()
	var want *param.InvalidParamError
	r.ErrorSink = expectErrSink(t, &want)

	type input struct {
		Field routey.Query[int] `default:"1."`
	}
	h := func(input) (any, error) { return nil, nil }
	routey.Get(r, "/", h)

	wantMsg := param.ErrUnparsableDefault + ": 1."
	if got := want.Message; got != wantMsg {
		t.Fatalf("got: %q, want: %q", got, wantMsg)
	}
}

func TestRouter_MountWithTrailingSlash(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	r := newTestRouter(t)
	subRouter := newTestRouter(t)

	routey.Get(subRouter, "/foo/", h)
	r.Mount("/v1", subRouter)

	got := r.Routes()
	want := []route.Info{{
		FullPattern: "/v1/foo/",
		Pattern:     "/foo/",
		Method:      http.MethodGet,
		Params:      []param.Info{},
		ReturnType:  reflect.TypeFor[any](),
	}}
	test.MatchAsJSON(t, got, want)
}

func TestRouter_CollectAllErrors(t *testing.T) {
	type input struct {
		Int      routey.Query[int]
		OtherInt routey.Query[int]
	}
	h := func(p input) (any, error) { return nil, nil }

	r := routey.New()
	r.Errors.CollectAll = true

	gotError := test.WantAfterTest(t, false, true, "expected an error, got none")
	r.Response = func(_ http.ResponseWriter, _ *http.Request, resp extractor.Response) {
		var want interface {
			Unwrap() []error
		}
		test.WantError(t, resp.Error, &want)
		*gotError = true

		for _, err := range want.Unwrap() {
			test.IsError(t, err, strconv.ErrSyntax)
		}
	}

	routey.Get(r, "/", h)
	req := newRequest(t, http.MethodGet, "/?int=a&other_int=b", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}
