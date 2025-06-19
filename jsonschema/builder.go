package jsonschema

import (
	"strings"

	"github.com/sv-tools/openapi"
	"github.com/zhamlin/routey/internal/stringz"
)

func newBuilderWithSchema(schema *openapi.Schema) Builder {
	return Builder{
		Schema:        schema,
		StringBuilder: StringBuilder{schema},
		NumberBuilder: NumberBuilder{schema},
		ArrayBuilder:  ArrayBuilder{schema},
		ObjectBuilder: ObjectBuilder{schema},
	}
}

// Builder provides functions to build a json schema.
type Builder struct {
	StringBuilder
	NumberBuilder
	ArrayBuilder
	ObjectBuilder

	Schema *openapi.Schema
}

// NewBuilder returns an empty [Builder].
func NewBuilder() Builder {
	s := New()
	return newBuilderWithSchema(&s.Schema)
}

// Reference will set the schema to be a $ref item pointing
// to the provided ref.
func (b Builder) Reference(ref string) Schema {
	schema := New()
	schema.Schema = *b.Schema
	schema.refPath = ref
	return schema
}

func (b Builder) Build() Schema {
	schema := New()
	schema.Schema = *b.Schema
	return schema
}

func (b Builder) Type(typs ...Type) Builder {
	types := make([]string, 0, len(typs))
	for _, typ := range typs {
		types = append(types, string(typ))
	}

	b.Schema.Type = openapi.NewSingleOrArray(types...)
	return b
}

func (b Builder) Description(desc string) Builder {
	b.Schema.Description = strings.Trim(stringz.TrimLinesSpace(desc), "\n")
	return b
}

func (b Builder) Title(value string) Builder {
	b.Schema.Title = value
	return b
}

func (b Builder) Default(value any) Builder {
	b.Schema.Default = value
	return b
}

func (b Builder) Const(value string) Builder {
	b.Schema.Const = value
	return b
}

func (b Builder) Examples(values ...any) Builder {
	b.Schema.Examples = values
	return b
}

func (b Builder) Enum(values ...any) Builder {
	b.Schema.Enum = values
	return b
}

func (b Builder) Deprecated(value bool) Builder {
	b.Schema.Deprecated = value
	return b
}

func (b Builder) WriteOnly(value bool) Builder {
	b.Schema.WriteOnly = value
	return b
}

func (b Builder) ReadOnly(value bool) Builder {
	b.Schema.ReadOnly = value
	return b
}

// ObjectBuilder provides functions for object related options on the schema.
type ObjectBuilder struct {
	Schema *openapi.Schema
}

func (o ObjectBuilder) Build() Schema {
	schema := New()
	schema.Schema = *o.Schema
	return schema
}

func (o ObjectBuilder) Property(name string, schema Schema) ObjectBuilder {
	if o.Schema.Properties == nil {
		o.Schema.Properties = map[string]*openapi.RefOrSpec[openapi.Schema]{}
	}

	if schema.refPath != "" {
		o.Schema.Properties[name] = openapi.NewRefOrSpec[openapi.Schema](schema.refPath)
	} else {
		o.Schema.Properties[name] = openapi.NewRefOrSpec[openapi.Schema](schema.Schema)
	}
	return o
}

func (o ObjectBuilder) Required(vals ...string) ObjectBuilder {
	o.Schema.Required = append(o.Schema.Required, vals...)
	return o
}

func (o ObjectBuilder) MaxProperties(n int) ObjectBuilder {
	o.Schema.MaxProperties = &n
	return o
}

func (o ObjectBuilder) MinProperties(n int) ObjectBuilder {
	o.Schema.MinProperties = &n
	return o
}

// StringBuilder provides functions for string related options on the schema.
type StringBuilder struct {
	Schema *openapi.Schema
}

func (s StringBuilder) Build() Schema {
	schema := New()
	schema.Schema = *s.Schema
	return schema
}

func (s StringBuilder) Format(value Format) StringBuilder {
	s.Schema.Format = string(value)
	return s
}

func (s StringBuilder) Pattern(value string) StringBuilder {
	s.Schema.Pattern = value
	return s
}

func (s StringBuilder) MaxLength(n int) StringBuilder {
	s.Schema.MaxLength = &n
	return s
}

func (s StringBuilder) MinLength(n int) StringBuilder {
	s.Schema.MinLength = &n
	return s
}

func (s StringBuilder) Length(n int) StringBuilder {
	s.MinLength(n)
	s.MaxLength(n)
	return s
}

// NumberBuilder provides functions for number related options on the schema.
type NumberBuilder struct {
	Schema *openapi.Schema
}

func (nb NumberBuilder) Build() Schema {
	schema := New()
	schema.Schema = *nb.Schema
	return schema
}

func (nb NumberBuilder) MultipleOf(n int) NumberBuilder {
	nb.Schema.MultipleOf = &n
	return nb
}

func (nb NumberBuilder) Maximum(n int) NumberBuilder {
	nb.Schema.Maximum = &n
	return nb
}

func (nb NumberBuilder) ExclusiveMaximum(n int) NumberBuilder {
	nb.Schema.ExclusiveMaximum = &n
	return nb
}

func (nb NumberBuilder) Minimum(n int) NumberBuilder {
	nb.Schema.Minimum = &n
	return nb
}

func (nb NumberBuilder) ExclusiveMinimum(n int) NumberBuilder {
	nb.Schema.ExclusiveMinimum = &n
	return nb
}

// ArrayBuilder provides functions for array related options on the schema.
type ArrayBuilder struct {
	Schema *openapi.Schema
}

func (a ArrayBuilder) Build() Schema {
	schema := New()
	schema.Schema = *a.Schema
	return schema
}

func (a ArrayBuilder) MaxContains(n int) ArrayBuilder {
	a.Schema.MaxContains = &n
	return a
}

func (a ArrayBuilder) MinContains(n int) ArrayBuilder {
	a.Schema.MinContains = &n
	return a
}

func (a ArrayBuilder) MaxItems(n int) ArrayBuilder {
	a.Schema.MaxItems = &n
	return a
}

func (a ArrayBuilder) MinItems(n int) ArrayBuilder {
	a.Schema.MinItems = &n
	return a
}

func (a ArrayBuilder) UniqueItems() ArrayBuilder {
	t := true
	a.Schema.UniqueItems = &t
	return a
}
