package param_test

import (
	"reflect"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/param"
)

func compareErrors(t *testing.T, err error, want string) {
	t.Helper()

	got := err.Error()
	const tabSize = 4
	test.VisuallyMatch(t, got, want, tabSize)
}

func TestInvalidParamError(t *testing.T) {
	type object struct {
		Field string
	}

	typ := reflect.TypeFor[object]()
	err := param.InvalidParamError{
		Struct:    typ,
		Field:     typ.Field(0),
		ParamType: reflect.TypeFor[string](),
		Message:   "",
		Err:       "",
	}

	want := `
error: cannot determine how to parse param
| type object struct {
|     Field string
|           ^^^^^^
|           |
|           cannot parse "string"
| }

help: r.Params.Parser defines how types are parsed
	  By default most go built in types and [encoding.TextUnmarshaler] are included
`
	compareErrors(t, err, want)
}

func TestNameFromField_OverrideWithTag(t *testing.T) {
	type Object struct {
		Field string `name:"new_name"`
	}

	f := reflect.TypeFor[Object]().Field(0)
	want := "new_name"
	got := param.NameFromField(f, param.NamerCapitals, "")

	if got != want {
		t.Errorf("got: %v, wanted: %v", got, want)
	}
}

func TestGetParamsFromStruct(t *testing.T) {
	type Params struct{ Value routey.Query[int] }
	got, err := param.InfoFromStruct[Params](param.NamerCapitals, param.ParseInt)
	test.NoError(t, err)

	want := []param.Info{
		{
			Name:   "value",
			Source: "query",
			Type:   reflect.TypeOf(int(0)),
			Field:  reflect.TypeFor[Params]().Field(0),
			Struct: reflect.TypeFor[Params](),
		},
	}
	test.MatchAsJSON(t, got, want)
}

func TestGetParamsFromStruct_DefaultValue(t *testing.T) {
	type Params struct {
		Value routey.Query[int] `default:"1"`
	}
	got, err := param.InfoFromStruct[Params](param.NamerCapitals, param.ParseInt)
	test.NoError(t, err)

	want := []param.Info{
		{
			Name:    "value",
			Source:  "query",
			Default: "1",
			Type:    reflect.TypeOf(int(0)),
			Field:   reflect.TypeFor[Params]().Field(0),
			Struct:  reflect.TypeFor[Params](),
		},
	}
	test.MatchAsJSON(t, got, want)
}

func TestGetParamsFromStruct_SkipNonParam(t *testing.T) {
	type Params struct{ Skipped int }
	got, err := param.InfoFromStruct[Params](param.NamerCapitals, param.ParseString)
	test.NoError(t, err)

	want := []param.Info{}
	test.MatchAsJSON(t, got, want)
}

func TestGetParamsFromStruct_InvalidParamErr(t *testing.T) {
	type Params struct{ Value routey.Query[int] }
	_, err := param.InfoFromStruct[Params](param.NamerCapitals, param.ParseString)

	var want *param.InvalidParamError
	test.WantError(t, err, &want)
}

func TestGetParamsFromStruct_NonStructError(t *testing.T) {
	_, err := param.InfoFromStruct[int](nil, nil)
	test.IsError(t, err, param.ErrNonStructArg)
}
