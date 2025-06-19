package structs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/zhamlin/routey/internal/color"
)

var NoErrorColors = Colors{
	Error: "",
	Reset: "",
}

type Colors struct {
	Error color.Color
	Reset color.Color
}

type Err struct {
	FieldType  reflect.Type
	FieldName  string
	Error      string
	Underliner func(name, typ string) (int, int)
}

func PrintStructWithErr(typ reflect.Type, err Err, colors Colors) string {
	if typ.Kind() != reflect.Struct {
		panic("provided type is not a struct")
	}

	maxNameLen := calculateMaxFieldNameLength(typ)
	maxTypeLen := calculateMaxFieldTypeLength(typ, typ.PkgPath())

	var sb strings.Builder
	writeStructHeader(&sb, typ)

	for i := range typ.NumField() {
		field := typ.Field(i)
		writeStructField(&sb, field, typ.PkgPath(), maxNameLen, maxTypeLen)

		if shouldShowError(field, err) {
			writeFieldError(&sb, field, typ.PkgPath(), err, colors, maxNameLen)
		}
	}

	sb.WriteString("}")
	return sb.String()
}

func calculateMaxFieldTypeLength(typ reflect.Type, pkgPath string) int {
	maxLen := 0
	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldType := formatFieldType(field.Type.String(), pkgPath)

		if len(fieldType) > maxLen {
			maxLen = len(fieldType)
		}
	}
	return maxLen
}

func calculateErrorUnderline(
	fieldName, fieldType, padding string,
	err Err,
) (string, string) {
	baseSpacing := "    "
	defaultLength := len(fieldName) + len(padding) + 1 + len(fieldType)
	underline := strings.Repeat("^", defaultLength)
	spacing := baseSpacing

	if err.Underliner != nil {
		start, end := err.Underliner(fieldName, fieldType)
		if start > 0 && end > 0 && end > start {
			underline = strings.Repeat("^", end-start)
			spacing = baseSpacing + strings.Repeat(" ", start+len(padding)+1)
		}
	}

	return underline, spacing
}

func calculateMaxFieldNameLength(typ reflect.Type) int {
	maxLen := 0
	for i := range typ.NumField() {
		if nameLen := len(typ.Field(i).Name); nameLen > maxLen {
			maxLen = nameLen
		}
	}
	return maxLen
}

func writeStructHeader(sb *strings.Builder, typ reflect.Type) {
	if name := typ.Name(); name != "" {
		fmt.Fprintf(sb, "type %s struct {\n", name)
	} else {
		fmt.Fprintln(sb, "struct {")
	}
}

func writeStructField(
	sb *strings.Builder,
	field reflect.StructField,
	pkgPath string,
	maxNameLen int,
	maxTypeLen int,
) {
	fieldName := field.Name
	fieldType := formatFieldType(field.Type.String(), pkgPath)
	fieldTag := formatFieldTag(field.Tag)

	namePadding := strings.Repeat(" ", maxNameLen-len(fieldName))
	typePadding := strings.Repeat(" ", maxTypeLen-len(fieldType))

	if fieldTag == "" {
		typePadding = ""
	}

	fmt.Fprintf(sb, "    %s%s %s%s%s\n", fieldName, namePadding, fieldType, typePadding, fieldTag)
}

func formatFieldType(fieldType, pkgPath string) string {
	if parsedType, err := parseGenericType(fieldType); err == nil {
		fieldType = parsedType
	}

	// Remove current package path from type names
	if pkgPath != "" {
		parts := strings.Split(pkgPath, "/")
		pkg := parts[len(parts)-1]
		fieldType = strings.ReplaceAll(fieldType, pkg+".", "")
	}

	return fieldType
}

func formatFieldTag(tag reflect.StructTag) string {
	if tag == "" {
		return ""
	}
	return fmt.Sprintf(" `%s`", string(tag))
}

var errMissingBrackets = errors.New("invalid generic type format: missing square brackets")

// parseGenericType takes an input string such as "Type[database/sql.Null[string]]"
// and returns "Type[sql.Null[string]]".
func parseGenericType(input string) (string, error) {
	openBracket := strings.Index(input, "[")
	closeBracket := strings.LastIndex(input, "]")

	if openBracket == -1 || closeBracket == -1 {
		return "", errMissingBrackets
	}

	// Extract the base type (before the square bracket)
	baseType := input[:openBracket]

	// Extract the inner type (between square brackets)
	innerType := input[openBracket+1 : closeBracket]
	isPtr := innerType[0] == '*'

	// Get the last part of the inner type (after the last slash if it exists)
	parts := strings.Split(innerType, "/")
	lastPart := parts[len(parts)-1]

	if isPtr {
		lastPart = "*" + lastPart
	}
	result := fmt.Sprintf("%s[%s]", baseType, lastPart)

	return result, nil
}

func shouldShowError(field reflect.StructField, err Err) bool {
	return err.FieldType == field.Type &&
		err.FieldName == field.Name &&
		err.Error != ""
}

func writeFieldError(
	sb *strings.Builder,
	field reflect.StructField,
	pkgPath string,
	err Err,
	colors Colors,
	maxNameLen int,
) {
	fieldName := field.Name
	fieldType := formatFieldType(field.Type.String(), pkgPath)
	padding := strings.Repeat(" ", maxNameLen-len(fieldName))

	underline, spacing := calculateErrorUnderline(fieldName, fieldType, padding, err)

	fmt.Fprintf(sb, "%s%s%s%s\n", spacing, colors.Error, underline, colors.Reset)
	fmt.Fprintf(sb, "%s%s|%s\n", spacing, colors.Error, colors.Reset)
	fmt.Fprintf(sb, "%s%s%s%s\n", spacing, colors.Error, err.Error, colors.Reset)
}
