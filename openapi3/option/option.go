package option

import (
	"fmt"

	"github.com/zhamlin/routey/internal/stringz"
	"github.com/zhamlin/routey/openapi3"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

type (
	Context struct {
		openapi3.Context

		Info *route.Info
		// set the content type on any option that has a mediaType
		contentType []string
		// inline any types seen vs trying to store them in schema and create ref
		noRef bool
	}
	Option func(*Context, *openapi3.Operation) error
)

func (ctx Context) getContentType(existingContentType []string) []string {
	if l := len(existingContentType); l == 0 && len(ctx.contentType) > 0 {
		return ctx.contentType
	} else if l == 0 && ctx.OpenAPI.DefaultContentType != "" {
		return []string{ctx.OpenAPI.DefaultContentType}
	}
	return existingContentType
}

func (ctx Context) newMediaType(obj any) (openapi3.MediaType, error) {
	v, err := ctx.OpenAPI.GetSchemaOrRef(obj, openapi3.SchemaRefOptions{
		ForceNoRef:            ctx.noRef,
		IgnoreAddSchemaErrors: true,
	})
	if err != nil {
		return openapi3.MediaType{}, fmt.Errorf("failed getting schema: %w", err)
	}

	mediaType := openapi3.NewMediaType()
	mediaType.Schema = v
	return mediaType, nil
}

// Params takes a struct and grabs all parameters defined on it,
// adding them to the operation.
func Params[T any]() route.Option {
	return New(func(ctx *Context, _ *openapi3.Operation) error {
		params, err := param.InfoFromStruct[T](
			ctx.Namer, ctx.Parser,
		)
		if err != nil {
			return err
		}

		ctx.Info.Params = params
		return nil
	})
}

func Body[T any](desc string, required bool, contentType ...string) route.Option {
	return New(func(ctx *Context, o *openapi3.Operation) error {
		var obj T
		body := openapi3.RequestBody{}
		body.Description = stringz.TrimLinesSpace(desc)
		mediaType, err := ctx.newMediaType(obj)

		if err != nil {
			return err
		}

		body.Required = required
		for _, typ := range ctx.getContentType(contentType) {
			body.SetContent(typ, mediaType)
		}

		o.SetRequestBody(body)
		return nil
	})
}

// ContentType sets the content type for the provided responses.
func ContentType(contentTypes []string, options ...route.Option) route.Option {
	return func(i *route.Info) error {
		ctx, err := ctxFromInfo(i)
		if err != nil {
			return err
		}

		oldContentType := ctx.contentType
		ctx.contentType = contentTypes

		for _, option := range options {
			err = option(i)
			if err != nil {
				return err
			}
		}

		ctx.contentType = oldContentType
		return nil
	}
}

// NoRef causes any supplied types to be inlined in the schema instead of
// creating them in the components schema and using a reference to that.
func NoRef(options ...route.Option) route.Option {
	return func(i *route.Info) error {
		ctx, err := ctxFromInfo(i)
		if err != nil {
			return err
		}

		oldNoRef := ctx.noRef
		ctx.noRef = true

		for _, option := range options {
			if err := option(i); err != nil {
				return err
			}
		}

		ctx.noRef = oldNoRef
		return nil
	}
}

// None represents no value.
type None struct{}

// Response takes a http status code and sets the response body for it to T.
func Response[T any](code int, desc string, contentType ...string) route.Option {
	return New(func(ctx *Context, o *openapi3.Operation) error {
		resp := openapi3.Response{}
		resp.Description = stringz.TrimLinesSpace(desc)

		var obj T
		switch any(&obj).(type) {
		case *any:
		case *None:
		default:
			mediaType, err := ctx.newMediaType(obj)
			if err != nil {
				return err
			}

			for _, typ := range ctx.getContentType(contentType) {
				resp.SetContent(typ, mediaType)
			}
		}

		o.AddResponse(code, resp)
		return nil
	})
}

// ID sets the operations id.
func ID(id string) route.Option {
	return New(func(_ *Context, o *openapi3.Operation) error {
		o.OperationID = id
		return nil
	})
}

// Ignore excludes the operation from the openapi spec.
func Ignore() route.Option {
	return New(func(_ *Context, o *openapi3.Operation) error {
		o.Ignore = true
		return nil
	})
}

// Deprecated marks the operation as deprecated.
func Deprecated() route.Option {
	return New(func(_ *Context, o *openapi3.Operation) error {
		o.Deprecated = true
		return nil
	})
}

// Summary sets the summary on the operation.
func Summary(summary string) route.Option {
	return New(func(_ *Context, o *openapi3.Operation) error {
		o.Summary = stringz.TrimLinesSpace(summary)
		return nil
	})
}

func ctxFromInfo(i *route.Info) (*Context, error) {
	const contextKey = "openapi3.option.context"
	if ctx, ok := i.Context[contextKey].(*Context); ok {
		return ctx, nil
	}

	openapiCtx, err := openapi3.ContextFromCtx(i.Context)
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		Info:        i,
		Context:     openapiCtx,
		contentType: []string{},
		noRef:       false,
	}
	i.Context[contextKey] = ctx
	return ctx, nil
}

func New(opt Option) route.Option {
	return func(i *route.Info) error {
		ctx, err := ctxFromInfo(i)
		if err != nil {
			return err
		}

		op := openapi3.OperationFromCtx(i.Context)
		if err := opt(ctx, op); err != nil {
			return err
		}

		return nil
	}
}
