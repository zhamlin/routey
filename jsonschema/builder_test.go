package jsonschema_test

import (
	"testing"

	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
)

func TestSchemaBuilderValues(t *testing.T) {
	s := jsonschema.NewBuilder().
		Type("string").
		Title("title").
		Description("description").
		Default("default").
		Const("const").
		Examples("examples").
		Enum("a", "b").
		Deprecated(true).
		WriteOnly(true).
		ReadOnly(true).
		Build()

	test.MatchAsJSON(t, s, `
{
 "type": "string",
 "default": "default",
 "title": "title",
 "description": "description",
 "const": "const",
 "enum": [
  "a",
  "b"
 ],
 "examples": [
  "examples"
 ],
 "readOnly": true,
 "writeOnly": true,
 "deprecated": true
}`)
}

func TestObjectBuilderValues(t *testing.T) {
	s := jsonschema.NewBuilder().
		ObjectBuilder.
		Property("property", jsonschema.NewDateTimeSchema()).
		Property("property2", jsonschema.New()).
		Required("property").
		Property("ref", jsonschema.NewBuilder().Reference("reference")).
		MinProperties(1).
		MaxProperties(2).
		Build()

	test.MatchAsJSON(t, s, `
{
 "properties": {
  "property2": {},
  "property": {
	"format": "date-time",
	"type": "string"
  },
  "ref": {
   "$ref": "reference"
  }
 },
 "minProperties": 1,
 "maxProperties": 2,
 "required": [
  "property"
 ]
}`)
}

func TestStringBuilderValues(t *testing.T) {
	s := jsonschema.NewBuilder().
		StringBuilder.
		Format("format").
		Length(1).
		Pattern("pattern").
		Build()

	test.MatchAsJSON(t, s, `
{
 "minLength": 1,
 "maxLength": 1,
 "pattern": "pattern",
 "format": "format"
}`)
}

func TestNumberBuilderValues(t *testing.T) {
	s := jsonschema.NewBuilder().
		NumberBuilder.
		ExclusiveMinimum(1).
		ExclusiveMaximum(1).
		Minimum(1).
		Maximum(1).
		MultipleOf(1).
		Build()

	test.MatchAsJSON(t, s, `
{
 "multipleOf": 1,
 "minimum": 1,
 "exclusiveMinimum": 1,
 "maximum": 1,
 "exclusiveMaximum": 1
}`)
}

func TestArrayBuilderValues(t *testing.T) {
	s := jsonschema.NewBuilder().
		ArrayBuilder.
		MinItems(1).
		MaxItems(1).
		MinContains(1).
		MaxContains(1).
		UniqueItems().
		Build()

	test.MatchAsJSON(t, s, `
{
 "maxItems": 1,
 "minContains": 1,
 "maxContains": 1,
 "minItems": 1,
 "uniqueItems": true
}`)
}
