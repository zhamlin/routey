package param_test

import (
	"net/http"
	"testing"

	"github.com/zhamlin/routey/internal/test"
	"github.com/zhamlin/routey/param"
)

func TestOpts_ParseDefault(t *testing.T) {
	want := "default"
	opts := param.Opts{
		Parser:  param.ParseString,
		Default: want,
	}

	var got string
	err := opts.Parse(&got, []string{})
	test.NoError(t, err)

	if got != want {
		t.Errorf("expected param value: %v, got: %v", want, got)
	}
}

func TestOpts_ParseNoParams(t *testing.T) {
	opts := param.Opts{}
	var v any
	err := opts.Parse(&v, []string{})
	test.NoError(t, err)
}

func TestOpts_Parse(t *testing.T) {
	opts := param.Opts{
		Parser: func(any, []string) error {
			return nil
		},
	}

	var v any
	err := opts.Parse(&v, []string{""})
	test.NoError(t, err)
}

type testPather struct {
	value string
}

func (tp testPather) Param(string, *http.Request) string {
	return tp.value
}

func TestOpts_PathValue(t *testing.T) {
	want := "test"
	opts := param.Opts{
		Pather: testPather{value: want},
	}

	got := opts.PathValue("", nil)
	if got != want {
		t.Errorf("expected param value: %v, got: %v", want, got)
	}
}
