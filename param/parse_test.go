package param_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/param"
)

func compareParsed(t *testing.T, want any, params []string, parser param.Parser) {
	t.Helper()

	got := reflect.New(reflect.TypeOf(want))
	err := parser(got.Interface(), params)

	if err != nil {
		t.Fatalf("compareParsed: expected no error, got: %v", err)
	}

	if got := got.Elem().Interface(); !reflect.DeepEqual(want, got) {
		t.Errorf("wanted: %v, got: %v", want, got)
	}
}

type textUnmarshaler struct {
	Value string
}

func (t *textUnmarshaler) UnmarshalText(text []byte) error {
	t.Value = string(text)
	return nil
}

func TestParseTextUnmarshaller(t *testing.T) {
	want := textUnmarshaler{Value: "test"}
	compareParsed(t, want, []string{"test"}, param.ParseTextUnmarshaller)
}

func TestParseBool(t *testing.T) {
	want := true
	compareParsed(t, want, []string{"true"}, param.ParseBool)

	want = false
	compareParsed(t, want, []string{"false"}, param.ParseBool)
}

func TestParseString(t *testing.T) {
	want := "test"
	compareParsed(t, want, []string{"test"}, param.ParseString)
}

func TestParseParamsInt(t *testing.T) {
	tests := []struct {
		want   any
		params []string
	}{
		{
			want:   int(1),
			params: []string{"1"},
		},
		{
			want:   int8(1),
			params: []string{"1"},
		},
		{
			want:   int16(1),
			params: []string{"1"},
		},
		{
			want:   int32(1),
			params: []string{"1"},
		},
		{
			want:   int64(1),
			params: []string{"1"},
		},
	}

	for _, test := range tests {
		compareParsed(t, test.want, test.params, param.ParseInt)
	}
}

func TestParseParamsUint(t *testing.T) {
	tests := []struct {
		want   any
		params []string
	}{
		{
			want:   uint(1),
			params: []string{"1"},
		},
		{
			want:   uint8(1),
			params: []string{"1"},
		},
		{
			want:   uint16(1),
			params: []string{"1"},
		},
		{
			want:   uint32(1),
			params: []string{"1"},
		},
		{
			want:   uint64(1),
			params: []string{"1"},
		},
	}

	for _, test := range tests {
		compareParsed(t, test.want, test.params, param.ParseUint)
	}
}

func TestParseParamsFloat(t *testing.T) {
	tests := []struct {
		want   any
		params []string
	}{
		{
			want:   float32(1.01),
			params: []string{"1.01"},
		},
		{
			want:   float64(1.01),
			params: []string{"1.01"},
		},
	}

	for _, test := range tests {
		compareParsed(t, test.want, test.params, param.ParseFloat)
	}
}

func TestParseParamReflect(t *testing.T) {
	tests := []struct {
		want   any
		params []string
	}{
		{
			want:   []int{1, 2, 3},
			params: []string{"1", "2", "3"},
		},
		{
			want:   []int{1, 2, 3},
			params: []string{"1,2,3"},
		},
	}

	parser := param.NewReflectParser(param.ParseInt)
	for _, test := range tests {
		compareParsed(t, test.want, test.params, parser)
	}
}

func TestParseParamReflect_ErrorParsingItem(t *testing.T) {
	parse := param.Parsers{
		param.NewReflectParser(param.ParseBool),
	}.Parse

	var b []bool
	err := parse(&b, []string{"fals"})
	test.IsError(t, err, strconv.ErrSyntax)
}

func TestParsersParse(t *testing.T) {
	parse := param.Parsers{
		param.ParseBool,
		param.ParseUint,
		param.ParseFloat,
		param.ParseString,
		param.ParseTextUnmarshaller,
		param.ParseInt,
	}.Parse

	want := 1
	input := []string{fmt.Sprint(want)}
	compareParsed(t, want, input, parse)
}

func TestParsers_ParseNoMatch(t *testing.T) {
	parse := param.Parsers{
		param.NewReflectParser(param.ParseBool),
	}.Parse

	var s string
	err := parse(&s, []string{""})
	test.IsError(t, err, param.ErrInvalidParamType)
}

func BenchmarkParsesParse(b *testing.B) {
	parse := param.Parsers{
		param.ParseBool,
		param.ParseUint,
		param.ParseFloat,
		param.ParseString,
		param.ParseTextUnmarshaller,
		param.ParseInt,
	}.Parse

	var value int
	input := []string{"1"}

	for b.Loop() {
		err := parse(&value, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
