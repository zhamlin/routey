package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/sv-tools/openapi"
)

func getTypeName(typ reflect.Type) string {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.Name()
}

// Schema represents a json schema object.
//
// https://json-schema.org/overview/what-is-jsonschema
type Schema struct {
	openapi.Schema

	// if set the schema will be marshalled as reference
	refPath string

	noRef bool
	name  string
}

// New returns an empty [Schema].
func New() Schema {
	return Schema{}
}

// NewDateTimeSchema returns a [Schema] representing
// strings in the `date-time` format.
func NewDateTimeSchema() Schema {
	return NewBuilder().
		Type(openapi.StringType).
		Format(openapi.DateTimeFormat).
		Build()
}

// MarshalJSON implements the [json.Marshaler] interface.
func (s Schema) MarshalJSON() ([]byte, error) {
	if ref := s.refPath; ref != "" {
		return json.Marshal(openapi.NewRefOrSpec[openapi.Schema](ref))
	}
	return json.Marshal(s.Schema)
}

// Property returns a [Builder] for the property matching
// the supplied name. The returned builder will modify the
// schema directly.
//
// This function panics if no property exists on the schema with the given
// name.
func (s *Schema) Property(name string) Builder {
	p, has := s.Properties[name]
	if !has {
		keys := slices.Collect(maps.Keys(s.Properties))
		errStr := fmt.Sprintf(
			"%s: property does not exist: %s\nhave: %v", s.name, name, keys,
		)
		panic(errStr)
	}

	if p.Spec == nil {
		msg := fmt.Sprintf("empty spec for %q, references(%v) not supported",
			name, p.Ref,
		)
		panic(msg)
	}

	return newBuilderWithSchema(p.Spec)
}

func (s *Schema) Name() string {
	return s.name
}

func (s *Schema) NoRef() bool {
	return s.noRef
}

func (s *Schema) GetType() []string {
	if s.Type == nil {
		return nil
	}

	return *s.Type
}

func (s *Schema) hasType() bool {
	return len(s.GetType()) > 0
}

// Schemer creates [Schema] from provided types.
type Schemer struct {
	// DefaultStructRequire will mark all struct fields
	// on the object schema as required unless it is a pointer.
	DefaultStructRequire bool

	// RefPath determines whether or not the schema will have
	// any $ref items in it. When empty all schemas will be inlined.
	// When set $ref will be $RefPath$TypeName.
	//
	// Defaults to `/schemas`.
	RefPath string

	// GetTypeName is used to get a name for the schema
	// from a given type.
	//
	// This defaults to the name from reflect.Type Name.
	GetTypeName func(reflect.Type) string

	types map[reflect.Type]Schema
}

// NewSchemer returns a [Schemer] with the default values set.
func NewSchemer() Schemer {
	return Schemer{
		types:                map[reflect.Type]Schema{},
		RefPath:              "/schemas/",
		GetTypeName:          getTypeName,
		DefaultStructRequire: false,
	}
}

func (s Schemer) Has(obj any) bool {
	var typ reflect.Type
	if t, ok := obj.(reflect.Type); ok {
		typ = t
	} else {
		typ = reflect.TypeOf(obj)
	}

	_, exists := s.types[typ]
	return exists
}

// Get returns a [Schema] from the provided type.
func (s Schemer) Get(obj any) (Schema, error) {
	if t, ok := obj.(reflect.Type); ok {
		return s.schemaFromType(t)
	}
	return s.schemaFromType(reflect.TypeOf(obj))
}

// Get returns a [Schema] from the provided type.
func (s Schemer) GetSchemaByRef(wantRef string) (Schema, bool) {
	for _, schema := range s.types {
		ref := s.NewRef(schema.Name())
		if ref == wantRef {
			return schema, true
		}
	}
	return Schema{}, false
}

// Set updates the provided types schema to the supplied one.
func (s Schemer) Set(obj any, schema Schema, options ...Option) Schema {
	var typ reflect.Type
	if t, ok := obj.(reflect.Type); ok {
		typ = t
	} else {
		typ = reflect.TypeOf(obj)
	}

	for _, option := range options {
		schema = option(schema)
	}

	if schema.name == "" {
		schema.name = s.GetTypeName(typ)
	}
	s.types[typ] = schema
	return schema
}

// NewRef returns string with [Schemer].RefPath prefixed to it.
func (s Schemer) NewRef(name string) string {
	if name == "" {
		return ""
	}
	return s.RefPath + name
}

func (s Schemer) refOrSpec(
	t reflect.Type,
	schema Schema,
	useRef bool,
) *openapi.RefOrSpec[openapi.Schema] {
	specOrRef := openapi.NewRefOrSpec[openapi.Schema](schema.Schema)
	// if this type already exists maybe create a ref
	if fieldSchema, has := s.types[t]; has && useRef {
		ref := s.NewRef(fieldSchema.name)
		specOrRef = openapi.NewRefOrSpec[openapi.Schema](ref)
	}
	return specOrRef
}

func (s Schemer) schemaFromType(typ reflect.Type) (Schema, error) {
	if typ == nil {
		return New(), nil
	}

	if schema, exists := s.types[typ]; exists {
		return schema, nil
	}

	if v, ok := reflect.New(typ).Interface().(schemer); ok {
		return s.handleCustomSchemer(v, typ), nil
	}

	schema, err := s.createSchemaByKind(typ)
	if err != nil {
		return schema, err
	}

	return s.applyExtensions(typ, schema)
}

func (s Schemer) handleCustomSchemer(schemer schemer, typ reflect.Type) Schema {
	schema := schemer.JSONSchema()
	if schema.name == "" {
		schema.name = s.GetTypeName(typ)
	}
	s.types[typ] = schema
	return schema
}

//nolint:cyclop
func (s Schemer) createSchemaByKind(typ reflect.Type) (Schema, error) {
	kind := typ.Kind()

	switch kind {
	case reflect.Bool:
		return createBoolSchema(), nil
	case reflect.String:
		return createStringSchema(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return createIntSchema(kind), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return createUintSchema(kind), nil
	case reflect.Float32, reflect.Float64:
		return createFloatSchema(), nil
	case reflect.Slice, reflect.Array:
		return s.createArraySchema(typ)
	case reflect.Ptr:
		return s.createPointerSchema(typ)
	case reflect.Map:
		return s.createMapSchema(typ)
	case reflect.Struct:
		return s.createStructSchema(typ)
	case reflect.Interface:
		return New(), nil
	default:
		err := fmt.Errorf(
			"type: %s: reflect.kind: %s: %w",
			typ.String(), kind.String(), ErrUnknowType,
		)
		return New(), err
	}
}

func (s Schemer) createArraySchema(typ reflect.Type) (Schema, error) {
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.ArrayType)

	arrayItemSchema, err := s.schemaFromType(typ.Elem())
	if err != nil {
		return schema, err
	}

	if arrayItemSchema.hasType() {
		specOrRef := s.refOrSpec(
			typ.Elem(),
			arrayItemSchema,
			s.useRefs() && !arrayItemSchema.noRef,
		)
		schema.Items = openapi.NewBoolOrSchema(specOrRef)
	}

	return schema, nil
}

func (s Schemer) createPointerSchema(typ reflect.Type) (Schema, error) {
	typeSchema, err := s.schemaFromType(typ.Elem())
	if err != nil {
		return typeSchema, err
	}

	// Pointers can be null
	typeSchema.Type.Add(openapi.NullType)
	return typeSchema, nil
}

func (s Schemer) createMapSchema(typ reflect.Type) (Schema, error) {
	// Only support maps with string keys
	if typ.Key().Kind() != reflect.String {
		return New(), errInvalidMapKey
	}

	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.ObjectType)

	mapItemSchema, err := s.schemaFromType(typ.Elem())
	if err != nil {
		return schema, err
	}

	if mapItemSchema.hasType() {
		refOrSpec := openapi.NewRefOrSpec[openapi.Schema](mapItemSchema.Schema)
		schema.AdditionalProperties = openapi.NewBoolOrSchema(refOrSpec)
	}

	return schema, nil
}

func (s Schemer) createStructSchema(typ reflect.Type) (Schema, error) {
	structSchema, err := s.schemaFromStruct(typ)
	if err != nil {
		return structSchema, err
	}

	structSchema.name = s.GetTypeName(typ)

	if typ.Implements(noReferType) {
		structSchema.noRef = true
	}

	s.types[typ] = structSchema
	return structSchema, nil
}

func (s Schemer) applyExtensions(typ reflect.Type, schema Schema) (Schema, error) {
	v, ok := reflect.New(typ).Interface().(schemerExtended)
	if !ok {
		return schema, nil
	}

	v.JSONSchemaExtend(&schema)

	if typ.Kind() == reflect.Struct {
		s.types[typ] = schema
	}

	return schema, nil
}

func (s Schemer) useRefs() bool {
	return s.RefPath != ""
}

func (s Schemer) addObjectRequired(field reflect.StructField, schema, fieldSchema Schema) Schema {
	fieldName := JSONFieldName(field)
	if fieldName != "" {
		shouldUseRef := s.useRefs() && !fieldSchema.noRef
		specOrRef := s.refOrSpec(field.Type, fieldSchema, shouldUseRef)
		schema.Properties[fieldName] = specOrRef

		if s.DefaultStructRequire && field.Type.Kind() != reflect.Ptr {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

func (s Schemer) schemaFromStruct(typ reflect.Type) (Schema, error) {
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.ObjectType)
	schema.Properties = map[string]*openapi.RefOrSpec[openapi.Schema]{}

	fieldCount := typ.NumField()
	updateSchema := func(field reflect.StructField) error {
		_, hasFieldType := s.types[field.Type]

		fieldSchema, err := s.schemaFromType(field.Type)
		if err != nil {
			return err
		}
		fieldSchema.Schema = loadSchemaOptions(field, fieldSchema.Schema)

		if field.Anonymous {
			if !hasFieldType {
				// remove anonymous field type from the schema map
				// if it did not already exist
				delete(s.types, field.Type)
			}

			if fieldCount == 1 {
				schema = fieldSchema
			} else {
				maps.Copy(schema.Properties, fieldSchema.Properties)
			}

			return nil
		}

		schema = s.addObjectRequired(field, schema, fieldSchema)
		return nil
	}

	for i := range fieldCount {
		field := typ.Field(i)
		if err := updateSchema(field); err != nil {
			return schema, err
		}
	}

	return schema, nil
}

type Option func(Schema) Schema

// NoRef will cause the schema to always be used
// directly instead of a reference to it.
func NoRef() Option {
	return func(s Schema) Schema {
		s.noRef = true
		return s
	}
}

func Name(name string) Option {
	return func(s Schema) Schema {
		s.name = name
		return s
	}
}

type noRefer interface {
	NoRef()
}

type schemer interface {
	JSONSchema() Schema
}

type schemerExtended interface {
	JSONSchemaExtend(s *Schema)
}

var (
	noReferType = reflect.TypeFor[noRefer]()

	errInvalidMapKey = errors.New("maps only support string keys")
)

var ErrUnknowType = errors.New("unable to create jsonschema for type")

func createBoolSchema() Schema {
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.BooleanType)
	return schema
}

func createStringSchema() Schema {
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.StringType)
	return schema
}

func createIntSchema(kind reflect.Kind) Schema {
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.IntegerType)

	switch kind {
	case reflect.Int32:
		schema.Format = openapi.Int32Format
	case reflect.Int64:
		schema.Format = openapi.Int64Format
	}

	return schema
}

func createUintSchema(kind reflect.Kind) Schema {
	var zeroInt = 0
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.IntegerType)
	schema.Minimum = &zeroInt

	switch kind {
	case reflect.Uint32:
		schema.Format = openapi.Int32Format
	case reflect.Uint64:
		schema.Format = openapi.Int64Format
	}

	return schema
}

func createFloatSchema() Schema {
	schema := New()
	schema.Type = openapi.NewSingleOrArray(openapi.NumberType)
	schema.Format = openapi.FloatFormat

	return schema
}

func loadSchemaOptions(field reflect.StructField, schema openapi.Schema) openapi.Schema {
	if v := field.Tag.Get("default"); v != "" {
		schema.Default = v
	}

	if v := field.Tag.Get("doc"); v != "" {
		schema.Description = v
	}

	return schema
}

func JSONFieldName(f reflect.StructField) string {
	jsonTag := f.Tag.Get("json")
	if jsonTag == "-" {
		return ""
	}

	name := strings.Split(jsonTag, ",")[0]
	if name == "" {
		name = f.Name
	}
	return name
}
