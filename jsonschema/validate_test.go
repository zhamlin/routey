package jsonschema_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	schema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
)

func validatorFromSchema(t *testing.T, s jsonschema.Schema) *jsonschema.Validator {
	t.Helper()

	v := jsonschema.NewValidator()
	data, err := s.MarshalJSON()
	test.NoError(t, err)

	err = v.Add("schema.json", string(data))
	test.NoError(t, err)

	return v
}

func TestValidate_AddInvalidJson(t *testing.T) {
	v := jsonschema.NewValidator()
	err := v.Add("schema.json", "{")
	test.IsError(t, err, io.ErrUnexpectedEOF)
}

func TestValidate_AddReferenceError(t *testing.T) {
	v := jsonschema.NewValidator()
	err := v.Add("schema.json", `{"$ref": "reference"}`)

	var want *schema.LoadURLError
	test.WantError(t, err, &want)
}

func TestValidation_Passes(t *testing.T) {
	s := jsonschema.NewBuilder().
		Type("object").
		Property("field", jsonschema.New()).
		Required("field").
		Build()

	v := validatorFromSchema(t, s)
	err := v.Validate("schema.json", []byte(`{
        "field": {}
    }`))

	test.NoError(t, err)
}

func TestValidation_ErrorsOnInvalidJSON(t *testing.T) {
	s := jsonschema.NewBuilder().
		Type("object").
		Build()

	v := validatorFromSchema(t, s)
	err := v.Validate("schema.json", []byte(`{`))

	var want *json.SyntaxError
	test.WantError(t, err, &want)
}

func TestValidation_ErrorMissingSchemaName(t *testing.T) {
	v := jsonschema.NewValidator()
	err := v.Validate("schema.json", nil)
	test.IsError(t, err, jsonschema.ErrSchemaNotFound)
}

func TestValidation_ErrorsContainDetailsAndLocation(t *testing.T) {
	schema := `
{
  "type": "object",
  "properties": {
    "body": {
      "$ref": "openapi.json#/components/schemas/Example"
    },
    "parameters": {
      "properties": {
        "query": {
          "properties": {
            "int": {
              "type": "integer"
            },
            "bool": {
              "type": "boolean"
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
  ]
}
    `
	openAPI := `
{
  "components": {
    "schemas": {
      "Example": {
        "properties": {},
        "type": "object"
      }
    }
  }
}
    `
	v := jsonschema.NewValidator()
	err := v.Add("openapi.json", openAPI)
	test.NoError(t, err)

	err = v.Add("schema.json", schema)
	test.NoError(t, err)

	err = v.Validate("schema.json", []byte(`
    {
        "parameters": {
            "query": {
                "int": false
            }
        }
    }`))
	var verr jsonschema.ValidationError
	test.WantError(t, err, &verr)

	expectedErrStr := `
[#/]
  [#/] missing property 'body'
  [#/parameters/query/int] got boolean, want integer
`
	got := verr.Error()
	want := strings.TrimSpace(expectedErrStr)

	if got != want {
		t.Fatalf("Errors do not match\nwant:\n%s\ngot:\n%s", want, got)
	}

	if l := len(verr.Causes); l != 2 {
		t.Fatalf("expected two causes for the validation failure, got: %d", l)
	}

	c := verr.Causes[0]
	if c.Message != "missing property 'body'" {
		t.Errorf("unexpected first error message, got: %v", c.Message)
	}

	// missing field has no location
	if c.Location != "/" {
		t.Errorf("unexpected first error location, got: %v", c.Location)
	}

	c = verr.Causes[1]
	if c.Message != "got boolean, want integer" {
		t.Errorf("unexpected second error message, got: %v", c.Message)
	}

	if c.Location != "/parameters/query/int" {
		t.Errorf("uexpected second error location, got: %v", c.Location)
	}
}

type object struct {
	Field string `json:"field"`
}

func (object) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Property("Field").
		MinLength(5)
}

func BenchmarkValidate(b *testing.B) {
	schemer := jsonschema.NewSchemer()
	schema, err := schemer.Get(object{})
	test.NoError(b, err)

	validator := jsonschema.NewValidator()
	data, err := schema.MarshalJSON()
	test.NoError(b, err)

	name := "schema.json"
	validator.Add(name, string(data))

	toJson := func(o object) []byte {
		data, err = json.Marshal(&o)
		test.NoError(b, err)
		return data
	}

	b.Run("no error", func(b *testing.B) {
		data := toJson(object{
			Field: "FieldValue",
		})

		for b.Loop() {
			validator.Validate(name, data)
		}
	})

	b.Run("with error", func(b *testing.B) {
		data := toJson(object{
			Field: "",
		})

		for b.Loop() {
			validator.Validate(name, data)
		}
	})
}
