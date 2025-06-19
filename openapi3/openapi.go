package openapi3

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/sv-tools/openapi"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/openapi3/param"
)

// RegisterType set the types schema in the spec.
// If the schema allows references, it will be added to the specs components.
func RegisterType[T any](spec *OpenAPI, schema jsonschema.Schema, opts ...jsonschema.Option) error {
	typ := reflect.TypeFor[T]()
	spec.Schemer.Set(typ, schema, opts...)
	_, err := spec.GetSchemaOrRef(typ, SchemaRefOptions{})
	return err
}

func SetDefaultResponse[T any](spec *OpenAPI, code int, contentType ...string) {
	if len(contentType) == 0 {
		contentType = []string{spec.DefaultContentType}
	}

	typ := reflect.TypeFor[T]()
	v, err := spec.GetSchemaOrRef(typ, SchemaRefOptions{
		IgnoreAddSchemaErrors: true,
	})

	if err != nil {
		panic(err)
	}

	mt := NewMediaType()
	mt.Schema = v

	resp := Response{}
	for _, ct := range contentType {
		resp.SetContent(ct, mt)
	}

	spec.SetDefaultResponse(code, resp)
}

type Info = openapi.Info

// func NewOpenAPI(info openapi.Info) OpenAPI {
// 	s := &openapi.OpenAPI{
// 		Info:       openapi.NewExtendable(&info),
// 		Components: openapi.NewComponents(),
// 		// ExternalDocs: openapi.NewExternalDocs(),
// 		Paths:    openapi.NewPaths(),
// 		WebHooks: map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.PathItem]]{},
// 		OpenAPI:  "3.1.0",
// 		// JsonSchemaDialect: "https://json-schema.org/draft/2020-12/schema",
// 	}
// 	s.Components.Spec.Schemas = map[string]*openapi.RefOrSpec[openapi.Schema]{}
// 	s.Components.Spec.Paths = map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.PathItem]]{}
// 	s.Components.Spec.RequestBodies = map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.RequestBody]]{}
// 	s.Components.Spec.Parameters = map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.Parameter]]{}
// 	return OpenAPI{
// 		OpenAPI: s,
// 	}
// }

func New() *OpenAPI {
	openAPI := &openapi.OpenAPI{}
	openAPI.Info = NewExtendable(&Info{})
	openAPI.OpenAPI = "3.1.1"

	schemer := jsonschema.NewSchemer()
	schemer.RefPath = "#/components/schemas/"

	return &OpenAPI{
		OpenAPI:            openAPI,
		Schemer:            schemer,
		DefaultContentType: JSONContentType,
	}
}

type Tag struct {
	*openapi.Extendable[openapi.Tag]
}

func NewTag() Tag {
	tag := openapi.NewTagBuilder().Build()
	return Tag{tag}
}

const JSONContentType = "application/json"

type OpenAPI struct {
	*openapi.OpenAPI

	Schemer            jsonschema.Schemer `json:"-"`
	DefaultContentType string             `json:"-"`
	Strict             bool               `json:"-"`
}

func (o OpenAPI) GetComponents() Components {
	if o.Components == nil {
		o.Components = openapi.NewComponents()
		o.Components.Spec.Schemas = map[string]*openapi.RefOrSpec[openapi.Schema]{}
	}
	return Components{
		Components: o.Components.Spec,
	}
}

func defaultRespName(code int) string {
	if code != 0 {
		return strconv.Itoa(code)
	}
	return "default"
}

func (o OpenAPI) SetDefaultResponse(code int, resp Response) {
	name := defaultRespName(code)
	o.GetComponents().AddResponse(name, resp)
}

func (o OpenAPI) GetDefaultResponse(code int) (Response, bool) {
	name := defaultRespName(code)
	return o.GetComponents().GetResponse(name)
}

type Schema struct {
	*openapi.Schema
}

func (s Schema) JSONSchema() jsonschema.Schema {
	schema := jsonschema.New()
	if s.Schema != nil {
		schema.Schema = *s.Schema
	}
	return schema
}

type PathItem struct {
	*openapi.PathItem
}

func NewPathItem() PathItem {
	return PathItem{
		PathItem: &openapi.PathItem{},
	}
}

type PathOperation struct {
	Operation Operation
	Method    string
}

func (p PathItem) GetOperations() []PathOperation {
	methods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodPost,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodTrace,
		http.MethodOptions,
	}

	var ops []PathOperation
	for _, method := range methods {
		if op, has := p.GetOperation(method); has {
			ops = append(ops, PathOperation{
				Operation: op,
				Method:    method,
			})
		}
	}
	return ops
}

func (p PathItem) GetOperation(method string) (Operation, bool) {
	var o *openapi.Extendable[openapi.Operation]

	switch method {
	case http.MethodGet:
		o = p.Get
	case http.MethodPut:
		o = p.Put
	case http.MethodPost:
		o = p.Post
	case http.MethodPatch:
		o = p.Patch
	case http.MethodDelete:
		o = p.Delete
	case http.MethodHead:
		o = p.Head
	case http.MethodTrace:
		o = p.Trace
	case http.MethodOptions:
		o = p.Options
	}

	var op Operation
	if o != nil {
		op = Operation{Operation: o.Spec}
	}

	return op, op.Operation != nil
}

func (p PathItem) SetOperation(method string, operation Operation) {
	op := NewExtendable(operation.Operation)

	switch method {
	case http.MethodGet:
		p.Get = op
	case http.MethodPut:
		p.Put = op
	case http.MethodPatch:
		p.Patch = op
	case http.MethodPost:
		p.Post = op
	case http.MethodDelete:
		p.Delete = op
	case http.MethodTrace:
		p.Trace = op
	case http.MethodOptions:
		p.Options = op
	case http.MethodHead:
		p.Head = op
	}
}

func schemaShouldBeRef(schema jsonschema.Schema) bool {
	hasName := schema.Name() != ""
	return hasName && !schema.NoRef()
}

// SchemaRefOptions configures how schema references are handled.
type SchemaRefOptions struct {
	// ForceNoRef prevents creating a reference even if the schema would normally be referenced
	ForceNoRef bool
	// IgnoreAddSchemaErrors continues processing even if AddSchema fails
	IgnoreAddSchemaErrors bool
}

// getRefSchemas iterates through all of the properties on a schema, and recursively
// finds all schemas that are references.
func getRefSchemas(schema jsonschema.Schema, schemer jsonschema.Schemer) []jsonschema.Schema {
	found := []jsonschema.Schema{}

	for _, prop := range schema.Properties {
		if ref := prop.Ref; ref != nil {
			if schema, ok := schemer.GetSchemaByRef(ref.Ref); ok {
				found = append(found, schema)
			}
		}

		if spec := prop.Spec; spec != nil {
			schema := jsonschema.Schema{Schema: *spec}
			found = append(found, getRefSchemas(schema, schemer)...)
		}
	}

	return found
}

// GetSchemaOrRef creates and adds returns a schema or a ref to the created schema.
// If a ref is returned then the schema will be added to the specs components.
func (o OpenAPI) GetSchemaOrRef(
	obj any,
	opts SchemaRefOptions,
) (*openapi.RefOrSpec[openapi.Schema], error) {
	schema, err := o.Schemer.Get(obj)
	if err != nil {
		return nil, fmt.Errorf("error getting schema: %w", err)
	}

	var typ reflect.Type
	if t, ok := obj.(reflect.Type); ok {
		typ = t
	} else {
		typ = reflect.TypeOf(obj)
	}

	schema.Extensions = map[string]any{
		// This _should_ not show up in the schema, as every
		// schema will have a type specified.
		"type": reflect.New(typ).Elem().Interface(),
	}

	c := o.GetComponents()
	for _, schema := range getRefSchemas(schema, o.Schemer) {
		if err := c.AddSchema(schema.Name(), schema); !opts.IgnoreAddSchemaErrors && err != nil {
			return nil, fmt.Errorf("components: %w", err)
		}
	}

	var value any = schema.Schema
	if schemaShouldBeRef(schema) && !opts.ForceNoRef {
		name := schema.Name()
		ref := o.Schemer.NewRef(name)

		if err := c.AddSchema(name, schema); !opts.IgnoreAddSchemaErrors && err != nil {
			return nil, fmt.Errorf("components: %w", err)
		}

		value = ref
	}

	return openapi.NewRefOrSpec[openapi.Schema](value), nil
}

func (o OpenAPI) GetPath(name string) (PathItem, bool) {
	if o.Paths == nil {
		return PathItem{}, false
	}

	p, has := o.Paths.Spec.Paths[name]
	if has {
		return PathItem{p.Spec.Spec}, true
	}

	return PathItem{}, false
}

// SetPath overrides any existing paths if they exist, if not
// it creates the pathItem.
func (o OpenAPI) SetPath(name string, pathItem PathItem) {
	if o.Paths == nil {
		o.Paths = openapi.NewPaths()
		o.Paths.Spec.Paths = map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.PathItem]]{}
	}

	if path, has := o.Paths.Spec.Paths[name]; has {
		path.Spec.Spec = pathItem.PathItem
	}

	item := NewExtendable(pathItem.PathItem)
	o.Paths.Spec.Paths[name] = openapi.NewRefOrSpec[openapi.Extendable[openapi.PathItem]](item)
}

func (o OpenAPI) getSchemaSource(src *openapi.RefOrSpec[openapi.Schema]) (Schema, error) {
	if src == nil {
		return Schema{}, nil
	}

	schema, err := src.GetSpec(o.Components)
	if err != nil {
		return Schema{}, err
	}

	return Schema{
		Schema: schema,
	}, nil
}

type MediaType struct {
	openapi.MediaType
}

func NewMediaType() MediaType {
	return MediaType{MediaType: openapi.MediaType{}}
}

func (m *MediaType) SetSchema(schema jsonschema.Schema) {
	m.Schema = openapi.NewRefOrSpec[openapi.Schema](schema.Schema)
}

func (m *MediaType) SetSchemaRef(ref string) {
	m.Schema = openapi.NewRefOrSpec[openapi.Schema](ref)
}

type RequestBody struct {
	openapi.RequestBody
}

func (r *RequestBody) SetContent(typ string, mediaType MediaType) {
	if r.Content == nil {
		r.Content = map[string]*openapi.Extendable[openapi.MediaType]{}
	}
	r.Content[typ] = openapi.NewExtendable(&mediaType.MediaType)
}

type Response struct {
	openapi.Response
}

func (r *Response) SetContent(typ string, mediaType MediaType) {
	if r.Content == nil {
		r.Content = map[string]*openapi.Extendable[openapi.MediaType]{}
	}
	r.Content[typ] = openapi.NewExtendable(&mediaType.MediaType)
}

type Parameter = param.Parameter

func NewParameter() param.Parameter {
	return param.New()
}

func NewExtendable[T any](t *T) *openapi.Extendable[T] {
	return openapi.NewExtendable(t)
}

type Components struct {
	*openapi.Components
}

type ComponentSchema struct {
	jsonschema.Schema
	Name string
}

func (c Components) GetSchemaByName(name string) (Schema, bool) {
	if schema, has := c.Schemas[name]; has {
		return Schema{Schema: schema.Spec}, true
	}
	return Schema{}, false
}

var ErrAlreadyExists = errors.New("already exists in the schema")

func (c Components) AddSchema(name string, schema jsonschema.Schema) error {
	if existingSchema, has := c.Schemas[name]; has {
		if reflect.DeepEqual(existingSchema.Spec, schema.Schema) {
			return nil
		}
		return fmt.Errorf("%s: %w", name, ErrAlreadyExists)
	}
	c.Schemas[name] = openapi.NewRefOrSpec[openapi.Schema](schema.Schema)
	return nil
}

func (c Components) AddResponse(name string, resp Response) {
	if c.Responses == nil {
		c.Responses = map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.Response]]{}
	}

	item := NewExtendable(&resp.Response)
	c.Responses[name] = openapi.NewRefOrSpec[openapi.Extendable[openapi.Response]](item)
}

func (c Components) GetResponse(name string) (Response, bool) {
	if c.Responses == nil {
		return Response{}, false
	}

	if resp, has := c.Responses[name]; has {
		return Response{*resp.Spec.Spec}, has
	}

	return Response{}, false
}
