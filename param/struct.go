package param

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/zhamlin/routey/internal/stringz"
	"github.com/zhamlin/routey/internal/structs"
)

type Info struct {
	Name    string
	Source  string
	Default string
	// Type of the param, can be different than the
	// fields type.
	Type reflect.Type
	// Field that contains the param.
	Field reflect.StructField
	// Struct containing the field.
	Struct reflect.Type
	// If the param was nested this will be set with the field.
	ParentFields []reflect.StructField
}

type sourcer interface {
	Source() string
}

type inner interface {
	Inner() any
}

func GetSourceAndType(typ reflect.Type) (string, reflect.Type, bool) {
	value := reflect.New(typ).Interface()

	sourcer, ok := value.(sourcer)
	if !ok {
		return "", nil, false
	}

	gotType := typ
	if typer, ok := value.(inner); ok {
		if t := reflect.TypeOf(typer.Inner()); t != nil {
			gotType = t
		}
	}

	return sourcer.Source(), gotType, true
}

var ErrNonStructArg = errors.New("handler argument should be a struct")

// InvalidParamError represents a param that cannot be parsed.
type InvalidParamError struct {
	Struct       reflect.Type
	Field        reflect.StructField
	ParamType    reflect.Type
	Message      string
	Err          string
	UnderlineAll bool
}

func (e InvalidParamError) Error() string {
	builder := &strings.Builder{}
	writeInvalidParamError(builder, &e, structs.NoErrorColors)
	return strings.TrimSuffix(builder.String(), "\n")
}

func (e InvalidParamError) Type() string {
	typ := e.ParamType
	if typ == nil {
		typ = e.Field.Type
	}

	if p := e.Struct.PkgPath(); p != "" {
		return strings.ReplaceAll(typ.String(), p+".", "")
	}

	return typ.String()
}

func (e InvalidParamError) ErrorWithColor(c structs.Colors) string {
	builder := &strings.Builder{}
	writeInvalidParamError(builder, &e, c)
	return strings.TrimSuffix(builder.String(), "\n")
}

const errInvalidParamHelp = `
	r.Params.Parser defines how types are parsed
	By default most go built in types and [encoding.TextUnmarshaler] are included
`

func writeInvalidParamError(
	msg *strings.Builder,
	invalidParam *InvalidParamError,
	colors structs.Colors,
) {
	errTitle := "cannot determine how to parse param"
	if invalidParam.Message != "" {
		errTitle = invalidParam.Message
	}

	errMsg := invalidParam.Err
	if errMsg == "" {
		errMsg = fmt.Sprintf("cannot parse %q", invalidParam.Type())
	}

	fmt.Fprintf(msg, "%serror%s: %s\n", colors.Error, colors.Reset, errTitle)

	structOutput := structs.PrintStructWithErr(invalidParam.Struct, structs.Err{
		FieldType: invalidParam.Field.Type,
		FieldName: invalidParam.Field.Name,
		Error:     errMsg,
		// TODO: Underliner not aware of changed type name later
		// formatFieldType
		Underliner: func(name, typ string) (int, int) {
			if invalidParam.UnderlineAll {
				return 0, 0
			}

			innerType := invalidParam.Type()
			pos := strings.Index(typ, innerType)
			if pos < 0 {
				return 0, 0
			}
			start := len(name) + pos
			return start, start + len(innerType)
		},
	}, colors)

	msg.WriteString(stringz.PrefixBorder("| ", structOutput) + "\n")
	fmt.Fprintln(msg)
	fmt.Fprintln(msg, stringz.FormatText("help: ", errInvalidParamHelp))
}

func NameFromField(f reflect.StructField, namer Namer, source string) string {
	name := f.Tag.Get("name")
	if name == "" {
		name = namer(f.Name, source)
	}
	return name
}

type CustomParser interface {
	CanParse(p Parser, source reflect.StructField, value any) error
}

// canParseType tests if the parser can handle the given type.
//
//nolint:wrapcheck // do not wrap errors to reduce allocations
func canParseType(
	parser Parser,
	value any,
	field reflect.StructField,
) error {
	if !errors.Is(parser(value, []string{""}), ErrInvalidParamType) {
		return nil
	}

	f := reflect.New(field.Type)
	if p, ok := f.Interface().(CustomParser); ok {
		return p.CanParse(parser, field, value)
	}

	if p, ok := f.Elem().Interface().(CustomParser); ok {
		return p.CanParse(parser, field, value)
	}

	return ErrInvalidParamType
}

func InfoFromStruct[T any](namer Namer, parser Parser) ([]Info, error) {
	structType := reflect.TypeFor[T]()
	return infoFromValue(structType, namer, parser)
}

var ErrUnparsableDefault = "default value cannot be parsed"

func getType(value any) reflect.Type {
	var structType reflect.Type
	if t, ok := value.(reflect.Type); ok {
		structType = t
	} else {
		structType = reflect.TypeOf(value)
	}

	return structType
}

func infoFromValue(value any, namer Namer, parser Parser) ([]Info, error) {
	structType := getType(value)
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: got: %q", ErrNonStructArg, structType)
	}

	params := make([]Info, 0, structType.NumField())
	for i := range structType.NumField() {
		field := structType.Field(i)
		info, err := infoFromField(structType, field, namer, parser)

		if err != nil {
			return nil, err
		}

		params = append(params, info...)
	}

	return params, nil
}

func infoFromField(
	structType reflect.Type,
	field reflect.StructField,
	namer Namer,
	parser Parser,
) ([]Info, error) {
	source, typ, isParam := GetSourceAndType(field.Type)
	if !isParam {
		return getParamsFromStruct(field, namer, parser)
	}

	value := reflect.New(typ).Interface()
	if err := canParseType(parser, value, field); err != nil {
		var want *InvalidParamError
		if errors.As(err, &want) {
			return nil, want
		}

		return nil, &InvalidParamError{
			Struct:    structType,
			Field:     field,
			ParamType: typ,
		}
	}

	defaultValue := field.Tag.Get("default")
	if defaultValue != "" {
		err := parser(value, []string{defaultValue})
		if err != nil {
			return nil, &InvalidParamError{
				Struct:    structType,
				ParamType: typ,
				Field:     field,
				Err:       err.Error(),
				Message:   ErrUnparsableDefault + ": " + defaultValue,
			}
		}
	}

	name := NameFromField(field, namer, source)
	return []Info{{
		Name:    name,
		Source:  source,
		Default: defaultValue,
		Type:    typ,
		Field:   field,
		Struct:  structType,
	}}, nil
}

var ErrNoParser = errors.New("no param parser provided")

func getParamsFromStruct(
	field reflect.StructField,
	namer Namer,
	parser Parser,
) ([]Info, error) {
	if parser == nil {
		return nil, ErrNoParser
	}

	if field.Type.Kind() != reflect.Struct {
		return nil, nil
	}

	infos, err := infoFromValue(field.Type, namer, parser)
	if err != nil {
		return nil, err
	}

	params := make([]Info, 0, len(infos))
	for _, info := range infos {
		info.ParentFields = append(info.ParentFields, field)
		params = append(params, info)
	}
	return params, nil
}
