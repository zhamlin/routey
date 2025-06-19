package extractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"unsafe"

	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
)

var ErrParamFailedToExtract = errors.New("failed to extract param")

func GetAndSetQueryValues(r *http.Request) url.Values {
	type cachedQueryKey struct{}

	ctx := r.Context()
	values, ok := ctx.Value(cachedQueryKey{}).(url.Values)

	if !ok {
		values = r.URL.Query()
		ctx = context.WithValue(ctx, cachedQueryKey{}, values)
		*r = *r.WithContext(ctx)
	}

	return values
}

var (
	_ ParamExtractor = &Query[string]{}
	_ ParamExtractor = &Path[string]{}
	_ Extractor      = &JSON[string]{}
)

// Path allows T to be parsed from the url path.
type Path[T any] struct {
	Value T
}

func (p *Path[T]) Extract(r *http.Request, _ *route.Info, opts param.Opts) error {
	value := opts.PathValue(opts.Name, r)
	err := opts.Parse(&p.Value, []string{value})

	if err != nil {
		return fmt.Errorf("%w: %w", ErrParamFailedToExtract, err)
	}
	return nil
}

func (Path[T]) Source() string {
	return "path"
}

func (p Path[T]) Inner() any {
	return p.Value
}

// Query allows T to be parsed from the url query params.
type Query[T any] struct {
	Value T
}

func (q *Query[T]) Extract(r *http.Request, _ *route.Info, opts param.Opts) error {
	values := GetAndSetQueryValues(r)
	err := opts.Parse(&q.Value, values[opts.Name])

	if err != nil {
		return fmt.Errorf("%w: %w", ErrParamFailedToExtract, err)
	}
	return nil
}

func (Query[T]) Source() string {
	return "query"
}

func (q Query[T]) Inner() any {
	return q.Value
}

// JSON allows T to be json decoded from the http request body.
type JSON[T any] struct{ V T }

func (v *JSON[T]) Extract(r *http.Request, _ *route.Info) error {
	return decodeBodyJSON(r, &v.V)
}

func (JSON[T]) Source() string {
	return "body"
}

func (v JSON[T]) Inner() any {
	return v.V
}

func (v JSON[T]) CanParse(_ param.Parser, _ reflect.StructField, value any) error {
	return nil
}

var ErrJSONDecode = errors.New("error decoding http request body as json")

func decodeBodyJSON(r *http.Request, dest any) error {
	hasBody := r.Body != nil && r.ContentLength > 0
	if hasBody {
		if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
			return fmt.Errorf("type: %T: %w: %w", dest, ErrJSONDecode, err)
		}
	}

	return nil
}

var extractors = sync.Map{}

type fnExtractor[T any] struct {
	fn func(*http.Request) (T, error)
}

func (f fnExtractor[T]) ExtractType(r *http.Request) (any, error) {
	return f.fn(r)
}

func Register[T any](f func(*http.Request) (T, error)) {
	t := reflect.TypeFor[T]()
	extractors.Store(t, fnExtractor[T]{f})
}

// Extractor is the interface implemented by an object that can
// create itself from a http request.
type Extractor interface {
	Extract(*http.Request, *route.Info) error
}

type ParamExtractor interface {
	Extract(*http.Request, *route.Info, param.Opts) error
	Source() string
}

var (
	extType      = reflect.TypeFor[Extractor]()
	paramExtType = reflect.TypeFor[ParamExtractor]()
	httpReqType  = reflect.TypeFor[*http.Request]()
	httpRespType = reflect.TypeFor[http.ResponseWriter]()
)

type Response struct {
	// Response from the handler
	Response any
	// Error from the handler
	Error error
	Info  *route.Info
}

type ResponseHandler func(http.ResponseWriter, *http.Request, Response)

type HandlerParams struct {
	Response         ResponseHandler
	ErrorSink        func(error)
	Parser           param.Parser
	Namer            param.Namer
	ParamPather      param.Pather
	Pattern          string
	RouteInfo        *route.Info
	CollectAllErrors bool
}

func Handler[T, R any](handler func(T) (R, error), params HandlerParams) http.HandlerFunc {
	typ := reflect.TypeFor[T]()
	extractInputs, err := extractorFor(typ, extractorForOpts{
		Parser:           params.Parser,
		Namer:            params.Namer,
		Pather:           params.ParamPather,
		RouteInfo:        params.RouteInfo,
		CollectAllErrors: params.CollectAllErrors,
	})

	if err != nil {
		params.ErrorSink(err)
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var out R
		var args T

		err := extractInputs(w, r, unsafe.Pointer(&args))
		if err == nil {
			out, err = handler(args)
		}

		if f := params.Response; f != nil {
			f(w, r, Response{
				Response: out,
				Error:    err,
				Info:     params.RouteInfo,
			})
		}
	}
}

type extractorFn func(http.ResponseWriter, *http.Request, unsafe.Pointer) error

type extractorForOpts struct {
	Namer            param.Namer
	Parser           param.Parser
	Pather           param.Pather
	RouteInfo        *route.Info
	CollectAllErrors bool
}

func findRelatedExtractors(f reflect.StructField, opts extractorForOpts) []reflect.Type {
	var related []reflect.Type
	typ := f.Type
	f.Type = reflect.PointerTo(typ)

	if fn, _ := extractorFromField(f, opts); fn != nil {
		related = append(related, f.Type)
	}

	if typ.Kind() == reflect.Pointer {
		f.Type = typ.Elem()
		if fn, _ := extractorFromField(f, opts); fn != nil {
			related = append(related, f.Type)
		}
	}
	return related
}

func extractorFor(argType reflect.Type, opts extractorForOpts) (extractorFn, error) {
	if kind := argType.Kind(); kind != reflect.Struct {
		return nil, fmt.Errorf("type: %s: %w", kind.String(), param.ErrNonStructArg)
	}

	numFields := argType.NumField()
	fns := make([]extractorFn, numFields)

	for i := range fns {
		f := argType.Field(i)
		fn, err := extractorFromFieldWithRelated(f, opts)

		var want *UnknownFieldTypeError
		if errors.As(err, &want) {
			want.setStruct(argType)
		}

		if err != nil {
			return nil, err
		}
		fns[i] = fn
	}

	return func(w http.ResponseWriter, r *http.Request, argsPtr unsafe.Pointer) error {
		var allErrors []error
		for _, fn := range fns {
			if err := fn(w, r, argsPtr); err != nil {
				if !opts.CollectAllErrors {
					return err
				}

				allErrors = append(allErrors, err)
			}
		}
		return errors.Join(allErrors...)
	}, nil
}

func fieldValue(field reflect.StructField, ptr unsafe.Pointer) reflect.Value {
	fieldPtr := unsafe.Add(ptr, field.Offset)
	return reflect.NewAt(field.Type, fieldPtr)
}

func fieldImplements(f reflect.StructField, t reflect.Type) bool {
	return reflect.PointerTo(f.Type).Implements(t)
}

func extractExtractor(field reflect.StructField, opts extractorForOpts) extractorFn {
	if !fieldImplements(field, extType) {
		return nil
	}

	return func(_ http.ResponseWriter, r *http.Request, argBasePtr unsafe.Pointer) error {
		field := fieldValue(field, argBasePtr).Interface()
		return field.(Extractor).Extract(r, opts.RouteInfo)
	}
}

func extractParamExtractor(field reflect.StructField, opts extractorForOpts) extractorFn {
	if !fieldImplements(field, paramExtType) {
		return nil
	}

	source := reflect.New(field.Type).Interface().(ParamExtractor).Source()
	name := param.NameFromField(field, opts.Namer, source)

	return func(_ http.ResponseWriter, r *http.Request, argBasePtr unsafe.Pointer) error {
		field := fieldValue(field, argBasePtr).Interface()
		return field.(ParamExtractor).Extract(r, opts.RouteInfo, param.Opts{
			Name:    name,
			Default: "",
			Pather:  opts.Pather,
			Parser:  opts.Parser,
		})
	}
}

func extractHTTPRequest(field reflect.StructField, _ extractorForOpts) extractorFn {
	if field.Type != httpReqType {
		return nil
	}

	return func(_ http.ResponseWriter, r *http.Request, argBasePtr unsafe.Pointer) error {
		field := fieldValue(field, argBasePtr).Interface()
		req := field.(**http.Request)
		*req = r

		return nil
	}
}

func extractHTTPResponse(field reflect.StructField, _ extractorForOpts) extractorFn {
	if field.Type != httpRespType {
		return nil
	}

	return func(w http.ResponseWriter, _ *http.Request, argBasePtr unsafe.Pointer) error {
		field := fieldValue(field, argBasePtr).Interface()
		resp := field.(*http.ResponseWriter)
		*resp = w
		return nil
	}
}

var ErrExtactType = errors.New("could not extract type")

func extractFromExtractors(field reflect.StructField, _ extractorForOpts) extractorFn {
	type typeExtractor interface {
		ExtractType(*http.Request) (any, error)
	}

	if extractor, has := extractors.Load(field.Type); has {
		return func(_ http.ResponseWriter, r *http.Request, argBasePtr unsafe.Pointer) error {
			field := fieldValue(field, argBasePtr)
			t, err := extractor.(typeExtractor).ExtractType(r)

			if err == nil {
				field.Elem().Set(reflect.ValueOf(t))
			} else {
				err = fmt.Errorf("%w: %w", ErrExtactType, err)
			}

			return err
		}
	}

	return nil
}

func extractFromStructOfExtractors(
	field reflect.StructField,
	opts extractorForOpts,
) (extractorFn, error) {
	if field.Type.Kind() != reflect.Struct {
		//nolint:nilnil
		return nil, nil
	}

	fn, err := extractorFor(field.Type, opts)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request, argsPtr unsafe.Pointer) error {
		fieldPtr := unsafe.Add(argsPtr, field.Offset)
		return fn(w, r, fieldPtr)
	}, nil
}

func extractorFromFieldWithRelated(
	field reflect.StructField,
	opts extractorForOpts,
) (extractorFn, error) {
	fn, err := extractorFromField(field, opts)

	var want *UnknownFieldTypeError
	if errors.As(err, &want) {
		want.RelatedFound = findRelatedExtractors(field, opts)
	}

	return fn, err
}

func extractorFromField(
	field reflect.StructField,
	opts extractorForOpts,
) (extractorFn, error) {
	type fn func(reflect.StructField, extractorForOpts) extractorFn

	// TODO: allow extractor to specify help
	fns := []fn{
		extractHTTPRequest,
		extractHTTPResponse,
		extractExtractor,
		extractParamExtractor,
		extractFromExtractors,
	}

	for _, extractor := range fns {
		if fn := extractor(field, opts); fn != nil {
			return fn, nil
		}
	}

	if f, err := extractFromStructOfExtractors(field, opts); f != nil || err != nil {
		return f, err
	}

	return nil, &UnknownFieldTypeError{
		Field:        field.Name,
		Type:         field.Type,
		Struct:       nil,
		RelatedFound: nil,
	}
}
