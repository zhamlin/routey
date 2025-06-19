package stringz

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// SplitByCapitals slices s into all substrings separated by upper case letters and returns a slice of
// the substrings between those letters.
func SplitByCapitals(s string) []string {
	if s == "" {
		return nil
	}

	start := 0
	var result []string

	for i := 1; i < len(s); i++ {
		if unicode.IsUpper(rune(s[i])) && !unicode.IsUpper(rune(s[i-1])) {
			result = append(result, s[start:i])
			start = i
		}
	}

	result = append(result, s[start:])
	return result
}

// TrimLinesSpace removes spaces from each line in the provided s.
func TrimLinesSpace(s string) string {
	strLines := strings.Split(s, "\n")
	for i, line := range strLines {
		strLines[i] = strings.TrimSpace(line)
	}
	return strings.Join(strLines, "\n")
}

// ShowWhitespace makes whitespace characters visible.
func ShowWhitespace(s string) string {
	s = strings.ReplaceAll(s, " ", "·")
	s = strings.ReplaceAll(s, "\t", "→")
	s = strings.ReplaceAll(s, "\n", "↵\n")

	return s
}

// VisuallyNormalize converts tabs to spaces based on tab stops.
func VisuallyNormalize(s string, tabWidth int) string {
	var result strings.Builder
	column := 0

	for _, r := range s {
		switch r {
		case '\t':
			// Calculate spaces needed to reach next tab stop
			spaces := tabWidth - (column % tabWidth)
			result.WriteString(strings.Repeat(" ", spaces))
			column += spaces
		case '\n':
			result.WriteRune(r)
			column = 0
		default:
			result.WriteRune(r)
			column++
		}
	}

	return result.String()
}

// PrefixBorder adds the prefix to the beginning of every line in s.
func PrefixBorder(prefix, s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}

	return strings.Join(lines, "\n")
}

// CountLeadingWhitespace returns the number of leading whitespace characters.
func CountLeadingWhitespace(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// FormatText formats a multi-line string by:
// 1. Removing the initial newline if present
// 2. Adding the prefix to the first line
// 3. Preserving relative indentation based on leading whitespace
// 4. Preserving empty lines.
func FormatText(prefix, s string) string {
	if s == "" {
		return s
	}

	prefixLen := len(prefix)

	// Split into lines and remove initial empty line if present
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}

	// Find the base indentation level
	baseIndent := findBaseIndentation(lines)

	// Process each line
	formatted := make([]string, 0, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			formatted = append(formatted, "")
			continue
		}

		// Calculate relative indentation
		origIndent := CountLeadingWhitespace(line)
		relativeIndent := max(origIndent-baseIndent, 0)

		// Format the line
		if i == 0 {
			formatted = append(formatted, prefix+strings.TrimSpace(line))
		} else {
			indent := strings.Repeat(" ", prefixLen+relativeIndent)
			formatted = append(formatted, indent+strings.TrimSpace(line))
		}
	}

	return strings.Join(formatted, "\n")
}

// findBaseIndentation returns the minimum indentation level among non-empty lines.
func findBaseIndentation(lines []string) int {
	baseIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := CountLeadingWhitespace(line)
		if baseIndent == -1 || indent < baseIndent {
			baseIndent = indent
		}
	}
	return baseIndent
}

// stringer interface for types that implement String() method.
type stringer interface {
	String() string
}

// toString converts any value to a string representation.
func toString(value any) string {
	if value == nil {
		return "<nil>"
	}

	// Check if it's already a string
	if str, ok := value.(string); ok {
		return str
	}

	// Check if it implements String() method
	if stringer, ok := value.(stringer); ok {
		return stringer.String()
	}

	// Check if it's a pointer to something that implements String()
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		if stringer, ok := v.Interface().(stringer); ok {
			return stringer.String()
		}
	}

	// Fallback to fmt.Sprintf
	return fmt.Sprintf("%v", value)
}

type TableOptions struct {
	MinWidth    int
	Padding     int
	BorderStyle string
}

// CreateASCIITableWithOptions creates an ASCII table with configurable options.
func CreateASCIITableWithOptions[T any](columnName string, data []T, opts TableOptions) string {
	if len(data) == 0 {
		return ""
	}

	opts = setDefaults(opts)
	stringData := convertToStrings(data)
	maxWidth := calculateMaxWidth(columnName, stringData, opts.MinWidth)
	borders := getBorderChars(opts.BorderStyle)

	return buildTable(columnName, stringData, maxWidth, opts.Padding, borders)
}

func setDefaults(opts TableOptions) TableOptions {
	if opts.Padding == 0 {
		opts.Padding = 1
	}

	if opts.BorderStyle == "" {
		opts.BorderStyle = "ascii"
	}
	return opts
}

func convertToStrings[T any](data []T) []string {
	stringData := make([]string, len(data))
	for i, item := range data {
		stringData[i] = toString(item)
	}
	return stringData
}

func calculateMaxWidth(columnName string, stringData []string, minWidth int) int {
	maxWidth := len(columnName)
	for _, item := range stringData {
		if len(item) > maxWidth {
			maxWidth = len(item)
		}
	}

	if maxWidth < minWidth {
		maxWidth = minWidth
	}
	return maxWidth
}

type borderChars struct {
	horizontal, vertical, corner string
}

func getBorderChars(style string) borderChars {
	if style == "unicode" {
		return borderChars{"─", "│", "+"}
	}
	return borderChars{"-", "|", "+"}
}

func buildTable(
	columnName string,
	stringData []string,
	maxWidth, padding int,
	borders borderChars,
) string {
	var result strings.Builder
	totalWidth := maxWidth + (padding * 2)

	writeHorizontalBorder(&result, borders.corner, borders.horizontal, totalWidth)
	writeRow(&result, columnName, maxWidth, padding, borders.vertical)
	writeSeparator(&result, borders.vertical, borders.horizontal, totalWidth)

	for _, item := range stringData {
		writeRow(&result, item, maxWidth, padding, borders.vertical)
	}

	writeHorizontalBorder(&result, borders.corner, borders.horizontal, totalWidth)
	return strings.TrimSuffix(result.String(), "\n")
}

func writeHorizontalBorder(result *strings.Builder, corner, horizontal string, totalWidth int) {
	result.WriteString(corner + strings.Repeat(horizontal, totalWidth) + corner + "\n")
}

func writeRow(result *strings.Builder, content string, maxWidth, padding int, vertical string) {
	result.WriteString(vertical)
	result.WriteString(strings.Repeat(" ", padding))
	result.WriteString(content)
	result.WriteString(strings.Repeat(" ", maxWidth-len(content)+padding))
	result.WriteString(vertical + "\n")
}

func writeSeparator(result *strings.Builder, vertical, horizontal string, totalWidth int) {
	result.WriteString(vertical + strings.Repeat(horizontal, totalWidth) + vertical + "\n")
}
