package option_test

import (
	"errors"
	"maps"
	"net/http"
	"reflect"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/openapi3"
	"github.com/zhamlin/routey/openapi3/option"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

func createInfo(t *testing.T) (*openapi3.OpenAPI, route.Info) {
	t.Helper()

	r, spec := openapi3.NewRouter()
	info := route.Info{}
	info.Context = maps.Clone(r.Context)

	return spec, info
}

func TestRouter_SpecWithOperationID(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	r, spec := openapi3.NewRouter()

	routey.Get(r, "/", h, option.ID("get"))

	test.MatchAsJSON(t, spec.Paths, `
	{
		"/": {
			"get": {
				"operationId": "get"
			}
		}
	}
	`)
}

func TestRouter_SpecWithMountOperationID(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }

	r, spec := openapi3.NewRouter()
	subRouter, _ := openapi3.NewRouter()

	routey.Get(subRouter, "/", h,
		option.ID("id"))
	r.Mount("/v1", subRouter)

	test.MatchAsJSON(t, spec, `
	{
	  "info": {
		"title": "",
		"version": ""
	  },
	  "openapi": "3.1.1",
	  "components": {},
	  "paths": {
		"/v1": {
		  "get": {
			"operationId": "id"
		  }
		}
	  }
	}
	`)
}

func TestRouter_SpecWithMountResponse(t *testing.T) {
	type Object struct {
		Field string
	}
	h := func(struct{}) (any, error) { return nil, nil }

	r, spec := openapi3.NewRouter()
	subRouter, _ := openapi3.NewRouter()

	routey.Get(subRouter, "/", h,
		option.Response[Object](http.StatusOK, "description"))
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
		"/v1": {
		  "get": {
			"responses": {
			  "200": {
				"content": {
				  "application/json": {
					"schema": {
					  "$ref": "#/components/schemas/Object"
					}
				  }
				},
				"description": "description"
			  }
			}
		  }
		}
	  }
	}
	`)
}

func TestRouter_SpecWithMountResponseRegisteredType(t *testing.T) {
	type Object struct {
		Field string
	}
	h := func(struct{}) (any, error) { return nil, nil }

	r, spec := openapi3.NewRouter()
	err := openapi3.RegisterType[Object](spec, jsonschema.NewBuilder().Type("object").Build())
	test.NoError(t, err)

	subRouter, _ := openapi3.NewRouter()

	routey.Get(subRouter, "/", h,
		option.Response[Object](http.StatusOK, "description"))
	r.Mount("/v1", subRouter)

	test.MatchAsJSON(t, spec, `
	{
	  "components": {
		"schemas": {
		  "Object": {
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
		"/v1": {
		  "get": {
			"responses": {
			  "200": {
				"content": {
				  "application/json": {
					"schema": {
					  "$ref": "#/components/schemas/Object"
					}
				  }
				},
				"description": "description"
			  }
			}
		  }
		}
	  }
	}
	`)
}

func TestRouter_SpecWithTwoMountsResponse(t *testing.T) {
	type Object struct {
		Field string
	}
	h := func(struct{}) (any, error) { return nil, nil }

	r, spec := openapi3.NewRouter()
	subRouter, _ := openapi3.NewRouter()
	subSubRouter, _ := openapi3.NewRouter()

	routey.Get(subSubRouter, "/", h,
		option.Response[Object](http.StatusOK, "description"))
	subRouter.Mount("/b", subSubRouter)
	r.Mount("/a", subRouter)

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
		"/a/b": {
		  "get": {
			"responses": {
			  "200": {
				"content": {
				  "application/json": {
					"schema": {
					  "$ref": "#/components/schemas/Object"
					}
				  }
				},
				"description": "description"
			  }
			}
		  }
		}
	  }
	}
	`)
}

type testResp[D any] struct {
	Data D `json:"data"`
}

func (testResp[D]) NoRef() {}

func TestRouter_SpecResponseWithNoRefTypes(t *testing.T) {
	type Object struct {
		Field string
	}
	h := func(struct{}) (any, error) { return nil, nil }

	r, spec := openapi3.NewRouter()
	subRouter, _ := openapi3.NewRouter()

	routey.Get(subRouter, "/", h, option.Response[testResp[Object]](http.StatusOK, ""))
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
		"/v1": {
		  "get": {
			"responses": {
			  "200": {
				"content": {
				  "application/json": {
					"schema": {
					  "properties": {
						"data": {
						  "$ref": "#/components/schemas/Object"
						}
					  },
					  "type": "object"
					}
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

func TestRouter_SpecResponseWithNestedTypes(t *testing.T) {
	type Object struct {
		Field string
	}

	type Response struct {
		Object Object
	}
	h := func(struct{}) (any, error) { return nil, nil }

	r, spec := openapi3.NewRouter()
	subRouter, _ := openapi3.NewRouter()

	routey.Get(subRouter, "/{id}", h, option.Response[Response](http.StatusOK, ""))
	r.Mount("/v1", subRouter)

	test.MatchAsJSON(t, spec, `
	{
	  "components": {
		"schemas": {
		  "Response": {
			"properties": {
			  "Object": {
				"$ref": "#/components/schemas/Object"
			  }
			},
			"type": "object"
		  },
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
			"responses": {
			  "200": {
				"content": {
				  "application/json": {
					"schema": {
						"$ref": "#/components/schemas/Response"
					}
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

func TestRouter_Ignored(t *testing.T) {
	h := func(struct{}) (any, error) { return nil, nil }
	r, spec := openapi3.NewRouter()
	routey.Get(r, "/", h, option.Ignore())

	if spec.Paths != nil {
		t.Errorf("expected no paths, got: %v", *spec.Paths.Spec)
	}
}

func TestRouter_ParamsOptMatchesHandler(t *testing.T) {
	type input struct{ Query routey.Query[int] }
	r, spec := openapi3.NewRouter()

	r.Get("/a", func(http.ResponseWriter, *http.Request) {}, option.Params[input]())
	h := func(input) (any, error) { return nil, nil }
	routey.Get(r, "/b", h)

	want := `
	{
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
	`

	aPath, _ := spec.GetPath("/a")
	test.MatchAsJSON(t, aPath, want)

	bPath, _ := spec.GetPath("/b")
	test.MatchAsJSON(t, bPath, want)
}

func TestOption_ErrorWhenMissingOpenAPI3Ctx(t *testing.T) {
	info := route.Info{}
	err := option.ID("")(&info)
	test.IsError(t, err, openapi3.ErrNoContext)
}

func TestOption_ErrorWhenOptionReturnsError(t *testing.T) {
	_, info := createInfo(t)
	var want = errors.New("test error")
	opt := option.New(func(*option.Context, *openapi3.Operation) error {
		return want
	})

	err := opt(&info)
	test.IsError(t, err, want)
}

func TestOption_NoRefErrorWhenOptionReturnsError(t *testing.T) {
	_, info := createInfo(t)
	var want = errors.New("test error")
	opt := option.New(func(*option.Context, *openapi3.Operation) error {
		return want
	})

	err := option.NoRef(opt)(&info)
	test.IsError(t, err, want)
}

func TestOption_ID(t *testing.T) {
	_, info := createInfo(t)

	err := option.ID("id")(&info)
	test.NoError(t, err)

	got := openapi3.OperationFromCtx(info.Context)
	want := openapi3.NewOperation()
	want.OperationID = "id"

	test.MatchAsJSON(t, got, want)
}

func TestOption_Params(t *testing.T) {
	_, info := createInfo(t)

	type input struct {
		Query routey.Query[int]
	}
	err := option.Params[input]()(&info)
	test.NoError(t, err)

	want := []param.Info{
		{
			Name:   "query",
			Source: "query",
			Type:   reflect.TypeOf(int(0)),
			Field:  reflect.TypeFor[input]().Field(0),
			Struct: reflect.TypeFor[input](),
		},
	}

	test.MatchAsJSON(t, info.Params, want)
}

func TestOption_ResponseDefaultContentType(t *testing.T) {
	spec, info := createInfo(t)

	type response struct {
		Field string
	}
	desc := "description"
	err := option.Response[response](http.StatusOK, desc)(&info)
	test.NoError(t, err)

	want := openapi3.NewOperation()
	{
		resp := openapi3.Response{}
		resp.Description = desc

		mt := openapi3.NewMediaType()
		mt.SetSchemaRef(spec.Schemer.RefPath + "response")

		resp.SetContent(openapi3.JSONContentType, mt)
		want.AddResponse(http.StatusOK, resp)
	}

	got := openapi3.OperationFromCtx(info.Context)
	test.MatchAsJSON(t, got, want)
}

func TestOption_ResponseContentType(t *testing.T) {
	spec, info := createInfo(t)

	type response struct {
		Field string
	}
	contentType := "contentType"
	desc := "description"
	err := option.Response[response](http.StatusOK, desc, contentType)(&info)
	test.NoError(t, err)

	want := openapi3.NewOperation()
	{
		resp := openapi3.Response{}
		resp.Description = desc

		mt := openapi3.NewMediaType()
		mt.SetSchemaRef(spec.Schemer.RefPath + "response")

		resp.SetContent(contentType, mt)
		want.AddResponse(http.StatusOK, resp)
	}

	got := openapi3.OperationFromCtx(info.Context)
	test.MatchAsJSON(t, got, want)
}

func TestOption_ContentTypeResponse(t *testing.T) {
	_, info := createInfo(t)

	contentType := "custom"
	err := option.ContentType(
		[]string{openapi3.JSONContentType, contentType},
		option.Response[struct{}](http.StatusOK, ""),
	)(&info)
	test.NoError(t, err)

	want := openapi3.NewOperation()
	{
		resp := openapi3.Response{}
		mt := openapi3.NewMediaType()
		mt.SetSchema(jsonschema.NewBuilder().Type("object").Build())

		resp.SetContent(openapi3.JSONContentType, mt)
		resp.SetContent(contentType, mt)
		want.AddResponse(http.StatusOK, resp)
	}

	got := openapi3.OperationFromCtx(info.Context)
	test.MatchAsJSON(t, got, want)
}

func TestOption_ContentTypeContext(t *testing.T) {
	info := route.Info{}
	opt := option.New(func(*option.Context, *openapi3.Operation) error {
		return nil
	})
	err := option.ContentType([]string{}, opt)(&info)
	test.IsError(t, err, openapi3.ErrNoContext)
}

func TestOption_ContentTypeOptionError(t *testing.T) {
	_, info := createInfo(t)
	want := errors.New("test error")
	opt := option.New(func(*option.Context, *openapi3.Operation) error {
		return want
	})
	err := option.ContentType([]string{}, opt)(&info)
	test.IsError(t, err, want)
}

func TestOption_NoRefResponse(t *testing.T) {
	spec, info := createInfo(t)

	type response struct {
		Field string
	}
	desc := "description"
	err := option.NoRef(
		option.Response[response](http.StatusOK, desc),
	)(&info)
	test.NoError(t, err)

	want := openapi3.NewOperation()
	{
		resp := openapi3.Response{}
		resp.Description = desc

		mt := openapi3.NewMediaType()
		schema, err := spec.Schemer.Get(reflect.TypeFor[response]())
		test.NoError(t, err)
		mt.SetSchema(schema)

		resp.SetContent(openapi3.JSONContentType, mt)
		want.AddResponse(http.StatusOK, resp)
	}

	got := openapi3.OperationFromCtx(info.Context)
	test.MatchAsJSON(t, got, want)
}

func TestOption_Ignore(t *testing.T) {
	_, info := createInfo(t)
	err := option.Ignore()(&info)
	test.NoError(t, err)

	got := openapi3.OperationFromCtx(info.Context)
	if !got.Ignore {
		t.Error("expected the operation to have ignore set")
	}
}

func TestOption_Deprecated(t *testing.T) {
	_, info := createInfo(t)
	err := option.Deprecated()(&info)
	test.NoError(t, err)

	got := openapi3.OperationFromCtx(info.Context)
	if !got.Deprecated {
		t.Error("expected the operation to have deprecated set")
	}
}

func TestOption_Summary(t *testing.T) {
	_, info := createInfo(t)
	want := "summary"
	err := option.Summary(want)(&info)
	test.NoError(t, err)

	got := openapi3.OperationFromCtx(info.Context)
	if got := got.Summary; got != want {
		t.Errorf("got summary: %q, expected: %q", got, want)
	}
}

func TestOption_Body(t *testing.T) {
	spec, info := createInfo(t)
	desc := "description"
	required := true

	type response struct{}
	err := option.Body[response](desc, required)(&info)
	test.NoError(t, err)

	want := openapi3.RequestBody{}
	{
		mt := openapi3.NewMediaType()
		mt.SetSchemaRef(spec.Schemer.RefPath + "response")

		want.Description = desc
		want.Required = required
		want.SetContent(openapi3.JSONContentType, mt)
	}

	got := openapi3.OperationFromCtx(info.Context).RequestBody
	test.MatchAsJSON(t, got, want)
}

func TestOption_NoRefNoContext(t *testing.T) {
	info := route.Info{}
	opt := option.New(func(*option.Context, *openapi3.Operation) error {
		return nil
	})
	err := option.NoRef(opt)(&info)
	test.IsError(t, err, openapi3.ErrNoContext)
}

func TestOption_ParamsError(t *testing.T) {
	_, info := createInfo(t)
	err := option.Params[int]()(&info)
	test.IsError(t, err, param.ErrNonStructArg)
}
