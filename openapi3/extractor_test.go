package openapi3_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/openapi3"
	openapiParam "github.com/zhamlin/routey/openapi3/param"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

func newParamTester(
	t *testing.T,
	p openapi3.Parameter,
	values url.Values,
) func(extractor.ParamExtractor, param.Opts) {
	t.Helper()

	method := http.MethodGet
	path := "/"
	r, spec := openapi3.NewRouter()

	{
		op := openapi3.NewOperation()
		op.AddParameter(p)

		pathItem := openapi3.NewPathItem()
		pathItem.SetOperation(method, op)
		spec.SetPath(path, pathItem)
	}

	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	req.URL.RawQuery = values.Encode()

	info := route.Info{
		FullPattern: path,
		Method:      method,
		Context:     r.Context,
	}

	return func(e extractor.ParamExtractor, opts param.Opts) {
		t.Helper()

		opts.Name = p.Name
		opts.Parser = r.Params.Parser
		err := e.Extract(req, &info, opts)
		test.NoError(t, err)
	}
}

func TestQuery_DeepObject(t *testing.T) {
	type Object struct {
		Field string `json:"field"`
	}

	name := "obj"
	want := "value"
	values := url.Values{}
	values.Add(fmt.Sprintf("%s[%s]", name, "field"), want)

	p := openapi3.NewParameter()
	p.Name = name
	p.Style = string(openapiParam.StyleDeepObject)
	p.In = string(openapiParam.LocationQuery)

	parse := newParamTester(t, p, values)
	q := openapi3.Query[Object]{}
	parse(&q, param.Opts{})

	if got := q.Value.Field; got != want {
		t.Errorf("got: %v, wanted: %v", got, want)
	}
}

func TestQuery_FormSlice(t *testing.T) {
	name := "obj"
	want := []string{"a", "b"}
	values := url.Values{}
	values.Add(name, strings.Join(want, ","))

	p := openapi3.NewParameter()
	p.Name = name
	p.Style = string(openapiParam.StyleForm)
	p.In = string(openapiParam.LocationQuery)

	parse := newParamTester(t, p, values)
	q := openapi3.Query[[]string]{}
	parse(&q, param.Opts{})

	got := q.Value
	test.MatchAsJSON(t, got, want)
}

func TestQuery_FormExplodeSlice(t *testing.T) {
	name := "obj"
	want := []string{"a", "b"}

	values := url.Values{}
	for _, v := range want {
		values.Add(name, v)
	}

	p := openapi3.NewParameter()
	p.Name = name
	p.Explode = true
	p.Style = string(openapiParam.StyleForm)
	p.In = string(openapiParam.LocationQuery)

	parse := newParamTester(t, p, values)
	q := openapi3.Query[[]string]{}
	parse(&q, param.Opts{})

	got := q.Value
	test.MatchAsJSON(t, got, want)
}

func TestQuery_FormDefaultValue(t *testing.T) {
	p := openapi3.NewParameter()
	p.Style = string(openapiParam.StyleForm)
	p.In = string(openapiParam.LocationQuery)

	parse := newParamTester(t, p, url.Values{})
	q := openapi3.Query[int]{}

	parse(&q, param.Opts{
		Default: "1",
	})

	got := q.Value
	want := 1
	test.MatchAsJSON(t, got, want)
}

type deepObject struct {
	Field int `json:"field"`
}

func (deepObject) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Property("field").
		Default("1.")
}

func TestQuery_UnparsableDeepObjectDefaultValueFromJsonschema(t *testing.T) {
	r, _ := openapi3.NewRouter()

	gotErr := false
	r.ErrorSink = func(err error) {
		var want *param.InvalidParamError
		test.WantError(t, err, &want)

		wantMsg := param.ErrUnparsableDefault + ": 1."
		if got := want.Message; got != wantMsg {
			t.Fatalf("got: %q, want: %q", got, wantMsg)
		}
		gotErr = true
	}

	type input struct {
		Field openapi3.Query[deepObject] `style:"deepObject"`
	}
	h := func(input) (any, error) { return nil, nil }
	routey.Get(r, "/", h)

	if !gotErr {
		t.Error("expected error, got none")
	}
}

func TestQuery_UnparsableDeepObjectDefaultValue(t *testing.T) {
	r, _ := openapi3.NewRouter()

	gotErr := false
	r.ErrorSink = func(err error) {
		var want *param.InvalidParamError
		test.WantError(t, err, &want)

		wantMsg := param.ErrUnparsableDefault + ": 1."
		if got := want.Message; got != wantMsg {
			t.Fatalf("got: %q, want: %q", got, wantMsg)
		}
		gotErr = true
	}

	type object struct {
		Field int `default:"1."`
	}
	type input struct {
		Field openapi3.Query[object] `style:"deepObject"`
	}
	h := func(input) (any, error) { return nil, nil }
	routey.Get(r, "/", h)

	if !gotErr {
		t.Error("expected error, got none")
	}
}
