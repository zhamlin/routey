package extractor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/zhamlin/routey/internal/stringz"
	"github.com/zhamlin/routey/internal/structs"
)

// UnknownFieldTypeError represents a field that has no way
// of being extracted.
type UnknownFieldTypeError struct {
	Struct       reflect.Type
	Field        string
	Type         reflect.Type
	RelatedFound []reflect.Type
}

func (e *UnknownFieldTypeError) Error() string {
	builder := &strings.Builder{}
	writeUnknownFieldError(builder, e, structs.NoErrorColors)
	return strings.TrimSuffix(builder.String(), "\n")
}

func (e *UnknownFieldTypeError) ErrorWithColor(c structs.Colors) string {
	builder := &strings.Builder{}
	writeUnknownFieldError(builder, e, c)
	return strings.TrimSuffix(builder.String(), "\n")
}

func (e *UnknownFieldTypeError) setStruct(structType reflect.Type) {
	if e.Struct == nil {
		e.Struct = structType
	}
}

func (e *UnknownFieldTypeError) typ() string {
	if p := e.Struct.PkgPath(); p != "" {
		return strings.ReplaceAll(e.Type.String(), p+".", "")
	}

	return e.Type.String()
}

const errUnknownFieldHelp = `
	field must implement either:
		- [extractor.Extractor]
		- [extractor.ParamExtractor]
	or %q requires an extractor func registered with [extractor.RegisterExtractor]
`

func writeUnknownFieldError(
	msg *strings.Builder,
	unknownField *UnknownFieldTypeError,
	colors structs.Colors,
) {
	errTitle := "cannot determine how to extract field"
	fmt.Fprintf(msg, "%serror%s: %s\n", colors.Error, colors.Reset, errTitle)
	errMsg := fmt.Sprintf("cannot extract %q", unknownField.typ())

	structOutput := structs.PrintStructWithErr(unknownField.Struct, structs.Err{
		FieldType: unknownField.Type,
		FieldName: unknownField.Field,
		Error:     errMsg,
		Underliner: func(name, typ string) (int, int) {
			fieldType := unknownField.typ()
			pos := strings.Index(typ, fieldType)
			if pos < 0 {
				return 0, 0
			}
			start := len(name) + pos
			return start, start + len(fieldType)
		},
	}, colors)

	msg.WriteString(stringz.PrefixBorder("| ", structOutput) + "\n")

	help := fmt.Sprintf(errUnknownFieldHelp, unknownField.typ())
	fmt.Fprintln(msg)
	fmt.Fprintln(msg, stringz.FormatText("help: ", help))

	writeRelatedTypes(msg, unknownField)
}

func writeRelatedTypes(msg *strings.Builder, unknownField *UnknownFieldTypeError) {
	if len(unknownField.RelatedFound) == 0 {
		return
	}

	types := make([]string, 0, len(unknownField.RelatedFound))
	for _, relatedType := range unknownField.RelatedFound {
		types = append(types, "\t- "+relatedType.String())
	}

	hint := "extractors found for the following types:\n"
	hints := strings.Join(types, "\n")
	fmt.Fprintln(msg, stringz.FormatText("hint: ", hint+hints))
	fmt.Fprintln(msg)
}
