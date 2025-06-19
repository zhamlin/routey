package openapi3

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/jsonschema"
	openAPIParam "github.com/zhamlin/routey/openapi3/param"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

func opFromCtx(ctx Context, info *route.Info) (Operation, error) {
	path, has := ctx.OpenAPI.GetPath(info.FullPattern)
	if !has {
		return Operation{}, fmt.Errorf(
			"no path found: %s: %w",
			info.FullPattern,
			extractor.ErrParamFailedToExtract,
		)
	}

	op, has := path.GetOperation(info.Method)
	if !has {
		return Operation{}, fmt.Errorf(
			"no op found: %s %s: %w",
			info.Method,
			info.FullPattern,
			extractor.ErrParamFailedToExtract,
		)
	}

	return op, nil
}

func parseable(parser param.Parser, value any) bool {
	return !errors.Is(parser(value, []string{""}), param.ErrInvalidParamType)
}

func getDefaultValue(f reflect.StructField, schema jsonschema.Schema) string {
	if def := f.Tag.Get("default"); def != "" {
		return def
	}

	fieldName := jsonschema.JSONFieldName(f)
	if prop, ok := schema.Properties[fieldName]; ok {
		def := prop.Spec.Default
		if s, ok := def.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", def)
	}

	return ""
}

func validDeepObjectType(parser param.Parser, typ reflect.Type) error {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return param.ErrInvalidParamType
	}

	schema, err := jsonschema.NewSchemer().Get(typ)
	if err != nil {
		return fmt.Errorf("error getting schema for deepObject(%s): %w", typ.String(), err)
	}

	n := typ.NumField()
	for i := range n {
		f := typ.Field(i)
		if !f.IsExported() {
			return &param.InvalidParamError{
				Struct:       typ,
				Field:        f,
				Err:          "field is not exported",
				UnderlineAll: true,
			}
		}

		value := reflect.New(f.Type).Interface()

		if !parseable(parser, value) {
			return &param.InvalidParamError{
				Struct: typ,
				Field:  f,
			}
		}

		defaultValue := getDefaultValue(f, schema)
		if defaultValue != "" {
			if err := parser(value, []string{defaultValue}); err != nil {
				return &param.InvalidParamError{
					Struct:  typ,
					Field:   f,
					Message: param.ErrUnparsableDefault + ": " + defaultValue,
					Err:     err.Error(),
				}
			}
		}
	}

	return nil
}

var (
	_ extractor.ParamExtractor = &Query[string]{}
	_ extractor.Extractor      = &JSON[string]{}
)

type JSON[T any] struct {
	routey.JSON[T]
}

func (q *JSON[T]) Extract(r *http.Request, info *route.Info) error {
	ctx, err := ContextFromCtx(info.Context)
	if err != nil {
		return fmt.Errorf("no context: %w", err)
	}

	op, err := opFromCtx(ctx, info)
	if err != nil {
		return err
	}

	if err := q.JSON.Extract(r, info); err != nil {
		return err
	}

	return validateJSONBodySchema(ctx, op, &q.V)
}

func validateJSONBodySchema(ctx Context, op Operation, value any) error {
	if ctx.Validator == nil {
		return nil
	}

	loc := "#/body"
	name := op.OperationID + ".body"

	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = ctx.Validator.Validate(name, b)
	var want jsonschema.ValidationError

	if errors.As(err, &want) {
		want.Location = loc
		return want
	}

	return err
}

type Query[T any] struct {
	routey.Query[T]
}

func (q *Query[T]) CanParse(
	parser param.Parser,
	source reflect.StructField,
	value any,
) error {
	if parseable(parser, value) {
		return nil
	}

	style, err := openAPIParam.GetStyleFromTag(source.Tag)
	if err != nil {
		// Return nil here to allow for better error reporting
		// when openAPIParam.FromInfo is called.
		//
		//nolint:nilerr
		return nil
	}

	if style != openAPIParam.StyleDeepObject {
		return param.ErrInvalidParamType
	}

	typ := reflect.TypeFor[T]()
	if err := validDeepObjectType(parser, typ); err != nil {
		return err
	}

	return nil
}

func (q *Query[T]) Extract(r *http.Request, info *route.Info, opts param.Opts) error {
	ctx, err := ContextFromCtx(info.Context)
	if err != nil {
		return fmt.Errorf("no context: %w", err)
	}

	op, err := opFromCtx(ctx, info)
	if err != nil {
		return err
	}

	param, has := op.GetParameter(opts.Name, q.Source())
	if !has {
		return fmt.Errorf(
			"no param found: %s %s: %w",
			info.Method,
			info.FullPattern,
			extractor.ErrParamFailedToExtract,
		)
	}

	values := extractor.GetAndSetQueryValues(r)
	return q.parse(values, opts, param, ctx)
}

func (q *Query[T]) parse(
	values url.Values,
	opts param.Opts,
	p openAPIParam.Parameter,
	ctx Context,
) error {
	var err error

	// TODO: handle required
	switch openAPIParam.Style(p.Style) {
	case openAPIParam.StyleForm:
		err = q.parseForm(values, opts, p)
	case openAPIParam.StyleDeepObject:
		err = q.parseDeepObject(values, opts, p, ctx.OpenAPI)
	default:
		// case openAPIParam.StyleSpaceDelimited:
		// case openAPIParam.StylePipeDelimited:
		return nil
	}

	if err != nil || ctx.Validator == nil {
		return err
	}

	return validateSchema(p.Name, ctx.Validator, &q.Value)
}

func (q *Query[T]) parseForm(values url.Values, opts param.Opts, p openAPIParam.Parameter) error {
	params := values[opts.Name]

	// params are separated by ,
	if !p.Explode && len(params) > 0 {
		params = strings.Split(params[0], ",")
	}

	err := opts.Parse(&q.Value, params)
	if err != nil {
		return fmt.Errorf("%w: %w", extractor.ErrParamFailedToExtract, err)
	}

	return nil
}

func (q *Query[T]) parseDeepObject(
	values url.Values,
	opts param.Opts,
	p openAPIParam.Parameter,
	spec *OpenAPI,
) error {
	val := reflect.ValueOf(&q.Value)
	typ := val.Elem().Type()
	s, err := spec.getSchemaSource(p.Schema)

	if err != nil {
		return err
	}
	schema := s.JSONSchema()

	n := typ.NumField()
	for i := range n {
		fType := typ.Field(i)
		f := val.Elem().Field(i)

		fieldName := jsonschema.JSONFieldName(fType)
		name := fmt.Sprintf("%s[%s]", p.Name, fieldName)
		params := values[name]

		opts.Default = getDefaultValue(fType, schema)
		if err := opts.Parse(f.Addr().Interface(), params); err != nil {
			return fmt.Errorf("opts.Parse(%s, %v): %w", name, params, err)
		}
	}

	return nil
}

func validateSchema(name string, validator *jsonschema.Validator, value any) error {
	loc := "#/parameters/query/" + name
	name = "param." + name
	b, err := json.Marshal(value)

	if err != nil {
		return err
	}

	err = validator.Validate(name, b)
	var want jsonschema.ValidationError

	if errors.As(err, &want) {
		want.Location = loc
		return want
	}

	return err
}
