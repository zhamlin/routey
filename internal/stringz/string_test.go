package stringz_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zhamlin/routey/internal/stringz"
)

func TestSplitByCapitals(t *testing.T) {
	tests := []struct {
		have string
		want []string
	}{
		{
			have: "",
			want: []string(nil),
		},
		{
			have: "first",
			want: []string{"first"},
		},
		{
			have: "FirstSecond",
			want: []string{"First", "Second"},
		},
		{
			have: "FIRSTSecond",
			want: []string{"FIRSTSecond"},
		},
	}

	for _, test := range tests {
		got := stringz.SplitByCapitals(test.have)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestTrimSpaceStringLines(t *testing.T) {
	tests := []struct {
		have string
		want string
	}{
		{
			have: "  test   ",
			want: "test",
		},
		{
			have: "  line 1\n  line 2",
			want: "line 1\nline 2",
		},
	}

	for _, test := range tests {
		got := stringz.TrimLinesSpace(test.have)
		if got != test.want {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestShowWhitespace(t *testing.T) {
	tests := []struct {
		have string
		want string
	}{
		{
			have: " ",
			want: "·",
		},
		{
			have: "\n",
			want: "↵\n",
		},
		{
			have: "\t",
			want: "→",
		},
	}

	for _, test := range tests {
		got := stringz.ShowWhitespace(test.have)
		if got != test.want {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestVisuallyNormalize(t *testing.T) {
	tabWidth := 4
	tab := strings.Repeat(" ", tabWidth)

	tests := []struct {
		have string
		want string
	}{
		{
			have: "\tTest",
			want: tab + "Test",
		},
		{
			have: "\nTest",
			want: "\nTest",
		},
		{
			have: "\t\nTest",
			want: tab + "\nTest",
		},
	}

	for _, test := range tests {
		got := stringz.VisuallyNormalize(test.have, tabWidth)
		if got != test.want {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestCountLeadingWhitespace(t *testing.T) {
	tests := []struct {
		have string
		want int
	}{
		{
			have: "test",
			want: 0,
		},
		{
			have: "\ttest",
			want: 1,
		},
		{
			have: "  test",
			want: 2,
		},
	}

	for _, test := range tests {
		got := stringz.CountLeadingWhitespace(test.have)
		if got != test.want {
			t.Errorf("wanted: %v, got: %v", test.want, got)
		}
	}
}

func TestCreateTable(t *testing.T) {
	columns := []string{"a", "b"}
	opts := stringz.TableOptions{}

	got := stringz.CreateASCIITableWithOptions("name", columns, opts)
	want := strings.TrimSpace(`
+------+
| name |
|------|
| a    |
| b    |
+------+
	`)

	if got != want {
		t.Errorf("got:\n%v\nwanted:\n%v", got, want)
	}
}
