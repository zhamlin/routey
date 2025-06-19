package extractor_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

	hasError := false
	t.Cleanup(func() {
		t.Helper()

		if !hasError {
			t.Fatal("expected an error, got none")
		}
	})

	return func(err error) {
		test.WantError(t, err, want)
		hasError = true
	}
}

func newRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), method, path, body)
}

func TestJSONExtractor_ValidJSON(t *testing.T) {
	type Body struct {
		Value int
	}
	r := newRequest(t, http.MethodPost, "/", strings.NewReader(`{"value": 1}`))

	got := routey.JSON[Body]{}
	err := got.Extract(r, nil)
	test.NoError(t, err)

	want := 1
	test.Equal(t, got.V.Value, want)
}

func TestJSONExtractor_InvalidJSON(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/", strings.NewReader(`{"key": }`))
	val := routey.JSON[struct{}]{}
	err := val.Extract(r, nil)

	var want *json.SyntaxError
	test.WantError(t, err, &want)
}

func TestQueryExtractor_ValidValue(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/?query=1", nil)
	got := routey.Query[int]{}

	err := got.Extract(r, &route.Info{}, param.Opts{
		Name:   "query",
		Parser: param.ParseInt,
	})
	test.NoError(t, err)

	want := 1
	test.Equal(t, got.Value, want)
}

func TestQueryExtractor_DefaultValue(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/", nil)
	got := routey.Query[int]{}
	want := 1

	err := got.Extract(r, &route.Info{}, param.Opts{
		Name:    "query",
		Default: fmt.Sprintf("%d", want),
		Parser:  param.ParseInt,
	})

	test.NoError(t, err)
	test.Equal(t, got.Value, want)
}

func TestQueryExtractor_ErrorParsing(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/?query=1.", nil)
	q := routey.Query[int]{}
	err := q.Extract(r, &route.Info{}, param.Opts{
		Name:   "query",
		Parser: param.ParseInt,
	})

	test.IsError(t, err, strconv.ErrSyntax)
}

func TestQueryExtractor_MissingParam(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/", nil)
	got := routey.Query[int]{}
	err := got.Extract(r, &route.Info{}, param.Opts{
		Name:   "query",
		Parser: param.ParseInt,
	})
	test.NoError(t, err)

	want := 0
	test.Equal(t, got.Value, want)
}

type testPather struct {
	value string
}

func (t testPather) Param(_ string, _ *http.Request) string {
	return t.value
}

func TestPathExtractor_ValidValue(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/", nil)

	want := 1
	pather := testPather{value: fmt.Sprint(want)}
	got := routey.Path[int]{}

	err := got.Extract(r, &route.Info{}, param.Opts{
		Parser: param.ParseInt,
		Pather: pather,
	})
	test.NoError(t, err)
	test.Equal(t, got.Value, want)
}

func TestPathExtractor_ErrorFailedToExtract(t *testing.T) {
	r := newRequest(t, http.MethodPost, "/", nil)

	pather := testPather{}
	got := routey.Path[struct{}]{}

	err := got.Extract(r, &route.Info{}, param.Opts{
		Parser: param.ParseInt,
		Pather: pather,
	})
	test.IsError(t, err, extractor.ErrParamFailedToExtract)
}

func TestHandler_ValidExtractor(t *testing.T) {
	type Input struct {
		Value routey.Query[int]
	}
	fn := func(i Input) (int, error) {
		return i.Value.Value, nil
	}

	params := extractor.HandlerParams{
		Response: func(w http.ResponseWriter, _ *http.Request, resp extractor.Response) {
			_, _ = fmt.Fprintf(w, "%v", resp.Response)
		},
		Parser:    param.ParseInt,
		Namer:     func(string, string) string { return "value" },
		RouteInfo: &route.Info{},
	}
	handler := extractor.Handler(fn, params)

	want := "1"
	r := newRequest(t, http.MethodGet, "/?value="+want, nil)
	w := httptest.NewRecorder()
	handler(w, r)

	got := w.Body.String()
	test.Equal(t, got, want)
}

func TestHandler_ExtractHttpRequest(t *testing.T) {
	type Input struct{ r *http.Request }
	fn := func(i Input) (any, error) {
		if i.r == nil {
			t.Error("Expected http.ResponseWriter, got nil")
		}
		return nil, nil
	}

	params := extractor.HandlerParams{
		ErrorSink: func(err error) {
			test.NoError(t, err)
		},
	}

	h := extractor.Handler(fn, params)
	r := newRequest(t, http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, r)
}

func TestHandler_ExtractHttpResponseWriter(t *testing.T) {
	type Input struct{ w http.ResponseWriter }
	fn := func(i Input) (any, error) {
		if i.w == nil {
			t.Error("Expected http.ResponseWriter, got nil")
		}
		return nil, nil
	}

	params := extractor.HandlerParams{
		ErrorSink: func(err error) {
			test.NoError(t, err)
		},
	}

	h := extractor.Handler(fn, params)
	r := newRequest(t, http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, r)
}

func TestHandler_ErrorRelatedExtractors(t *testing.T) {
	params := extractor.HandlerParams{
		ErrorSink: func(err error) {
			var want *extractor.UnknownFieldTypeError
			test.WantError(t, err, &want)

			if l := len(want.RelatedFound); l != 1 {
				t.Errorf("expected one related type, got: %v", l)
			}
		},
	}

	fn1 := func(struct{ r http.Request }) (any, error) { return nil, nil }
	extractor.Handler(fn1, params)

	fn2 := func(struct{ r **http.Request }) (any, error) { return nil, nil }
	extractor.Handler(fn2, params)
}

func TestHandler_ErrorExtracting(t *testing.T) {
	type Input struct{ Query routey.Query[int] }
	fn := func(Input) (any, error) {
		t.Error("extractor should have failed")
		return nil, nil
	}

	want := errors.New("test error")
	hasErr := false
	params := extractor.HandlerParams{
		Parser: func(any, []string) error {
			return want
		},
		Response: func(_ http.ResponseWriter, _ *http.Request, resp extractor.Response) {
			test.IsError(t, resp.Error, want)
			hasErr = true
		},
		Namer:     func(string, string) string { return "query" },
		RouteInfo: &route.Info{},
	}

	h := extractor.Handler(fn, params)
	r := newRequest(t, http.MethodGet, "/?query=no", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, r)

	if !hasErr {
		t.Error("expected an error, got none")
	}
}

func TestHandler_ErrorNoExtractorForField(t *testing.T) {
	type Input struct {
		Value int
	}

	var want *extractor.UnknownFieldTypeError
	params := extractor.HandlerParams{ErrorSink: expectErrSink(t, &want)}

	fn := func(Input) (any, error) { return nil, nil }
	extractor.Handler(fn, params)
}

func TestHandler_ErrorNonStruct(t *testing.T) {
	params := extractor.HandlerParams{ErrorSink: func(err error) {
		test.IsError(t, err, param.ErrNonStructArg)
	}}

	fn := func(int) (any, error) { return nil, nil }
	extractor.Handler(fn, params)
}
