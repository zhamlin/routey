package route

import (
	"reflect"

	"github.com/zhamlin/routey/param"
)

type Option func(*Info) error

type Context map[any]any

// RouteInfo contains information about a given pattern and its handler.
type Info struct {
	Handler any `json:"-"`
	// Method is optional, a handler could be for all methods.
	Method string
	// Full pattern that of the route. If mounted this will contain
	// the mount prefix.
	FullPattern string
	// Pattern used to register the handler.
	Pattern    string
	Params     []param.Info
	ReturnType reflect.Type

	// Stored values provided during the route registering.
	Context Context `json:"-"`
	Options []Option
}
