package structs_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/zhamlin/routey/internal/structs"
	"github.com/zhamlin/routey/internal/test"
)

func compareErrors[T any](t *testing.T, err structs.Err, want string) {
	t.Helper()

	typ := reflect.TypeFor[T]()
	got := structs.PrintStructWithErr(typ, err, structs.NoErrorColors)

	const tabSize = 4
	test.VisuallyMatch(t, got, want, tabSize)
}

func TestPrintStructWithErr_CurrentPackage(t *testing.T) {
	//nolint:unused
	type object struct {
		field *object
	}

	want := `
type object struct {
	field *object
}
	`

	compareErrors[object](t, structs.Err{}, want)
}

func TestPrintStructWithErr_GenericField(t *testing.T) {
	//nolint:unused
	type object struct {
		field sql.Null[sql.Null[string]]
	}

	want := `
type object struct {
	field sql.Null[sql.Null[string]]
}
	`
	compareErrors[object](t, structs.Err{}, want)
}
