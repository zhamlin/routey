package jsonschema_test

import (
	"testing"
	"time"

	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
)

func matchJSON(t *testing.T, s jsonschema.Schemer, obj any, want string) {
	t.Helper()

	schema, err := s.Get(obj)
	if err != nil {
		t.Fatalf("schemer.Get: expected no error, got: %v", err)
	}

	test.MatchAsJSON(t, schema, want)
}

func TestSchemaPropertyMissing(t *testing.T) {
	s := jsonschema.NewBuilder().
		Property("field", jsonschema.New()).
		Build()

	paniced := false

	func() {
		defer func() {
			if r := recover(); r != nil {
				paniced = true
			}
		}()
		s.Property("invalid")
	}()

	if !paniced {
		t.Error("expected schema.Property to panic for non existing property")
	}
}

func TestSchemaMarshalJSON(t *testing.T) {
	s := jsonschema.NewBuilder().Type("string").Build()
	test.MatchAsJSON(t, s, `{"type": "string"}`)
}

func TestSchemaMarshalJSONRef(t *testing.T) {
	s := jsonschema.NewBuilder().Reference("example")
	test.MatchAsJSON(t, s, `{"$ref": "example"}`)
}

func TestSchema(t *testing.T) {
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			obj: true,
			want: `{
                "type": "boolean"
            }`,
		},
		{
			obj: int(1),
			want: `{
                "type": "integer"
            }`,
		},
		{
			obj: int8(1),
			want: `{
                "type": "integer"
            }`,
		},
		{
			obj: int16(1),
			want: `{
                "type": "integer"
            }`,
		},
		{
			obj: int32(1),
			want: `{
                "type": "integer",
                "format": "int32"
            }`,
		},
		{
			obj: int64(1),
			want: `{
                "type": "integer",
                "format": "int64"
            }`,
		},
		{
			obj: float32(1.0),
			want: `{
                "type": "number",
                "format": "float"
            }`,
		},
		{
			obj: float64(1.0),
			want: `{
                "type": "number",
                "format": "float"
            }`,
		},
		{
			obj: uint(1),
			want: `{
                "type": "integer",
                "minimum": 0
            }`,
		},
		{
			obj: uint8(1),
			want: `{
                "type": "integer",
                "minimum": 0
            }`,
		},
		{
			obj: uint16(1),
			want: `{
                "type": "integer",
                "minimum": 0
            }`,
		},
		{
			obj: uint32(1),
			want: `{
                "type": "integer",
                "minimum": 0,
                "format": "int32"
            }`,
		},
		{
			obj: uint64(1),
			want: `{
                "type": "integer",
                "minimum": 0,
                "format": "int64"
            }`,
		},
		{
			obj: "string",
			want: `{
                "type": "string"
            }`,
		},
		{
			obj: []string{"a", "b", "c"},
			want: `{
                "type": "array",
                "items": {
                    "type": "string"
                }
            }`,
		},
		{
			obj: map[string]string{
				"a": "1",
				"b": "2",
			},
			want: `{
                "type": "object",
                "additionalProperties": {
                    "type": "string"
                }
            }`,
		},
		{
			obj: map[string]any{
				"a": 1,
				"b": "2",
			},
			want: `{
                "type": "object"
            }`,
		},
		{
			obj: struct {
				F string
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "F": {
                        "type": "string"
                    }
                }
            }`,
		},
		{
			name: "pointer is nullable",
			obj: struct {
				F *string
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "F": {
                        "type": ["string", "null"]
                    }
                }
            }`,
		},
		{
			name: "field ignored",
			obj: struct {
				F string `json:"-"`
			}{},
			want: `{
                "type": "object"
            }`,
		},
		{
			name: "json tag used for name",
			obj: struct {
				F string `json:"field"`
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "field": {
                        "type": "string"
                    }
                }
            }`,
		},
		{
			name: "nested structs",
			obj: struct {
				F      string `json:"field"`
				Nested struct {
					F int `json:"nested_field"`
				} `json:"nested"`
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "field": {
                        "type": "string"
                    },
                    "nested": {
                        "properties": {
                            "nested_field": {
                                "type": "integer"
                            }
                        },
                        "type": "object"
                    }
                }
            }`,
		},
		{
			name: "omitempty ignored",
			obj: struct {
				F string `json:"f,omitempty"`
				A string `json:"a,omitempty"`
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "f": {
                        "type": "string"
                    },
                    "a": {
                        "type": "string"
                    }
                }
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()
	schemer.RefPath = ""

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

type extendedSchema struct {
	Items []string `json:"items"`
}

func (extendedSchema) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Property("items").
		Description("Array of items").
		MaxLength(50)
	s.Description = "ExtendedSchema"
}

func TestSchemaExtended(t *testing.T) {
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			obj: extendedSchema{},
			want: `{
                "type": "object",
                "description": "ExtendedSchema",
                "properties": {
                    "items": {
                        "type": "array",
                        "description": "Array of items",
                        "maxLength": 50,
                        "items": {
                            "type": "string"
                        }
                    }
                }
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()
	schemer.RefPath = ""

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

type testJSONSchema struct{}

func (testJSONSchema) JSONSchema() jsonschema.Schema {
	return jsonschema.NewBuilder().
		Type(jsonschema.TypeString).
		Description("Example description").
		Build()
}

func TestJSONSchema(t *testing.T) {
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			obj: testJSONSchema{},
			want: `{
                "type": "string",
                "description": "Example description"
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()
	schemer.RefPath = ""

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

func TestSchemaEmbeded(t *testing.T) {
	type Foo struct {
		F string
	}
	type Bar struct {
		B string
	}
	type FooBar struct {
		Foo
		Bar
	}
	type EmbeddedFoo struct {
		Foo
	}

	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			obj: FooBar{},
			want: `{
                "type": "object",
                "properties": {
                    "F": {
                        "type": "string"
                    },
                    "B": {
                        "type": "string"
                    }
                }
            }`,
		},
		{
			obj: EmbeddedFoo{},
			want: `{
                "type": "object",
                "description": "FooObject",
                "properties": {
                    "F": {
                        "type": "string"
                    }
                }
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()
	schemer.RefPath = ""
	{
		s, err := schemer.Get(Foo{})
		if err != nil {
			t.Fatal(err)
		}
		// Show that EmbeddedFoo will have the description
		// as well
		s.Description = "FooObject"
		schemer.Set(Foo{}, s)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

func TestSchemaRef(t *testing.T) {
	type A struct {
		Name string
	}
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			name: "show reference is created when seeing struct for the first time",
			obj: struct {
				A A
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "A": {
                        "$ref": "/schemas/A"
                    }
                }
            }`,
		},
		{
			name: "show the objects schema is returned vs a reference",
			obj:  A{},
			want: `{
                "type": "object",
                "properties": {
                    "Name": {
                        "type": "string"
                    }
                }
            }`,
		},
		{
			name: "array items use a ref",
			obj:  []A{},
			want: `{
                "type": "array",
                "items": {
                    "$ref": "/schemas/A"
                }
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

func TestSchemaCustomTypes(t *testing.T) {
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			name: "show reference is created when seeing struct for the first time",
			obj:  time.Time{},
			want: `{
                "type": "string",
                "format": "date-time"
            }`,
		},
		{
			name: "show inline option is respected",
			obj: struct {
				DateCreated time.Time `json:"date_created"`
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "date_created": {
                        "type": "string",
                        "format": "date-time"
                    }
                }
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()
	dateTime := jsonschema.NewBuilder().
		Type(jsonschema.TypeString).
		Format(jsonschema.FormatDateTime).
		Build()
	schemer.Set(time.Time{}, dateTime, jsonschema.NoRef())

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

func TestSchemaModifiers(t *testing.T) {
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			name: "show reference is created when seeing struct for the first time",
			obj: struct {
				Default string `json:"default" default:"hello world"`
			}{},
			want: `{
                "type": "object",
                "properties": {
                    "default": {
                        "type": "string",
                        "default": "hello world"
                    }
                }
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}

func TestSchemaStructFieldsRequired(t *testing.T) {
	tests := []struct {
		name string
		obj  any
		want string
	}{
		{
			obj: struct {
				Field         int
				OptionalField *int
			}{},
			want: `{
                "properties": {
                    "Field": {
                        "type": "integer"
                    },
                    "OptionalField": {
                        "type": [
                            "integer",
                            "null"
                        ]
                    }
                },
                "required": [
                    "Field"
                ],
                "type": "object"
            }`,
		},
	}

	schemer := jsonschema.NewSchemer()
	schemer.RefPath = ""
	schemer.DefaultStructRequire = true

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matchJSON(t, schemer, test.obj, test.want)
		})
	}
}
