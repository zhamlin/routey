package param

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/sv-tools/openapi"
	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/jsonschema"
	"github.com/zhamlin/routey/param"
)

type Parameter struct {
	*openapi.Parameter
}

func New() Parameter {
	return Parameter{
		Parameter: &openapi.Parameter{},
	}
}

func (p *Parameter) SetSchema(schema jsonschema.Schema) {
	p.Schema = openapi.NewRefOrSpec[openapi.Schema](schema.Schema)
}

// https://spec.openapis.org/oas/v3.1.0#styleValues
type Style string

const (
	StyleMatrix         Style = "matrix"
	StyleLabel          Style = "label"
	StyleForm           Style = "form"
	StyleSimple         Style = "simple"
	StyleSpaceDelimited Style = "spaceDelimited"
	StylePipeDelimited  Style = "pipeDelimited"
	StyleDeepObject     Style = "deepObject"
)

var (
	ErrInvalidStyle    = errors.New("invalid parameter style")
	ErrInvalidLocation = errors.New("invalid parameter location")
)

func StyleFromString(str string) (Style, error) {
	switch Style(str) {
	case StyleMatrix:
		return StyleMatrix, nil
	case StyleLabel:
		return StyleLabel, nil
	case StyleForm:
		return StyleForm, nil
	case StyleSimple:
		return StyleSimple, nil
	case StyleSpaceDelimited:
		return StyleSpaceDelimited, nil
	case StylePipeDelimited:
		return StylePipeDelimited, nil
	case StyleDeepObject:
		return StyleDeepObject, nil
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidStyle, str)
}

type Location string

const (
	LocationPath   Location = openapi.InPath
	LocationQuery  Location = openapi.InQuery
	LocationHeader Location = openapi.InHeader
	LocationCookie Location = openapi.InCookie
)

func LocationFromString(str string) (Location, error) {
	switch Location(str) {
	case LocationPath:
		return LocationPath, nil
	case LocationQuery:
		return LocationQuery, nil
	case LocationHeader:
		return LocationHeader, nil
	case LocationCookie:
		return LocationCookie, nil
	}
	return Location(""), fmt.Errorf("%w: %s", ErrInvalidLocation, str)
}

func defaultStyle(in Location) (Style, error) {
	// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md#parameterStyle
	switch in {
	case LocationHeader:
		return StyleSimple, nil
	case LocationPath:
		return StyleSimple, nil
	case LocationQuery:
		return StyleForm, nil
	case LocationCookie:
		return StyleForm, nil
	default:
		return Style(""), ErrInvalidLocation
	}
}

func fromInfoError(i param.Info, p Parameter, dataType DataType, err error) error {
	var typ reflect.Type
	if i.Field.Type != nil {
		_, typ, _ = param.GetSourceAndType(i.Field.Type)
	}

	invalidParam := InvalidParamStyleError{
		Struct:   i.Struct,
		Field:    i.Field,
		Type:     typ,
		Style:    Style(p.Style),
		DataType: dataType,
		Location: Location(p.In),
		err:      err,
	}

	var tagErr updateFromTagsError
	switch {
	case errors.Is(err, errInvalidLocForStyle):
		invalidParam.Type = i.Field.Type
		invalidParam.Msg = fmt.Sprintf("invalid style %q for location %q", p.Style, p.In)
		invalidParam.UnderlineMsg = "invalid location for style"
	case errors.Is(err, errInvalidTypeForLocation):
		invalidParam.Msg = fmt.Sprintf("invalid type %q with style %q", dataType, p.Style)
		invalidParam.UnderlineMsg = "invalid type for style"
	case errors.As(err, &tagErr):
		invalidParam.Type = i.Field.Type
		invalidParam.Msg = fmt.Sprintf("failed to parse: %q", tagErr.Name)
		invalidParam.UnderlineMsg = err.Error()
	default:
		invalidParam.Type = i.Field.Type
		invalidParam.Msg = err.Error()
		invalidParam.UnderlineMsg = err.Error()
	}

	handleErr := routey.HandlerError{
		Err: invalidParam,
	}
	return handleErr
}

func FromInfo(info param.Info, schemer jsonschema.Schemer) (Parameter, error) {
	p := New()
	p.Name = info.Name
	p.In = info.Source

	schema, err := schemer.Get(info.Type)
	if err != nil {
		return p, fmt.Errorf("failed getting schema: %w", err)
	}

	infoHasDefault := info.Default != ""
	schemaHasDefault := schema.Default != nil

	if infoHasDefault && !schemaHasDefault {
		schema.Default = info.Default
	}

	dataType, _ := getSchemasDataType(schema)
	tags := getTags(info.Field.Tag)

	p.SetSchema(schema)

	if err := updateFromTags(tags, p); err != nil {
		return p, fromInfoError(info, p, dataType, err)
	}

	p, err = setDefaults(p, tags)
	if err != nil {
		return p, err
	}

	// https://spec.openapis.org/oas/v3.1.0#styleValues
	validationRule, has := styleValidation[Style(p.Style)]
	if !has {
		return p, fromInfoError(info, p, dataType, ErrInvalidStyle)
	}

	if err := validationRule.Validate(Location(p.In), dataType); err != nil {
		return p, fromInfoError(info, p, dataType, err)
	}

	return p, nil
}

func setDefaults(p Parameter, tags tags) (Parameter, error) {
	if p.Style == "" {
		defaultStyle, err := defaultStyle(Location(p.In))
		if err != nil {
			return p, err
		}
		p.Style = string(defaultStyle)
	}

	if p.Style == string(StyleForm) && tags.explode == "" {
		// form style defaults to explode=true
		p.Explode = true
	}

	// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md#parameter-object
	// Required must be true for path params.
	if p.In == string(LocationPath) {
		p.Required = true
	}

	return p, nil
}

func getSchemasDataType(schema jsonschema.Schema) (DataType, bool) {
	types := map[string]DataType{
		openapi.IntegerType: DataTypePrimitive,
		openapi.NumberType:  DataTypePrimitive,
		openapi.StringType:  DataTypePrimitive,
		openapi.BooleanType: DataTypePrimitive,
		openapi.NullType:    DataTypePrimitive,
		openapi.ObjectType:  DataTypeObject,
		openapi.ArrayType:   DataTypeArray,
	}

	for _, typ := range schema.GetType() {
		if dataType, has := types[typ]; has {
			return dataType, true
		}
	}
	return DataType(""), false
}

// DataType represents the parameter data type.
type DataType string

const (
	DataTypePrimitive DataType = "primitive"
	DataTypeArray     DataType = "array"
	DataTypeObject    DataType = "object"
)

type paramValidation struct {
	types []DataType
	in    []Location
}

var (
	errInvalidLocForStyle     = errors.New("location does not support provided style")
	errInvalidTypeForLocation = errors.New("location does not support provided type")
)

func (p paramValidation) Validate(loc Location, typ DataType) error {
	if !slices.Contains(p.in, loc) {
		return errInvalidLocForStyle
	}

	if !slices.Contains(p.types, typ) {
		return errInvalidTypeForLocation
	}

	return nil
}

type styleValidationRules map[Style]paramValidation

func (s styleValidationRules) ValidTypes(style Style) []DataType {
	validation, has := s[style]
	if !has {
		return nil
	}

	return validation.types
}

func (s styleValidationRules) ValidStylesForType(typ DataType) []Style {
	var styles []Style
	for style, opts := range s {
		if slices.Contains(opts.types, typ) {
			styles = append(styles, style)
		}
	}

	slices.Sort(styles)
	return styles
}

func (s styleValidationRules) ValidStylesForLocation(loc Location) []Style {
	var styles []Style
	for style, opts := range s {
		if slices.Contains(opts.in, loc) {
			styles = append(styles, style)
		}
	}

	slices.Sort(styles)
	return styles
}

func sort[T cmp.Ordered](items ...T) []T {
	slices.Sort(items)
	return items
}

// https://spec.openapis.org/oas/v3.1.0#styleValues
// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#style-values
var styleValidation = styleValidationRules{
	StyleDeepObject: {
		in: []Location{
			LocationQuery,
		},
		types: sort(DataTypeObject),
	},
	StyleSpaceDelimited: {
		in: []Location{
			LocationQuery,
		},
		types: sort(DataTypeArray, DataTypeObject),
	},
	StylePipeDelimited: {
		in: []Location{
			LocationQuery,
		},
		types: sort(DataTypeArray, DataTypeObject),
	},
	StyleSimple: {
		in: []Location{
			LocationPath,
			LocationHeader,
		},
		types: sort(DataTypePrimitive, DataTypeArray, DataTypeObject),
	},
	StyleForm: {
		in: []Location{
			LocationQuery,
			LocationCookie,
		},
		types: sort(DataTypePrimitive, DataTypeArray, DataTypeObject),
	},
	StyleLabel: {
		in: []Location{
			LocationPath,
		},
		types: sort(DataTypePrimitive, DataTypeArray, DataTypeObject),
	},
	StyleMatrix: {
		in: []Location{
			LocationPath,
		},
		types: sort(DataTypePrimitive, DataTypeArray, DataTypeObject),
	},
}

func GetStyleFromTag(tag reflect.StructTag) (Style, error) {
	tags := getTags(tag)
	if tags.style == "" {
		return Style(""), nil
	}

	return StyleFromString(tags.style)
}

type tags struct {
	explode    string
	deprecated string
	style      string
	required   string
	reserved   string
	minimum    string
}

func getTags(tag reflect.StructTag) tags {
	return tags{
		minimum:    tag.Get("minimum"),
		explode:    tag.Get("explode"),
		deprecated: tag.Get("deprecated"),
		style:      tag.Get("style"),
		required:   tag.Get("required"),
		reserved:   tag.Get("reserved"),
	}
}

func parse[T any, F func(string) (T, error)](input string, value *T, fn F) error {
	if input == "" {
		return nil
	}

	result, err := fn(input)
	if err != nil {
		return err
	}

	if value == nil {
		var x T
		value = &x
	}

	*value = result
	return nil
}

func parseBool(input string, value *bool) error {
	return parse(input, value, strconv.ParseBool)
}

func parseInt(input string, value *int) error {
	return parse(input, value, func(s string) (int, error) {
		i, err := strconv.ParseInt(s, 0, 0)
		return int(i), err
	})
}

func parseStyle(input string, value *string) error {
	var style Style
	if err := parse(input, &style, StyleFromString); err != nil {
		return err
	}
	*value = string(style)
	return nil
}

type updateFromTagsError struct {
	Name string
	Err  error
}

func (e updateFromTagsError) Error() string {
	return e.Err.Error()
}

func (e updateFromTagsError) Unwrap() error {
	return e.Err
}

func updateFromTags(tags tags, p Parameter) error {
	wrap := func(tag string, err error) error {
		if err != nil {
			return fmt.Errorf("failed to parse tag %q: %w", tag, err)
		}
		return nil
	}

	if tags.minimum != "" {
		n := 0
		p.Schema.Spec.Minimum = &n
	}

	return cmp.Or(
		wrap("explode", parseBool(tags.explode, &p.Explode)),
		wrap("deprecated", parseBool(tags.deprecated, &p.Deprecated)),
		wrap("required", parseBool(tags.required, &p.Required)),
		wrap("reserved", parseBool(tags.reserved, &p.AllowReserved)),
		wrap("style", parseStyle(tags.style, &p.Style)),
		wrap("minimum", parseInt(tags.minimum, p.Schema.Spec.Minimum)),
	)
}
