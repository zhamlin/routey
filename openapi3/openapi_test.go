package openapi3_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/openapi3"
)

func TestOpenAPI_SetDefaultResponse(t *testing.T) {
	spec := openapi3.New()
	type DefaultResponse struct {
		Error string
	}
	openapi3.SetDefaultResponse[DefaultResponse](spec, 0)

	test.MatchAsJSON(t, spec.Components, `
	{
	  "responses": {
		"default": {
		  "content": {
			"application/json": {
			  "schema": {
				"$ref": "#/components/schemas/DefaultResponse"
			  }
			}
		  }
		}
	  },
	  "schemas": {
		"DefaultResponse": {
		  "properties": {
			"Error": {
			  "type": "string"
			}
		  },
		  "type": "object"
		}
	  }
	}
	`)
}

func TestOpenAPI_SetDefaultResponseWithCode(t *testing.T) {
	spec := openapi3.New()
	type DefaultResponse struct {
		Error string
	}
	openapi3.SetDefaultResponse[DefaultResponse](spec, http.StatusBadRequest)

	test.MatchAsJSON(t, spec.Components, `
	{
	  "responses": {
		"400": {
		  "content": {
			"application/json": {
			  "schema": {
				"$ref": "#/components/schemas/DefaultResponse"
			  }
			}
		  }
		}
	  },
	  "schemas": {
		"DefaultResponse": {
		  "properties": {
			"Error": {
			  "type": "string"
			}
		  },
		  "type": "object"
		}
	  }
	}
	`)
}

func TestOpenAPI_JsonHasInfo(t *testing.T) {
	spec := openapi3.New()
	spec.Info.Spec.Title = "Title"
	spec.Info.Spec.Version = "0.0.1"

	test.MatchAsJSON(t, spec, `
	{
	  "info": {
		"title": "Title",
		"version": "0.0.1"
	  },
	  "openapi": "3.1.1"
	}
	`)
}

func TestOpenAPI_Path(t *testing.T) {
	spec := openapi3.New()

	op := openapi3.NewOperation()
	resp := openapi3.Response{}
	resp.Description = "description"

	mediaType := openapi3.NewMediaType()
	mediaType.SetSchemaRef("response")

	resp.SetContent(openapi3.JSONContentType, mediaType)
	op.AddResponse(http.StatusOK, resp)

	path := openapi3.NewPathItem()
	path.SetOperation(http.MethodGet, op)
	spec.SetPath("/", path)

	test.MatchAsJSON(t, spec, `
	{
	  "info": {
		"title": "",
		"version": ""
	  },
	  "openapi": "3.1.1",
	  "paths": {
		"/": {
		  "get": {
			"responses": {
			  "200": {
				"description": "description",
				"content": {
				  "application/json": {
					"schema": {
					  "$ref": "response"
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

func TestOpenAPI_RegisterType(t *testing.T) {
	spec := openapi3.New()
	openapi3.RegisterType[time.Time](spec, jsonschema.NewDateTimeSchema())

	test.MatchAsJSON(t, spec.Components.Spec.Schemas, `
	{
		"Time": {
			"format": "date-time",
			"type": "string"
		}
	}
	`)
}

func TestOpenAPI_RegisterTypeCustomName(t *testing.T) {
	spec := openapi3.New()
	openapi3.RegisterType[time.Time](spec,
		jsonschema.NewDateTimeSchema(),
		jsonschema.Name("date"),
	)

	test.MatchAsJSON(t, spec.Components.Spec.Schemas, `
	{
		"date": {
			"format": "date-time",
			"type": "string"
		}
	}
	`)
}

func TestOpenAPI_RegisterTypeNoRef(t *testing.T) {
	spec := openapi3.New()
	openapi3.RegisterType[time.Time](spec,
		jsonschema.NewDateTimeSchema(),
		jsonschema.NoRef(),
	)

	type foo struct{ Date time.Time }
	_, err := spec.GetSchemaOrRef(foo{}, openapi3.SchemaRefOptions{})
	test.NoError(t, err)

	test.MatchAsJSON(t, spec.Components.Spec.Schemas, `
	{
		"foo": {
			"properties": {
				"Date": {
					"format": "date-time",
					"type": "string"
				}
			},
			"type": "object"
		}
	}
	`)
}

func TestGetSchemaOrRef_MultipleStructs(t *testing.T) {
	type Bar struct {
		Field string `json:"bar"`
	}
	type Foo struct {
		Bar Bar `json:"bar"`
	}
	spec := openapi3.New()
	_, err := spec.GetSchemaOrRef(Foo{}, openapi3.SchemaRefOptions{})
	test.NoError(t, err)

	test.MatchAsJSON(t, spec.Components, `
	{
	  "schemas": {
		"Bar": {
		  "properties": {
			"bar": {
			  "type": "string"
			}
		  },
		  "type": "object"
		},
		"Foo": {
		  "properties": {
			"bar": {
			  "$ref": "#/components/schemas/Bar"
			}
		  },
		  "type": "object"
		}
	  }
	}
	`)
}

func TestGetSchemaOrRef_NoRefTypes(t *testing.T) {
	type Bar struct {
		Field string `json:"bar"`
	}
	type Foo struct {
		Bar Bar `json:"bar"`
	}
	spec := openapi3.New()

	_, err := spec.GetSchemaOrRef(Foo{}, openapi3.SchemaRefOptions{
		ForceNoRef:            true,
		IgnoreAddSchemaErrors: true,
	})
	test.NoError(t, err)

	test.MatchAsJSON(t, spec.Components, `
	{
	  "schemas": {
		"Bar": {
		  "properties": {
			"bar": {
			  "type": "string"
			}
		  },
		  "type": "object"
		}
	  }
	}
	`)
}
