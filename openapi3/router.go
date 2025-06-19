package openapi3

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/sv-tools/openapi"
	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal"
	"github.com/zhamlin/routey/internal/stringz"
	"github.com/zhamlin/routey/jsonschema"
	openAPIParam "github.com/zhamlin/routey/openapi3/param"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

func OperationFromCtx(ctx route.Context) *Operation {
	type operationContextKey struct{}

	if op, ok := ctx[operationContextKey{}].(*Operation); ok {
		return op
	}

	op := NewOperation()
	ctx[operationContextKey{}] = &op
	return &op
}

type Context struct {
	OpenAPI   *OpenAPI
	Validator *jsonschema.Validator
	Namer     param.Namer
	Parser    param.Parser
}

type contextKey struct{}

var ErrNoContext = errors.New("openapi3.Context not found in OptionsContext")

func ContextFromCtx(ctx route.Context) (Context, error) {
	if ctx, ok := ctx[contextKey{}].(Context); ok {
		return ctx, nil
	}
	return Context{}, ErrNoContext
}

func updateRequestBodyFromTags(field reflect.StructField, r RequestBody) (RequestBody, error) {
	if v := field.Tag.Get("description"); v != "" {
		r.Description = stringz.TrimLinesSpace(v)
	}

	if v := field.Tag.Get("required"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return r, err
		}
		r.Required = b
	}

	return r, nil
}

func compileBodySchema(ctx Context, op *Operation, s *openapi.RefOrSpec[openapi.Schema]) error {
	if ctx.Validator == nil {
		return nil
	}

	// TODO: include content-type
	name := op.OperationID + ".body"
	schema, err := ctx.OpenAPI.getSchemaSource(s)

	if err != nil {
		return err
	}

	b, err := json.Marshal(schema.JSONSchema())
	if err != nil {
		return err
	}

	if err := ctx.Validator.Add(name, string(b)); err != nil {
		return fmt.Errorf("compling schema(%s) failed: %w", name, err)
	}

	return nil
}

func addBodyToOp(ctx Context, info param.Info, o *Operation) error {
	s, err := ctx.OpenAPI.GetSchemaOrRef(info.Type, SchemaRefOptions{
		IgnoreAddSchemaErrors: true,
	})
	if err != nil {
		return err
	}

	mt := NewMediaType()
	mt.Schema = s

	body := RequestBody{}
	body.SetContent(JSONContentType, mt)
	body, err = updateRequestBodyFromTags(info.Field, body)

	if err != nil {
		return err
	}

	o.SetRequestBody(body)
	return compileBodySchema(ctx, o, s)
}

func compileParamSchema(ctx Context, p Parameter) error {
	if ctx.Validator == nil {
		return nil
	}

	name := "param." + p.Name
	schema, err := ctx.OpenAPI.getSchemaSource(p.Schema)

	if err != nil {
		return err
	}

	b, err := json.Marshal(schema.JSONSchema())
	if err != nil {
		return err
	}

	if err := ctx.Validator.Add(name, string(b)); err != nil {
		return fmt.Errorf("compling schema(%s) failed: %w", name, err)
	}

	return nil
}

func addParamToOp(ctx Context, i param.Info, o *Operation) error {
	spec := ctx.OpenAPI
	p, err := openAPIParam.FromInfo(i, spec.Schemer)

	if err != nil {
		return fmt.Errorf("openapi.FromInfo: %w", err)
	}

	if i.Default != "" {
		v := reflect.New(i.Type)
		if err := ctx.Parser(v.Interface(), []string{i.Default}); err != nil {
			return fmt.Errorf("failed parsing default: %w", err)
		}
		p.Schema.Spec.Default = v.Elem().Interface()
	}

	if !o.HasParameter(p) {
		isDeepObject := p.Style == string(openAPIParam.StyleDeepObject)
		if isDeepObject {
			p.Schema, err = spec.GetSchemaOrRef(
				i.Type,
				SchemaRefOptions{IgnoreAddSchemaErrors: true},
			)
			if err != nil {
				return err
			}
		}

		o.AddParameter(p)

		if err := compileParamSchema(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func getPublicFunctionName(fn any) string {
	if fn == nil {
		return ""
	}

	getFunctionName := func(temp any) string {
		strs := strings.Split((runtime.FuncForPC(reflect.ValueOf(temp).Pointer()).Name()), ".")
		return strs[len(strs)-1]
	}

	funcName := getFunctionName(fn)
	if unicode.IsLower(rune(funcName[0])) {
		funcName = ""
	}
	return funcName
}

func addParam(p param.Info, ctx Context, o *Operation, info *route.Info) error {
	var err error
	if p.Source == "body" {
		err = addBodyToOp(ctx, p, o)
	} else {
		err = addParamToOp(ctx, p, o)
	}

	var hErr routey.HandlerError
	if errors.As(err, &hErr) {
		hErr.Pattern = info.Method + " " + info.FullPattern
		hErr.Handler = internal.GetFnInfo(info.Handler)
		err = hErr
	}

	return err
}

var (
	ErrNoOperationID        = errors.New("operation id required")
	ErrDuplicateOperationID = errors.New("operation id already exists")
)

func ensureNoDupOpID(spec *OpenAPI, operation *Operation) error {
	if spec.Paths == nil {
		return nil
	}

	haveSameID := func(op Operation) bool {
		return op.OperationID == operation.OperationID
	}

	for pattern, path := range spec.Paths.Spec.Paths {
		path := PathItem{PathItem: path.Spec.Spec}

		for _, op := range path.GetOperations() {
			if haveSameID(op.Operation) {
				return fmt.Errorf(
					"%w: path=%q",
					ErrDuplicateOperationID, op.Method+" "+pattern,
				)
			}
		}
	}

	return nil
}

func ensureOperationID(spec *OpenAPI, operation *Operation, info *route.Info) error {
	if operation.OperationID == "" {
		// TODO: make configurable
		operation.OperationID = getPublicFunctionName(info.Handler)
	}

	if spec.Strict {
		if operation.OperationID == "" {
			return routey.HandlerError{
				Pattern: info.Method + " " + info.FullPattern,
				Handler: internal.GetFnInfo(info.Handler),
				Err:     fmt.Errorf("error: openapi: %w", ErrNoOperationID),
			}
		}

		if err := ensureNoDupOpID(spec, operation); err != nil {
			return routey.HandlerError{
				Pattern: info.Method + " " + info.FullPattern,
				Handler: internal.GetFnInfo(info.Handler),
				Err:     fmt.Errorf("error: openapi: %q %w", operation.OperationID, err),
			}
		}
	}

	return nil
}

func setDefaultResponseIfAvailable(spec *OpenAPI, operation *Operation) {
	// TODO: get all default responses
	if resp, has := spec.GetDefaultResponse(0); has {
		operation.SetDefaultResponse(resp)
	}
}

func newOnRouteAdd(spec *OpenAPI) func(*route.Info) error {
	return func(info *route.Info) error {
		path, has := spec.GetPath(info.FullPattern)
		if !has {
			path = NewPathItem()
		}

		operation := OperationFromCtx(info.Context)
		if operation.Ignore {
			return nil
		}

		for _, opt := range info.Options {
			if err := opt(info); err != nil {
				return err
			}
		}

		if err := ensureOperationID(spec, operation, info); err != nil {
			return err
		}

		c, err := ContextFromCtx(info.Context)
		if err != nil {
			return err
		}

		for _, p := range info.Params {
			if err := addParam(p, c, operation, info); err != nil {
				return err
			}
		}

		setDefaultResponseIfAvailable(spec, operation)
		path.SetOperation(info.Method, *operation)
		spec.SetPath(info.FullPattern, path)

		return nil
	}
}

type AddSpecToRouterOpts struct {
	DefaultContentType string
	ValidateRequests   bool
	// Strict determines whether or not an error is thrown
	// if required properties are not set on OpenAPI resources.
	Strict bool
}

func AddSpecToRouter(r *routey.Router, opts AddSpecToRouterOpts) *OpenAPI {
	spec := New()
	spec.Strict = opts.Strict

	if typ := opts.DefaultContentType; typ != "" {
		spec.DefaultContentType = typ
	}

	ctx := Context{
		OpenAPI: spec,
		Parser:  r.Params.Parser,
		Namer:   r.Params.Namer,
	}

	if opts.ValidateRequests {
		spec.Strict = true
		ctx.Validator = jsonschema.NewValidator()
	}

	r.Context = route.Context{
		contextKey{}: ctx,
	}
	r.OnRouteAdd = newOnRouteAdd(spec)

	return spec
}

func NewRouter() (*routey.Router, *OpenAPI) {
	r := routey.New()
	spec := AddSpecToRouter(r, AddSpecToRouterOpts{})

	return r, spec
}
