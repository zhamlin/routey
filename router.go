package routey

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/internal"
	"github.com/zhamlin/routey/param"
	"github.com/zhamlin/routey/route"
	"github.com/zhamlin/routey/std"
)

type Path[T any] = extractor.Path[T]
type Query[T any] = extractor.Query[T]
type JSON[T any] = extractor.JSON[T]

// Mux is the interface implemented by an object that can
// be used as a http handler.
type Mux interface {
	http.Handler
	param.Pather

	Handle(method, pattern string, handler http.Handler)
}

func newParamParsers() param.Parser {
	parsers := param.Parsers{
		param.ParseTextUnmarshaller,
		param.ParseInt,
		param.ParseUint,
		param.ParseFloat,
		param.ParseString,
		param.ParseBool,
	}

	reflectParser := param.NewReflectParser(parsers.Parse)
	finalParser := param.Parsers{parsers.Parse, reflectParser}
	return finalParser.Parse
}

// New returns a ready to use [Router] with the default settings.
func New() *Router {
	return &Router{
		pattern:  "",
		isNested: false,
		routes:   &sharedRoutes{},
		middleware: middlewareConfig{
			global: []Middleware{},
			route:  []Middleware{},
		},
		Mux: std.Mux{ServeMux: http.NewServeMux()},
		ErrorSink: func(err error) {
			fmt.Println(err.Error())
			os.Exit(1)
		},
		Params: param.Config{
			Parser: newParamParsers(),
			Namer:  param.NamerCapitals,
		},
		Errors: ErrorConfig{
			Colored:    false,
			CallerSkip: 1,
		},
		Response: nil,
		Context:  route.Context{},
	}
}

type Middleware func(http.Handler) http.Handler

type middlewareConfig struct {
	global []Middleware
	route  []Middleware
}

func applyMiddleware(h http.Handler, mw ...Middleware) http.Handler {
	for _, mw := range slices.Backward(mw) {
		h = mw(h)
	}
	return h
}

type sharedRoutes struct {
	Routes []*route.Info
}

func (sb *sharedRoutes) Append(infos ...*route.Info) {
	sb.Routes = append(sb.Routes, infos...)
}

func (sb *sharedRoutes) Pop() (*route.Info, bool) {
	if len(sb.Routes) > 0 {
		last := sb.Routes[len(sb.Routes)-1]
		sb.Routes = sb.Routes[:len(sb.Routes)-1]
		return last, true
	}
	return nil, false
}

func (sb *sharedRoutes) Last() *route.Info {
	if len(sb.Routes) > 0 {
		return sb.Routes[len(sb.Routes)-1]
	}
	return nil
}

type Router struct {
	silentAdd  bool
	pattern    string
	isNested   bool
	middleware middlewareConfig
	routes     *sharedRoutes
	// The base router used to register handlers with.
	Mux Mux
	// Called when there is an error while registering handlers.
	ErrorSink func(error)
	// Called when a new route is added to the router.
	OnRouteAdd func(*route.Info) error
	// Default values to set on route.Info.
	Context  route.Context
	Response extractor.ResponseHandler
	Params   param.Config
	Errors   ErrorConfig
}

func (r *Router) Routes() []*route.Info {
	return r.routes.Routes
}

// Mount handles nested routers by applying global middleware to the mounted handler.
func (r *Router) Mount(pattern string, handler http.Handler) {
	newPattern, err := url.JoinPath(pattern, "/")
	if err != nil {
		r.handleError(err)
	}

	handle := r.Handle
	router, ok := handler.(*Router)

	if ok {
		handle = r.silentHandle
	}

	handle("", newPattern, http.StripPrefix(pattern, handler))

	if ok {
		// remove the route added from handle call above
		r.routes.Pop()

		for _, route := range router.routes.Routes {
			route.FullPattern = joinPatterns(newPattern, route.FullPattern)
			route.Context = maps.Clone(r.Context)

			r.routes.Append(route)
			r.onRouteAdd(route)
		}
	}
}

// Use appends the middlware onto the router middleware stack.
func (r *Router) Use(mw ...Middleware) {
	if r.isNested {
		r.middleware.route = append(r.middleware.route, mw...)
	} else {
		r.middleware.global = append(r.middleware.global, mw...)
	}
}

func (r *Router) Route(pattern string, fn func(*Router)) {
	cloned := r.clone()
	cloned.pattern = pattern
	fn(cloned)
}

// With appends the middlware onto the handlers middleware stack.
func (r *Router) With(mw ...Middleware) *Router {
	cloned := r.clone()
	cloned.isNested = true
	cloned.Use(mw...)
	return cloned
}

// Group creates a new router that will use any middleware declared
// in the group and the parent groups.
func (r *Router) Group(fn func(*Router)) {
	cloned := r.clone()
	cloned.isNested = true
	fn(cloned)
}

// ServeHTTP implments the [http.Handler] interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Mux.ServeHTTP(w, req)
}

func joinPatterns(prefix, pattern string) string {
	if prefix == "" {
		return pattern
	}

	if pattern == "/" {
		return strings.TrimRight(prefix, "/")
	}

	// Normalize paths by ensuring exactly one slash between parts
	prefix = strings.TrimRight(prefix, "/")
	pattern = strings.TrimLeft(pattern, "/")

	if pattern == "" {
		return prefix
	}
	return prefix + "/" + pattern
}

func (r *Router) Handle(method, pattern string, handler http.Handler, opts ...route.Option) {
	pattern = joinPatterns(r.pattern, pattern)
	info := r.getOrAddRouteInfo(route.Info{
		Method:      method,
		FullPattern: pattern,
		Pattern:     pattern,
		Context:     maps.Clone(r.Context),
		Options:     opts,
	})

	for _, opt := range opts {
		if err := opt(info); err != nil {
			err = fmt.Errorf("option returned an error: %w", err)
			err = maybeToHandlerErr(err, method, pattern, info.Handler)
			r.handleError(err)
		}
	}

	handler = applyMiddleware(handler, r.middleware.route...)
	handler = applyMiddleware(handler, r.middleware.global...)

	r.Mux.Handle(method, pattern, handler)
	r.onRouteAdd(info)
}

func (r *Router) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	r.Handle(method, pattern, handler)
}

func (r *Router) Get(pattern string, handler http.HandlerFunc, opts ...route.Option) {
	r.Handle(http.MethodGet, pattern, handler, opts...)
}

func (r *Router) Put(pattern string, handler http.HandlerFunc, opts ...route.Option) {
	r.Handle(http.MethodPut, pattern, handler, opts...)
}

func (r *Router) Post(pattern string, handler http.HandlerFunc, opts ...route.Option) {
	r.Handle(http.MethodPost, pattern, handler, opts...)
}

func (r *Router) Patch(pattern string, handler http.HandlerFunc, opts ...route.Option) {
	r.Handle(http.MethodPatch, pattern, handler, opts...)
}

func (r *Router) Delete(pattern string, handler http.HandlerFunc, opts ...route.Option) {
	r.Handle(http.MethodDelete, pattern, handler, opts...)
}

func (r *Router) silentHandle(method, pattern string, handler http.Handler, opts ...route.Option) {
	r.silentAdd = true
	r.Handle(method, pattern, handler, opts...)
	r.silentAdd = false
}

func (r *Router) clone() *Router {
	return &Router{
		isNested: false,
		routes:   r.routes,
		pattern:  r.pattern,
		middleware: middlewareConfig{
			global: slices.Clone(r.middleware.global),
			route:  slices.Clone(r.middleware.route),
		},
		Mux:        r.Mux,
		ErrorSink:  r.ErrorSink,
		Response:   r.Response,
		Params:     r.Params,
		Errors:     r.Errors,
		OnRouteAdd: r.OnRouteAdd,
		Context:    maps.Clone(r.Context),
	}
}

func (r *Router) onRouteAdd(info *route.Info) {
	if fn := r.OnRouteAdd; !r.silentAdd && fn != nil {
		if err := fn(info); err != nil {
			r.handleError(err)
		}
	}
}

func (r *Router) getOrAddRouteInfo(i route.Info) *route.Info {
	if last := r.routes.Last(); last != nil {
		needsInfo := last.Method != i.Method || last.FullPattern != i.FullPattern

		if !needsInfo {
			return last
		}
	}

	r.routes.Append(&i)
	return &i
}

func (r *Router) handleError(err error) {
	var hErr HandlerError
	if errors.As(err, &hErr) {
		hErr.CallerSkip = r.Errors.CallerSkip + 1
		err = hErr
	}

	if r.ErrorSink != nil {
		r.ErrorSink(coloredError{
			err:    err,
			colors: r.Errors.color(),
		})
	}
}

func (r *Router) handlerParams(pattern string) extractor.HandlerParams {
	return extractor.HandlerParams{
		Response:         r.Response,
		ErrorSink:        r.handleError,
		Parser:           r.Params.Parser,
		Namer:            r.Params.Namer,
		ParamPather:      r.Mux,
		Pattern:          pattern,
		CollectAllErrors: r.Errors.CollectAll,
	}
}

func maybeToHandlerErr(err error, method, pattern string, handler any) error {
	if errors.As(err, &HandlerError{}) {
		return err
	}

	if method != "" {
		pattern = method + " " + pattern
	}

	err = HandlerError{
		Err:     err,
		Pattern: pattern,
		Handler: internal.GetFnInfo(handler),
	}
	return err
}

func Handle[T, R any](
	r *Router,
	method, pattern string,
	handler func(T) (R, error),
	opts ...route.Option,
) {
	prefixPattern := joinPatterns(r.pattern, pattern)

	// only used when being displayed in errors
	errPattern := prefixPattern
	if method != "" {
		errPattern = method + " " + errPattern
	}
	hParmas := r.handlerParams(errPattern)

	params, err := param.InfoFromStruct[T](r.Params.Namer, r.Params.Parser)
	if err != nil {
		r.handleError(HandlerError{
			Err:     err,
			Pattern: hParmas.Pattern,
			Handler: internal.GetFnInfo(handler),
		})
		return
	}

	info := route.Info{
		Params:      params,
		Method:      method,
		FullPattern: prefixPattern,
		Pattern:     pattern,
		ReturnType:  reflect.TypeFor[R](),
		Context:     maps.Clone(r.Context),
		Options:     opts,
	}
	hParmas.RouteInfo = r.getOrAddRouteInfo(info)
	hParmas.ErrorSink = func(err error) {
		r.handleError(HandlerError{
			Err:        err,
			Pattern:    hParmas.Pattern,
			Handler:    internal.GetFnInfo(handler),
			CallerSkip: r.Errors.CallerSkip + 1,
		})
	}

	h := extractor.Handler(handler, hParmas)
	if h == nil {
		return
	}
	hParmas.RouteInfo.Handler = handler

	r.Handle(method, pattern, h, opts...)
}

func Get[T, R any](r *Router, pattern string, fn func(T) (R, error), opts ...route.Option) {
	Handle(r, http.MethodGet, pattern, fn, opts...)
}

func Put[T, R any](r *Router, pattern string, fn func(T) (R, error), opts ...route.Option) {
	Handle(r, http.MethodPut, pattern, fn, opts...)
}

func Post[T, R any](r *Router, pattern string, fn func(T) (R, error), opts ...route.Option) {
	Handle(r, http.MethodPost, pattern, fn, opts...)
}

func Patch[T, R any](r *Router, pattern string, fn func(T) (R, error), opts ...route.Option) {
	Handle(r, http.MethodPatch, pattern, fn, opts...)
}

func Delete[T, R any](r *Router, pattern string, fn func(T) (R, error), opts ...route.Option) {
	Handle(r, http.MethodDelete, pattern, fn, opts...)
}
