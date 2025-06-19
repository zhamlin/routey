package openapi3_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/openapi3"
	"github.com/zhamlin/routey/openapi3/option"
	"github.com/zhamlin/routey/route"
)

func newTestRouter(t *testing.T) (*routey.Router, *openapi3.OpenAPI) {
	t.Helper()

	r, spec := openapi3.NewRouter()
	r.Response = func(w http.ResponseWriter, _ *http.Request, resp extractor.Response) {
		test.NoError(t, resp.Error, "newTestRouter: Response")
	}
	r.ErrorSink = func(err error) {
		test.NoError(t, err, "newTestRouter: ErrorSink")
	}
	return r, spec
}

func HandlerForTests(struct{}) (any, error) { return nil, nil }

func TestRouter_DefaultOperationID(t *testing.T) {
	r, spec := newTestRouter(t)
	routey.Handle(r, http.MethodGet, "/", HandlerForTests)

	want := "HandlerForTests"
	got := spec.Paths.Spec.Paths["/"].Spec.Spec.Get.Spec.OperationID

	if got != want {
		t.Errorf("got operationID: %v, want: %v", got, want)
	}
}

func TestRouter_DefaultOperationIDDoesNotOverrideID(t *testing.T) {
	r, spec := newTestRouter(t)

	want := "testOperationID"
	routey.Handle(r, http.MethodGet, "/", HandlerForTests, option.ID(want))

	got := spec.Paths.Spec.Paths["/"].Spec.Spec.Get.Spec.OperationID
	if got != want {
		t.Errorf("got operationID: %v, want: %v", got, want)
	}
}

func TestRouter_DefaultResponse(t *testing.T) {
	type DefaultResponse struct {
		Error string
	}

	r, spec := newTestRouter(t)
	openapi3.SetDefaultResponse[DefaultResponse](spec, 0)
	routey.Handle(r, http.MethodGet, "/", HandlerForTests)

	want := openapi3.Response{}
	{
		mt := openapi3.NewMediaType()
		mt.SetSchemaRef(spec.Schemer.RefPath + "DefaultResponse")
		want.SetContent(openapi3.JSONContentType, mt)
	}
	got := spec.Paths.Spec.Paths["/"].Spec.Spec.Get.Spec.Responses.Spec.Default
	test.MatchAsJSON(t, got, want)
}

func TestRouter_ValidJSONBodyParam(t *testing.T) {
	type Body struct{ Field string }
	type Input struct {
		Body routey.JSON[Body] `description:"test" required:"true"`
	}
	fn := func(Input) (any, error) { return nil, nil }

	r, spec := newTestRouter(t)
	routey.Handle(r, http.MethodGet, "/", fn)

	want := openapi3.RequestBody{}
	{
		mt := openapi3.NewMediaType()
		mt.SetSchemaRef(spec.Schemer.RefPath + "Body")

		want.Description = "test"
		want.Required = true
		want.SetContent(openapi3.JSONContentType, mt)
	}
	got := spec.Paths.Spec.Paths["/"].Spec.Spec.Get.Spec.RequestBody
	test.MatchAsJSON(t, got, want)
}

func TestRouter_SpecWithParam(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }
	r, spec := newTestRouter(t)

	routey.Get(r, "/", h)

	test.MatchAsJSON(t, spec.Paths, `
	{
		"/": {
			"get": {
				"parameters": [
					{
						"in": "query",
						"explode": true,
						"name": "query",
						"style": "form",
						"schema": {
							"type": "integer"
						}
					}
				]
			}
		}
	}
	`)
}

func TestRouter_SpecWithPaths(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	r, spec := newTestRouter(t)

	routey.Get(r, "/", h)
	routey.Post(r, "/", h)

	test.MatchAsJSON(t, spec.Paths, `
	{
		"/": {
			"get": {},
			"post": {}
		}
	}
	`)
}

func TestRouter_InvalidParamStructTag(t *testing.T) {
	type input struct {
		Field routey.Query[int] `explode:"no"`
	}
	h := func(input) (any, error) { return nil, nil }

	r := routey.New()
	haveErr := false
	r.ErrorSink = func(err error) {
		haveErr = true
		test.IsError(t, err, strconv.ErrSyntax)
	}
	openapi3.AddSpecToRouter(r, openapi3.AddSpecToRouterOpts{})
	routey.Get(r, "/", h)

	if !haveErr {
		t.Errorf("expected an error, got none")
	}
}

func TestContextFromInfo_GetsContext(t *testing.T) {
	r := routey.New()
	want := "title"
	spec := openapi3.AddSpecToRouter(r, openapi3.AddSpecToRouterOpts{})
	spec.Info.Spec.Title = "title"

	i := route.Info{Context: r.Context}
	got, err := openapi3.ContextFromCtx(i.Context)
	test.NoError(t, err)

	if got := got.OpenAPI.Info.Spec.Title; got != want {
		t.Errorf("wanted openapi from context: %v, got: %v", want, got)
	}
}

func TestContextFromInfo_ErrorWhenMissingContext(t *testing.T) {
	info := route.Info{}
	_, err := openapi3.ContextFromCtx(info.Context)
	test.IsError(t, err, openapi3.ErrNoContext)
}

func TestOperationFromInfo_UsesExistingOp(t *testing.T) {
	i := route.Info{Context: route.Context{}}
	op := openapi3.OperationFromCtx(i.Context)
	want := "id"
	op.OperationID = want

	got := openapi3.OperationFromCtx(i.Context).OperationID
	if got != want {
		t.Errorf("wanted operation id: %v, got: %v", want, got)
	}
}

func TestRouter_RouteInfoWithRoute(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r, spec := newTestRouter(t)
	r.Route("/v1", func(r *routey.Router) {
		routey.Get(r, "/foo", h)
	})

	test.MatchAsJSON(t, spec.Paths, `
	{
	  "/v1/foo": {
		"get": {
		  "parameters": [
			{
			  "in": "query",
		   	  "explode": true,
			  "name": "query",
			  "schema": {
				"type": "integer"
			  },
			  "style": "form"
			}
		  ]
		}
	  }
	}
	`)
}

func TestRouter_RouteInfoWithWith(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r, spec := newTestRouter(t)

	mw := func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	}
	routey.Get(r.With(mw), "/foo", h)

	test.MatchAsJSON(t, spec.Paths, `
	{
	  "/foo": {
		"get": {
		  "parameters": [
			{
			  "in": "query",
		   	  "explode": true,
			  "name": "query",
			  "schema": {
				"type": "integer"
			  },
			  "style": "form"
			}
		  ]
		}
	  }
	}
	`)
}

func TestRouter_RouteInfoWithGroup(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	h := func(input) (any, error) { return nil, nil }

	r, spec := newTestRouter(t)
	r.Group(func(r *routey.Router) {
		routey.Get(r, "/foo", h)
	})

	test.MatchAsJSON(t, spec.Paths, `
	{
	  "/foo": {
		"get": {
		  "parameters": [
			{
			  "in": "query",
		   	  "explode": true,
			  "name": "query",
			  "schema": {
				"type": "integer"
			  },
			  "style": "form"
			}
		  ]
		}
	  }
	}
	`)
}

func TestRouter_SpecWithMount(t *testing.T) {
	type Object struct{}
	type input struct {
		Query routey.Query[int]
		Body  routey.JSON[Object]
	}
	h := func(input) (any, error) { return nil, nil }

	r, spec := newTestRouter(t)
	{
		strSchema := jsonschema.NewBuilder().Type("string").Build()
		schema := jsonschema.NewBuilder().
			Type("object").
			Property("Field", strSchema).
			Build()
		openapi3.RegisterType[Object](spec, schema)
	}

	subRouter, _ := newTestRouter(t)

	routey.Get(subRouter, "/{id}", h)
	r.Mount("/v1", subRouter)

	test.MatchAsJSON(t, spec, `
	{
	  "components": {
		"schemas": {
		  "Object": {
			"properties": {
			  "Field": {
				"type": "string"
			  }
			},
			"type": "object"
		  }
		}
	  },
	  "info": {
		"title": "",
		"version": ""
	  },
	  "openapi": "3.1.1",
	  "paths": {
		"/v1/{id}": {
		  "get": {
			"parameters": [
			  {
				"in": "query",
		   	    "explode": true,
				"name": "query",
				"schema": {
				  "type": "integer"
				},
				"style": "form"
			  }
			],
			"requestBody": {
			  "content": {
				"application/json": {
				  "schema": {
					"$ref": "#/components/schemas/Object"
				  }
				}
			  }
			}
		  }
		}
	  }
	}
	`)
}

type object struct {
	Field string `json:"field"`
}

func (object) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Property("field").
		MinLength(5)
}

func TestRouterValidateRequest_MiddlewareDeepObjectError(t *testing.T) {
	type input struct {
		Name openapi3.Query[object] `style:"deepObject"`
	}
	h := func(p input) (any, error) { return nil, nil }

	r := routey.New()
	openapi3.AddSpecToRouter(r, openapi3.AddSpecToRouterOpts{
		ValidateRequests: true,
	})
	subRouter, _ := newTestRouter(t)

	gotError := test.WantAfterTest(t, false, true, "expected an error, got none")
	subRouter.Response = func(_ http.ResponseWriter, _ *http.Request, resp extractor.Response) {
		var want jsonschema.ValidationError
		test.WantError(t, resp.Error, &want)
		*gotError = true
	}

	routey.Get(subRouter, "/bar", h, option.ID("id"))
	r.Mount("/foo", subRouter)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/foo/bar?name[field]=test",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestRouterValidateRequest_QueryError(t *testing.T) {
	type input struct {
		Int openapi3.Query[int] `minimum:"2"`
	}
	h := func(p input) (any, error) { return nil, nil }

	r := routey.New()
	openapi3.AddSpecToRouter(r, openapi3.AddSpecToRouterOpts{
		ValidateRequests: true,
	})

	gotError := test.WantAfterTest(t, false, true, "expected an error, got none")
	r.Response = func(_ http.ResponseWriter, _ *http.Request, resp extractor.Response) {
		var want jsonschema.ValidationError
		test.WantError(t, resp.Error, &want)
		*gotError = true
	}

	routey.Get(r, "/", h, option.ID("id"))
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?int=1",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestRouter_DuplicateOperationIDs(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	r, spec := newTestRouter(t)
	spec.Strict = true

	gotError := test.WantAfterTest(t, false, true, "expected an error, got none")
	r.ErrorSink = func(err error) {
		test.IsError(t, err, openapi3.ErrDuplicateOperationID)
		*gotError = true
	}

	routey.Get(r, "/foo", h, option.ID("id"))
	routey.Get(r, "/bar", h, option.ID("id"))
}
