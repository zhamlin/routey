package openapi3_test

import (
	"maps"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/openapi3"
	"github.com/zhamlin/routey/openapi3/option"
	"github.com/zhamlin/routey/route"
)

func Named() {}

func newOptionsCtx() (*routey.Router, option.Context) {
	r := routey.New()
	openapi := openapi3.AddSpecToRouter(r, openapi3.AddSpecToRouterOpts{})
	openapi.Schemer.RefPath = "/schemas/"

	info := route.Info{}
	info.Handler = Named
	info.Context = maps.Clone(r.Context)

	return r, option.Context{
		Context: openapi3.Context{
			OpenAPI: openapi,
			Namer:   r.Params.Namer,
			Parser:  r.Params.Parser,
		},
		Info: &info,
	}
}

func compareOpSchema(t *testing.T, op openapi3.Operation, want any) {
	t.Helper()

	s, err := openapi3.SchemaFromOp(op, openapi3.JSONContentType)
	test.NoError(t, err)

	test.MatchAsJSON(t, s, want)
}

func TestSchemaFromOp(t *testing.T) {
	r, ctx := newOptionsCtx()

	err := option.Body[struct {
		Field string `json:"field"`
	}]("description", true)(ctx.Info)
	test.NoError(t, err)

	err = option.Params[struct {
		Param routey.Path[string]
	}]()(ctx.Info)
	test.NoError(t, err)

	err = r.OnRouteAdd(ctx.Info)
	test.NoError(t, err)

	got := openapi3.OperationFromCtx(ctx.Info.Context)
	want := `{
        "description": "Contains the request body and all parameters",
        "properties": {
            "body": {
                "properties": {
                    "field": {
                        "type": "string"
                    }
                },
                "type": "object"
            },
            "parameters": {
                "description": "Contains the parameters",
                "properties": {
                    "path": {
                        "properties": {
                            "param": {
                                "type": "string"
                            }
                        },
                        "required": [
                            "param"
                        ],
                        "type": "object"
                    }
                },
                "required": [
                    "path"
                ],
                "type": "object"
            }
        },
        "required": [
            "parameters",
            "body"
        ],
        "type": "object"
    }`
	compareOpSchema(t, *got, want)
}

func TestSchemaFromOpWithRef(t *testing.T) {
	r, ctx := newOptionsCtx()
	type reqBody struct {
		Field string `json:"field"`
	}

	err := option.Body[reqBody]("description", true)(ctx.Info)
	test.NoError(t, err)

	err = option.Params[struct {
		Field routey.Query[string]
	}]()(ctx.Info)
	test.NoError(t, err)

	err = r.OnRouteAdd(ctx.Info)
	test.NoError(t, err)

	got := openapi3.OperationFromCtx(ctx.Info.Context)
	want := `{
        "description": "Contains the request body and all parameters",
        "properties": {
            "body": {
                "$ref": "/schemas/reqBody"
            },
            "parameters": {
                "description": "Contains the parameters",
                "properties": {
                    "query": {
                        "properties": {
                            "field": {
								"type": "string"
                            }
                        },
                        "type": "object"
                    }
                },
                "type": "object"
            }
        },
        "required": [
            "parameters",
            "body"
        ],
        "type": "object"
    }`
	compareOpSchema(t, *got, want)
}
