package openapi3

import (
	"errors"
	"strconv"

	"github.com/sv-tools/openapi"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/openapi3/param"
)

type Operation struct {
	*openapi.Operation

	Ignore bool `json:"-"`
}

func NewOperation() Operation {
	return Operation{
		Operation: &openapi.Operation{},
	}
}

func (o *Operation) SetDefaultResponse(resp Response) {
	if o.Responses == nil {
		o.Responses = openapi.NewExtendable(&openapi.Responses{})
	}

	ext := openapi.NewExtendable(&resp.Response)
	response := openapi.NewRefOrSpec[openapi.Extendable[openapi.Response]](ext)
	o.Responses.Spec.Default = response
}

func (o *Operation) SetRequestBody(body RequestBody) {
	item := openapi.NewExtendable(&body.RequestBody)
	o.RequestBody = openapi.NewRefOrSpec[openapi.Extendable[openapi.RequestBody]](item)
}

func (o *Operation) AddResponse(code int, schema Response) {
	if o.Responses == nil {
		o.Responses = openapi.NewResponsesBuilder().Build().Spec
		o.Responses.Spec.Response = map[string]*openapi.RefOrSpec[openapi.Extendable[openapi.Response]]{}
	}
	statusCode := strconv.Itoa(code)

	item := openapi.NewExtendable(&schema.Response)
	o.Responses.Spec.Response[statusCode] = openapi.NewRefOrSpec[openapi.Extendable[openapi.Response]](
		item,
	)
}

func (o *Operation) GetParameter(name, in string) (param.Parameter, bool) {
	if o.Parameters == nil {
		return param.Parameter{}, false
	}

	for _, p := range o.Parameters {
		if p.Ref != nil {
			panic("TODO: handle param ref in operations")
		}

		hasLocation := in != ""
		sourceMatch := hasLocation && in == p.Spec.Spec.In
		nameMatch := p.Spec.Spec.Name == name

		if nameMatch && (sourceMatch || !hasLocation) {
			return param.Parameter{Parameter: p.Spec.Spec}, true
		}
	}
	return param.Parameter{}, false
}

func (o *Operation) HasParameter(param param.Parameter) bool {
	_, has := o.GetParameter(param.Name, param.In)
	return has
}

func (o *Operation) AddParameter(param param.Parameter) {
	if o.Parameters == nil {
		o.Parameters = []*openapi.RefOrSpec[openapi.Extendable[openapi.Parameter]]{}
	}

	item := openapi.NewExtendable(param.Parameter)
	p := openapi.NewRefOrSpec[openapi.Extendable[openapi.Parameter]](item)
	o.Parameters = append(o.Parameters, p)
}

// SchemaFromOp takes an operation and returns a json schema that can be used
// to validate a request.
func SchemaFromOp(op Operation, contentType string) (jsonschema.Schema, error) {
	schema := jsonschema.NewBuilder().
		Type("object").
		Description("Contains the request body and all parameters").
		ObjectBuilder

	addParamsToSchema(schema, op.Parameters)

	if err := addBodyToSchema(schema, op.RequestBody, contentType); err != nil {
		return jsonschema.New(), err
	}

	return schema.Build(), nil
}

var ErrRequestBodyMissingSchema = errors.New("no schema or ref on request body")

func addBodyToSchema(
	schema jsonschema.ObjectBuilder,
	requestBody *openapi.RefOrSpec[openapi.Extendable[openapi.RequestBody]],
	contentType string,
) error {
	if requestBody == nil {
		return nil
	}

	var bodyJSONSchema jsonschema.Schema
	bodySchema := requestBody.Spec.Spec.Content[contentType].Spec.Schema

	switch {
	case bodySchema.Ref != nil:
		ref := bodySchema.Ref.Ref
		bodyJSONSchema = jsonschema.NewBuilder().Reference(ref)
	case bodySchema.Spec != nil:
		bodyJSONSchema = jsonschema.Schema{Schema: *bodySchema.Spec}
	default:
		return ErrRequestBodyMissingSchema
	}

	schema = schema.Property("body", bodyJSONSchema)

	if requestBody.Spec.Spec.Required {
		schema.
			Required("body")
	}

	return nil
}

func addParamsToSchema(
	schema jsonschema.ObjectBuilder,
	parameters []*openapi.RefOrSpec[openapi.Extendable[openapi.Parameter]],
) {
	if len(parameters) == 0 {
		return
	}

	paramsSchemas := jsonschema.NewBuilder().
		Type("object").
		Description("Contains the parameters").
		ObjectBuilder

	pByIn := map[string][]opParamInfo{}

	getOperationParamInfo(parameters, pByIn)

	// group params by location
	for loc, params := range pByIn {
		locSchema := jsonschema.NewBuilder().
			Type("object").
			ObjectBuilder

		for _, param := range params {
			locSchema = locSchema.Property(param.Name, param.Schema)

			if param.Required {
				locSchema = locSchema.Required(param.Name)
				paramsSchemas = paramsSchemas.Required(loc)
			}
		}

		paramsSchemas = paramsSchemas.Property(loc, locSchema.Build())
	}

	schema.
		Required("parameters").
		Property("parameters", paramsSchemas.Build())
}

type opParamInfo struct {
	Name     string
	Required bool
	Schema   jsonschema.Schema
}

func getOperationParamInfo(
	parameters []*openapi.RefOrSpec[openapi.Extendable[openapi.Parameter]],
	pByIn map[string][]opParamInfo,
) {
	for _, p := range parameters {
		s := p.Spec.Spec
		var schema jsonschema.Schema

		if r := s.Schema.Ref; r != nil {
			schema = jsonschema.NewBuilder().Reference(r.Ref)
		} else if s := s.Schema.Spec; s != nil {
			schema = jsonschema.Schema{Schema: *s}
		}

		locParams, has := pByIn[s.In]
		if !has {
			locParams = []opParamInfo{}
		}

		locParams = append(locParams, opParamInfo{
			Name:     s.Name,
			Required: s.Required,
			Schema:   schema,
		})
		pByIn[s.In] = locParams
	}
}
