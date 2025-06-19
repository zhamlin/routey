package param_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/jsonschema"
	openAPIParam "github.com/zhamlin/routey/openapi3/param"
	"github.com/zhamlin/routey/param"
)

func TestFromInfo_ValidStylesForLocations(t *testing.T) {
	object := struct{}{}
	array := []struct{}{}
	primitive := ""

	tests := []struct {
		source openAPIParam.Location
		style  openAPIParam.Style
		types  []any
	}{
		{
			style:  openAPIParam.StyleDeepObject,
			source: openAPIParam.LocationQuery,
			types:  []any{object},
		},
		{
			style:  openAPIParam.StylePipeDelimited,
			source: openAPIParam.LocationQuery,
			types:  []any{object, array},
		},
		{
			style:  openAPIParam.StyleSpaceDelimited,
			source: openAPIParam.LocationQuery,
			types:  []any{object, array},
		},
		{
			style:  openAPIParam.StyleSimple,
			source: openAPIParam.LocationPath,
			types:  []any{primitive, array, object},
		},
		{
			style:  openAPIParam.StyleSimple,
			source: openAPIParam.LocationHeader,
			types:  []any{primitive, array, object},
		},
		{
			style:  openAPIParam.StyleForm,
			source: openAPIParam.LocationQuery,
			types:  []any{primitive, array, object},
		},
		{
			style:  openAPIParam.StyleForm,
			source: openAPIParam.LocationCookie,
			types:  []any{primitive, array, object},
		},
		{
			style:  openAPIParam.StyleLabel,
			source: openAPIParam.LocationPath,
			types:  []any{primitive, array, object},
		},
		{
			style:  openAPIParam.StyleMatrix,
			source: openAPIParam.LocationPath,
			types:  []any{primitive, array, object},
		},
	}

	schemer := jsonschema.NewSchemer()
	for _, have := range tests {
		tag := fmt.Sprintf(`style:"%s"`, have.style)

		for _, typ := range have.types {
			info := param.Info{
				Source: string(have.source),
				Field:  reflect.StructField{Tag: reflect.StructTag(tag)},
				Type:   reflect.TypeOf(typ),
			}

			_, err := openAPIParam.FromInfo(info, schemer)
			test.NoError(t, err)
		}
	}
}

func TestFromInfo_InvalidSource(t *testing.T) {
	schemer := jsonschema.NewSchemer()
	tests := []struct {
		source string
	}{
		{source: ""},
		{source: "invalid"},
	}

	for _, have := range tests {
		info := param.Info{Source: have.source}
		_, err := openAPIParam.FromInfo(info, schemer)
		test.IsError(t, err, openAPIParam.ErrInvalidLocation)
	}
}

func TestFromInfo_DefaultStyle(t *testing.T) {
	schemer := jsonschema.NewSchemer()
	tests := []struct {
		source    openAPIParam.Location
		wantStyle openAPIParam.Style
	}{
		{
			source:    openAPIParam.LocationQuery,
			wantStyle: openAPIParam.StyleForm,
		},
		{
			source:    openAPIParam.LocationCookie,
			wantStyle: openAPIParam.StyleForm,
		},
		{
			source:    openAPIParam.LocationHeader,
			wantStyle: openAPIParam.StyleSimple,
		},
		{
			source:    openAPIParam.LocationPath,
			wantStyle: openAPIParam.StyleSimple,
		},
	}

	for _, have := range tests {
		info := param.Info{
			Source: string(have.source),
			Type:   reflect.TypeFor[int](),
		}
		got, err := openAPIParam.FromInfo(info, schemer)
		test.NoError(t, err, "source: %q", have.source)

		if got.Style != string(have.wantStyle) {
			t.Errorf("expected style: %v, got: %v", have.wantStyle, got.Style)
		}
	}
}

func TestInfoToOpenAPIParam_ValidTags(t *testing.T) {
	schemer := jsonschema.NewSchemer()
	withTag := func(tag string) param.Info {
		return param.Info{
			Source: "query",
			Type:   reflect.TypeFor[struct{}](),
			Field: reflect.StructField{
				Tag: reflect.StructTag(tag),
			},
		}
	}

	tests := []struct {
		info     param.Info
		validate func(openAPIParam.Parameter) bool
	}{
		{
			info:     withTag(`explode:"true"`),
			validate: func(p openAPIParam.Parameter) bool { return p.Explode },
		},
		{
			info:     withTag(`deprecated:"true"`),
			validate: func(p openAPIParam.Parameter) bool { return p.Deprecated },
		},
		{
			info:     withTag(`required:"true"`),
			validate: func(p openAPIParam.Parameter) bool { return p.Required },
		},
		{
			info:     withTag(`style:"deepObject"`),
			validate: func(p openAPIParam.Parameter) bool { return p.Style == "deepObject" },
		},
	}

	for _, have := range tests {
		got, err := openAPIParam.FromInfo(have.info, schemer)
		test.NoError(t, err)

		if !have.validate(got) {
			t.Errorf("%s validator failed", have.info.Field.Tag)
		}
	}
}

func TestInfoToOpenAPIParam_InvalidTags(t *testing.T) {
	schemer := jsonschema.NewSchemer()
	withTag := func(tag string) param.Info {
		return param.Info{
			Field: reflect.StructField{
				Tag: reflect.StructTag(tag),
			},
		}
	}

	tests := []struct {
		info param.Info
		want error
	}{
		{
			info: withTag(`explode:"invalid"`),
			want: strconv.ErrSyntax,
		},
		{
			info: withTag(`deprecated:"invalid"`),
			want: strconv.ErrSyntax,
		},
		{
			info: withTag(`required:"invalid"`),
			want: strconv.ErrSyntax,
		},
		{
			info: withTag(`style:"invalid"`),
			want: openAPIParam.ErrInvalidStyle,
		},
	}

	for _, have := range tests {
		_, err := openAPIParam.FromInfo(have.info, schemer)
		test.IsError(t, err, have.want)
	}
}

func TestInfoToOpenAPIParam_DefaultValue(t *testing.T) {
	params, err := param.InfoFromStruct[struct {
		FieldName routey.Query[int] `default:"1"`
	}](param.NamerCapitals, param.ParseInt)
	test.NoError(t, err)

	schemer := jsonschema.NewSchemer()
	intSchema, err := schemer.Get(int(0))
	test.NoError(t, err)

	got, err := openAPIParam.FromInfo(params[0], schemer)
	test.NoError(t, err)

	want := openAPIParam.New()
	want.Name = "field_name"
	want.Style = "form"
	want.In = "query"
	want.Explode = true
	want.SetSchema(intSchema)
	want.Schema.Spec.Default = "1"

	test.MatchAsJSON(t, got, want)
}

func TestInfoToOpenAPIParam_ValidParam(t *testing.T) {
	params, err := param.InfoFromStruct[struct {
		FieldName routey.Query[int] `style:"form"`
	}](param.NamerCapitals, param.ParseInt)
	test.NoError(t, err)

	schemer := jsonschema.NewSchemer()
	intSchema, err := schemer.Get(int(0))
	test.NoError(t, err)

	got, err := openAPIParam.FromInfo(params[0], schemer)
	test.NoError(t, err)

	want := openAPIParam.New()
	want.Name = "field_name"
	want.Style = "form"
	want.In = "query"
	want.Explode = true
	want.SetSchema(intSchema)

	test.MatchAsJSON(t, got, want)
}

func TestInfoToOpenAPIParam_PathAlwaysRequired(t *testing.T) {
	params, err := param.InfoFromStruct[struct {
		Path routey.Path[int]
	}](param.NamerCapitals, param.ParseInt)
	test.NoError(t, err)

	schemer := jsonschema.NewSchemer()
	intSchema, err := schemer.Get(int(0))
	test.NoError(t, err)

	got, err := openAPIParam.FromInfo(params[0], schemer)
	test.NoError(t, err)

	want := openAPIParam.New()
	want.Name = "path"
	want.Required = true
	want.Style = "simple"
	want.In = "path"
	want.SetSchema(intSchema)

	test.MatchAsJSON(t, got, want)
}

func TestStyleFromString(t *testing.T) {
	tests := []struct {
		have string
		want openAPIParam.Style
	}{
		{
			have: "form",
			want: openAPIParam.StyleForm,
		},
		{
			have: "matrix",
			want: openAPIParam.StyleMatrix,
		},
		{
			have: "simple",
			want: openAPIParam.StyleSimple,
		},
		{
			have: "label",
			want: openAPIParam.StyleLabel,
		},
		{
			have: "spaceDelimited",
			want: openAPIParam.StyleSpaceDelimited,
		},
		{
			have: "pipeDelimited",
			want: openAPIParam.StylePipeDelimited,
		},
		{
			have: "deepObject",
			want: openAPIParam.StyleDeepObject,
		},
	}

	for _, test := range tests {
		got, err := openAPIParam.StyleFromString(test.have)
		if err != nil {
			t.Fatal(err)
		}

		if got != test.want {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestStyleFromString_ErrorInvalid(t *testing.T) {
	_, err := openAPIParam.StyleFromString("invalid")
	test.IsError(t, err, openAPIParam.ErrInvalidStyle)
}

func TestLocationFromString(t *testing.T) {
	tests := []struct {
		have string
		want openAPIParam.Location
	}{
		{
			have: "path",
			want: openAPIParam.LocationPath,
		},
		{
			have: "query",
			want: openAPIParam.LocationQuery,
		},
		{
			have: "header",
			want: openAPIParam.LocationHeader,
		},
		{
			have: "cookie",
			want: openAPIParam.LocationCookie,
		},
	}

	for _, test := range tests {
		got, err := openAPIParam.LocationFromString(test.have)
		if err != nil {
			t.Fatal(err)
		}

		if got != test.want {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestLocationFromString_ErrorInvalid(t *testing.T) {
	_, err := openAPIParam.LocationFromString("invalid")
	test.IsError(t, err, openAPIParam.ErrInvalidLocation)
}

func TestParam_QueryDefaultExplodeOvverideInJSON(t *testing.T) {
	schemer := jsonschema.NewSchemer()
	info := param.Info{
		Source: "query",
		Type:   reflect.TypeFor[struct{}](),
		Field: reflect.StructField{
			Tag: reflect.StructTag(`explode:"false"`),
		},
	}

	got, err := openAPIParam.FromInfo(info, schemer)
	test.NoError(t, err)

	// Query params with a style of form default to explode=true,
	// verify when setting to false it is included in the json.
	test.MatchAsJSON(t, got, `
	{
		"in": "query",
		"explode": false,
		"name": "",
		"schema": {
			"type": "object"
		},
		"style": "form"
	}
	`)
}
