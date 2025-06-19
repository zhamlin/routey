package param_test

import (
	"reflect"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/openapi3/param"
)

func compareErrors(t *testing.T, err error, want string) {
	t.Helper()

	got := err.Error()
	tabSize := 4

	test.VisuallyMatch(t, got, want, tabSize)
}

func TestInvalidParamStyleError_Display(t *testing.T) {
	type source struct {
		Field routey.Query[string]
	}
	typ := reflect.TypeFor[source]()
	field, _ := typ.FieldByName("Field")
	err := param.InvalidParamStyleError{
		Struct:       typ,
		Type:         reflect.TypeFor[string](),
		Field:        field,
		Style:        param.StyleDeepObject,
		DataType:     param.DataTypePrimitive,
		Location:     param.LocationQuery,
		Msg:          `invalid style "deepObject" with type: "primitive"`,
		UnderlineMsg: "invalid type for style",
	}

	want := `
error: openapi: invalid style "deepObject" with type: "primitive"
| type source struct {
|     Field extractor.Query[string]
|                           ^^^^^^
|                           |
|                           invalid type for style
| }

help: style "deepObject" supports:
      +--------+
      | type   |
      |--------|
      | object |
      +--------+

      type "primitive" supports:
      +--------+
      | style  |
      |--------|
      | form   |
      | label  |
      | matrix |
      | simple |
      +--------+

      location "query" supports:
      +----------------+
      | style          |
      |----------------|
      | deepObject     |
      | form           |
      | pipeDelimited  |
      | spaceDelimited |
      +----------------+
`
	compareErrors(t, err, want)
}
