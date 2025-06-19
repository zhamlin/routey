package param

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/zhamlin/routey/internal/stringz"
	"github.com/zhamlin/routey/internal/structs"
)

type InvalidParamStyleError struct {
	Struct reflect.Type
	Field  reflect.StructField
	// Type of the parameter, can be different than the field type.
	Type         reflect.Type
	Style        Style
	DataType     DataType
	Location     Location
	Msg          string
	UnderlineMsg string
	err          error
}

func (e InvalidParamStyleError) Unwrap() error {
	return e.err
}

func (e InvalidParamStyleError) Error() string {
	msg := &strings.Builder{}
	writeInvalidParamStyleError(msg, e, structs.NoErrorColors)
	return msg.String()
}

func (e InvalidParamStyleError) ErrorWithColor(c structs.Colors) string {
	msg := &strings.Builder{}
	writeInvalidParamStyleError(msg, e, c)
	return strings.TrimSuffix(msg.String(), "\n")
}

func writeInvalidParamStyleError(
	msg *strings.Builder,
	invalidParam InvalidParamStyleError,
	colors structs.Colors,
) {
	errMsg := "openapi: " + invalidParam.Msg
	fmt.Fprintf(msg, "%serror%s: %s\n", colors.Error, colors.Reset, errMsg)

	structOutput := structs.PrintStructWithErr(invalidParam.Struct, structs.Err{
		FieldType: invalidParam.Field.Type,
		FieldName: invalidParam.Field.Name,
		Error:     invalidParam.UnderlineMsg,
		Underliner: func(name, typ string) (int, int) {
			innerType := invalidParam.Type.String()
			pos := strings.Index(typ, innerType)
			if pos < 0 {
				return 0, 0
			}
			start := len(name) + pos
			return start, start + len(innerType)
		},
	}, colors)

	helpTxt := buildHelpText(invalidParam.Style, invalidParam.DataType, invalidParam.Location)

	msg.WriteString(stringz.PrefixBorder("| ", structOutput) + "\n")
	fmt.Fprintln(msg)
	fmt.Fprintln(msg, stringz.FormatText("help: ", helpTxt))
}

func buildHelpText(style Style, dataType DataType, location Location) string {
	validTypesTable := validTypesTable(style)
	validStyleTable := validStylesTable(dataType)
	validLocStyleTable := validStylesFromLocTable(location)

	sections := []struct {
		name  string
		value string
		table string
	}{
		{"style", string(style), validTypesTable},
		{"type", string(dataType), validStyleTable},
		{"location", string(location), validLocStyleTable},
	}

	var parts []string
	for _, section := range sections {
		if section.value != "" {
			parts = append(parts, fmt.Sprintf("%s %q supports:\n%v",
				section.name, section.value, section.table))
		}
	}

	return strings.Join(parts, "\n\n")
}

func validTypesTable(style Style) string {
	valid := styleValidation.ValidTypes(style)
	return stringz.CreateASCIITableWithOptions("type", valid, stringz.TableOptions{})
}

func validStylesTable(dataType DataType) string {
	valid := styleValidation.ValidStylesForType(dataType)
	return stringz.CreateASCIITableWithOptions("style", valid, stringz.TableOptions{})
}

func validStylesFromLocTable(loc Location) string {
	valid := styleValidation.ValidStylesForLocation(loc)
	return stringz.CreateASCIITableWithOptions("style", valid, stringz.TableOptions{})
}
