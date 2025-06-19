package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/nsf/jsondiff"
	"github.com/zhamlin/routey/internal/stringz"
)

func argsToFormat(args ...any) (string, []any) {
	if len(args) > 0 {
		if msg, ok := args[0].(string); ok {
			return msg, args[1:]
		}
	}
	return "", nil
}

type logFn func(format string, args ...any)

func log(tb testing.TB, fn logFn, format string, args ...any) func(args ...any) {
	tb.Helper()

	msg, args := argsToFormat(args...)
	if msg == "" {
		return func(in ...any) {
			tb.Helper()
			fn(format, in...)
		}
	}

	return func(in ...any) {
		tb.Helper()
		fn(msg+"\n"+format, append(args, in...)...)
	}
}

func Equal[T comparable](tb testing.TB, got, want T, args ...any) {
	tb.Helper()

	if got != want {
		log(tb, tb.Errorf, "got: %v, wanted: %v", args...)(got, want)
	}
}

func NoError(tb testing.TB, err error, args ...any) {
	tb.Helper()

	if err != nil {
		log(tb, tb.Fatalf, "expected no error, got: %v", args...)(err)
	}
}

func WantError(tb testing.TB, err error, want any) {
	tb.Helper()

	if !errors.As(err, want) {
		tb.Fatalf("got: %T, wanted: %T error, ", err, want)
	}
}

func IsError(tb testing.TB, err, want error) {
	tb.Helper()

	if !errors.Is(err, want) {
		tb.Fatalf("got: %v, wanted: %q error", err, want)
	}
}

func mustMarshal(tb testing.TB, obj any) string {
	tb.Helper()

	switch val := obj.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	}

	val, err := json.MarshalIndent(&obj, "", " ")
	NoError(tb, err, "json.MarshalIndent")
	return string(val)
}

func jsonDiff(input, expected string) string {
	opts := jsondiff.DefaultConsoleOptions()
	diff, show := jsondiff.Compare([]byte(input), []byte(expected), &opts)

	if diff.String() != "FullMatch" {
		return fmt.Sprintf("%v:\n%v", diff, show)
	}
	return ""
}

func MatchAsJSON(tb testing.TB, got, want any, args ...any) {
	tb.Helper()

	gotStr := mustMarshal(tb, got)
	wantStr := mustMarshal(tb, want)

	if diff := jsonDiff(gotStr, wantStr); diff != "" {
		log(tb, tb.Errorf, "%T does not match %T\n%s", args...)(got, want, diff)
	}
}

func compareVisualAlignment(a, b string, tabWidth int) bool {
	normA := stringz.VisuallyNormalize(a, tabWidth)
	normB := stringz.VisuallyNormalize(b, tabWidth)
	return normA == normB
}

// printLineDiff shows line-by-line differences for multi-line strings.
func printLineDiff(tb testing.TB, s1, s2 string, tabWidth int) {
	tb.Helper()

	lines1 := strings.Split(s1, "\n")
	lines2 := strings.Split(s2, "\n")
	maxLines := max(len(lines2), len(lines1))

	for i := range maxLines {
		var line1, line2 string

		if i < len(lines1) {
			line1 = lines1[i]
		}

		if i < len(lines2) {
			line2 = lines2[i]
		}

		if !compareVisualAlignment(line1, line2, tabWidth) {
			tb.Logf("Line %d is different:\n", i+1)
			tb.Logf("  1: '%s'\n", stringz.ShowWhitespace(line1))
			tb.Logf("  2: '%s'\n", stringz.ShowWhitespace(line2))
		}
	}
}

func VisuallyMatch(tb testing.TB, got, want string, tabSize int) {
	tb.Helper()

	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)

	if !compareVisualAlignment(got, want, tabSize) {
		printLineDiff(tb, got, want, tabSize)
		tb.Errorf("got:\n%s\nwanted:\n%s", got, want)
	}
}

func WantAfterTest[T comparable](tb testing.TB, got, want T, args ...any) *T {
	tb.Helper()

	tb.Cleanup(func() {
		tb.Helper()

		if got != want {
			log(tb, tb.Errorf, "got: %v, wanted: %v", args...)(got, want)
		}
	})

	return &got
}
