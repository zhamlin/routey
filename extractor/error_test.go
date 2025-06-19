package extractor_test

import (
	"reflect"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/extractor"
	"github.com/zhamlin/routey/internal/test"
)

type testHandlerInput struct {
	Value      int
	QueryValue routey.Query[int]
}

func compareErrors(t *testing.T, err error, want string) {
	t.Helper()

	got := err.Error()
	const tabSize = 4
	test.VisuallyMatch(t, got, want, tabSize)
}

func TestCreateHandlerErrMsgUnknownField(t *testing.T) {
	typ := reflect.TypeOf(testHandlerInput{})
	field, _ := typ.FieldByName("Value")
	err := &extractor.UnknownFieldTypeError{
		Struct: typ,
		Field:  field.Name,
		Type:   field.Type,
	}

	want := `
error: cannot determine how to extract field
| type testHandlerInput struct {
|     Value      int
|                ^^^
|                |
|                cannot extract "int"
|     QueryValue extractor.Query[int]
| }

help: field must implement either:
	   - [extractor.Extractor]
	   - [extractor.ParamExtractor]
	  or "int" requires an extractor func registered with [extractor.RegisterExtractor]
`
	compareErrors(t, err, want)
}

func TestCreateHandlerErrMsgUnknownFieldWithRelated(t *testing.T) {
	typ := reflect.TypeOf(testHandlerInput{})
	field, _ := typ.FieldByName("Value")
	err := &extractor.UnknownFieldTypeError{
		Struct:       typ,
		Field:        field.Name,
		Type:         field.Type,
		RelatedFound: []reflect.Type{reflect.TypeOf(new(int))},
	}

	want := `
error: cannot determine how to extract field
| type testHandlerInput struct {
|     Value      int
|                ^^^
|                |
|                cannot extract "int"
|     QueryValue extractor.Query[int]
| }

help: field must implement either:
	   - [extractor.Extractor]
	   - [extractor.ParamExtractor]
	  or "int" requires an extractor func registered with [extractor.RegisterExtractor]

hint: extractors found for the following types:
	   - *int
`
	compareErrors(t, err, want)
}
